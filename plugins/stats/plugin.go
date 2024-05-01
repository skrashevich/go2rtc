package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/AlexxIT/go2rtc/internal/api"
	"github.com/AlexxIT/go2rtc/pkg/plugin"
)

type StatsPlugin struct{}

type HostInfo struct {
	Platform string `json:"platform"`
	Family   string `json:"family"`
	Version  string `json:"version"`
}

func Init() plugin.Plugin {

	return &StatsPlugin{}
}

func (p *StatsPlugin) Start() {

	fmt.Println("Плагин запущен")
}

func (p *StatsPlugin) RegisterHandlers() {
	api.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]any{}

		cpuUsage, err := GetCPUUsage(0)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Failed to get CPU usage."}`))
			fmt.Printf("[plugin-stats] cpu stat: %v", err)
			return
		}

		memUsage, err := GetRAMUsage()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Failed to get memory usage."}`))
			fmt.Printf("[plugin-stats] ram stat: %v", err)
			return
		}

		hostInfo, err := GetHostInfo()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Failed to get CPU usage."}`))
			//log.Warn().Err(err).Msg("[api] cpu stat")
			return
		}
		stats["cpu"] = cpuUsage
		stats["mem"] = memUsage
		stats["host"] = hostInfo
		ResponseJSON(w, stats)
	})
}

func ResponseJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

var New StatsPlugin

// GetCPUUsage calculates the CPU usage percentage over a specified interval.
// It returns the average CPU usage as a float64 and any error encountered.
//
// The function works by first fetching the CPU usage percentage before the sleep interval,
// then calculating the average CPU usage over that interval. This is useful for monitoring
// or logging system performance metrics.
//
// Parameters:
// - interval: A time.Duration value specifying how long to measure CPU usage for.
//
// Returns:
// - A float64 representing the average CPU usage percentage over the interval.
// - An error if there was an issue fetching the CPU usage data.
func GetCPUUsage(interval time.Duration) (float64, error) {
	percentages, err := cpu.Percent(interval, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU usage data available")
	}

	var total float64
	for _, percent := range percentages {
		total += percent
	}
	avgCPUUsage := total / float64(len(percentages))

	return avgCPUUsage, nil
}

// GetRAMUsage fetches the current virtual memory statistics.
// It returns a pointer to a VirtualMemoryStat struct containing detailed memory usage stats
// and any error encountered.
//
// This function is useful for retrieving comprehensive memory usage data, such as total and available RAM,
// used and free amounts, and various other metrics related to system memory performance.
//
// Returns:
// - A pointer to a mem.VirtualMemoryStat struct containing the memory usage statistics.
// - An error if there was an issue fetching the memory usage data.
func GetRAMUsage() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}
func GetHostInfo() (HostInfo, error) {

	platform, family, version, err := host.PlatformInformation()

	return HostInfo{
		Platform: platform,
		Family:   family,
		Version:  version,
	}, err
}
