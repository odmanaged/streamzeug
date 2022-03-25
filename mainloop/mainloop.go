/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package mainloop

import (
	"context"
	"sync"
	"time"

	"code.videolan.org/rist/ristgo"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/output"
	"github.com/rs/zerolog"
)

type inputstatus struct {
	packetcount        int
	packetcountsince   int
	bytesSince         int
	discontinuitycount int
	lastPacketTime     time.Time
}

type Mainloop struct {
	ctx                context.Context
	flow               ristgo.ReceiverFlow
	logger             zerolog.Logger
	outputs            map[int]*out
	outPutAdd          chan output.Output
	outPutRemove       chan output.Output
	outRemoveIdx       chan int
	wg                 sync.WaitGroup
	statusLock         sync.Mutex
	primaryInputStatus inputstatus
	lastStatusCall     time.Time
}

func (m *Mainloop) removeOutputByID(idx int) {
	select {
	case <-m.ctx.Done():
		return
	default:
		//
	}
	m.outRemoveIdx <- idx
}

func (m *Mainloop) RemoveOutput(output output.Output) {
	select {
	case <-m.ctx.Done():
		return
	default:
		//
	}
	m.outPutRemove <- output
}

func (m *Mainloop) deleteOutput(idx int, output output.Output) {
	m.logger.Info().Msgf("deleting output: %s", output.String())
	close(m.outputs[idx].dataChan)
	delete(m.outputs, idx)
}

func (m *Mainloop) AddOutput(output output.Output) {
	m.logger.Info().Msgf("adding output %s", output.String())
	select {
	case <-m.ctx.Done():
		return
	default:
		//
	}
	m.outPutAdd <- output
}

func (m *Mainloop) Wait(timeout time.Duration) {
	c := make(chan bool)
	go func() {
		m.wg.Wait()
		c <- true
	}()
	select {
	case <-c:
		return
	case <-time.After(timeout):
		return
	}
}

func NewMainloop(ctx context.Context, flow ristgo.ReceiverFlow, identifier string) *Mainloop {
	m := &Mainloop{
		ctx:          ctx,
		flow:         flow,
		logger:       logging.Log.With().Str("identifier", identifier).Logger(),
		outputs:      make(map[int]*out),
		outPutAdd:    make(chan output.Output, 4),
		outPutRemove: make(chan output.Output, 4),
		outRemoveIdx: make(chan int, 16),
	}
	go receiveLoop(m)
	return m
}

func receiveLoop(m *Mainloop) {
	outputidx := 0
	m.primaryInputStatus.lastPacketTime = time.Now()
	m.lastStatusCall = m.primaryInputStatus.lastPacketTime
	expectedSec := uint16(0)
	m.logger.Info().Msg("receiver mainloop started")
	m.wg.Add(1)
	lastDiscontinuityMsg := time.Time{}
	discontinuitiesSinceLastMsg := int(0)
main:
	for {
		select {
		case <-m.ctx.Done():
			break main

		case rb, ok := <-m.flow.DataChannel():
			if !ok {
				break main
			}
			discontinuity := false
			if rb.Discontinuity {
				discontinuity = true
			}
			if rb.SeqNo != uint32(expectedSec) {
				discontinuity = true
			}
			if discontinuity {
				m.primaryInputStatus.discontinuitycount++
				discontinuitiesSinceLastMsg++
			}

			if discontinuitiesSinceLastMsg > 0 && time.Since(lastDiscontinuityMsg) >= time.Duration(5)*time.Second {
				m.logger.Error().Int("count", discontinuitiesSinceLastMsg).Msg("discontinuity!")
				lastDiscontinuityMsg = time.Now()
				discontinuitiesSinceLastMsg = 0
			}
			expectedSec = uint16(rb.SeqNo) + 1
			m.statusLock.Lock()
			m.primaryInputStatus.packetcount++
			m.primaryInputStatus.packetcountsince++
			m.primaryInputStatus.lastPacketTime = time.Now()
			m.primaryInputStatus.bytesSince += len(rb.Data)
			m.statusLock.Unlock()
			m.writeOutputs(rb)
		case output := <-m.outPutAdd:
			m.statusLock.Lock()
			m.addOutput(output, outputidx)
			outputidx++
			m.statusLock.Unlock()
		case idx := <-m.outRemoveIdx:
			m.statusLock.Lock()
			output, ok := m.outputs[idx]
			if ok {
				m.deleteOutput(idx, output.w)
			} else {
				m.logger.Error().Msgf("couldn't delete output at index: %d, notfound", idx)
			}
			m.statusLock.Unlock()
		case output := <-m.outPutRemove:
			found := false
			m.statusLock.Lock()
			for idx, o := range m.outputs {
				if o.w == output {
					found = true
					delete(m.outputs, idx)
					break
				}
			}
			m.statusLock.Unlock()
			if !found {
				m.logger.Error().Msgf("couldn't delete output: %s, notfound", output.String())
			}
		}
	}
	close(m.outPutAdd)
	close(m.outPutRemove)
	close(m.outRemoveIdx)
	m.logger.Info().Msg("mainloop terminated")
	m.wg.Done()
}
