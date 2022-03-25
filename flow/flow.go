/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package flow

import (
	"context"
	"sync"
	"time"

	"code.videolan.org/rist/ristgo"
	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/input"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/mainloop"
	"github.com/odmedia/streamzeug/stats"
)

type Flow struct {
	rcontext          context.Context
	context           context.Context
	cancel            context.CancelFunc
	receiver          ristgo.Receiver
	configuredOutputs map[string]outhandle
	configLock        sync.Mutex
	config            config.Flow
	configuredInputs  map[string]input.Input
	m                 *mainloop.Mainloop
	outputWait        *sync.WaitGroup
	statsConfig       *stats.Stats
	identifier        string
}

func (f *Flow) Status() *mainloop.Status {
	mlStatus := f.m.Status()
	f.configLock.Lock()
	defer f.configLock.Unlock()
	if f.config.MinimalBitrate > 0 && f.config.MaxPacketTimeMS > 0 {
		if mlStatus.Bitrate < f.config.MinimalBitrate || mlStatus.MsSinceLastPacket > f.config.MaxPacketTimeMS {
			mlStatus.Status = "NOT-OK"
			mlStatus.OK = false
		}
	}
	return mlStatus
}

func (f *Flow) Stop() {
	f.cancel()
	for _, o := range f.configuredOutputs {
		o.out.Close()
	}
}

func (f *Flow) Wait(timeout time.Duration) {
	f.m.Wait(timeout)
	c := make(chan bool)
	go func() {
		f.receiver.Destroy()
		f.outputWait.Wait()
		c <- true
	}()
	select {
	case <-c:
		logging.Log.Info().Str("identifier", f.identifier).Msg("cleanup complete")
		return
	case <-time.After(timeout):
		logging.Log.Error().Str("identifier", f.identifier).Msg("cleanup timeout")
		return
	}
}
