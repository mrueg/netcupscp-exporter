// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package metrics implements a metric collector to gather metrics from the NetCup API
package metrics

import (
	"encoding/xml"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mrueg/netcupscp-exporter/pkg/scpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xhit/go-str2duration/v2"
)

const requestURL = "http://enduser.service.web.vcp.netcup.de/"

// ScpCollector struct includes all the information to gather metrics
type ScpCollector struct {
	client              scpclient.WSEndUser
	logger              *slog.Logger
	loginName           *string
	password            *string
	cpuCores            *prometheus.Desc
	memory              *prometheus.Desc
	monthlyTrafficIn    *prometheus.Desc
	monthlyTrafficOut   *prometheus.Desc
	monthlyTrafficTotal *prometheus.Desc
	serverStartTime     *prometheus.Desc
	ipInfo              *prometheus.Desc
	ifaceThrottled      *prometheus.Desc
	serverStatus        *prometheus.Desc
	rescueActive        *prometheus.Desc
	rebootRecommended   *prometheus.Desc
	diskCapacity        *prometheus.Desc
	diskUsed            *prometheus.Desc
	diskOptimization    *prometheus.Desc
}

// NewScpCollector returns a collector object
func NewScpCollector(client scpclient.WSEndUser, logger *slog.Logger, loginName *string, password *string) *ScpCollector {
	var prefix = "scp_"
	return &ScpCollector{
		client:    client,
		logger:    logger,
		loginName: loginName,
		password:  password,
		cpuCores: prometheus.NewDesc(prefix+"cpu_cores",
			"Number of CPU cores",
			[]string{"vserver"},
			nil),
		memory: prometheus.NewDesc(prefix+"memory_bytes",
			"Amount of Memory in Bytes",
			[]string{"vserver"},
			nil),
		monthlyTrafficIn: prometheus.NewDesc(prefix+"monthlytraffic_in_bytes",
			"Monthly traffic incoming in Bytes (only gigabyte-level resolution)",
			[]string{"vserver", "month", "year"},
			nil),
		monthlyTrafficOut: prometheus.NewDesc(prefix+"monthlytraffic_out_bytes",
			"Monthly traffic outgoing in Bytes (only gigabyte-level resolution)",
			[]string{"vserver", "month", "year"},
			nil),
		monthlyTrafficTotal: prometheus.NewDesc(prefix+"monthlytraffic_total_bytes",
			"Total monthly traffic in Bytes (only gigabyte-level resolution)",
			[]string{"vserver", "month", "year"},
			nil),
		serverStartTime: prometheus.NewDesc(prefix+"server_start_time_seconds",
			"Start time of the vserver in seconds (only minute-level resolution)",
			[]string{"vserver"},
			nil),
		ipInfo: prometheus.NewDesc(prefix+"ip_info", "IPs assigned to this server",
			[]string{"vserver", "ip"},
			nil),
		ifaceThrottled: prometheus.NewDesc(prefix+"interface_throttled", "Interface's traffic is throttled (1) or not (0)",
			[]string{"vserver", "driver", "id", "ip", "ip_type", "mac", "throttle_message"},
			nil),
		serverStatus: prometheus.NewDesc(prefix+"server_status", "Online (1) / Offline (0) status",
			[]string{"vserver", "status", "nickname"},
			nil),
		rescueActive: prometheus.NewDesc(prefix+"rescue_active", "Rescue system active (1) / inactive (0)",
			[]string{"vserver", "message"},
			nil),
		rebootRecommended: prometheus.NewDesc(prefix+"reboot_recommended", "Reboot recommended (1) / not recommended (0)",
			[]string{"vserver", "message"},
			nil),
		diskCapacity: prometheus.NewDesc(prefix+"disk_capacity_bytes", "Available storage space in Bytes",
			[]string{"vserver", "driver", "name"},
			nil),
		diskUsed: prometheus.NewDesc(prefix+"disk_used_bytes", "Used storage space in Bytes",
			[]string{"vserver", "driver", "name"},
			nil),
		diskOptimization: prometheus.NewDesc(prefix+"disk_optimization", "Optimization recommended (1) / not recommended (0)",
			[]string{"vserver", "driver", "name", "message"},
			nil),
	}
}

// Describe implements prometheus.Describe for ScpCollector
func (collector *ScpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.cpuCores
	ch <- collector.memory
	ch <- collector.monthlyTrafficIn
	ch <- collector.monthlyTrafficOut
	ch <- collector.monthlyTrafficTotal
	ch <- collector.serverStartTime
	ch <- collector.ipInfo
	ch <- collector.ifaceThrottled
	ch <- collector.serverStatus
	ch <- collector.rescueActive
	ch <- collector.diskCapacity
	ch <- collector.diskUsed
	ch <- collector.diskOptimization
}

