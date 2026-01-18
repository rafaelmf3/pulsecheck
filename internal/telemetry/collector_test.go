package telemetry

import (
	"testing"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	if thresholds.CPUWarn != 70.0 {
		t.Errorf("DefaultThresholds() CPUWarn = %f, want 70.0", thresholds.CPUWarn)
	}

	if thresholds.CPUCritical != 90.0 {
		t.Errorf("DefaultThresholds() CPUCritical = %f, want 90.0", thresholds.CPUCritical)
	}

	if thresholds.RAMWarn != 80.0 {
		t.Errorf("DefaultThresholds() RAMWarn = %f, want 80.0", thresholds.RAMWarn)
	}

	if thresholds.RAMCritical != 95.0 {
		t.Errorf("DefaultThresholds() RAMCritical = %f, want 95.0", thresholds.RAMCritical)
	}

	if thresholds.DiskWarn != 85.0 {
		t.Errorf("DefaultThresholds() DiskWarn = %f, want 85.0", thresholds.DiskWarn)
	}

	if thresholds.DiskCritical != 95.0 {
		t.Errorf("DefaultThresholds() DiskCritical = %f, want 95.0", thresholds.DiskCritical)
	}
}

func TestCalculateStatusOK(t *testing.T) {
	thresholds := Thresholds{
		CPUWarn:     70.0,
		CPUCritical: 90.0,
		RAMWarn:     80.0,
		RAMCritical: 95.0,
		DiskWarn:    85.0,
		DiskCritical: 95.0,
	}

	testCases := []struct {
		name    string
		metrics *Metrics
	}{
		{"All low", &Metrics{CPUPercent: 50.0, RAMPercent: 60.0, DiskPercent: 70.0}},
		{"CPU at warn boundary", &Metrics{CPUPercent: 69.9, RAMPercent: 50.0, DiskPercent: 50.0}},
		{"RAM at warn boundary", &Metrics{CPUPercent: 50.0, RAMPercent: 79.9, DiskPercent: 50.0}},
		{"Disk at warn boundary", &Metrics{CPUPercent: 50.0, RAMPercent: 50.0, DiskPercent: 84.9}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := CalculateStatus(tc.metrics, thresholds)
			if status != StatusOK {
				t.Errorf("CalculateStatus() = %d, want %d (StatusOK)", status, StatusOK)
			}
		})
	}
}

func TestCalculateStatusWarn(t *testing.T) {
	thresholds := Thresholds{
		CPUWarn:     70.0,
		CPUCritical: 90.0,
		RAMWarn:     80.0,
		RAMCritical: 95.0,
		DiskWarn:    85.0,
		DiskCritical: 95.0,
	}

	testCases := []struct {
		name    string
		metrics *Metrics
	}{
		{"CPU at warn", &Metrics{CPUPercent: 70.0, RAMPercent: 50.0, DiskPercent: 50.0}},
		{"CPU above warn", &Metrics{CPUPercent: 75.0, RAMPercent: 50.0, DiskPercent: 50.0}},
		{"RAM at warn", &Metrics{CPUPercent: 50.0, RAMPercent: 80.0, DiskPercent: 50.0}},
		{"RAM above warn", &Metrics{CPUPercent: 50.0, RAMPercent: 85.0, DiskPercent: 50.0}},
		{"Disk at warn", &Metrics{CPUPercent: 50.0, RAMPercent: 50.0, DiskPercent: 85.0}},
		{"Disk above warn", &Metrics{CPUPercent: 50.0, RAMPercent: 50.0, DiskPercent: 90.0}},
		{"Multiple at warn", &Metrics{CPUPercent: 75.0, RAMPercent: 85.0, DiskPercent: 50.0}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := CalculateStatus(tc.metrics, thresholds)
			if status != StatusWarn {
				t.Errorf("CalculateStatus() = %d, want %d (StatusWarn)", status, StatusWarn)
			}
		})
	}
}

func TestCalculateStatusCritical(t *testing.T) {
	thresholds := Thresholds{
		CPUWarn:     70.0,
		CPUCritical: 90.0,
		RAMWarn:     80.0,
		RAMCritical: 95.0,
		DiskWarn:    85.0,
		DiskCritical: 95.0,
	}

	testCases := []struct {
		name    string
		metrics *Metrics
	}{
		{"CPU at critical", &Metrics{CPUPercent: 90.0, RAMPercent: 50.0, DiskPercent: 50.0}},
		{"CPU above critical", &Metrics{CPUPercent: 95.0, RAMPercent: 50.0, DiskPercent: 50.0}},
		{"RAM at critical", &Metrics{CPUPercent: 50.0, RAMPercent: 95.0, DiskPercent: 50.0}},
		{"RAM above critical", &Metrics{CPUPercent: 50.0, RAMPercent: 98.0, DiskPercent: 50.0}},
		{"Disk at critical", &Metrics{CPUPercent: 50.0, RAMPercent: 50.0, DiskPercent: 95.0}},
		{"Disk above critical", &Metrics{CPUPercent: 50.0, RAMPercent: 50.0, DiskPercent: 99.0}},
		{"Multiple critical", &Metrics{CPUPercent: 95.0, RAMPercent: 98.0, DiskPercent: 99.0}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := CalculateStatus(tc.metrics, thresholds)
			if status != StatusCritical {
				t.Errorf("CalculateStatus() = %d, want %d (StatusCritical)", status, StatusCritical)
			}
		})
	}
}

func TestCalculateStatusPriority(t *testing.T) {
	thresholds := Thresholds{
		CPUWarn:     70.0,
		CPUCritical: 90.0,
		RAMWarn:     80.0,
		RAMCritical: 95.0,
		DiskWarn:    85.0,
		DiskCritical: 95.0,
	}

	// Critical should take priority over Warn
	metrics := &Metrics{
		CPUPercent:  75.0, // Warn
		RAMPercent:  98.0, // Critical
		DiskPercent: 50.0, // OK
	}

	status := CalculateStatus(metrics, thresholds)
	if status != StatusCritical {
		t.Errorf("CalculateStatus() = %d, want %d (StatusCritical takes priority)", status, StatusCritical)
	}
}

func TestCalculateStatusEdgeCases(t *testing.T) {
	thresholds := Thresholds{
		CPUWarn:     70.0,
		CPUCritical: 90.0,
		RAMWarn:     80.0,
		RAMCritical: 95.0,
		DiskWarn:    85.0,
		DiskCritical: 95.0,
	}

	testCases := []struct {
		name    string
		metrics *Metrics
		want    StatusCode
	}{
		{"All zero", &Metrics{CPUPercent: 0.0, RAMPercent: 0.0, DiskPercent: 0.0}, StatusOK},
		{"All at exact boundaries", &Metrics{CPUPercent: 70.0, RAMPercent: 80.0, DiskPercent: 85.0}, StatusWarn},
		{"All critical boundaries", &Metrics{CPUPercent: 90.0, RAMPercent: 95.0, DiskPercent: 95.0}, StatusCritical},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := CalculateStatus(tc.metrics, thresholds)
			if status != tc.want {
				t.Errorf("CalculateStatus() = %d, want %d", status, tc.want)
			}
		})
	}
}

// Note: CollectMetrics() is tested indirectly through integration tests
// as it requires actual system resources and may behave differently
// across platforms.
