/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package srt

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/haivision/srtgo"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/mainloop"
	"github.com/odmedia/streamzeug/output"
	"github.com/odmedia/streamzeug/stats"
	"github.com/rs/zerolog"
)

var (
	logger zerolog.Logger
)

func init() {
	logger = logging.Log.With().Str("module", "srt-input").Logger()
	srtgo.InitSRT()
	srtgo.SrtSetLogLevel(srtgo.SrtLogLevelNotice)
	srtgo.SrtSetLogHandler(srtLogCB)
}

type srtoutput struct {
	ctx               context.Context
	cancel            context.CancelFunc
	srt               *srtgo.SrtSocket
	host              string
	timeout           int
	identifier        string
	output_identifier string
	Url               *url.URL
	SanitisedURL      *url.URL
	m                 *mainloop.Mainloop
	stats             *stats.Stats
	wg                *sync.WaitGroup
	parent            *srtoutput
	index             int
	clientsLock       *sync.Mutex
	clients           map[int]*srtoutput
}

func (s *srtoutput) String() string {
	if s.srt.Mode() == srtgo.ModeCaller {
		return "srt: " + s.SanitisedURL.String()
	}
	return "srt: " + s.host + "@" + s.SanitisedURL.String()
}

func (s *srtoutput) Count() int {
	if s.srt.Mode() == srtgo.ModeCaller {
		return 1
	}
	return len(s.clients)
}

func (s *srtoutput) Write(block *libristwrapper.RistDataBlock) (n int, e error) {
	n, e = s.srt.Write(block.Data)
	if e != nil {
		if s.srt.Mode() == srtgo.ModeFailure {
			logger.Info().Str("identifier", s.identifier).Str("output_identifier", s.output_identifier).Str("srt-url", s.SanitisedURL.String()).Str("client", s.host).Msgf("SRT client %s disconnected", s.host)
			s.parent.clientsLock.Lock()
			delete(s.parent.clients, s.index)
			s.parent.clientsLock.Unlock()
		} else if s.srt.Mode() == srtgo.ModeCaller {
			logger.Info().Str("identifier", s.identifier).Str("output_identifier", s.output_identifier).Str("srt-url", s.SanitisedURL.String()).Str("client", s.host).Msgf("Lost connection to SRT server: %s", s.host)
			s.srt.Close()
			go s.reconnect()
		}
	}
	return
}

func (s *srtoutput) Close() error {
	s.cancel()
	if s.srt != nil {
		s.srt.Close()
	}
	return nil
}

func (s *srtoutput) listenAccept() {
	s.clients = make(map[int]*srtoutput, 5)
	s.clientsLock = new(sync.Mutex)
	clientIndex := 0
	for {
		srtSocket, u, err := s.srt.Accept()
		if err != nil {
			logger.Error().Str("identifier", s.identifier).Str("output_identifier", s.output_identifier).Str("srt-url", s.SanitisedURL.String()).Err(err).Msg("error in srtsocket listen")
			break
		}
		srtoutput := *s
		srtoutput.srt = srtSocket
		srtoutput.parent = s
		srtoutput.host = u.IP.String()
		srtoutput.index = clientIndex
		s.clientsLock.Lock()
		s.clients[clientIndex] = &srtoutput
		s.clientsLock.Unlock()
		clientIndex++
		s.m.AddOutput(&srtoutput)
		go srtoutput.statsLoop()
	}

	s.clientsLock.Lock()
	for _, o := range s.clients {
		o.Close()
	}
	s.clientsLock.Unlock()
	s.wg.Done()
}

func (s *srtoutput) statsLoop() {
	for {
		time.Sleep(time.Duration(stats.StatsIntervalSeconds) * time.Second)
		stats, err := s.srt.Stats()
		if err != nil {
			if errors.Is(err, srtgo.SRTErrno(srtgo.ENoConn)) || errors.Is(err, srtgo.SRTErrno(srtgo.EInvSock)) {
				break
			}
			logger.Error().Str("identifier", s.identifier).Str("output_identifier", s.output_identifier).Str("srt-url", s.SanitisedURL.String()).Err(err).Msg("error in srt statsloop")
			break
		}
		go s.stats.HandleStats(s.host, s.output_identifier, s.Url, stats)
	}
}

