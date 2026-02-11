// This Source Code Form is subject to the terms of the Mozilla Public
// License, version 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package metrics implements a metric collector to gather metrics from the NetCup API
package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/mrueg/netcupscp-exporter/scpclient"
	"github.com/prometheus/client_golang/prometheus"
)

// ScpCollector struct includes all the information to gather metrics
type ScpCollector struct {
	client              *scpclient.ClientWithResponses
	logger              *slog.Logger
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
	diskCapacity        *prometheus.Desc
	diskUsed            *prometheus.Desc
	diskOptimization    *prometheus.Desc
	snapshotCount       *prometheus.Desc
	configChanged       *prometheus.Desc
	interfaceSpeed      *prometheus.Desc
	cpuMaxCount         *prometheus.Desc
	memoryMax           *prometheus.Desc
	disksAvailableSpace *prometheus.Desc
	autostartEnabled    *prometheus.Desc
	uefiEnabled         *prometheus.Desc
	latestQemu          *prometheus.Desc
	disabled            *prometheus.Desc
	snapshotAllowed     *prometheus.Desc
	maintenanceStart    *prometheus.Desc
	maintenanceFinish   *prometheus.Desc
	taskInfo            *prometheus.Desc
	tasksPending        *prometheus.Desc
	apiUp               *prometheus.Desc
}

