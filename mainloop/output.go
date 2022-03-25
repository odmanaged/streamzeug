/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package mainloop

import (
	"context"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/output"
)

type out struct {
	c        context.Context
	w        output.Output
	i        int
	m        *Mainloop
	dataChan chan *libristwrapper.RistDataBlock
}

func (m *Mainloop) addOutput(w output.Output, i int) {
	o := &out{
		m.ctx,
		w,
		i,
		m,
		make(chan *libristwrapper.RistDataBlock, 256),
	}
	go o.loop()
	m.outputs[i] = o
}

func (o *out) write(rb *libristwrapper.RistDataBlock) error {
	defer rb.Return()
	_, err := o.w.Write(rb)
	if err != nil {
		return err
	}
	return nil
}

func (o *out) loop() {
	for {
		select {
		case <-o.c.Done():
			return
		case rb := <-o.dataChan:
			err := o.write(rb)
			if err != nil {
				logging.Log.Error().Err(err).Msg("error writing to output")
				o.m.removeOutputByID(o.i)
				for rb := range o.dataChan {
					rb.Return()
				}
				return
			}
		}
	}
}

func (m *Mainloop) writeOutputs(rb *libristwrapper.RistDataBlock) {
	if len(rb.Data) == 0 {
		return
	}
	if len(m.outputs) == 0 {
		rb.Return()
		return
	}
	for _, out := range m.outputs {
		rb.Increment()
		select {
		case out.dataChan <- rb:
			//
		default:
			rb.Return()
		}
	}
	rb.Return()
}
