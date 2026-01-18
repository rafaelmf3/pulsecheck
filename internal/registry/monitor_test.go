package registry

import (
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m := NewMonitor()

	if m == nil {
		t.Fatal("NewMonitor() returned nil")
	}

	// Verify all shards are initialized
	for i := 0; i < numShards; i++ {
		if m.shards[i] == nil {
			t.Errorf("NewMonitor() shard[%d] is nil", i)
		}
		if m.shards[i].nodes == nil {
			t.Errorf("NewMonitor() shard[%d].nodes is nil", i)
		}
	}
}

func TestMonitorUpdate(t *testing.T) {
	m := NewMonitor()
	addr := "192.168.1.100:9999"

	m.Update(addr)

	info, ok := m.GetNodeInfo(addr)
	if !ok {
		t.Fatal("GetNodeInfo() returned false after Update()")
	}

	if info.Address != addr {
		t.Errorf("GetNodeInfo() Address = %s, want %s", info.Address, addr)
	}

	if time.Since(info.LastSeen) > time.Second {
		t.Error("GetNodeInfo() LastSeen is too old")
	}
}

func TestMonitorUpdateWithStatus(t *testing.T) {
	m := NewMonitor()
	addr := "192.168.1.200:9999"
	statusCode := uint8(1)
	timestamp := time.Now().UnixNano()

	m.UpdateWithStatus(addr, statusCode, timestamp)

	info, ok := m.GetNodeInfo(addr)
	if !ok {
		t.Fatal("GetNodeInfo() returned false after UpdateWithStatus()")
	}

	if info.StatusCode != statusCode {
		t.Errorf("GetNodeInfo() StatusCode = %d, want %d", info.StatusCode, statusCode)
	}

	if info.PacketTime != timestamp {
		t.Errorf("GetNodeInfo() PacketTime = %d, want %d", info.PacketTime, timestamp)
	}

	// LastSeen should be recent (local time)
	if time.Since(info.LastSeen) > time.Second {
		t.Error("GetNodeInfo() LastSeen is too old")
	}
}

func TestMonitorUpdateWithTelemetry(t *testing.T) {
	m := NewMonitor()
	addr := "192.168.1.300:9999"
	cpuPercent := 75.5
	ramPercent := 85.2
	diskPercent := 90.1
	statusCode := uint8(2)

	m.UpdateWithTelemetry(addr, cpuPercent, ramPercent, diskPercent, statusCode)

	info, ok := m.GetNodeInfo(addr)
	if !ok {
		t.Fatal("GetNodeInfo() returned false after UpdateWithTelemetry()")
	}

	if info.CPUPercent != cpuPercent {
		t.Errorf("GetNodeInfo() CPUPercent = %f, want %f", info.CPUPercent, cpuPercent)
	}

	if info.RAMPercent != ramPercent {
		t.Errorf("GetNodeInfo() RAMPercent = %f, want %f", info.RAMPercent, ramPercent)
	}

	if info.DiskPercent != diskPercent {
		t.Errorf("GetNodeInfo() DiskPercent = %f, want %f", info.DiskPercent, diskPercent)
	}

	if info.StatusCode != statusCode {
		t.Errorf("GetNodeInfo() StatusCode = %d, want %d", info.StatusCode, statusCode)
	}
}

func TestMonitorGetNodeCount(t *testing.T) {
	m := NewMonitor()

	if count := m.GetNodeCount(); count != 0 {
		t.Errorf("GetNodeCount() = %d, want 0", count)
	}

	// Add nodes to different shards
	for i := 0; i < 10; i++ {
		m.UpdateWithStatus("192.168.1.10:9999", 0, time.Now().UnixNano())
	}

	// Update same node (should still be 1)
	m.UpdateWithStatus("192.168.1.10:9999", 0, time.Now().UnixNano())

	if count := m.GetNodeCount(); count != 1 {
		t.Errorf("GetNodeCount() = %d, want 1", count)
	}

	// Add more nodes
	for i := 0; i < 5; i++ {
		addr := "192.168.1.10" + string(rune(i)) + ":9999"
		m.Update(addr)
	}

	if count := m.GetNodeCount(); count != 6 {
		t.Errorf("GetNodeCount() = %d, want 6", count)
	}
}

