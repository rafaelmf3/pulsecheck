package display

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rafaelmarinho/pulsecheck/internal/registry"
)

// Reporter handles status reporting in various formats
type Reporter struct {
	monitor   *registry.Monitor
	jsonMode  bool
	output    *os.File
	stopChan  chan struct{}
}

// StatusReport represents the JSON output structure
type StatusReport struct {
	Timestamp time.Time              `json:"timestamp"`
	NodeCount int                    `json:"node_count"`
	Nodes     map[string]NodeStatus  `json:"nodes"`
}

// NodeStatus represents a single node's status in JSON output
type NodeStatus struct {
	Address     string        `json:"address"`
	Status      string        `json:"status"`
	StatusCode  uint8         `json:"status_code"`
	LastSeen    time.Time     `json:"last_seen"`
	Age         string        `json:"age"`
	CPUPercent  float64       `json:"cpu_percent,omitempty"`
	RAMPercent  float64       `json:"ram_percent,omitempty"`
	DiskPercent float64       `json:"disk_percent,omitempty"`
	RTT         string        `json:"rtt,omitempty"`
}

// NewReporter creates a new status reporter
func NewReporter(monitor *registry.Monitor, jsonMode bool) *Reporter {
	return &Reporter{
		monitor:  monitor,
		jsonMode: jsonMode,
		output:   os.Stdout,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic status reporting
func (r *Reporter) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.Report()
		}
	}
}

// Stop stops the reporter
func (r *Reporter) Stop() {
	close(r.stopChan)
}

// Report outputs the current status
func (r *Reporter) Report() {
	if r.jsonMode {
		r.reportJSON()
	} else {
		r.reportHuman()
	}
}

// reportHuman outputs human-readable status
func (r *Reporter) reportHuman() {
	nodes := r.monitor.GetNodes()
	count := r.monitor.GetNodeCount()

	fmt.Fprintf(r.output, "\n=== PulseCheck Status (Nodes: %d) ===\n", count)

	if count == 0 {
		fmt.Fprintln(r.output, "No active nodes")
		return
	}

	for addr, info := range nodes {
		statusStr := statusCodeToString(info.StatusCode)
		age := time.Since(info.LastSeen)

		fmt.Fprintf(r.output, "Node: %s | Status: %s | Age: %v", 
			addr, statusStr, age.Round(time.Second))

		if info.CPUPercent > 0 || info.RAMPercent > 0 || info.DiskPercent > 0 {
			fmt.Fprintf(r.output, " | CPU: %.1f%% RAM: %.1f%% Disk: %.1f%%",
				info.CPUPercent, info.RAMPercent, info.DiskPercent)
		}

		if info.RTT > 0 {
			fmt.Fprintf(r.output, " | RTT: %v", info.RTT.Round(time.Millisecond))
		}

		fmt.Fprintln(r.output)
	}
}

// reportJSON outputs JSON-formatted status
func (r *Reporter) reportJSON() {
	nodes := r.monitor.GetNodes()
	count := r.monitor.GetNodeCount()

	report := StatusReport{
		Timestamp: time.Now(),
		NodeCount: count,
		Nodes:     make(map[string]NodeStatus, count),
	}

	for addr, info := range nodes {
		age := time.Since(info.LastSeen)
		nodeStatus := NodeStatus{
			Address:    addr,
			Status:      statusCodeToString(info.StatusCode),
			StatusCode:  info.StatusCode,
			LastSeen:   info.LastSeen,
			Age:        age.Round(time.Second).String(),
		}

		if info.CPUPercent > 0 || info.RAMPercent > 0 || info.DiskPercent > 0 {
			nodeStatus.CPUPercent = info.CPUPercent
			nodeStatus.RAMPercent = info.RAMPercent
			nodeStatus.DiskPercent = info.DiskPercent
		}

		if info.RTT > 0 {
			nodeStatus.RTT = info.RTT.Round(time.Millisecond).String()
		}

		report.Nodes[addr] = nodeStatus
	}

	encoder := json.NewEncoder(r.output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// statusCodeToString converts status code to string
func statusCodeToString(code uint8) string {
	switch code {
	case 0:
		return "OK"
	case 1:
		return "WARN"
	case 2:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}
