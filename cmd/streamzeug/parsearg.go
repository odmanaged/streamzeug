/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/stats"
	"github.com/odmedia/streamzeug/version"
)

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, strings.TrimSpace(value))
	return nil
}

func parseArguments() (*config.Config, error) {
	var (
		inputs, outputs                        arrayFlags
		ristRecoverySize, statsIntervalSeconds int
		statsFile                              string
		influxDBUrl                            string
		influxDBToken                          string
		influxDBOrg                            string
		influxDBBucket                         string
		influxDBIdentifier                     string
		configTest                             bool
		statsStdOut                            bool
		conf                                   *config.Config
		showVersion                            bool
	)
	flag.StringVar(&configFile, "configfile", "", "config file")
	flag.BoolVar(&configTest, "configtest", false, "don't load config, just validate it")
	flag.Var(&inputs, "input", "input url, multiple instances of -input may be defined, with a minimum of 1")
	flag.Var(&outputs, "output", "output url, multiple instances of -output may be defined, with a minimum of 1")
	flag.IntVar(&ristRecoverySize, "rist-recoverysize", 1000, "recovery buffer size in ms")
	flag.IntVar(&statsIntervalSeconds, "stats-interval", 10, "stats reporting interval in seconds")
	flag.StringVar(&statsFile, "stats-file", "", "base name for stats file")
	flag.StringVar(&influxDBUrl, "influxdb-url", "", "url of influxdb server to write stats to")
	flag.StringVar(&influxDBToken, "influxdb-token", "", "influxdb token")
	flag.StringVar(&influxDBOrg, "influxdb-org", "", "influxdb org")
	flag.StringVar(&influxDBBucket, "influxdb-bucket", "", "influxdb bucket")
	flag.StringVar(&influxDBIdentifier, "influxdb-identifier", "DEFAULTID", "identifier used in influxdb as identifier tag")
	flag.BoolVar(&statsStdOut, "stats-stdout", false, "print stats to stdout")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.Parse()

	if showVersion {
		fmt.Printf("Streamzeug: %s git: %s\n", version.ProjectVersion, version.GitVersion)
		os.Exit(0)
	}
	if configTest && configFile == "" {
		return nil, errors.New("cannot test config without configfile")
	}

	stats.StatsIntervalSeconds = statsIntervalSeconds
	if configFile == "" {
		if len(inputs) < 1 || len(outputs) < 1 {
			fmt.Println("ERROR: at least one input and one output need to be defined when using CLI arg")
			flag.PrintDefaults()
			os.Exit(1)
		}
		conf = new(config.Config)

		conf.Identifier = influxDBIdentifier
		if influxDBUrl != "" {
			conf.InfluxDB = &config.InfluxDBConfig{
				Url:    influxDBUrl,
				Token:  influxDBToken,
				Bucket: influxDBBucket,
				Org:    influxDBOrg,
			}
		}

		flowConf := config.Flow{
			Identifier:  influxDBIdentifier,
			InputType:   "RIST",
			RistProfile: libristwrapper.RistProfileSimple,
			Latency:     ristRecoverySize,
			StreamID:    0,
			StatsStdOut: statsStdOut,
			StatsFile:   statsFile,
		}

		for _, input := range inputs {
			flowConf.Inputs = append(flowConf.Inputs, config.Input{Url: input})
		}

		for _, output := range outputs {
			u, err := url.Parse(output)
			if err != nil {
				logging.Log.Error().Err(err)
			}
			identifier := u.Query().Get("identifier")
			flowConf.Outputs = append(flowConf.Outputs, config.Output{
				Url:        output,
				Identifier: identifier,
			})
		}
		conf.Flows = append(conf.Flows, flowConf)
	} else {
		var err error
		conf, err = config.LoadFromFile(configFile)
		if err != nil {
			return nil, err
		}
	}

	if conf == nil {
		return nil, errors.New("conf is nil")
	}
	if err := config.ValidateConfig(conf); err != nil {
		return nil, err
	}
	if configTest {
		logging.Log.Info().Msg("config OK")
		os.Exit(0)
	}
	return conf, nil
}
