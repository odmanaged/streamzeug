/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package rist

import (
	"context"
	"net/url"
	"sync"

	"code.videolan.org/rist/ristgo"
	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/odmedia/streamzeug/mainloop"
	"github.com/odmedia/streamzeug/output"
	"github.com/odmedia/streamzeug/stats"
)

type ristoutput struct {
	sender    ristgo.Sender
	clientUrl string
}

func ParseRistOutput(ctx context.Context, u *url.URL, identifier, output_identifier string, m *mainloop.Mainloop, stats *stats.Stats, wait *sync.WaitGroup) (output.Output, error) {

	return &ristoutput{}, nil
}

func (r *ristoutput) Close() error {
	r.sender.Close()
	return nil
}

func (r *ristoutput) Count() int {
	return 1
}

func (r *ristoutput) String() string {
	return r.clientUrl
}

func (r *ristoutput) Write(block *libristwrapper.RistDataBlock) (n int, e error) {
	return r.sender.Write(block.Data)
}
