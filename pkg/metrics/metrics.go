// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/mrueg/netcupscp-exporter/pkg/scpclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xhit/go-str2duration/v2"
)

const requestURL = "http://enduser.service.web.vcp.netcup.de/"

type scpCollector struct {
	client               scpclient.WSEndUser
	logger               log.Logger
	loginName            *string
	password             *string
	cpuCores             *prometheus.Desc
	memory               *prometheus.Desc
	monthlytraffic_in    *prometheus.Desc
	monthlytraffic_out   *prometheus.Desc
	monthlytraffic_total *prometheus.Desc
	server_start_time    *prometheus.Desc
	ip_info              *prometheus.Desc
	iface_throttled      *prometheus.Desc
	server_status        *prometheus.Desc
	rescue_active        *prometheus.Desc
	reboot_recommended   *prometheus.Desc
	disk_capacity        *prometheus.Desc
	disk_used            *prometheus.Desc
	disk_optimization    *prometheus.Desc
}

func NewScpCollector(client scpclient.WSEndUser, logger log.Logger, loginName *string, password *string) *scpCollector {
	var prefix = "scp_"
	return &scpCollector{
		client:    client,
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
		monthlytraffic_in: prometheus.NewDesc(prefix+"monthlytraffic_in_bytes",
			"Monthly traffic incoming in Bytes (only gigabyte-level resolution)",
			[]string{"vserver", "month", "year"},
			nil),
		monthlytraffic_out: prometheus.NewDesc(prefix+"monthlytraffic_out_bytes",
			"Monthly traffic outgoing in Bytes (only gigabyte-level resolution)",
			[]string{"vserver", "month", "year"},
			nil),
		monthlytraffic_total: prometheus.NewDesc(prefix+"monthlytraffic_total_bytes",
			"Total monthly traffic in Bytes (only gigabyte-level resolution)",
			[]string{"vserver", "month", "year"},
			nil),
		server_start_time: prometheus.NewDesc(prefix+"server_start_time_seconds",
			"Start time of the vserver in seconds (only minute-level resolution)",
			[]string{"vserver"},
			nil),
		ip_info: prometheus.NewDesc(prefix+"ip_info", "IPs assigned to this server",
			[]string{"vserver", "ip"},
			nil),
		iface_throttled: prometheus.NewDesc(prefix+"interface_throttled", "Interface's traffic is throttled (1) or not (0)",
			[]string{"vserver", "driver", "id", "ip", "ip_type", "mac", "throttle_message"},
			nil),
		server_status: prometheus.NewDesc(prefix+"server_status", "Online (1) / Offline (0) status",
			[]string{"vserver", "status", "nickname"},
			nil),
		rescue_active: prometheus.NewDesc(prefix+"rescue_active", "Rescue system active (1) / inactive (0)",
			[]string{"vserver", "message"},
			nil),
		reboot_recommended: prometheus.NewDesc(prefix+"reboot_recommended", "Reboot recommended (1) / not recommended (0)",
			[]string{"vserver", "message"},
			nil),
		disk_capacity: prometheus.NewDesc(prefix+"disk_capacity_bytes", "Available storage space in Bytes",
			[]string{"vserver", "driver", "name"},
			nil),
		disk_used: prometheus.NewDesc(prefix+"disk_used_bytes", "Used storage space in Bytes",
			[]string{"vserver", "driver", "name"},
			nil),
		disk_optimization: prometheus.NewDesc(prefix+"disk_optimization", "Optimization recommended (1) / not recommended (0)",
			[]string{"vserver", "driver", "name", "message"},
			nil),
	}
}

func (collector *scpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.cpuCores
	ch <- collector.memory
	ch <- collector.monthlytraffic_in
	ch <- collector.monthlytraffic_out
	ch <- collector.monthlytraffic_total
	ch <- collector.server_start_time
	ch <- collector.ip_info
	ch <- collector.iface_throttled
	ch <- collector.server_status
	ch <- collector.rescue_active
	ch <- collector.disk_capacity
	ch <- collector.disk_used
	ch <- collector.disk_optimization
}

