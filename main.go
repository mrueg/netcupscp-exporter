// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package main parses config arguments and starts the exporter
package main

import (
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hooklift/gowsdl/soap"
	"github.com/mrueg/netcupscp-exporter/pkg/metrics"
	"github.com/mrueg/netcupscp-exporter/pkg/scpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
)

var (
	loginName = kingpin.Flag("login-name", "User ID").Envar("SCP_LOGINNAME").Default("").String()
	password  = kingpin.Flag("password", "API Password").Envar("SCP_PASSWORD").Default("").String()
	addr      = kingpin.Flag("listen-address", "The address to listen on for HTTP requests.").Envar("SCP_LISTENADDRESS").Default(":9757").String()
	tlsConfig = kingpin.Flag("tls-config", "Path to TLS config file.").Envar("SCP_TLSCONFIG").Default("").String()
	logLevel  = kingpin.Flag("log-level", "Log level (debug, info, warn, error)").Envar("SCP_LOGLEVEL").Default("info").String()
)

const netcupWSUrl = "https://www.servercontrolpanel.de/SCP/WSEndUser" //nolint:gosec

func main() {

	kingpin.Version(version.Version + " git " + version.Revision)
	kingpin.Parse()

	var logger log.Logger

	var metricsPath = "/metrics"
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.Allow(level.ParseDefault(*logLevel, level.InfoValue())))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	_ = level.Info(logger).Log("msg", "Starting SCP Exporter version "+version.Version+" git "+version.Revision)
	client := soap.NewClient(netcupWSUrl)
	wsclient := scpclient.NewWSEndUser(client)
	scpCollector := metrics.NewScpCollector(wsclient, logger, loginName, password)
	prometheus.DefaultRegisterer.MustRegister(scpCollector)
	prometheus.DefaultRegisterer.MustRegister(version.NewCollector("scp"))
	metricsServer := http.Server{
		ReadHeaderTimeout: 5 * time.Second}

	landingConfig := web.LandingConfig{
		Name:        "Netcup SCP Exporter",
		Description: "Exporting Metrics from Netcup's ServerControlPanel",
		Version:     version.Version + " git " + version.Revision,
		Links: []web.LandingLinks{
			{
				Address: metricsPath,
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		_ = level.Error(logger).Log(err, "failed to create landing page")
	}
	http.Handle(metricsPath, promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		}))
	http.Handle("/", landingPage)

	flags := web.FlagConfig{
		WebListenAddresses: &[]string{*addr},
		WebSystemdSocket:   new(bool),
		WebConfigFile:      tlsConfig,
	}
	err = web.ListenAndServe(&metricsServer, &flags, logger)
	if err != nil {
		_ = level.Error(logger).Log("msg", "Run into bad state", "error", err)
		os.Exit(1)
	}

}
