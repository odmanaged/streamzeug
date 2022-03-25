/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

type InfluxDBConfig struct {
	Url                    string `yaml:"url"`
	Token                  string `yaml:"token"`
	Bucket                 string `yaml:"bocket"`
	Org                    string `yaml:"org"`
	SrtMeasurement         string `yaml:"srt"`
	RistRXMeasurement      string `yaml:"ristrx"`
	RistTXMeasurement      string `yaml:"risttx"`
	ApplicationMeasurement string `yaml:"application"`
}

func ValidateInfluxDBConfig(c *InfluxDBConfig) error {
	if c == nil {
		return nil
	}
	return validateURL(c.Url)
}