// Collect implements prometheus.Collect for ScpCollector
func (collector *ScpCollector) Collect(ch chan<- prometheus.Metric) {
	genericRequest := &scpclient.GetVServers{
		Xmlns:     requestURL,
		LoginName: *collector.loginName,
		Password:  *collector.password,
	}
	genericResponse, err := collector.client.GetVServers(genericRequest)
	if err != nil {
		collector.logger.Error("Unable to get servers", "error", err.Error())
	}

	debug, _ := xml.Marshal(genericResponse)
	collector.logger.Debug(string(debug))

	vservers := genericResponse.Return_

	for _, vserver := range vservers {
		infoRequest := &scpclient.GetVServerInformation{
			Xmlns:       requestURL,
			LoginName:   *collector.loginName,
			Password:    *collector.password,
			Vservername: *vserver,
		}
		infoResponse, err := collector.client.GetVServerInformation(infoRequest)
		debug, _ := xml.Marshal(infoResponse)
		collector.logger.Debug(string(debug))
		if err != nil {
			collector.logger.Error("Unable to get Server Information", "error", err.Error())
		}
		// Create CPU / Memory info metrics
		ch <- prometheus.MustNewConstMetric(collector.cpuCores, prometheus.GaugeValue, float64(infoResponse.Return_.CpuCores), *vserver)
		ch <- prometheus.MustNewConstMetric(collector.memory, prometheus.GaugeValue, float64(infoResponse.Return_.Memory*1024*1024), *vserver)

		// Create traffic metrics
		ch <- prometheus.MustNewConstMetric(collector.monthlyTrafficIn, prometheus.GaugeValue, float64(infoResponse.Return_.CurrentMonth.In*1024*1024), *vserver, strconv.Itoa(int(infoResponse.Return_.CurrentMonth.Month)), strconv.Itoa(int(infoResponse.Return_.CurrentMonth.Year)))
		ch <- prometheus.MustNewConstMetric(collector.monthlyTrafficOut, prometheus.GaugeValue, float64(infoResponse.Return_.CurrentMonth.Out*1024*1024), *vserver, strconv.Itoa(int(infoResponse.Return_.CurrentMonth.Month)), strconv.Itoa(int(infoResponse.Return_.CurrentMonth.Year)))
		ch <- prometheus.MustNewConstMetric(collector.monthlyTrafficTotal, prometheus.GaugeValue, float64(infoResponse.Return_.CurrentMonth.Total*1024*1024), *vserver, strconv.Itoa(int(infoResponse.Return_.CurrentMonth.Month)), strconv.Itoa(int(infoResponse.Return_.CurrentMonth.Year)))

		// Create server status metric
		var online float64
		if infoResponse.Return_.Status == "online" {
			online = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.serverStatus, prometheus.GaugeValue, online, *vserver, infoResponse.Return_.Status, infoResponse.Return_.VServerNickname)

		var rescue float64
		if infoResponse.Return_.RescueEnabled {
			rescue = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.rescueActive, prometheus.GaugeValue, rescue, *vserver, infoResponse.Return_.RescueEnabledMessage)

		var reboot float64
		if infoResponse.Return_.RebootRecommended {
			reboot = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.rebootRecommended, prometheus.GaugeValue, reboot, *vserver, infoResponse.Return_.RebootRecommendedMessage)

		// Create IP info metric
		for _, ip := range infoResponse.Return_.Ips {
			ch <- prometheus.MustNewConstMetric(collector.ipInfo, prometheus.GaugeValue, 1, *vserver, *ip)
		}

		// Create Interface throttling metric
		for _, iface := range infoResponse.Return_.ServerInterfaces {
			var throttled float64
			if iface.TrafficThrottled {
				throttled = 1
			}
			seenIPs := make(map[string]bool)
			for _, ip := range iface.Ipv4IP {
				if _, seen := seenIPs[*ip]; !seen {
					seenIPs[*ip] = true
					ch <- prometheus.MustNewConstMetric(collector.ifaceThrottled, prometheus.GaugeValue, throttled, *vserver, iface.Driver, iface.Id, *ip, "ipv4", iface.Mac, iface.TrafficThrottledMessage)
				}
			}
			for _, ip := range iface.Ipv6IP {
				if _, seen := seenIPs[*ip]; !seen {
					seenIPs[*ip] = true
					ch <- prometheus.MustNewConstMetric(collector.ifaceThrottled, prometheus.GaugeValue, throttled, *vserver, iface.Driver, iface.Id, *ip, "ipv6", iface.Mac, iface.TrafficThrottledMessage)
				}
			}
		}

		// Create Disk metrics
		for _, disk := range infoResponse.Return_.ServerDisks {
			ch <- prometheus.MustNewConstMetric(collector.diskCapacity, prometheus.GaugeValue, float64(disk.Capacity*1024*1024*1024), *vserver, disk.Driver, disk.Name)
			ch <- prometheus.MustNewConstMetric(collector.diskUsed, prometheus.GaugeValue, float64(disk.Used*1024*1024*1024), *vserver, disk.Driver, disk.Name)

			var optimize float64
			if disk.OptimizationRecommended {
				optimize = 1
			}
			ch <- prometheus.MustNewConstMetric(collector.diskOptimization, prometheus.GaugeValue, optimize, *vserver, disk.Driver, disk.Name, disk.OptimizationRecommendedMessage)

		}
		// Create start time metric
		uptime, err := parseUptimeString(&infoResponse.Return_.Uptime)
		if err != nil {
			collector.logger.Error("Unable to parse uptime", "error", err.Error())
		}
		ch <- prometheus.MustNewConstMetric(collector.serverStartTime, prometheus.GaugeValue, float64(time.Now().Add(-uptime).Unix()), *vserver)
	}
}

func parseUptimeString(uptime *string) (parsed time.Duration, err error) {
	tmp := strings.Replace(*uptime, " days ", "d", 1)
	tmp = strings.Replace(tmp, " day ", "d", 1)
	tmp = strings.Replace(tmp, " hours ", "h", 1)
	tmp = strings.Replace(tmp, " hour ", "h", 1)
	tmp = strings.Replace(tmp, " minutes", "m", 1)
	tmp = strings.Replace(tmp, " minute", "m", 1)
	return str2duration.ParseDuration(tmp)
}
