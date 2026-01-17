package main

import (
	"crypto/rand"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafaelmarinho/pulsecheck/internal/display"
	"github.com/rafaelmarinho/pulsecheck/internal/registry"
	"github.com/rafaelmarinho/pulsecheck/internal/telemetry"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 9999, "UDP port to listen on")
	heartbeatInterval := flag.Duration("heartbeat-interval", 5*time.Second, "Time between heartbeats")
	timeout := flag.Duration("timeout", 15*time.Second, "Time before marking node offline")
	nodeID := flag.String("node-id", "", "Unique identifier for this node (default: hostname)")
	seedNode := flag.String("seed-node", "", "Seed node address (e.g., 192.168.1.100:9999) for peer discovery")
	jsonOutput := flag.Bool("json", false, "Output status in JSON format (for tool consumption)")
	
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
	
	// Connect to seed node if provided (for peer discovery)
	if *seedNode != "" {
		// Collect initial metrics for seed node connection
		metrics, err := telemetry.CollectMetrics()
		if err != nil {
			log.Printf("Warning: Failed to collect metrics for seed node: %v", err)
			metrics = &telemetry.Metrics{} // Use zero values
		}
		statusCode := telemetry.CalculateStatus(metrics, thresholds)
		
		// Send initial heartbeat to seed node
		if err := udpNode.SendToSeedNode(*seedNode, uint8(statusCode)); err != nil {
			log.Printf("Warning: Failed to connect to seed node %s: %v", *seedNode, err)
			log.Println("Continuing without seed node - peer discovery may be limited")
		} else {
			log.Printf("Connected to seed node: %s", *seedNode)
		}
	}
	
	// Start reaper goroutine
	go monitor.StartReaper(1*time.Second, *timeout)
	
	// Initialize status reporter
	reporter := display.NewReporter(monitor, *jsonOutput)
	go reporter.Start(10 * time.Second)
	defer reporter.Stop()
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Start heartbeat ticker
	heartbeatTicker := time.NewTicker(*heartbeatInterval)
	defer heartbeatTicker.Stop()
	
	log.Printf("PulseCheck node started (UUID: %x, Port: %d)", nodeUUID, *port)
	log.Printf("Heartbeat interval: %v, Timeout: %v", *heartbeatInterval, *timeout)
	if *seedNode != "" {
		log.Printf("Seed node: %s", *seedNode)
	}
	if *jsonOutput {
		log.Println("JSON output mode enabled")
	}
	
	// Main loop - handles heartbeat and shutdown
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

