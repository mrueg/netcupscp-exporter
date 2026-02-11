// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package main parses config arguments and starts the exporter
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/mrueg/netcupscp-exporter/metrics"
	"github.com/mrueg/netcupscp-exporter/scpclient"
	"github.com/prometheus/client_golang/prometheus"
	cversion "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"golang.org/x/oauth2"
)

var (
	refreshToken = kingpin.Flag("refresh-token", "API Refresh Token").Envar("SCP_REFRESHTOKEN").Default("").String()
	addr         = kingpin.Flag("listen-address", "The address to listen on for HTTP requests.").Envar("SCP_LISTENADDRESS").Default(":9757").String()
	tlsConfig    = kingpin.Flag("tls-config", "Path to TLS config file.").Envar("SCP_TLSCONFIG").Default("").String()
)

const (
	tokenURL = "https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token"
	apiURL   = "https://www.servercontrolpanel.de/scp-core"
)

func main() {

	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Version + " git " + version.Revision)
	kingpin.Parse()

	var logger *slog.Logger

	var metricsPath = "/metrics"
	logger = promslog.New(promslogConfig)
	logger.Debug("Starting SCP Exporter version " + version.Version + " git " + version.Revision)

	if *refreshToken == "" {
		logger.Error("Refresh token is required")
		os.Exit(1)
	}

	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID: "scp",
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenURL,
		},
	}

	token := &oauth2.Token{
		RefreshToken: *refreshToken,
	}

	httpClient := conf.Client(ctx, token)
	client, err := scpclient.NewClientWithResponses(apiURL, scpclient.WithHTTPClient(httpClient))
	if err != nil {
		logger.Error("failed to create API client", "error", err.Error())
		os.Exit(1)
	}

	scpCollector := metrics.NewScpCollector(client, logger)
	prometheus.DefaultRegisterer.MustRegister(scpCollector)
	prometheus.DefaultRegisterer.MustRegister(cversion.NewCollector("scp"))
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
		logger.Error("failed to create landing page", "error", err.Error())
		os.Exit(1)
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
		logger.Error("Run into bad state", "error", err)
		os.Exit(1)
	}

}
