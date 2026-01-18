package display

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rafaelmarinho/pulsecheck/internal/registry"
)

func TestNewReporter(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, false)

	if reporter == nil {
		t.Fatal("NewReporter() returned nil")
	}

	if reporter.monitor != monitor {
		t.Error("NewReporter() monitor mismatch")
	}

	if reporter.jsonMode != false {
		t.Error("NewReporter() jsonMode = true, want false")
	}

	if reporter.output == nil {
		t.Error("NewReporter() output is nil")
	}
}

func TestReporterHumanOutput(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, false)

	// Capture output
	var buf bytes.Buffer
	reporter.output = &buf

	// Add a node
	monitor.UpdateWithTelemetry("192.168.1.100:9999", 75.5, 80.2, 85.1, 1)

	reporter.Report()

	output := buf.String()

	// Verify output contains expected elements
	if !strings.Contains(output, "PulseCheck Status") {
		t.Error("Human output missing 'PulseCheck Status'")
	}

	if !strings.Contains(output, "192.168.1.100:9999") {
		t.Error("Human output missing node address")
	}

	if !strings.Contains(output, "WARN") {
		t.Error("Human output missing status")
	}
}

func TestReporterJSONOutput(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, true)

	// Capture output
	var buf bytes.Buffer
	reporter.output = &buf

	// Add nodes with telemetry
	monitor.UpdateWithTelemetry("192.168.1.100:9999", 75.5, 80.2, 85.1, 1)
	monitor.UpdateWithTelemetry("192.168.1.101:9999", 45.0, 60.0, 70.0, 0)

	reporter.Report()

	output := buf.String()

	// Verify it's valid JSON
	var report StatusReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}

	// Verify structure
	if report.NodeCount != 2 {
		t.Errorf("StatusReport.NodeCount = %d, want 2", report.NodeCount)
	}

	if len(report.Nodes) != 2 {
		t.Errorf("StatusReport.Nodes length = %d, want 2", len(report.Nodes))
	}

	// Verify node data
	node1, ok := report.Nodes["192.168.1.100:9999"]
	if !ok {
		t.Error("StatusReport.Nodes missing node 192.168.1.100:9999")
	} else {
		if node1.Status != "WARN" {
			t.Errorf("Node1.Status = %s, want WARN", node1.Status)
		}
		if node1.StatusCode != 1 {
			t.Errorf("Node1.StatusCode = %d, want 1", node1.StatusCode)
		}
		if node1.CPUPercent != 75.5 {
			t.Errorf("Node1.CPUPercent = %f, want 75.5", node1.CPUPercent)
		}
	}

	node2, ok := report.Nodes["192.168.1.101:9999"]
	if !ok {
		t.Error("StatusReport.Nodes missing node 192.168.1.101:9999")
	} else {
		if node2.Status != "OK" {
			t.Errorf("Node2.Status = %s, want OK", node2.Status)
		}
		if node2.StatusCode != 0 {
			t.Errorf("Node2.StatusCode = %d, want 0", node2.StatusCode)
		}
	}
}

func TestReporterEmptyNodes(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, false)

	var buf bytes.Buffer
	reporter.output = &buf

	reporter.Report()

	output := buf.String()

	if !strings.Contains(output, "No active nodes") {
		t.Error("Empty nodes output missing 'No active nodes'")
	}
}

func TestReporterJSONEmptyNodes(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, true)

	var buf bytes.Buffer
	reporter.output = &buf

	reporter.Report()

	output := buf.String()

	var report StatusReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}

	if report.NodeCount != 0 {
		t.Errorf("StatusReport.NodeCount = %d, want 0", report.NodeCount)
	}

	if len(report.Nodes) != 0 {
		t.Errorf("StatusReport.Nodes length = %d, want 0", len(report.Nodes))
	}
}

func TestStatusCodeToString(t *testing.T) {
	testCases := []struct {
		code uint8
		want string
	}{
		{0, "OK"},
		{1, "WARN"},
		{2, "CRITICAL"},
		{99, "UNKNOWN"},
	}

	for _, tc := range testCases {
		t.Run(tc.want, func(t *testing.T) {
			result := statusCodeToString(tc.code)
			if result != tc.want {
				t.Errorf("statusCodeToString(%d) = %s, want %s", tc.code, result, tc.want)
			}
		})
	}
}

func TestReporterStop(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, false)

	// Start reporter in goroutine
	done := make(chan bool)
	go func() {
		reporter.Start(10 * time.Millisecond)
		done <- true
	}()

	// Stop after a short delay
	time.Sleep(50 * time.Millisecond)
	reporter.Stop()

	// Wait for goroutine to finish
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Reporter did not stop within timeout")
	}
}

func TestReporterJSONTimestamp(t *testing.T) {
	monitor := registry.NewMonitor()
	reporter := NewReporter(monitor, true)

	var buf bytes.Buffer
	reporter.output = &buf

	before := time.Now()
	reporter.Report()
	after := time.Now()

	var report StatusReport
	if err := json.Unmarshal([]byte(buf.Bytes()), &report); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}

	// Timestamp should be between before and after
	if report.Timestamp.Before(before) || report.Timestamp.After(after) {
		t.Errorf("StatusReport.Timestamp = %v, should be between %v and %v",
			report.Timestamp, before, after)
	}
}
