/*
 * SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
 * SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package stats

import (
	"context"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"

	"code.videolan.org/rist/ristgo/libristwrapper"
	"github.com/Showmax/go-fqdn"
	"github.com/haivision/srtgo"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/odmedia/streamzeug/config"
	"github.com/odmedia/streamzeug/logging"
	"github.com/odmedia/streamzeug/output/dektecasi/dtstats"
	"github.com/odmedia/streamzeug/version"
	"github.com/sam-kamerer/go-runtime-metrics/v2/pkg/collector"
)

var (
	configlock             sync.RWMutex
	influxDBWriteApi       api.WriteAPIBlocking = nil
	hostname               string
	applicationidentifier  string
	srtmeasurement         string
	ristrxmeasurement      string
	risttxmeasurement      string
	applicationmeasurement string
)

func SetupInfluxDB(ctx context.Context, c *config.InfluxDBConfig, identifier string) error {
	configlock.Lock()
	defer configlock.Unlock()
	var err error
	srtmeasurement = "srt"
	ristrxmeasurement = "rist-receive"
	risttxmeasurement = "rist-sender"
	applicationmeasurement = "streamzeug"
	if c.SrtMeasurement != "" {
		srtmeasurement = c.SrtMeasurement
	}
	if c.RistRXMeasurement != "" {
		ristrxmeasurement = c.RistRXMeasurement
	}
	if c.RistTXMeasurement != "" {
		risttxmeasurement = c.RistTXMeasurement
	}
	if c.ApplicationMeasurement != "" {
		applicationmeasurement = c.ApplicationMeasurement
	}
	client := influxdb2.NewClient(c.Url, c.Token)
	influxDBWriteApi = client.WriteAPIBlocking(c.Org, c.Bucket)
	hostname, err = fqdn.FqdnHostname()
	if err != nil {
		return err
	}
	go InfluxDBPeriodic(ctx)
	return nil
}

func InfluxDisable() {
	configlock.Lock()
	influxDBWriteApi = nil
	configlock.Unlock()
}

func structToMap(s interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	elem := reflect.ValueOf(s).Elem()
	relType := elem.Type()
	for i := 0; i < relType.NumField(); i++ {
		m[relType.Field(i).Name] = elem.Field(i).Interface()
	}
	return m
}

func (s *Stats) writeInfluxStats(host, output_identifier string, u *url.URL, stats interface{}) {
	configlock.RLock()
	defer configlock.RUnlock()
	values := structToMap(stats)
	var (
		measurement string
		cname       string
	)
	tags := map[string]string{"identifier": s.identifier, "hostname": hostname}
	switch stats.(type) {
	case *libristwrapper.ReceiverFlowStats:
		measurement = ristrxmeasurement
		cname = values["CName"].(string)
		delete(values, "CName")
	case *libristwrapper.SenderPeerStats:
		measurement = risttxmeasurement
		cname = values["CName"].(string)
		delete(values, "CName")
	case *srtgo.SrtStats:
		measurement = srtmeasurement
	case *dtstats.DektecAsiStats:
		measurement = "dektekasi"
		tags["port"] = strconv.FormatInt(int64(values["AsiPortno"].(int)), 10)
		delete(values, "AsiPortno")
	default:
		panic("wrong interface")
	}
	if host != "" {
		tags["remotehost"] = host
	}
	if u != nil {
		tags["localurl"] = u.Host
	}
	if output_identifier != "" {
		tags["output_identifier"] = output_identifier
	}
	if cname != "" {
		tags["cname"] = cname
	}
	point := influxdb2.NewPoint(
		measurement,
		tags,
		values,
		time.Now(),
	)
	err := influxDBWriteApi.WritePoint(context.Background(), point)
	if err != nil {
		logging.Log.Error().Str("module", "influxdb-stats").Err(err)
	}
}

func InfluxDBPeriodic(ctx context.Context) {
	collector := collector.New(nil)
	tickCH := time.NewTicker(collector.PauseDur).C
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickCH:
			configlock.RLock()
			fields := collector.CollectStats()
			tags := fields.Tags()
			tags["identifier"] = applicationidentifier
			tags["hostname"] = hostname
			values := fields.Values()
			values["go.os"] = tags["go.os"]
			values["go.arch"] = tags["go.arch"]
			values["go.version"] = tags["go.version"]
			values["streamzeug.version"] = version.CombinedVersion
			delete(tags, "go.os")
			delete(tags, "go.arch")
			delete(tags, "go.version")
			point := influxdb2.NewPoint(
				applicationmeasurement,
				tags,
				values,
				time.Now(),
			)
			err := influxDBWriteApi.WritePoint(context.Background(), point)
			if err != nil {
				logging.Log.Error().Str("module", "influxdb-stats").Err(err)
			}
			configlock.RUnlock()
		}
	}
}
