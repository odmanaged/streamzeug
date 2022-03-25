/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/flow"
	"github.com/odmedia/streamzeug/logging"
)

type arrayFlags []string

type flowhandle struct {
	f *flow.Flow
}

var (
	influxcancel  context.CancelFunc
	configFile    string
	configLock    sync.Mutex
	runningConfig *config.Config
	flowsLock     sync.Mutex
	flows         map[string]*flowhandle
	httpsrv       *http.Server
)

func init() {
	flows = make(map[string]*flowhandle)
}

func SignalHandler(ctx context.Context, cancel context.CancelFunc) {
	signalChan := make(chan os.Signal, 1)
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	if configFile != "" {
		signals = append(signals, syscall.SIGHUP)
	}
	signal.Notify(signalChan, signals...)

	go func() {
		for {
			select {
			case s := <-signalChan:
				if s == syscall.SIGTERM || s == syscall.SIGINT {
					logging.Log.Info().Msg("received termination signal, shutting down")
					cancel()
					return
				}
				if s == syscall.SIGHUP {
					logging.Log.Info().Msg("got SIGHUP, reloading config")
					go reloadConfigfile(ctx)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func main() {

	c := context.Background()
	ctx, cancel := context.WithCancel(c)

	conf, err := parseArguments()
	if err != nil {
		logging.Log.Error().Err(err).Msgf("failed to configure application %s", err)
		os.Exit(1)
	}

	if err := applyConfig(ctx, conf); err != nil {
		logging.Log.Error().Err(err).Msgf("failed to configure application %s", err)
		os.Exit(1)
	}

	SignalHandler(ctx, cancel)

	<-ctx.Done()

	var wg sync.WaitGroup
	for _, flow := range flows {
		flow.f.Stop()
	}

	for _, flow := range flows {
		wg.Add(1)
		go func(f *flowhandle) {
			f.f.Wait(1 * time.Second)
			wg.Done()
		}(flow)
	}
	wg.Wait()
}
