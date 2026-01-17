package registry

import (
	"log"
	"sync"
	"time"
)

type NodeInfo struct {
	LastSeen    time.Time
	Address     string
	CPUPercent  float64
	RAMPercent  float64
	DiskPercent float64
	StatusCode  uint8
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
func (m *Monitor) UpdateWithStatus(addr string, statusCode uint8, timestamp int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodes == nil {
		m.nodes = make(map[string]NodeInfo)
	}

	// Convert nano timestamp to time.Time
	lastSeen := time.Unix(0, timestamp)
	if timestamp == 0 {
		lastSeen = time.Now()
	}

	info := m.nodes[addr]
	info.LastSeen = lastSeen
	info.Address = addr
	info.StatusCode = statusCode
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
