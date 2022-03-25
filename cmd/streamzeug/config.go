/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"context"
	"reflect"
	"time"

	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/flow"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/stats"
)

func createFlow(ctx context.Context, f *config.Flow) error {
	flow, err := flow.CreateFlow(ctx, f)
	if err != nil {
		return err
	}
	flows[f.Identifier] = &flowhandle{
		f: flow,
	}
	return nil
}

func applyConfig(ctx context.Context, c *config.Config) error {
	var (
		influxctx context.Context
		err       error
	)

	influxctx, influxcancel = context.WithCancel(ctx)
	if c.InfluxDB != nil {
		if err := stats.SetupInfluxDB(influxctx, c.InfluxDB, c.Identifier); err != nil {
			return err
		}
	}
	if c.ListenHTTP != "" {
		httpsrv, err = startHttpServer(c.ListenHTTP)
		if err != nil {
			return err
		}
	}
	flowsLock.Lock()
	defer flowsLock.Unlock()
	for _, f := range c.Flows {
		err := createFlow(ctx, &f)
		if err != nil {
			return err
		}
	}
	configLock.Lock()
	runningConfig = c
	configLock.Unlock()
	return nil
}

func reloadConfigfile(ctx context.Context) {
	conf, err := config.LoadFromFile(configFile)
	if err != nil {
		logging.Log.Error().Err(err).Msg("failed to read configfile")
		return
	}

	if err := config.ValidateConfig(conf); err != nil {
		logging.Log.Error().Err(err).Msgf("failed to validate config file, not reloading: %s", err)
		return
	}

	configLock.Lock()
	defer configLock.Unlock()

	if reflect.DeepEqual(runningConfig, conf) {
		logging.Log.Info().Msg("config unchanged")
		return
	}

	if !reflect.DeepEqual(runningConfig.InfluxDB, conf.InfluxDB) {
		influxcancel()
		var influxctx context.Context
		influxctx, influxcancel = context.WithCancel(ctx)
		if conf.InfluxDB != nil {
			if err := stats.SetupInfluxDB(influxctx, conf.InfluxDB, conf.Identifier); err != nil {
				logging.Log.Error().Err(err).Msg("failed to reconfigure influxdb")
				return
			}
		} else {
			stats.InfluxDisable()
		}
	}

	if runningConfig.ListenHTTP != conf.ListenHTTP {
		if httpsrv != nil {
			shutdownctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
			defer cancel()
			if err := httpsrv.Shutdown(shutdownctx); err != nil {
				logging.Log.Error().Err(err).Msg("error stopping webserver")
				return
			}
			httpsrv = nil
		}
		if conf.ListenHTTP != "" {
			httpsrv, err = startHttpServer(conf.ListenHTTP)
			if err != nil {
				logging.Log.Error().Err(err).Msg("failed to start webserv")
				return
			}
		}
	}

	flowsLock.Lock()
	defer flowsLock.Unlock()

	checkDelete := make(map[string]int)

	for _, fc := range conf.Flows {
		checkDelete[fc.Identifier] = 1
	}

	//delete first, as flow might use same inputs/outputs
	for i, fh := range flows {
		if _, ok := checkDelete[i]; !ok {
			fh.f.Stop()
			fh.f.Wait(500 * time.Millisecond)
			delete(flows, i)
		}
	}

	for _, fc := range conf.Flows {
		if fh, ok := flows[fc.Identifier]; ok {
			if err := fh.f.UpdateConfig(&fc); err != nil {
				logging.Log.Error().Err(err).Msg("error updatinf flow config")
				return
			}
		} else {
			err := createFlow(ctx, &fc)
			if err != nil {
				logging.Log.Error().Err(err).Msgf("couldn't create flow %s: %s", fc.Identifier, err)
				return
			}
		}
	}

	runningConfig = conf
}
