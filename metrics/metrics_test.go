// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package metrics

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mrueg/netcupscp-exporter/scpclient"
	"github.com/prometheus/client_golang/prometheus"
)

func TestScpCollector_Describe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewScpCollector(nil, logger)

	ch := make(chan *prometheus.Desc, 100)
	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}

	if count == 0 {
		t.Errorf("expected > 0 metric descriptions, got 0")
	}
}

func TestScpCollector_Collect(t *testing.T) {
	serverName := "v1234"
	nickname := "my-server"
	status := scpclient.RUNNING
	cpuCount := int32(4)
	memInMiB := int64(8192)
	gpuAvailable := false
	gaAvailable := true

	mux := http.NewServeMux()

	// Ping
	mux.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Maintenance
	mux.HandleFunc("/api/v1/maintenance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scpclient.Maintenance{})
	})

	// Tasks
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tasks := []scpclient.TaskInfoMinimal{}
		_ = json.NewEncoder(w).Encode(tasks)
	})

	// Servers list
	mux.HandleFunc("/api/v1/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/servers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		id := int32(42)
		servers := []scpclient.ServerMinimal{
			{
				Id:   &id,
				Name: &serverName,
			},
		}
		_ = json.NewEncoder(w).Encode(servers)
	})

	// Server detail
	mux.HandleFunc("/api/v1/servers/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := int32(42)
		srv := scpclient.Server{
			Id:                 &id,
			Name:               &serverName,
			Nickname:           &nickname,
			GpuDriverAvailable: &gpuAvailable,
			ServerLiveInfo: &scpclient.ServerInfo{
				State:                    &status,
				CpuCount:                 &cpuCount,
				CurrentServerMemoryInMiB: &memInMiB,
			},
		}
		_ = json.NewEncoder(w).Encode(srv)
	})

	// Guest agent status
	mux.HandleFunc("/api/v1/servers/42/guest-agent/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scpclient.GuestAgentStatus{
			Available: &gaAvailable,
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := scpclient.NewClientWithResponses(ts.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewScpCollector(client, logger)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	metricsCount := 0
	for range ch {
		metricsCount++
	}

	if metricsCount == 0 {
		t.Errorf("expected collected metrics, got 0")
	}
}
