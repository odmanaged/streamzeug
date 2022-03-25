/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"fmt"
	"net/url"
)

type Output struct {
	Identifier string `yaml:"identifier"`
	Url        string `yaml:"url"`
}

func validateOutputConfig(c *Output) error {
	if err := validateURL(c.Url); err != nil {
		return err
	}
	u, err := url.Parse(c.Url)
	if err != nil {
		panic(err) //if url parsing goes bad after doing the same in validateURL, panic
	}
	switch u.Scheme {
	case "srt", "udp", "rtp", "dektecasi":
		return nil
	default:
		return fmt.Errorf("output type %s not supported", u.Scheme)
	}
}
