/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Identifier string          `yaml:"identifier"`
	ListenHTTP string          `yaml:"listenhttp"`
	InfluxDB   *InfluxDBConfig `yaml:"influxdb,omitempty"`
	Flows      []Flow          `yaml:"flows"`
}

func ValidateConfig(c *Config) error {
	if c.Identifier == "" {
		return errors.New("config identifier empty")
	}

	for _, f := range c.Flows {
		err := ValidateFlowConfig(&f)
		if err != nil {
			return fmt.Errorf("flow %s validation failed: %w", f.Identifier, err)
		}
	}
	if err := checkDuplicates(c.Flows); err != nil {
		return fmt.Errorf("duplicate flow identifier: %w", err)
	}
	if err := ValidateInfluxDBConfig(c.InfluxDB); err != nil {
		return fmt.Errorf("influx-db validation failed: %w", err)
	}
	return nil
}

func LoadFromFile(filename string) (*Config, error) {
	yamlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	conf := Config{}

	if err := yaml.Unmarshal(yamlData, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}
