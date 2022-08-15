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
	"github.com/prometheus/exporter-toolkit/web"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	loginName = kingpin.Flag("login-name", "User ID").Envar("SCP_LOGINNAME").Default("").String()
	password  = kingpin.Flag("password", "API Password").Envar("SCP_PASSWORD").Default("").String()
	addr      = kingpin.Flag("listen-address", "The address to listen on for HTTP requests.").Envar("SCP_LISTENADDRESS").Default(":9757").String()
	tlsConfig = kingpin.Flag("tls-config", "Path to TLS config file.").Envar("SCP_TLSCONFIG").Default("").String()
)

const netcupWSUrl = "https://www.servercontrolpanel.de/SCP/WSEndUser"

func main() {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	kingpin.Version(version.Version + " git " + version.Revision)
	kingpin.Parse()
	_ = logger.Log("msg", "Starting SCP Exporter version "+version.Version+" git "+version.Revision)
	client := soap.NewClient(netcupWSUrl)
	wsclient := scpclient.NewWSEndUser(client)
	scpCollector := metrics.NewScpCollector(wsclient, logger, loginName, password)
	prometheus.DefaultRegisterer.MustRegister(scpCollector)
	prometheus.DefaultRegisterer.MustRegister(version.NewCollector("scp"))
	metricsServer := http.Server{Handler: promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		}), Addr: *addr}
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))
	err := web.ListenAndServe(&metricsServer, *tlsConfig, logger)
	if err != nil {
		_ = logger.Log("msg", "Run into bad state", "error", err)
		os.Exit(1)
	}

}
