/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package rist

import (
	"context"
	"net/url"
	"strings"

	"github.com/odmedia/streamzeug/input"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/stats"

	"code.videolan.org/rist/ristgo"
	"code.videolan.org/rist/ristgo/libristwrapper"
)

func init() {
	logger := logging.Log.With().Str("module", "rist-input").Str("identifier", "global-log").Logger()
	globalLogCB := func(loglevel libristwrapper.RistLogLevel, logmessage string) {
		logmessage = strings.TrimSuffix(logmessage, "\n")
		switch loglevel {
		case libristwrapper.LogLevelError:
			logger.Error().Msg(logmessage)
		case libristwrapper.LogLevelWarn:
			logger.Warn().Msg(logmessage)
		case libristwrapper.LogLevelNotice:
			logger.Info().Msg(logmessage)
		case libristwrapper.LogLevelInfo:
			logger.Info().Msg(logmessage)
		case libristwrapper.LogLevelDebug:
			logger.Debug().Msg(logmessage)
		}
	}
	if err := libristwrapper.SetGlobalLoggingCB(globalLogCB); err != nil {
		logging.Log.Panic().Err(err)
	}
}

type ristinput struct {
	r ristgo.Receiver
	p int
}

func createStatsCB(s *stats.Stats) libristwrapper.StatsCallbackFunc {
	return func(stats *libristwrapper.StatsContainer) {
		if stats.ReceiverFlowStats != nil {
			s.HandleStats("", "", nil, stats.ReceiverFlowStats)
		} else if stats.SenderStats != nil {
			s.HandleStats("", "", nil, stats.SenderStats)
		}
	}
}

func createLogCB(indentifier string) libristwrapper.LogCallbackFunc {
	logger := logging.Log.With().Str("module", "rist-input").Str("identifier", indentifier).Logger()
	return func(loglevel libristwrapper.RistLogLevel, logmessage string) {
		logmessage = strings.TrimSuffix(logmessage, "\n")
		switch loglevel {
		case libristwrapper.LogLevelError:
			logger.Error().Msg(logmessage)
		case libristwrapper.LogLevelWarn:
			logger.Warn().Msg(logmessage)
		case libristwrapper.LogLevelNotice:
			logger.Info().Msg(logmessage)
		case libristwrapper.LogLevelInfo:
			logger.Info().Msg(logmessage)
		case libristwrapper.LogLevelDebug:
			logger.Debug().Msg(logmessage)
		}
	}
}

func SetupReceiver(ctx context.Context, identifier string, profile libristwrapper.RistProfile, recoverysize int, s *stats.Stats) (ristgo.Receiver, error) {
	logging.Log.Info().Str("identifier", identifier).Msg("starting rist receiver")
	return ristgo.ReceiverCreate(ctx, &ristgo.ReceiverConfig{
		RistProfile:             profile,
		LoggingCallbackFunction: createLogCB(identifier),
		StatsCallbackFunction:   createStatsCB(s),
		StatsInterval:           stats.StatsIntervalSeconds * 1000,
		RecoveryBufferSize:      recoverysize,
	})
}

func SetupRistInput(u *url.URL, identifier string, r ristgo.Receiver) (input.Input, error) {
	logging.Log.Info().Str("identifier", identifier).Msgf("setting up RIST input: %s", u.String())
	peerConfig, err := ristgo.ParseRistURL(u)
	if err != nil {
		return nil, err
	}
	id, err := r.AddPeer(peerConfig)
	if err != nil {
		return nil, err
	}
	return &ristinput{
		r,
		id,
	}, nil
}

func (i *ristinput) Close() {
	if err := i.r.RemovePeer(i.p); err != nil {
		logging.Log.Error().Err(err).Msg("error removing rist peer")
	}
}
