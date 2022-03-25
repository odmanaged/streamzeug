/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package flow

import (
	"context"
	"fmt"
	"sync"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/input"
	"github.com/odmedia/streamzeug/input/rist"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/mainloop"
	"github.com/odmedia/streamzeug/stats"
)

func CreateFlow(ctx context.Context, c *config.Flow) (*Flow, error) {
	var flow Flow
	var err error
	flow.rcontext = ctx
	flow.identifier = c.Identifier
	flow.outputWait = new(sync.WaitGroup)
	if err := config.ValidateFlowConfig(c); err != nil {
		return nil, fmt.Errorf("config validation failed %w", err)
	}
	flow.config = *c
	logging.Log.Info().Str("identifier", c.Identifier).Msg("setting up flow")
	flow.context, flow.cancel = context.WithCancel(ctx)

	flow.statsConfig, err = stats.SetupStats(c.StatsStdOut, c.Identifier, c.StatsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to setup stats %w", err)
	}

	if c.Latency == 0 {
		logging.Log.Info().Str("identifier", c.Identifier).Msg("setting latency to default of 1000ms")
		c.Latency = 1000
	}

	flow.receiver, err = rist.SetupReceiver(flow.context, c.Identifier, c.RistProfile, c.Latency, flow.statsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to setup rist receiver %w", err)
	}
	flow.configuredInputs = make(map[string]input.Input)
	for _, i := range c.Inputs {
		err = flow.setupInput(&i)
		if err != nil {
			return nil, fmt.Errorf("failed to setup input %s: %w", i, err)
		}
	}
	destinationPort := uint16(0)
	if c.RistProfile != libristwrapper.RistProfileSimple {
		destinationPort = uint16(c.StreamID)
	}
	err = flow.receiver.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start rist receiver %w", err)
	}
	rf, err := flow.receiver.ConfigureFlow(destinationPort)
	if err != nil {
		return nil, fmt.Errorf("failed to configure rist flow %w", err)
	}

	m := mainloop.NewMainloop(flow.context, rf, c.Identifier)
	flow.m = m

	flow.configuredOutputs = make(map[string]outhandle)
	for _, o := range c.Outputs {
		err := flow.setupOutput(&o)
		if err != nil {
			return nil, fmt.Errorf("failed to setup output %s: %w", o, err)
		}
	}
	return &flow, nil
}