func (collector *scpCollector) Collect(ch chan<- prometheus.Metric) {
	generic_request := &scpclient.GetVServers{
		Xmlns:     requestURL,
		LoginName: *collector.loginName,
		Password:  *collector.password,
	}
	generic_response, err := collector.client.GetVServers(generic_request)
	if err != nil {
		_ = collector.logger.Log("msg", "Unable to get servers", "err", err)
	}
	vservers := generic_response.Return_
	for _, vserver := range vservers {
		info_request := &scpclient.GetVServerInformation{
			Xmlns:       requestURL,
			LoginName:   *collector.loginName,
			Password:    *collector.password,
			Vservername: *vserver,
		}
		info_response, err := collector.client.GetVServerInformation(info_request)
		if err != nil {
			_ = collector.logger.Log("msg", "Unable to get Server Information", "err", err)
		}
		// Create CPU / Memory info metrics
		ch <- prometheus.MustNewConstMetric(collector.cpuCores, prometheus.GaugeValue, float64(info_response.Return_.CpuCores), *vserver)
		ch <- prometheus.MustNewConstMetric(collector.memory, prometheus.GaugeValue, float64(info_response.Return_.Memory*1024*1024), *vserver)

		// Create traffic metrics
		ch <- prometheus.MustNewConstMetric(collector.monthlytraffic_in, prometheus.GaugeValue, float64(info_response.Return_.CurrentMonth.In*1024*1024), *vserver, strconv.Itoa(int(info_response.Return_.CurrentMonth.Month)), strconv.Itoa(int(info_response.Return_.CurrentMonth.Year)))
		ch <- prometheus.MustNewConstMetric(collector.monthlytraffic_out, prometheus.GaugeValue, float64(info_response.Return_.CurrentMonth.Out*1024*1024), *vserver, strconv.Itoa(int(info_response.Return_.CurrentMonth.Month)), strconv.Itoa(int(info_response.Return_.CurrentMonth.Year)))
		ch <- prometheus.MustNewConstMetric(collector.monthlytraffic_total, prometheus.GaugeValue, float64(info_response.Return_.CurrentMonth.Total*1024*1024), *vserver, strconv.Itoa(int(info_response.Return_.CurrentMonth.Month)), strconv.Itoa(int(info_response.Return_.CurrentMonth.Year)))

		// Create server status metric
		var online float64 = 0
		if info_response.Return_.Status == "online" {
			online = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.server_status, prometheus.GaugeValue, online, *vserver, info_response.Return_.Status, info_response.Return_.VServerNickname)

		var rescue float64 = 0
		if info_response.Return_.RescueEnabled {
			rescue = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.rescue_active, prometheus.GaugeValue, rescue, *vserver, info_response.Return_.RescueEnabledMessage)

		var reboot float64 = 0
		if info_response.Return_.RebootRecommended {
			reboot = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.reboot_recommended, prometheus.GaugeValue, reboot, *vserver, info_response.Return_.RebootRecommendedMessage)

		// Create IP info metric
		for _, ip := range info_response.Return_.Ips {
			ch <- prometheus.MustNewConstMetric(collector.ip_info, prometheus.GaugeValue, 1, *vserver, *ip)
		}

		// Create Interface throttling metric
		for _, iface := range info_response.Return_.ServerInterfaces {
			var throttled float64 = 0
			if iface.TrafficThrottled {
				throttled = 1
			}
			for _, ip := range iface.Ipv4IP {
				ch <- prometheus.MustNewConstMetric(collector.iface_throttled, prometheus.GaugeValue, throttled, *vserver, iface.Driver, iface.Id, *ip, "ipv4", iface.Mac, iface.TrafficThrottledMessage)
			}
			for _, ip := range iface.Ipv6IP {
				ch <- prometheus.MustNewConstMetric(collector.iface_throttled, prometheus.GaugeValue, throttled, *vserver, iface.Driver, iface.Id, *ip, "ipv6", iface.Mac, iface.TrafficThrottledMessage)
			}
		}

		// Create Disk metrics
		for _, disk := range info_response.Return_.ServerDisks {
			ch <- prometheus.MustNewConstMetric(collector.disk_capacity, prometheus.GaugeValue, float64(disk.Capacity*1024*1024*1024), *vserver, disk.Driver, disk.Name)
			ch <- prometheus.MustNewConstMetric(collector.disk_used, prometheus.GaugeValue, float64(disk.Used*1024*1024*1024), *vserver, disk.Driver, disk.Name)

			var optimize float64 = 0
			if disk.OptimizationRecommended {
				optimize = 1
			}
			ch <- prometheus.MustNewConstMetric(collector.disk_optimization, prometheus.GaugeValue, optimize, *vserver, disk.Driver, disk.Name, disk.OptimizationRecommendedMessage)

		}
		// Create start time metric
		uptime, err := parseUptimeString(&info_response.Return_.Uptime)
		if err != nil {
			_ = collector.logger.Log("msg", "Unable to parse uptime", "err", err)
		}
		ch <- prometheus.MustNewConstMetric(collector.server_start_time, prometheus.GaugeValue, float64(time.Now().Add(-uptime).Unix()), *vserver)
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