func (s *srtoutput) reconnect() {
	select {
	case <-s.ctx.Done():
		return
	default:
		//
	}
	if err := setupSrtSocket(s); err != nil {
		logging.Log.Error().Err(err).Msg("this should be impossible in reconnect loop")
	}
}

func setupSrtSocket(s *srtoutput) error {
	host := s.Url.Hostname()
	if host == "" {
		host = "0.0.0.0"
	}
	s.host = host
	port, err := strconv.Atoi(s.Url.Port())
	if err != nil {
		return err
	}
	options := make(map[string]string)
	for key := range s.Url.Query() {
		options[key] = s.Url.Query().Get(key)
	}
	delete(options, "identifier")
	options["blocking"] = "0"
	options["transtype"] = "live"
	if host == "0.0.0.0" {
		options["mode"] = "listener"
	}
	srtSocket := srtgo.NewSrtSocket(host, uint16(port), options)
	if srtSocket == nil {
		return errors.New("got nil srtSocket")
	}
	s.srt = srtSocket
	if srtSocket.Mode() == srtgo.ModeListener {
		if err := srtSocket.Listen(5); err != nil {
			return err
		}
		s.wg.Add(1)
		go s.listenAccept()
	} else {
		if err := srtSocket.Connect(); err != nil {
			if _, ok := err.(*srtgo.SrtSocketClosed); ok {
				srtSocket.Close()
				go s.reconnect()
				return nil
			}
			return err
		}
		logger.Info().Str("output_identifier", s.identifier).Str("srt-url", s.Url.String()).Str("client", s.host).Msgf("SRT Connected to: %s", s.host)
		go s.statsLoop()
		s.m.AddOutput(s)
	}
	return nil
}

func ParseSrtOutput(ctx context.Context, u *url.URL, identifier, output_identifier string, m *mainloop.Mainloop, stats *stats.Stats, wait *sync.WaitGroup) (output.Output, error) {
	context, cancel := context.WithCancel(ctx)
	var srtout srtoutput
	srtout.Url = u
	srtout.SanitisedURL = u
	if u.Query().Get("passphrase") != "" {
		sanitised, _ := url.Parse(u.String())
		q := sanitised.Query()
		q.Set("passphrase", "REDACTED")
		sanitised.RawQuery = q.Encode()
		srtout.SanitisedURL = sanitised
	}
	logging.Log.Info().Str("identifier", identifier).Msgf("setting up srt output: %s", srtout.SanitisedURL)
	srtout.identifier = identifier
	srtout.output_identifier = output_identifier
	srtout.ctx = context
	srtout.cancel = cancel
	srtout.timeout = 0
	srtout.m = m
	srtout.stats = stats
	srtout.wg = wait
	err := setupSrtSocket(&srtout)
	if err != nil {
		return nil, err
	}
	return &srtout, nil
}

func srtLogCB(level srtgo.SrtLogLevel, file string, line int, area, message string) {
	//this strips the start of the SRT log message, which we don't need
	index := strings.Index(message, "c:")
	if index > 0 {
		index += 3
		message = message[index:]
	}
	message = strings.TrimSuffix(message, "\n")
	switch level {
	case srtgo.SrtLogLevelCrit:
		logger.Panic().Str("file", file).Int("line", line).Str("area", area).Msg(message)
	case srtgo.SrtLogLevelErr:
		logger.Error().Str("file", file).Int("line", line).Str("area", area).Msg(message)
	case srtgo.SrtLogLevelWarning:
		logger.Warn().Str("file", file).Int("line", line).Str("area", area).Msg(message)
	case srtgo.SrtLogLevelNotice:
		logger.Info().Str("file", file).Int("line", line).Str("area", area).Msg(message)
	case srtgo.SrtLogLevelInfo:
		logger.Info().Str("file", file).Int("line", line).Str("area", area).Msg(message)
	case srtgo.SrtLogLevelDebug:
		logger.Debug().Str("file", file).Int("line", line).Str("area", area).Msg(message)
	default:
	}
}
