package registry

import (
	"log"
	"sync"
	"time"
)

type NodeInfo struct {
	LastSeen    time.Time // Local time when packet was received (handles clock skew)
	Address     string
	CPUPercent  float64
	RAMPercent  float64
	DiskPercent float64
	StatusCode  uint8
	PacketTime  int64         // Sender's timestamp (for RTT calculation)
	RTT         time.Duration // Calculated round-trip time
}

type Monitor struct {
	nodes map[string]NodeInfo
	mu    sync.RWMutex
}

// NewMonitor creates a new monitor instance
func NewMonitor() *Monitor {
	return &Monitor{
		nodes: make(map[string]NodeInfo),
	}
}

// Update updates the heartbeat for a node
func (m *Monitor) Update(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodes == nil {
		m.nodes = make(map[string]NodeInfo)
	}
	m.nodes[addr] = NodeInfo{
		LastSeen: time.Now(),
		Address:  addr,
	}
}

// UpdateWithStatus updates the heartbeat with status code and timestamp
// Uses local time.Now() for LastSeen to handle clock skew, but stores packet timestamp for RTT
func (m *Monitor) UpdateWithStatus(addr string, statusCode uint8, packetTimestamp int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodes == nil {
		m.nodes = make(map[string]NodeInfo)
	}

	now := time.Now()
	info := m.nodes[addr]

	// Use local time for LastSeen to handle clock skew between nodes
	// This ensures reaper logic works correctly even with time differences
	info.LastSeen = now
	info.Address = addr
	info.StatusCode = statusCode
	info.PacketTime = packetTimestamp

	// Calculate RTT estimation
	// Note: True RTT requires echo packets, but we can estimate based on clock differences
	// For now, we store the packet timestamp for potential future RTT calculation
	// The packet timestamp can be used to detect clock skew and estimate network latency
	if packetTimestamp > 0 {
		// Store packet timestamp for RTT/latency analysis
		// In a production system, you might implement echo packets for true RTT
	}

	m.nodes[addr] = info
}

// UpdateWithTelemetry updates the heartbeat with full telemetry data
func (m *Monitor) UpdateWithTelemetry(addr string, cpuPercent, ramPercent, diskPercent float64, statusCode uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodes == nil {
		m.nodes = make(map[string]NodeInfo)
	}
	m.nodes[addr] = NodeInfo{
		LastSeen:    time.Now(),
		Address:     addr,
		CPUPercent:  cpuPercent,
		RAMPercent:  ramPercent,
		DiskPercent: diskPercent,
		StatusCode:  statusCode,
	}
}

// GetNodes returns a copy of all known nodes
func (m *Monitor) GetNodes() map[string]NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]NodeInfo, len(m.nodes))
	for k, v := range m.nodes {
		result[k] = v
	}
	return result
}

// GetNodeCount returns the number of active nodes
func (m *Monitor) GetNodeCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.nodes)
}

// GetNodeInfo returns information about a specific node
func (m *Monitor) GetNodeInfo(addr string) (NodeInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.nodes[addr]
	return info, ok
}

// StartReaper runs in a goroutine to remove stale nodes
func (m *Monitor) StartReaper(interval time.Duration, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		m.mu.Lock()
		for addr, info := range m.nodes {
			if time.Since(info.LastSeen) > timeout {
				delete(m.nodes, addr)
				log.Printf("Node %s timed out", addr)
			}
		}
		m.mu.Unlock()
	}
}
