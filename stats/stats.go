/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package stats

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/haivision/srtgo"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/output/dektecasi/dtstats"
)

var (
	StatsIntervalSeconds = 10
)

type Stats struct {
	stdout     bool
	identifier string
	statsFile  *rotatelogs.RotateLogs
}

func SetupStats(stdout bool, identifier, filename string) (*Stats, error) {
	logging.Log.Info().Str("identifier", identifier).Msg("setting up stats for")
	var stats Stats
	var err error
	if filename != "" {
		stats.statsFile, err = setupStatsFile(filename)
		if err != nil {
			return nil, err
		}
	}
	stats.stdout = stdout
	stats.identifier = identifier
	return &stats, nil
}

func setupStatsFile(file string) (*rotatelogs.RotateLogs, error) {
	logging.Log.Info().Msgf("setting up stats file: %s", file)
	var err error
	statsFilePattern := file + ".%Y%m%d"
	statsFile, err := rotatelogs.New(
		statsFilePattern,
		rotatelogs.WithClock(rotatelogs.Local),
		rotatelogs.WithLinkName(file),
	)
	if err != nil {
		return nil, err
	}
	return statsFile, nil
}

type statsPrepend struct {
	Timestamp string
	Type      string
	Host      string
}

type wrappedSrtStats struct {
	*statsPrepend
	*srtgo.SrtStats
}

type wrappedRistReceiverFlowStats struct {
	*statsPrepend
	*libristwrapper.ReceiverFlowStats
}

type wrappedRistSenderStats struct {
	*statsPrepend
	*libristwrapper.SenderPeerStats
}

type wrappedDektecAsiStats struct {
	*statsPrepend
	*dtstats.DektecAsiStats
}

func (s *Stats) HandleStats(Host, identifier string, u *url.URL, stats interface{}) {
	now := time.Now()
	prepend := &statsPrepend{now.Format("2006-01-02T15:04:05-0700"), "", Host}
	if s.stdout || s.statsFile != nil {
		var wrappedStats interface{}
		switch v := stats.(type) {
		case *srtgo.SrtStats:
			prepend.Type = "SrtStats"
			wrappedStats = &wrappedSrtStats{prepend, v}
		case *libristwrapper.ReceiverFlowStats:
			prepend.Type = "RistReceiverStats"
			wrappedStats = &wrappedRistReceiverFlowStats{prepend, v}
		case *libristwrapper.SenderPeerStats:
			prepend.Type = "RistSenderStats"
			wrappedStats = &wrappedRistSenderStats{prepend, v}
		case *dtstats.DektecAsiStats:
			prepend.Type = "DektecAsiStats"
			wrappedStats = &wrappedDektecAsiStats{prepend, v}
		default:
			panic("unhandled stats")
		}
		statsJson, err := json.Marshal(wrappedStats)
		statsString := string(statsJson) + "\n"
		if err != nil {
			panic(err)
		}
		if s.stdout {
			fmt.Print(statsString)
		}
		if s.statsFile != nil {
			if _, err := s.statsFile.Write([]byte(statsString)); err != nil {
				logging.Log.Error().Str("module", "streamzeug-stats").Err(err).Msgf("error writing to stats file", s.statsFile)
			}
		}
	}

	if influxDBWriteApi != nil {
		s.writeInfluxStats(Host, identifier, u, stats)
	}
}
