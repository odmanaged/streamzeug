/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"errors"
	"fmt"
	"math"
	"os"

	"code.videolan.org/rist/ristgo/libristwrapper"
)

type Flow struct {
	Identifier      string                     `yaml:"identifier"`
	InputType       string                     `yaml:"type"`
	RistProfile     libristwrapper.RistProfile `yaml:"ristprofile"`
	Latency         int                        `yaml:"latency"`
	StreamID        int                        `yaml:"streamid"`
	Inputs          []Input                    `yaml:"inputs"`
	Outputs         []Output                   `yaml:"outputs"`
	StatsStdOut     bool                       `yaml:"statsstdout"`
	StatsFile       string                     `yaml:"statsfile"`
	MinimalBitrate  int                        `yaml:"minimalbitrate"`
	MaxPacketTimeMS int                        `yaml:"maxpackettime"`
}

func ValidateFlowConfig(c *Flow) error {
	if len(c.Inputs) < 1 {
		return errors.New("at least 1 input required")
	}

	if err := checkDuplicates(c.Inputs); err != nil {
		return err
	}

	for _, i := range c.Inputs {
		if err := validateInputConfig(&i); err != nil {
			return fmt.Errorf("input validation failed: %w", err)
		}
	}
	if err := checkDuplicates(c.Outputs); err != nil {
		return err
	}

	for _, o := range c.Outputs {
		if err := validateOutputConfig(&o); err != nil {
			return fmt.Errorf("output validation failed: %w", err)
		}
	}

	if c.Identifier == "" {
		return errors.New("flow must have non-empty Identifier")
	}

	if c.InputType != "RIST" {
		return errors.New("only Type RIST supported atm")
	}

	if c.RistProfile > libristwrapper.RistProfileMain {
		return errors.New("invalid RistProfile")
	}

	if c.StatsFile != "" {
		if _, err := os.Stat(c.StatsFile); err != nil {
			return fmt.Errorf("statssfile: %s error: %w", c.StatsFile, err)
		}
	}

	if c.InputType == "RIST" {
		if c.StreamID > math.MaxUint16 {
			return fmt.Errorf("StreamID: %d must be smaller than: %d", c.StreamID, math.MaxUint16)
		}
	}

	if c.MaxPacketTimeMS > 0 && c.MinimalBitrate == 0 || c.MinimalBitrate > 0 && c.MaxPacketTimeMS == 0 {
		return errors.New("when using MaxpacketTime or MinimalBitrate both have to be set higher than 0")
	}
	return nil
}
