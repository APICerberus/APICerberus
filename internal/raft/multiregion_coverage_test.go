package raft

import (
	"testing"
	"time"
)

func TestParseRegionID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		nodeID   string
		expected string
	}{
		{"node-us-east-1-01", "us-east"},        // "us"(len2) + "east"(len4)
		{"node-eu-west-2-03", "eu-west"},        // "eu"(len2) + "west"(len4)
		{"node-ap-southeast-1", "ap-southeast"}, // "ap"(len2) + "southeast"(len9)
		{"node-1", "default"},
		{"single", "default"},
		{"", "default"},
		{"us-east-1-prod-42", "us-east"},        // "us"(len2) + "east"(len4)
	}
	for _, tt := range tests {
		t.Run(tt.nodeID, func(t *testing.T) {
			t.Parallel()
			got := ParseRegionID(tt.nodeID)
			if got != tt.expected {
				t.Errorf("ParseRegionID(%q) = %q, want %q", tt.nodeID, got, tt.expected)
			}
		})
	}
}

func TestRegionStatus_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status   RegionStatus
		expected string
	}{
		{RegionStatusHealthy, "healthy"},
		{RegionStatusDegraded, "degraded"},
		{RegionStatusUnreachable, "unreachable"},
		{RegionStatus(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("RegionStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestNewMultiRegionManager_NilConfig(t *testing.T) {
	t.Parallel()
	_, err := NewMultiRegionManager(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestNewMultiRegionManager_Disabled(t *testing.T) {
	t.Parallel()
	mgr, err := NewMultiRegionManager(&MultiRegionConfig{Enabled: false}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestMultiRegionManager_GetRegionForNode_Disabled(t *testing.T) {
	t.Parallel()
	mgr, _ := NewMultiRegionManager(&MultiRegionConfig{Enabled: false}, nil)
	if region := mgr.GetRegionForNode("any"); region != "" {
		t.Errorf("expected empty region for disabled config, got %q", region)
	}
}

func TestMultiRegionManager_GetLatencyToRegion_Disabled(t *testing.T) {
	t.Parallel()
	mgr, _ := NewMultiRegionManager(&MultiRegionConfig{Enabled: false}, nil)
	if latency := mgr.GetLatencyToRegion("any"); latency != 0 {
		t.Errorf("expected 0 latency for disabled config, got %d", latency)
	}
}

func TestMultiRegionManager_GetRegionReplicationStatus_Disabled2(t *testing.T) {
	t.Parallel()
	mgr, _ := NewMultiRegionManager(&MultiRegionConfig{Enabled: false}, nil)
	status := mgr.GetRegionReplicationStatus()
	if len(status) != 0 {
		t.Errorf("expected 0 statuses for disabled config, got %d", len(status))
	}
}

func TestMultiRegionManager_GetQuorumRegions_Disabled(t *testing.T) {
	t.Parallel()
	mgr, _ := NewMultiRegionManager(&MultiRegionConfig{Enabled: false}, nil)
	// Even disabled, returns a default region set
	regions := mgr.GetQuorumRegions()
	// Function returns at least something for safe fallback
	_ = regions // just verify it doesn't panic
}

func TestMultiRegionManager_GetRegionAwareTimeout_Disabled2(t *testing.T) {
	t.Parallel()
	mgr, _ := NewMultiRegionManager(&MultiRegionConfig{Enabled: false}, nil)
	timeout := mgr.GetRegionAwareTimeout("node-1", 5*time.Second)
	if timeout <= 0 {
		t.Errorf("timeout = %d, want > 0", timeout)
	}
}
