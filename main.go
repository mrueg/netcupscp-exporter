// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"net/http"
	"os"

	"github.com/go-kit/log"
	"github.com/hooklift/gowsdl/soap"
	"github.com/mrueg/netcupscp-exporter/pkg/metrics"
	"github.com/mrueg/netcupscp-exporter/pkg/scpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/spf13/pflag"
)

var (
	loginName = pflag.String("login-name", "", "User ID")
	password  = pflag.String("password", "", "API Password")
	addr      = pflag.String("listen-address", ":9757", "The address to listen on for HTTP requests.")
)

const netcupWSUrl = "https://www.servercontrolpanel.de/SCP/WSEndUser"

func main() {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	_ = logger.Log("msg", "Starting SCP Exporter version "+version.Version+" git "+version.Revision)
	pflag.Parse()
	client := soap.NewClient(netcupWSUrl)
	wsclient := scpclient.NewWSEndUser(client)
	scpCollector := metrics.NewScpCollector(wsclient, logger, loginName, password)
	prometheus.DefaultRegisterer.MustRegister(scpCollector)
	prometheus.DefaultRegisterer.MustRegister(version.NewCollector("scp"))
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		}))

	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		_ = logger.Log("msg", "Run into bad state", "error", err)
		os.Exit(1)
	}

}
