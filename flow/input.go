/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package flow

import (
	"net/url"

	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/input/rist"
)

func (f *Flow) setupInput(c *config.Input) error {
	u, err := url.Parse(c.Url)
	if err != nil {
		return err
	}
	input, err := rist.SetupRistInput(u, f.config.Identifier, f.receiver)
	if err != nil {
		return err
	}
	f.configuredInputs[c.Url] = input
	return nil
}
