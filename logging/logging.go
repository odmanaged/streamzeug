/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package logging

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var (
	Log zerolog.Logger
)

func init() {
	var output io.Writer
	output = os.Stdout
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	Log = zerolog.New(output).With().Timestamp().Logger()
}
