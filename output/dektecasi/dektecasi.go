/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package dektecasi

/*
#cgo CFLAGS: -I ${SRCDIR}/
#cgo LDFLAGS: -L${SRCDIR}/../../ -ldektec -lstdc++ -lm -ldl
#include "../../dektec/asi.h"

extern void dektekAsiLoggingCB(void *cookie, bool isErr, char *message);

*/
import "C"

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"
	"unsafe"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/mainloop"
	"github.com/odmedia/streamzeug/output"
	"github.com/odmedia/streamzeug/output/dektecasi/dtstats"
	"github.com/odmedia/streamzeug/stats"
)

type dektecasi struct {
	m                 *mainloop.Mainloop
	ctx               context.Context
	cancel            context.CancelFunc
	port              int
	identifier        string
	output_identifier string
	name              string
	logCBPtr          unsafe.Pointer
	stats             *stats.Stats
	dektecCtx         C.dektec_asi_ctx_t
}

func (d *dektecasi) statsloop() {
	var fetchStats C.struct_DektecAsiStats
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-time.After(10 * time.Second):
			C.dektec_asi_get_stats(d.dektecCtx, &fetchStats)
			stat := dtstats.DektecAsiStats{
				AsiPortno:         d.port,
				Fifobytes:         int(fetchStats.FifoBytes),
				BytesWrittenTotal: int(fetchStats.BytesWritten),
				BytesWritten:      int(fetchStats.BytesSinceLastCall),
			}
			d.stats.HandleStats("", d.output_identifier, nil, &stat)
		}
	}
}

func (d *dektecasi) String() string {
	return d.name
}

func (d *dektecasi) Count() int {
	return 1
}

func (d *dektecasi) Write(block *libristwrapper.RistDataBlock) (n int, err error) {
	select {
	case <-d.ctx.Done():
		return 0, errors.New("output stopped")
	default:
		//
	}
	n_out := C.dektec_asi_write(d.dektecCtx, (*C.char)(unsafe.Pointer(&block.Data[0])), C.size_t(len(block.Data)))
	return int(n_out), nil
}

func (d *dektecasi) Close() error {
	d.cancel()
	C.dektec_asi_destroy(d.dektecCtx)
	unsetLoggingCB(d.logCBPtr)
	return nil
}

func ParseURL(ctx context.Context, u *url.URL, identifier, output_identifier string, m *mainloop.Mainloop, stats *stats.Stats) (output.Output, error) {
	var err error
	logging.Log.Info().Str("identifier", identifier).Msgf("setting up dektec asi output: %s", u.String())
	sDektecport := u.Port()
	if sDektecport == "" {
		return nil, errors.New("port musn't be empty")
	}
	dektecport, err := strconv.Atoi(sDektecport)
	if err != nil {
		return nil, err
	}

	sBitrate := u.Query().Get("bitrate")
	bitrate, err := strconv.Atoi(sBitrate)
	if err != nil {
		return nil, err
	}
	logCBPtr := storeLoggingCB(func(isErr bool, msg string) {
		if isErr {
			logging.Log.Error().Str("module", "dektec-asi-output").Msg(msg)
			return
		}
		logging.Log.Info().Str("module", "dektec-asi-output").Msg(msg)
	})

	dektecasictx := C.setup_dektec_asi_output(C.int(dektecport), C.int(bitrate), (C.log_cb_func_t)(C.dektekAsiLoggingCB), logCBPtr)
	if dektecasictx == nil {
		unsetLoggingCB(logCBPtr)
		return nil, errors.New("unable to setup dektec-asi")
	}

	ctx, cancel := context.WithCancel(ctx)
	out := &dektecasi{
		m:                 m,
		ctx:               ctx,
		cancel:            cancel,
		port:              dektecport,
		identifier:        identifier,
		output_identifier: output_identifier,
		name:              u.String(),
		logCBPtr:          logCBPtr,
		stats:             stats,
		dektecCtx:         dektecasictx,
	}
	go out.statsloop()
	out.m.AddOutput(out)
	return out, nil
}
