/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package flow

import (
	"fmt"
	"net/url"

	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/output"
	"github.com/odmedia/streamzeug/output/dektecasi"
	"github.com/odmedia/streamzeug/output/srt"
	"github.com/odmedia/streamzeug/output/udp"
)

type outhandle struct {
	out  output.Output
	conf config.Output
}

func (f *Flow) setupOutput(c *config.Output) (err error) {
	var (
		outputurl *url.URL
		out       output.Output
	)
	outputurl, err = url.Parse(c.Url)
	if err != nil {
		return fmt.Errorf("couldn't parse output url %s: %w", c.Url, err)
	}
	switch outputurl.Scheme {
	case "udp", "rtp":
		out, err = udp.ParseUdpOutput(f.context, outputurl, f.identifier, f.m)
	case "srt":
		out, err = srt.ParseSrtOutput(f.context, outputurl, f.identifier, c.Identifier, f.m, f.statsConfig, f.outputWait)
	case "dektecasi":
		out, err = dektecasi.ParseURL(f.context, outputurl, f.identifier, c.Identifier, f.m, f.statsConfig)
	default:
		return fmt.Errorf("output url scheme: %s not implemented", outputurl.Scheme)
	}
	if err != nil {
		return fmt.Errorf("couldn't setup %s output: %s: %w", outputurl.Scheme, outputurl, err)
	}
	f.configuredOutputs[c.Url] = outhandle{out: out, conf: *c}
	return nil
}