// NewScpCollector returns a collector object
func NewScpCollector(client *scpclient.ClientWithResponses, logger *slog.Logger) *ScpCollector {
	var prefix = "scp_"
	return &ScpCollector{
		client: client,
		logger: logger,
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
			[]string{"vserver", "status", "nickname", "architecture", "site_city"},
			nil),
		rescueActive: prometheus.NewDesc(prefix+"rescue_active", "Rescue system active (1) / inactive (0)",
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
		snapshotCount: prometheus.NewDesc(prefix+"snapshot_count", "Total number of snapshots",
			[]string{"vserver"},
			nil),
		configChanged: prometheus.NewDesc(prefix+"config_changed", "Pending configuration changes (1) / none (0)",
			[]string{"vserver"},
			nil),
		interfaceSpeed: prometheus.NewDesc(prefix+"interface_speed_mbits", "Interface link speed in Mbits/s",
			[]string{"vserver", "mac", "driver"},
			nil),
		cpuMaxCount: prometheus.NewDesc(prefix+"cpu_max_count", "Maximum number of CPU cores",
			[]string{"vserver"},
			nil),
		memoryMax: prometheus.NewDesc(prefix+"memory_max_bytes", "Maximum amount of Memory in Bytes",
			[]string{"vserver"},
			nil),
		disksAvailableSpace: prometheus.NewDesc(prefix+"disks_available_space_bytes", "Available space for new disks in Bytes",
			[]string{"vserver"},
			nil),
		autostartEnabled: prometheus.NewDesc(prefix+"autostart_enabled", "Autostart enabled (1) / disabled (0)",
			[]string{"vserver"},
			nil),
		uefiEnabled: prometheus.NewDesc(prefix+"uefi_enabled", "UEFI enabled (1) / disabled (0)",
			[]string{"vserver"},
			nil),
		latestQemu: prometheus.NewDesc(prefix+"latest_qemu", "Server is running latest QEMU version (1) / older (0)",
			[]string{"vserver"},
			nil),
		disabled: prometheus.NewDesc(prefix+"disabled", "Server is disabled (1) / enabled (0)",
			[]string{"vserver"},
			nil),
		snapshotAllowed: prometheus.NewDesc(prefix+"snapshot_allowed", "Snapshot creation allowed (1) / disallowed (0)",
			[]string{"vserver"},
			nil),
		maintenanceStart: prometheus.NewDesc(prefix+"maintenance_start_time_seconds", "Next maintenance window start time",
			nil, nil),
		maintenanceFinish: prometheus.NewDesc(prefix+"maintenance_finish_time_seconds", "Next maintenance window finish time",
			nil, nil),
		taskInfo: prometheus.NewDesc(prefix+"task_info", "Current task information",
			[]string{"uuid", "name", "state"},
			nil),
		tasksPending: prometheus.NewDesc(prefix+"tasks_pending_count", "Number of pending or running tasks",
			nil, nil),
		apiUp: prometheus.NewDesc(prefix+"api_up", "API is reachable (1) / unreachable (0)",
			nil, nil),
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
	ch <- collector.snapshotCount
	ch <- collector.configChanged
	ch <- collector.interfaceSpeed
	ch <- collector.cpuMaxCount
	ch <- collector.memoryMax
	ch <- collector.disksAvailableSpace
	ch <- collector.autostartEnabled
	ch <- collector.uefiEnabled
	ch <- collector.latestQemu
	ch <- collector.disabled
	ch <- collector.snapshotAllowed
	ch <- collector.maintenanceStart
	ch <- collector.maintenanceFinish
	ch <- collector.taskInfo
	ch <- collector.tasksPending
	ch <- collector.apiUp
}

// Collect implements prometheus.Collect for ScpCollector
func (collector *ScpCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	// API Ping
	var apiUp float64
	pResp, err := collector.client.GetApiPingWithResponse(ctx)
	if err == nil && pResp.StatusCode() == http.StatusOK {
		apiUp = 1
	}
	ch <- prometheus.MustNewConstMetric(collector.apiUp, prometheus.GaugeValue, apiUp)

	// Maintenance info
	mResp, err := collector.client.GetApiV1MaintenanceWithResponse(ctx)
	if err != nil {
		collector.logger.Error("Unable to get maintenance information", "error", err.Error())
	} else if mResp.JSON200 != nil {
		if mResp.JSON200.StartAt != nil {
			ch <- prometheus.MustNewConstMetric(collector.maintenanceStart, prometheus.GaugeValue, float64(mResp.JSON200.StartAt.Unix()))
		}
		if mResp.JSON200.FinishAt != nil {
			ch <- prometheus.MustNewConstMetric(collector.maintenanceFinish, prometheus.GaugeValue, float64(mResp.JSON200.FinishAt.Unix()))
		}
	}

	resp, err := collector.client.GetApiV1ServersWithResponse(ctx, &scpclient.GetApiV1ServersParams{})
	if err != nil {
		collector.logger.Error("Unable to get servers", "error", err.Error())
		return
	}

	if resp.JSON200 == nil {
		collector.logger.Error("Unable to get servers", "status", resp.Status())
		return
	}

	// Tasks info
	tResp, err := collector.client.GetApiV1TasksWithResponse(ctx, &scpclient.GetApiV1TasksParams{})
	if err != nil {
		collector.logger.Error("Unable to get tasks", "error", err.Error())
	} else if tResp.JSON200 != nil {
		var pendingCount float64
		for _, task := range *tResp.JSON200 {
			state := ""
			if task.State != nil {
				state = string(*task.State)
				if *task.State == scpclient.TaskStatePENDING || *task.State == scpclient.TaskStateRUNNING {
					pendingCount++
				}
			}
			uuid := ""
			if task.Uuid != nil {
				uuid = *task.Uuid
			}
			name := ""
			if task.Name != nil {
				name = *task.Name
			}
			ch <- prometheus.MustNewConstMetric(collector.taskInfo, prometheus.GaugeValue, 1, uuid, name, state)
		}
		ch <- prometheus.MustNewConstMetric(collector.tasksPending, prometheus.GaugeValue, pendingCount)
	}

	now := time.Now()
	month := strconv.Itoa(int(now.Month()))
	year := strconv.Itoa(now.Year())

	for _, s := range *resp.JSON200 {
		serverID := s.Id
		vserverName := ""
		if s.Name != nil {
			vserverName = *s.Name
		}
		nickname := ""
		if s.Nickname != nil {
			nickname = *s.Nickname
		}

		infoResp, err := collector.client.GetApiV1ServersServerIdWithResponse(ctx, *serverID, &scpclient.GetApiV1ServersServerIdParams{})
		if err != nil {
			collector.logger.Error("Unable to get Server Information", "vserver", vserverName, "error", err.Error())
			continue
		}

		if infoResp.JSON200 == nil {
			collector.logger.Error("Unable to get Server Information", "vserver", vserverName, "status", infoResp.Status())
			continue
		}

		server := infoResp.JSON200
		liveInfo := server.ServerLiveInfo

		if server.Disabled != nil {
			var disabled float64
			if *server.Disabled {
				disabled = 1
			}
			ch <- prometheus.MustNewConstMetric(collector.disabled, prometheus.GaugeValue, disabled, vserverName)
		}

		if server.MaxCpuCount != nil {
			ch <- prometheus.MustNewConstMetric(collector.cpuMaxCount, prometheus.GaugeValue, float64(*server.MaxCpuCount), vserverName)
		}

		if server.DisksAvailableSpaceInMiB != nil {
			ch <- prometheus.MustNewConstMetric(collector.disksAvailableSpace, prometheus.GaugeValue, float64(*server.DisksAvailableSpaceInMiB*1024*1024), vserverName)
		}

		if server.SnapshotAllowed != nil {
			var allowed float64
			if *server.SnapshotAllowed {
				allowed = 1
			}
			ch <- prometheus.MustNewConstMetric(collector.snapshotAllowed, prometheus.GaugeValue, allowed, vserverName)
		}

		if server.SnapshotCount != nil {
			ch <- prometheus.MustNewConstMetric(collector.snapshotCount, prometheus.GaugeValue, float64(*server.SnapshotCount), vserverName)
		}

		if liveInfo != nil {
			// Create CPU / Memory info metrics
			if liveInfo.CpuCount != nil {
				ch <- prometheus.MustNewConstMetric(collector.cpuCores, prometheus.GaugeValue, float64(*liveInfo.CpuCount), vserverName)
			}
			if liveInfo.CurrentServerMemoryInMiB != nil {
				ch <- prometheus.MustNewConstMetric(collector.memory, prometheus.GaugeValue, float64(*liveInfo.CurrentServerMemoryInMiB*1024*1024), vserverName)
			}
			if liveInfo.MaxServerMemoryInMiB != nil {
				ch <- prometheus.MustNewConstMetric(collector.memoryMax, prometheus.GaugeValue, float64(*liveInfo.MaxServerMemoryInMiB*1024*1024), vserverName)
			}

			if liveInfo.Autostart != nil {
				var autostart float64
				if *liveInfo.Autostart {
					autostart = 1
				}
				ch <- prometheus.MustNewConstMetric(collector.autostartEnabled, prometheus.GaugeValue, autostart, vserverName)
			}

			if liveInfo.Uefi != nil {
				var uefi float64
				if *liveInfo.Uefi {
					uefi = 1
				}
				ch <- prometheus.MustNewConstMetric(collector.uefiEnabled, prometheus.GaugeValue, uefi, vserverName)
			}

			if liveInfo.LatestQemu != nil {
				var latestQemu float64
				if *liveInfo.LatestQemu {
					latestQemu = 1
				}
				ch <- prometheus.MustNewConstMetric(collector.latestQemu, prometheus.GaugeValue, latestQemu, vserverName)
			}

			if liveInfo.ConfigChanged != nil {
				var changed float64
				if *liveInfo.ConfigChanged {
					changed = 1
				}
				ch <- prometheus.MustNewConstMetric(collector.configChanged, prometheus.GaugeValue, changed, vserverName)
			}

			// Create traffic metrics
			var totalIn, totalOut float64
			if liveInfo.Interfaces != nil {
				for _, iface := range *liveInfo.Interfaces {
					if iface.RxMonthlyInMiB != nil {
						totalIn += float64(*iface.RxMonthlyInMiB) * 1024 * 1024
					}
					if iface.TxMonthlyInMiB != nil {
						totalOut += float64(*iface.TxMonthlyInMiB) * 1024 * 1024
					}
				}
			}
			ch <- prometheus.MustNewConstMetric(collector.monthlyTrafficIn, prometheus.GaugeValue, totalIn, vserverName, month, year)
			ch <- prometheus.MustNewConstMetric(collector.monthlyTrafficOut, prometheus.GaugeValue, totalOut, vserverName, month, year)
			ch <- prometheus.MustNewConstMetric(collector.monthlyTrafficTotal, prometheus.GaugeValue, totalIn+totalOut, vserverName, month, year)

			// Create server status metric
			var online float64
			status := ""
			if liveInfo.State != nil {
				status = string(*liveInfo.State)
				if *liveInfo.State == scpclient.RUNNING {
					online = 1
				}
			}
			arch := ""
			if server.Architecture != nil {
				arch = string(*server.Architecture)
			}
			city := ""
			if server.Site != nil {
				city = server.Site.City
			}
			ch <- prometheus.MustNewConstMetric(collector.serverStatus, prometheus.GaugeValue, online, vserverName, status, nickname, arch, city)

			// Create start time metric
			if liveInfo.UptimeInSeconds != nil {
				startTime := now.Add(-time.Duration(*liveInfo.UptimeInSeconds) * time.Second)
				ch <- prometheus.MustNewConstMetric(collector.serverStartTime, prometheus.GaugeValue, float64(startTime.Unix()), vserverName)
			}

			// Create Interface throttling metric
			if liveInfo.Interfaces != nil {
				for _, iface := range *liveInfo.Interfaces {
					var throttled float64
					if iface.TrafficThrottled != nil && *iface.TrafficThrottled {
						throttled = 1
					}
					mac := ""
					if iface.Mac != nil {
						mac = *iface.Mac
					}
					driver := ""
									if iface.Driver != nil {
										driver = *iface.Driver
									}
					
									if iface.SpeedInMBits != nil {
										ch <- prometheus.MustNewConstMetric(collector.interfaceSpeed, prometheus.GaugeValue, float64(*iface.SpeedInMBits), vserverName, mac, driver)
									}
					
									if iface.Ipv4Addresses != nil {
					
						for _, ip := range *iface.Ipv4Addresses {
							ch <- prometheus.MustNewConstMetric(collector.ifaceThrottled, prometheus.GaugeValue, throttled, vserverName, driver, "", ip, "ipv4", mac, "")
						}
					}
					if iface.Ipv6LinkLocalAddresses != nil {
						for _, ip := range *iface.Ipv6LinkLocalAddresses {
							ch <- prometheus.MustNewConstMetric(collector.ifaceThrottled, prometheus.GaugeValue, throttled, vserverName, driver, "", ip, "ipv6", mac, "")
						}
					}
					if iface.Ipv6NetworkPrefixes != nil {
						for _, prefix := range *iface.Ipv6NetworkPrefixes {
							ch <- prometheus.MustNewConstMetric(collector.ifaceThrottled, prometheus.GaugeValue, throttled, vserverName, driver, "", prefix, "ipv6", mac, "")
						}
					}
				}
			}

			// Create Disk metrics
			if liveInfo.Disks != nil {
				for _, disk := range *liveInfo.Disks {
					dev := ""
					if disk.Dev != nil {
						dev = *disk.Dev
					}
					driver := ""
					if disk.Driver != nil {
						driver = *disk.Driver
					}
					capacity := float64(0)
					if disk.CapacityInMiB != nil {
						capacity = float64(*disk.CapacityInMiB) * 1024 * 1024
					}
					allocation := float64(0)
					if disk.AllocationInMiB != nil {
						allocation = float64(*disk.AllocationInMiB) * 1024 * 1024
					}

					ch <- prometheus.MustNewConstMetric(collector.diskCapacity, prometheus.GaugeValue, capacity, vserverName, driver, dev)
					ch <- prometheus.MustNewConstMetric(collector.diskUsed, prometheus.GaugeValue, allocation, vserverName, driver, dev)

					var optimize float64
					msg := ""
					if liveInfo.RequiredStorageOptimization != nil && *liveInfo.RequiredStorageOptimization != scpclient.NO {
						optimize = 1
						msg = string(*liveInfo.RequiredStorageOptimization)
					}
					ch <- prometheus.MustNewConstMetric(collector.diskOptimization, prometheus.GaugeValue, optimize, vserverName, driver, dev, msg)
				}
			}
		}

		// Create rescue active metric
		var rescue float64
		if server.RescueSystemActive != nil && *server.RescueSystemActive {
			rescue = 1
		}
		ch <- prometheus.MustNewConstMetric(collector.rescueActive, prometheus.GaugeValue, rescue, vserverName, "")

		// Create IP info metric
		if server.Ipv4Addresses != nil {
			for _, ip := range *server.Ipv4Addresses {
				if ip.Ip != nil {
					ch <- prometheus.MustNewConstMetric(collector.ipInfo, prometheus.GaugeValue, 1, vserverName, *ip.Ip)
				}
			}
		}
		if server.Ipv6Addresses != nil {
			for _, ip := range *server.Ipv6Addresses {
				if ip.NetworkPrefix != nil {
					ch <- prometheus.MustNewConstMetric(collector.ipInfo, prometheus.GaugeValue, 1, vserverName, *ip.NetworkPrefix)
				}
			}
		}
	}
}