func TestMonitorGetNodes(t *testing.T) {
	m := NewMonitor()

	nodes := m.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("GetNodes() length = %d, want 0", len(nodes))
	}

	// Add multiple nodes
	testNodes := []string{
		"192.168.1.100:9999",
		"192.168.1.101:9999",
		"192.168.1.102:9999",
	}

	for _, addr := range testNodes {
		m.Update(addr)
	}

	nodes = m.GetNodes()
	if len(nodes) != len(testNodes) {
		t.Errorf("GetNodes() length = %d, want %d", len(nodes), len(testNodes))
	}

	for _, addr := range testNodes {
		if _, ok := nodes[addr]; !ok {
			t.Errorf("GetNodes() missing node %s", addr)
		}
	}
}

func TestMonitorConcurrentUpdates(t *testing.T) {
	m := NewMonitor()
	numGoroutines := 100
	numUpdates := 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numUpdates; j++ {
				addr := "192.168.1.10:9999"
				m.UpdateWithStatus(addr, uint8(id%3), time.Now().UnixNano())
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Should still have exactly one node
	if count := m.GetNodeCount(); count != 1 {
		t.Errorf("GetNodeCount() after concurrent updates = %d, want 1", count)
	}
}

func TestMonitorReaper(t *testing.T) {
	m := NewMonitor()
	shortTimeout := 100 * time.Millisecond
	reaperInterval := 50 * time.Millisecond

	// Start reaper
	go m.StartReaper(reaperInterval, shortTimeout)

	// Add a node
	addr := "192.168.1.100:9999"
	m.Update(addr)

	// Verify node exists
	if count := m.GetNodeCount(); count != 1 {
		t.Errorf("GetNodeCount() before timeout = %d, want 1", count)
	}

	// Wait for reaper to clean up
	time.Sleep(shortTimeout + reaperInterval + 50*time.Millisecond)

	// Node should be removed
	if count := m.GetNodeCount(); count != 0 {
		t.Errorf("GetNodeCount() after timeout = %d, want 0", count)
	}

	info, ok := m.GetNodeInfo(addr)
	if ok {
		t.Errorf("GetNodeInfo() after timeout = %v, want not found", info)
	}
}

func TestMonitorReaperKeepsActiveNodes(t *testing.T) {
	m := NewMonitor()
	shortTimeout := 200 * time.Millisecond
	reaperInterval := 50 * time.Millisecond

	// Start reaper
	go m.StartReaper(reaperInterval, shortTimeout)

	addr := "192.168.1.100:9999"
	m.Update(addr)

	// Keep updating the node to prevent timeout
	for i := 0; i < 5; i++ {
		time.Sleep(shortTimeout / 3)
		m.Update(addr)
	}

	// Node should still exist
	if count := m.GetNodeCount(); count != 1 {
		t.Errorf("GetNodeCount() with active updates = %d, want 1", count)
	}
}

func TestMonitorShardDistribution(t *testing.T) {
	m := NewMonitor()

	// Add many nodes to test shard distribution
	numNodes := 100
	for i := 0; i < numNodes; i++ {
		addr := "192.168.1.10" + string(rune(i)) + ":9999"
		m.Update(addr)
	}

	// Count nodes per shard
	shardCounts := make(map[int]int)
	for i := 0; i < numShards; i++ {
		m.shards[i].mu.RLock()
		shardCounts[i] = len(m.shards[i].nodes)
		m.shards[i].mu.RUnlock()
	}

	// Verify distribution (should be somewhat even)
	total := 0
	for _, count := range shardCounts {
		total += count
	}

	if total != numNodes {
		t.Errorf("Total nodes across shards = %d, want %d", total, numNodes)
	}

	// At least some shards should have nodes
	activeShards := 0
	for _, count := range shardCounts {
		if count > 0 {
			activeShards++
		}
	}

	if activeShards == 0 {
		t.Error("No shards have nodes, distribution may be broken")
	}
}
