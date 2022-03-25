/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package udp

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/mainloop"
	"github.com/odmedia/streamzeug/output"
	"github.com/odmedia/streamzeug/vectorio"
	"golang.org/x/sys/unix"
)

type socketOptFunc func(sc syscall.RawConn) error

type udpoutput struct {
	c          *net.UDPConn
	m          *mainloop.Mainloop
	ctx        context.Context
	cancel     context.CancelFunc
	float      bool
	identifier string
	source     *net.UDPAddr
	target     *net.UDPAddr
	name       string
	isRtp      bool
	rtpSeq     uint16
	rtpSSRC    uint32
	rtpHeader  []byte
	sc         syscall.RawConn
	ss         []socketOptFunc
}

func (u *udpoutput) String() string {
	return u.name
}

func (u *udpoutput) Count() int {
	return 1
}

func (u *udpoutput) writeRTP(block *libristwrapper.RistDataBlock) (int, error) {
	rtptime := (block.TimeStamp * 90000) >> 32
	u.rtpHeader[0] = 0x80
	u.rtpHeader[1] = 0x21 & 0x7f //MPEG-TS
	u.rtpHeader[2] = byte(u.rtpSeq >> 8)
	u.rtpHeader[3] = byte(u.rtpSeq & 0xff)
	u.rtpSeq++
	u.rtpHeader[4] = byte((rtptime >> 24) & 0xff)
	u.rtpHeader[5] = byte((rtptime >> 16) & 0xff)
	u.rtpHeader[6] = byte((rtptime >> 8) & 0xff)
	u.rtpHeader[7] = byte((rtptime) & 0xff)
	u.rtpHeader[8] = byte(u.rtpSSRC >> 24 & 0xff)
	u.rtpHeader[9] = byte(u.rtpSSRC >> 16 & 0xff)
	u.rtpHeader[10] = byte(u.rtpSSRC >> 8 & 0xff)
	u.rtpHeader[11] = byte(u.rtpSSRC & 0xff)
	bufs := make([][]byte, 2)
	bufs[0] = u.rtpHeader
	bufs[1] = block.Data
	return vectorio.WritevSC(u.sc, bufs)
}

func (u *udpoutput) Write(block *libristwrapper.RistDataBlock) (n int, err error) {
	if !u.isRtp {
		n, err = u.c.Write(block.Data)
	} else {
		n, err = u.writeRTP(block)
	}
	if err != nil {
		if errors.Is(err, error(syscall.EPERM)) || errors.Is(err, error(syscall.ECONNREFUSED)) {
			err = nil
			return
		}
		if u.float {
			logging.Log.Info().Str("identifier", u.identifier).Msgf("floating udp output: %s entered inactive state", u.name)
			go func() {
				go u.connectloop()
			}()
		}
	}
	return
}

func (u *udpoutput) Close() error {
	u.cancel()
	if u.c != nil {
		return u.c.Close()
	}
	return nil
}

func (u *udpoutput) connectloop() {
	for {
		select {
		case <-u.ctx.Done():
			return
		default:
			//
		}
		err := u.connect()
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		logging.Log.Info().Str("identifier", u.identifier).Msgf("floating udp output: %s entered active state", u.name)
		u.m.AddOutput(u)
		return
	}
}

func (u *udpoutput) connect() (err error) {
	u.c, err = net.DialUDP("udp", u.source, u.target)
	if err != nil {
		return
	}
	u.sc, err = u.c.SyscallConn()
	if err != nil {
		return
	}
	for _, s := range u.ss {
		err = s(u.sc)
		if err != nil {
			return
		}
	}
	return
}

func ParseUdpOutput(ctx context.Context, u *url.URL, identifier string, m *mainloop.Mainloop) (output.Output, error) {
	logging.Log.Info().Str("identifier", identifier).Msgf("setting up udp output: %s", u.String())
	var out udpoutput
	out.name = u.String()
	out.identifier = identifier
	out.ctx, out.cancel = context.WithCancel(ctx)
	out.m = m
	out.float = false
	out.ss = make([]socketOptFunc, 0)
	mcastIface := u.Query().Get("iface")
	float := u.Query().Get("float")
	if float != "" {
		out.float = true
	}
	if u.Scheme == "rtp" {
		out.isRtp = true
		out.rtpSSRC = rand.Uint32()
		out.rtpHeader = make([]byte, 12)
	}
	ttl := 255
	ttlVal := u.Query().Get("ttl")
	if ttlVal != "" {
		var err error
		ttl, err = strconv.Atoi(ttlVal)
		if err != nil {
			return nil, err
		}
	}

	var sourceIP *net.UDPAddr = nil
	if mcastIface != "" {
		if iface, err := net.InterfaceByName(mcastIface); err == nil {
			addrs, err := iface.Addrs()
			if err != nil {
				return nil, err
			}
			ipnet := addrs[0].(*net.IPNet)
			ip := ipnet.IP
			sourceIP, err = net.ResolveUDPAddr("udp", ip.String()+":0")
			if err != nil {
				return nil, err
			}
		} else if strings.Contains(mcastIface, ":") {
			if sourceIP, err = net.ResolveUDPAddr("udp", mcastIface); err != nil {
				return nil, err
			}
		} else if sourceIP, err = net.ResolveUDPAddr("udp", mcastIface+":0"); err != nil {
			return nil, err
		}
	}
	target, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return nil, err
	}
	out.source = sourceIP
	out.target = target
	if target.IP.IsMulticast() {
		ttlFunc := func(sc syscall.RawConn) (err error) {
			var scerr error
			err = sc.Control(func(fd uintptr) {
				scerr = syscall.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_MULTICAST_TTL, ttl)
			})
			if err != nil {
				return
			}
			err = scerr
			return
		}
		out.ss = append(out.ss, ttlFunc)
	}
	err = out.connect()
	if err != nil {
		if out.float && (errors.Is(err, error(unix.EADDRNOTAVAIL)) || errors.Is(err, error(unix.ENETUNREACH))) {
			go out.connectloop()
			return &out, nil
		}
		return nil, err
	}
	out.m.AddOutput(&out)
	return &out, nil
}
