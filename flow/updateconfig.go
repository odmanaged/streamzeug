/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package flow

import (
	"reflect"
	"time"

	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/logging"
)

func (f *Flow) UpdateConfig(c *config.Flow) (err error) {
	f.configLock.Lock()
	shouldUnlock := true
	defer func() {
		if shouldUnlock {
			f.configLock.Unlock()
		}
	}()
	if reflect.DeepEqual(f.config, *c) {
		return nil
	}
	logging.Log.Info().Str("identifier", f.config.Identifier).Msg("updating flow config")
	defer func() {
		if err == nil {
			logging.Log.Info().Str("identifier", f.config.Identifier).Msg("done updating config")
			return
		}
		logging.Log.Error().Str("identifier", f.config.Identifier).Err(err).Msgf("error configuring: %s", err)
	}()

	if c.Latency != f.config.Latency || c.RistProfile != f.config.RistProfile || c.StreamID != f.config.StreamID {
		logging.Log.Info().Str("identifier", f.config.Identifier).Msg("rist settings changed, re-creating")
		f.Stop()
		f.Wait(5 * time.Millisecond)
		newflow, err := CreateFlow(f.rcontext, c)
		if err != nil {
			return err
		}
		shouldUnlock = false
		*f = *newflow
		return nil
	}

	if !reflect.DeepEqual(c.Inputs, f.config.Inputs) {
		checkDelete := make(map[string]int)
		for _, ic := range c.Inputs {
			checkDelete[ic.Url] = 1
		}

		for url, i := range f.configuredInputs {
			if _, ok := checkDelete[url]; !ok {
				i.Close()
				delete(f.configuredInputs, url)
			}
		}

		for _, ic := range c.Inputs {
			if _, ok := f.configuredInputs[ic.Url]; !ok {
				err := f.setupInput(&ic)
				if err != nil {
					return err
				}
			}
		}
	}
	f.config.Inputs = c.Inputs
	if reflect.DeepEqual(f.config, *c) {
		return nil
	}
	if !reflect.DeepEqual(c.Outputs, f.config.Outputs) {
		checkDelete := make(map[string]int)
		for _, oc := range c.Outputs {
			checkDelete[oc.Url] = 1
		}

		for url, oh := range f.configuredOutputs {
			if _, ok := checkDelete[url]; !ok {
				oh.out.Close()
				delete(f.configuredOutputs, url)
			}
		}

		for _, oc := range c.Outputs {
			if oh, ok := f.configuredOutputs[oc.Url]; !ok {
				err := f.setupOutput(&oc)
				if err != nil {
					return err
				}
			} else {
				if !reflect.DeepEqual(oh.conf, oc) {
					oh.out.Close()
					delete(f.configuredOutputs, oc.Url)
					err := f.setupOutput(&oc)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	f.config = *c
	return nil
}
