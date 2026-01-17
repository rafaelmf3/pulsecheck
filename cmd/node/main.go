package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafaelmarinho/pulsecheck/internal/registry"
	"github.com/rafaelmarinho/pulsecheck/internal/telemetry"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 9999, "UDP port to listen on")
	heartbeatInterval := flag.Duration("heartbeat-interval", 5*time.Second, "Time between heartbeats")
	timeout := flag.Duration("timeout", 15*time.Second, "Time before marking node offline")
	nodeID := flag.String("node-id", "", "Unique identifier for this node (default: hostname)")
	
	// Telemetry thresholds
	cpuWarn := flag.Float64("cpu-warn-threshold", 70.0, "CPU percentage for Warn status")
	cpuCritical := flag.Float64("cpu-critical-threshold", 90.0, "CPU percentage for Critical status")
	ramWarn := flag.Float64("ram-warn-threshold", 80.0, "RAM percentage for Warn status")
	ramCritical := flag.Float64("ram-critical-threshold", 95.0, "RAM percentage for Critical status")
	diskWarn := flag.Float64("disk-warn-threshold", 85.0, "Disk percentage for Warn status")
	diskCritical := flag.Float64("disk-critical-threshold", 95.0, "Disk percentage for Critical status")
	
	flag.Parse()
	
	// Generate or use node UUID
	nodeUUID := generateNodeUUID(*nodeID)
	
	// Create thresholds
	thresholds := telemetry.Thresholds{
		CPUWarn:     *cpuWarn,
		CPUCritical: *cpuCritical,
		RAMWarn:     *ramWarn,
		RAMCritical: *ramCritical,
		DiskWarn:    *diskWarn,
		DiskCritical: *diskCritical,
	}
	
	// Initialize monitor
	monitor := registry.NewMonitor()
	
	// Create UDP node
	udpNode, err := registry.NewUDPNode(*port, nodeUUID, monitor)
	if err != nil {
		log.Fatalf("Failed to create UDP node: %v", err)
	}
	
	// Start UDP listener in background
	go udpNode.Start()
	
	// Start reaper goroutine
	go monitor.StartReaper(1*time.Second, *timeout)
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Start heartbeat ticker
	heartbeatTicker := time.NewTicker(*heartbeatInterval)
	defer heartbeatTicker.Stop()
	
	// Start status display ticker
	statusTicker := time.NewTicker(10 * time.Second)
	defer statusTicker.Stop()
	
	log.Printf("PulseCheck node started (UUID: %x, Port: %d)", nodeUUID, *port)
	log.Printf("Heartbeat interval: %v, Timeout: %v", *heartbeatInterval, *timeout)
	
	// Main loop
	for {
		select {
		case <-sigChan:
			log.Println("Shutting down...")
			udpNode.Stop()
			return
			
		case <-heartbeatTicker.C:
			// Collect telemetry
			metrics, err := telemetry.CollectMetrics()
			if err != nil {
				log.Printf("Failed to collect metrics: %v", err)
				continue
			}
			
			// Calculate status
			statusCode := telemetry.CalculateStatus(metrics, thresholds)
			
			// Update local monitor with telemetry (use local address)
			localAddr := udpNode.Conn().LocalAddr().String()
			monitor.UpdateWithTelemetry(
				localAddr,
				metrics.CPUPercent,
				metrics.RAMPercent,
				metrics.DiskPercent,
				uint8(statusCode),
			)
			
			// Broadcast heartbeat
			if err := udpNode.BroadcastHeartbeat(uint8(statusCode)); err != nil {
				log.Printf("Failed to broadcast heartbeat: %v", err)
			}
			
		case <-statusTicker.C:
			// Display status
			displayStatus(monitor)
		}
	}
}

// generateNodeUUID generates a 16-byte UUID from node ID or random
func generateNodeUUID(nodeID string) [16]byte {
	var uuid [16]byte
	
	if nodeID == "" {
		// Use hostname
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		nodeID = hostname
	}
	
	// Generate UUID from node ID (simple hash-based approach)
	// For production, consider using a proper UUID library
	hash := simpleHash(nodeID)
	copy(uuid[:], hash[:16])
	
	// Fill remaining bytes with random if needed
	if len(nodeID) < 16 {
		rand.Read(uuid[len(nodeID):])
	}
	
	return uuid
}

// simpleHash creates a simple hash from string
func simpleHash(s string) []byte {
	hash := make([]byte, 16)
	for i := 0; i < len(s) && i < 16; i++ {
		hash[i] = s[i]
	}
	return hash
}

// displayStatus displays the current status of all nodes
func displayStatus(monitor *registry.Monitor) {
	nodes := monitor.GetNodes()
	count := monitor.GetNodeCount()
	
	fmt.Printf("\n=== PulseCheck Status (Nodes: %d) ===\n", count)
	
	if count == 0 {
		fmt.Println("No active nodes")
		return
	}
	
	for addr, info := range nodes {
		statusStr := "OK"
		switch info.StatusCode {
		case 1:
			statusStr = "WARN"
		case 2:
			statusStr = "CRITICAL"
		}
		
		age := time.Since(info.LastSeen)
		fmt.Printf("Node: %s | Status: %s | Age: %v", addr, statusStr, age.Round(time.Second))
		if info.CPUPercent > 0 || info.RAMPercent > 0 || info.DiskPercent > 0 {
			fmt.Printf(" | CPU: %.1f%% RAM: %.1f%% Disk: %.1f%%", 
				info.CPUPercent, info.RAMPercent, info.DiskPercent)
		}
		fmt.Println()
	}
}
