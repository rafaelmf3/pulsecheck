package telemetry

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Metrics represents system resource metrics
type Metrics struct {
	CPUPercent float64
	RAMPercent float64
	DiskPercent float64
}

// Thresholds defines warning and critical thresholds for metrics
type Thresholds struct {
	CPUWarn     float64
	CPUCritical float64
	RAMWarn     float64
	RAMCritical float64
	DiskWarn    float64
	DiskCritical float64
}

// DefaultThresholds returns sensible default thresholds
func DefaultThresholds() Thresholds {
	return Thresholds{
		CPUWarn:     70.0,
		CPUCritical: 90.0,
		RAMWarn:     80.0,
		RAMCritical: 95.0,
		DiskWarn:    85.0,
		DiskCritical: 95.0,
	}
}

// StatusCode represents the health status of a node
type StatusCode uint8

const (
	StatusOK StatusCode = iota
	StatusWarn
	StatusCritical
)

// CollectMetrics gathers current system metrics
func CollectMetrics() (*Metrics, error) {
	// Collect CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, err
	}
	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	// Collect RAM usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	// Collect disk usage (root partition)
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	return &Metrics{
		CPUPercent:  cpuUsage,
		RAMPercent:  memInfo.UsedPercent,
		DiskPercent: diskInfo.UsedPercent,
	}, nil
}

// CalculateStatus determines the health status based on metrics and thresholds
func CalculateStatus(metrics *Metrics, thresholds Thresholds) StatusCode {
	// Check for critical conditions first
	if metrics.CPUPercent >= thresholds.CPUCritical ||
		metrics.RAMPercent >= thresholds.RAMCritical ||
		metrics.DiskPercent >= thresholds.DiskCritical {
		return StatusCritical
	}

	// Check for warning conditions
	if metrics.CPUPercent >= thresholds.CPUWarn ||
		metrics.RAMPercent >= thresholds.RAMWarn ||
		metrics.DiskPercent >= thresholds.DiskWarn {
		return StatusWarn
	}

	// All metrics are below warning thresholds
	return StatusOK
}
