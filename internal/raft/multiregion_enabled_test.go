package raft

import (
	"testing"
	"time"
)

func newEnabledMultiRegionManager() *MultiRegionManager {
	cfg := &MultiRegionConfig{
		Enabled:          true,
		RegionID:         "us-east",
		LeaderPreference: "priority",
		ReplicationMode:  "async",
		WANTimeoutFactor: 2.0,
		MaxCrossRegionLag: 30 * time.Second,
		Regions: []Region{
			{ID: "us-east", Name: "US East", Nodes: []string{"node-1", "node-2"}, Priority: 1},
			{ID: "us-west", Name: "US West", Nodes: []string{"node-3", "node-4"}, Priority: 2},
			{ID: "eu-west", Name: "EU West", Nodes: []string{"node-5"}, Priority: 3},
		},
	}
	mgr, _ := NewMultiRegionManager(cfg, nil)
	return mgr
}

func TestMultiRegionManager_Enabled_IsEnabled(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	if !mgr.IsEnabled() {
		t.Error("should be enabled")
	}
}

func TestMultiRegionManager_GetLocalRegion(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	if mgr.GetLocalRegion() != "us-east" {
		t.Errorf("local region = %q, want us-east", mgr.GetLocalRegion())
	}
}

func TestMultiRegionManager_GetRegionForNode(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	tests := []struct {
		nodeID string
		want   string
	}{
		{"node-1", "us-east"},
		{"node-3", "us-west"},
		{"node-5", "eu-west"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.nodeID, func(t *testing.T) {
			t.Parallel()
			got := mgr.GetRegionForNode(tt.nodeID)
			if got != tt.want {
				t.Errorf("GetRegionForNode(%q) = %q, want %q", tt.nodeID, got, tt.want)
			}
		})
	}
}

func TestMultiRegionManager_IsLocalNode(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	if !mgr.IsLocalNode("node-1") {
		t.Error("node-1 should be local")
	}
	if mgr.IsLocalNode("node-3") {
		t.Error("node-3 should not be local")
	}
}

func TestMultiRegionManager_IsCrossRegion(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	if mgr.IsCrossRegion("node-1") {
		t.Error("node-1 should not be cross-region")
	}
	if !mgr.IsCrossRegion("node-3") {
		t.Error("node-3 should be cross-region")
	}
}

func TestMultiRegionManager_RecordLatency(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	mgr.RecordLatency("node-3", 50*time.Millisecond)
	got := mgr.GetLatencyToRegion("us-west")
	if got != 50*time.Millisecond {
		t.Errorf("latency = %v, want 50ms", got)
	}
}

func TestMultiRegionManager_RecordLatency_UnknownNode(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	mgr.RecordLatency("unknown-node", 10*time.Millisecond)
	got := mgr.GetLatencyToRegion("")
	if got != 0 {
		t.Errorf("latency for unknown region should be 0, got %v", got)
	}
}

func TestMultiRegionManager_RecordLatency_DisabledNoop(t *testing.T) {
	t.Parallel()
	cfg := &MultiRegionConfig{Enabled: false}
	mgr, _ := NewMultiRegionManager(cfg, nil)
	mgr.RecordLatency("node-3", 50*time.Millisecond)
	// Should be a no-op
}

func TestMultiRegionManager_ShouldPreferLocalLeader(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	if mgr.ShouldPreferLocalLeader() {
		t.Error("priority mode should not prefer local leader")
	}
	mgr.config.LeaderPreference = "local"
	if !mgr.ShouldPreferLocalLeader() {
		t.Error("local mode should prefer local leader")
	}
}

func TestMultiRegionManager_GetLeaderPriorityScore(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	tests := []struct {
		nodeID string
		want   int
	}{
		{"node-1", -100}, // local region
		{"node-3", 2},    // us-west priority
		{"node-5", 3},    // eu-west priority
		{"unknown", 1000},
	}
	for _, tt := range tests {
		t.Run(tt.nodeID, func(t *testing.T) {
			t.Parallel()
			got := mgr.GetLeaderPriorityScore(tt.nodeID)
			if got != tt.want {
				t.Errorf("GetLeaderPriorityScore(%q) = %d, want %d", tt.nodeID, got, tt.want)
			}
		})
	}
}

func TestMultiRegionManager_GetLeaderPriorityScore_Disabled(t *testing.T) {
	t.Parallel()
	cfg := &MultiRegionConfig{Enabled: false}
	mgr, _ := NewMultiRegionManager(cfg, nil)
	if mgr.GetLeaderPriorityScore("any") != 0 {
		t.Error("disabled should return 0")
	}
}

