package registry

import (
	"hash/fnv"
	"log"
	"sync"
	"time"
)

const (
	// numShards is the number of shards for the sharded map
	// Using a power of 2 (16) allows efficient modulo operation via bitwise AND
	numShards = 16
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

// shard represents a single shard of the sharded map
type shard struct {
	nodes map[string]NodeInfo
	mu    sync.RWMutex
}

// Monitor uses a sharded map to reduce lock contention
// Operations on different shards can proceed concurrently
type Monitor struct {
	shards [numShards]*shard
}

// NewMonitor creates a new monitor instance with sharded map
func NewMonitor() *Monitor {
	m := &Monitor{}
	for i := 0; i < numShards; i++ {
		m.shards[i] = &shard{
			nodes: make(map[string]NodeInfo),
		}
	}
	return m
}

// getShard returns the shard for a given address
// Uses FNV-1a hash for good distribution
func (m *Monitor) getShard(addr string) *shard {
	h := fnv.New32a()
	h.Write([]byte(addr))
	// Use bitwise AND instead of modulo for efficiency (numShards is power of 2)
	shardIndex := h.Sum32() & (numShards - 1)
	return m.shards[shardIndex]
}

// Update updates the heartbeat for a node
func (m *Monitor) Update(addr string) {
	shard := m.getShard(addr)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.nodes == nil {
		shard.nodes = make(map[string]NodeInfo)
	}
	shard.nodes[addr] = NodeInfo{
		LastSeen: time.Now(),
		Address:  addr,
	}
}

// UpdateWithStatus updates the heartbeat with status code and timestamp
// Uses local time.Now() for LastSeen to handle clock skew, but stores packet timestamp for RTT
func (m *Monitor) UpdateWithStatus(addr string, statusCode uint8, packetTimestamp int64) {
	shard := m.getShard(addr)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.nodes == nil {
		shard.nodes = make(map[string]NodeInfo)
	}

	now := time.Now()
	info := shard.nodes[addr]

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

	shard.nodes[addr] = info
}

// UpdateWithTelemetry updates the heartbeat with full telemetry data
func (m *Monitor) UpdateWithTelemetry(addr string, cpuPercent, ramPercent, diskPercent float64, statusCode uint8) {
	shard := m.getShard(addr)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.nodes == nil {
		shard.nodes = make(map[string]NodeInfo)
	}
	shard.nodes[addr] = NodeInfo{
		LastSeen:    time.Now(),
		Address:     addr,
		CPUPercent:  cpuPercent,
		RAMPercent:  ramPercent,
		DiskPercent: diskPercent,
		StatusCode:  statusCode,
	}
}

// GetNodes returns a copy of all known nodes from all shards
func (m *Monitor) GetNodes() map[string]NodeInfo {
	// Lock all shards for reading (could be optimized with concurrent reads)
	result := make(map[string]NodeInfo)

	for i := 0; i < numShards; i++ {
		shard := m.shards[i]
		shard.mu.RLock()
		for k, v := range shard.nodes {
			result[k] = v
		}
		shard.mu.RUnlock()
	}

	return result
}

// GetNodeCount returns the total number of active nodes across all shards
func (m *Monitor) GetNodeCount() int {
	total := 0
	for i := 0; i < numShards; i++ {
		shard := m.shards[i]
		shard.mu.RLock()
		total += len(shard.nodes)
		shard.mu.RUnlock()
	}
	return total
}

// GetNodeInfo returns information about a specific node
func (m *Monitor) GetNodeInfo(addr string) (NodeInfo, bool) {
	shard := m.getShard(addr)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	info, ok := shard.nodes[addr]
	return info, ok
}

// StartReaper runs in a goroutine to remove stale nodes
// With sharded map, reaper processes each shard independently, reducing lock contention
func (m *Monitor) StartReaper(interval time.Duration, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		// Process each shard independently - allows concurrent operations on other shards
		for i := 0; i < numShards; i++ {
			shard := m.shards[i]
			shard.mu.Lock()
			for addr, info := range shard.nodes {
				if time.Since(info.LastSeen) > timeout {
					delete(shard.nodes, addr)
					log.Printf("Node %s timed out", addr)
				}
			}
			shard.mu.Unlock()
		}
	}
}