func TestMultiRegionManager_UpdateReplicationStatus(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	mgr.UpdateReplicationStatus("us-west", 100)

	status := mgr.GetRegionReplicationStatus()
	if len(status) != 1 {
		t.Fatalf("expected 1 region status, got %d", len(status))
	}
	if status["us-west"].MatchIndex != 100 {
		t.Errorf("MatchIndex = %d, want 100", status["us-west"].MatchIndex)
	}
	if status["us-west"].Status != RegionStatusHealthy {
		t.Errorf("Status = %v, want Healthy", status["us-west"].Status)
	}
}

func TestMultiRegionManager_GetQuorumRegions(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	regions := mgr.GetQuorumRegions()
	// No status yet — should return all configured regions
	if len(regions) != 3 {
		t.Errorf("expected 3 regions, got %d: %v", len(regions), regions)
	}
}

func TestMultiRegionManager_GetQuorumRegions_WithStatus(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	mgr.UpdateReplicationStatus("us-west", 100)
	mgr.UpdateReplicationStatus("eu-west", 50)

	regions := mgr.GetQuorumRegions()
	if len(regions) != 2 {
		t.Errorf("expected 2 healthy regions, got %d: %v", len(regions), regions)
	}
}

func TestMultiRegionManager_ShouldReplicateToRegion(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	if !mgr.ShouldReplicateToRegion("us-west") {
		t.Error("should replicate to unknown region")
	}
	mgr.UpdateReplicationStatus("us-west", 100)
	if !mgr.ShouldReplicateToRegion("us-west") {
		t.Error("should replicate to healthy region")
	}
}

func TestMultiRegionManager_GetReplicationTimeout(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	// Local node — base timeout
	got := mgr.GetReplicationTimeout("node-1", 5*time.Second)
	if got != 5*time.Second {
		t.Errorf("local timeout = %v, want 5s", got)
	}
	// Cross-region — multiplied by WAN factor
	got2 := mgr.GetReplicationTimeout("node-3", 5*time.Second)
	if got2 != 10*time.Second {
		t.Errorf("cross-region timeout = %v, want 10s", got2)
	}
}

func TestMultiRegionManager_GetSortedPeersByPriority(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	mgr.RecordLatency("node-3", 50*time.Millisecond)
	mgr.RecordLatency("node-5", 100*time.Millisecond)

	peers := []string{"node-5", "node-3", "node-1"}
	sorted := mgr.GetSortedPeersByPriority(peers)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 peers, got %d", len(sorted))
	}
	// Local node (node-1) should come first
	if sorted[0] != "node-1" {
		t.Errorf("first peer = %q, want node-1", sorted[0])
	}
}

func TestMultiRegionManager_GetSortedPeersByPriority_DisabledNoop(t *testing.T) {
	t.Parallel()
	cfg := &MultiRegionConfig{Enabled: false}
	mgr, _ := NewMultiRegionManager(cfg, nil)
	peers := []string{"a", "b", "c"}
	sorted := mgr.GetSortedPeersByPriority(peers)
	if len(sorted) != 3 {
		t.Errorf("expected 3 peers, got %d", len(sorted))
	}
}

func TestMultiRegionManager_StartStop(t *testing.T) {
	mgr := newEnabledMultiRegionManager()
	if err := mgr.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	mgr.Stop()
}

func TestMultiRegionManager_DefaultConfig(t *testing.T) {
	cfg := DefaultMultiRegionConfig()
	if cfg.Enabled {
		t.Error("default should be disabled")
	}
	if cfg.WANTimeoutFactor != 2.0 {
		t.Errorf("WANTimeoutFactor = %v, want 2.0", cfg.WANTimeoutFactor)
	}
}

func TestMultiRegionManager_NewMissingRegionID(t *testing.T) {
	cfg := &MultiRegionConfig{
		Enabled:  true,
		RegionID: "",
	}
	_, err := NewMultiRegionManager(cfg, nil)
	if err == nil {
		t.Error("expected error for missing region_id")
	}
}

func TestMultiRegionManager_NewMissingLocalRegion(t *testing.T) {
	cfg := &MultiRegionConfig{
		Enabled:  true,
		RegionID: "nonexistent",
		Regions:  []Region{{ID: "us-east", Name: "US East", Nodes: []string{"n1"}}},
	}
	_, err := NewMultiRegionManager(cfg, nil)
	if err == nil {
		t.Error("expected error for missing local region")
	}
}

func TestMultiRegionManager_GetRegionAwareTimeout(t *testing.T) {
	t.Parallel()
	mgr := newEnabledMultiRegionManager()
	// Local node — base timeout
	got := mgr.GetRegionAwareTimeout("node-1", 5*time.Second)
	if got != 5*time.Second {
		t.Errorf("local = %v, want 5s", got)
	}
	// Cross-region
	got2 := mgr.GetRegionAwareTimeout("node-3", 5*time.Second)
	if got2 <= 5*time.Second {
		t.Errorf("cross-region should be > 5s, got %v", got2)
	}
}
