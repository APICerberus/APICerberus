package raft

import (
	"testing"
	"time"
)

// setupCluster creates a 3-node cluster with in-memory transport.
func setupCluster(t *testing.T) ([]*Node, []*InmemTransport) {
	t.Helper()

	ids := []string{"node-1", "node-2", "node-3"}
	addrs := []string{"127.0.0.1:20001", "127.0.0.1:20002", "127.0.0.1:20003"}

	transports := make([]*InmemTransport, 3)
	for i := range transports {
		transports[i] = NewInmemTransport()
	}

	// Connect all transports to each other
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i != j {
				transports[i].Connect(ids[j], transports[j])
			}
		}
	}

	nodes := make([]*Node, 3)
	for i := 0; i < 3; i++ {
		cfg := DefaultConfig()
		cfg.NodeID = ids[i]
		cfg.BindAddress = addrs[i]
		cfg.ElectionTimeoutMin = 50 * time.Millisecond
		cfg.ElectionTimeoutMax = 150 * time.Millisecond
		cfg.HeartbeatInterval = 20 * time.Millisecond

		fsm := NewGatewayFSM()
		node, err := NewNode(cfg, fsm, transports[i])
		if err != nil {
			t.Fatalf("Failed to create node %s: %v", ids[i], err)
		}

		// Add peers
		for j := 0; j < 3; j++ {
			if i != j {
				node.Peers[ids[j]] = addrs[j]
			}
		}

		nodes[i] = node
	}

	return nodes, transports
}

func startNodes(t *testing.T, nodes []*Node) {
	t.Helper()
	for _, n := range nodes {
		if err := n.Start(); err != nil {
			t.Fatalf("Failed to start node %s: %v", n.ID, err)
		}
	}
}

func stopNodes(nodes []*Node) {
	for _, n := range nodes {
		n.Stop()
	}
}

func waitForLeader(t *testing.T, nodes []*Node, timeout time.Duration) *Node {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("Timed out waiting for leader election")
			return nil
		case <-ticker.C:
			for _, n := range nodes {
				if n.IsLeader() {
					return n
				}
			}
		}
	}
}

func TestLeaderElection(t *testing.T) {
	nodes, _ := setupCluster(t)
	startNodes(t, nodes)
	defer stopNodes(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)

	// Verify exactly one leader
	leaderCount := 0
	for _, n := range nodes {
		if n.IsLeader() {
			leaderCount++
		}
	}
	if leaderCount != 1 {
		t.Errorf("Expected exactly 1 leader, got %d", leaderCount)
	}

	// Verify followers know the leader
	for _, n := range nodes {
		if n.ID == leader.ID {
			continue
		}
		// Give time for heartbeat to propagate
		time.Sleep(100 * time.Millisecond)
		leaderID := n.GetLeaderID()
		if leaderID != leader.ID {
			t.Errorf("Node %s expected leader %s, got %s", n.ID, leader.ID, leaderID)
		}
	}
}

func TestLogReplication(t *testing.T) {
	nodes, _ := setupCluster(t)
	startNodes(t, nodes)
	defer stopNodes(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)

	// Propose an entry
	cmd := FSMCommand{
		Type:    CmdAddRoute,
		Payload: []byte(`{"id":"route-test","name":"Test Route"}`),
	}

	index, err := leader.AppendEntry(cmd)
	if err != nil {
		t.Fatalf("AppendEntry failed: %v", err)
	}

	// Wait for commit
	if err := leader.WaitForCommit(index, 3*time.Second); err != nil {
		t.Fatalf("WaitForCommit failed: %v", err)
	}

	// Verify all nodes have the entry
	time.Sleep(200 * time.Millisecond) // Allow replication
	for _, n := range nodes {
		n.mu.RLock()
		logLen := len(n.Log)
		commitIdx := n.CommitIndex
		n.mu.RUnlock()

		if logLen < 2 { // dummy + at least 1 entry
			t.Errorf("Node %s has %d log entries, expected at least 2", n.ID, logLen)
		}
		if commitIdx < index {
			t.Errorf("Node %s commitIndex=%d, expected >= %d", n.ID, commitIdx, index)
		}
	}
}

func TestPropose(t *testing.T) {
	nodes, _ := setupCluster(t)
	startNodes(t, nodes)
	defer stopNodes(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)

	// Create cluster manager for leader
	fsm := leader.fsm.(*GatewayFSM)
	cm := NewClusterManager(leader, fsm, "", "")

	// Propose a route addition
	cmd := FSMCommand{
		Type:    CmdAddRoute,
		Payload: []byte(`{"id":"route-cm","name":"CM Route","paths":["/cm"],"methods":["GET"]}`),
	}

	if err := cm.Propose(cmd); err != nil {
		t.Fatalf("Propose failed: %v", err)
	}

	// Verify FSM applied the route on leader
	if _, ok := fsm.GetRoute("route-cm"); !ok {
		t.Error("Expected route to be applied to leader FSM")
	}
}

func TestInmemStorage(t *testing.T) {
	s := NewInmemStorage()

	// Test state
	if err := s.SaveState(5, "node-1"); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	term, votedFor, err := s.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if term != 5 || votedFor != "node-1" {
		t.Errorf("LoadState = (%d, %s), want (5, node-1)", term, votedFor)
	}

	// Test log
	entries := []LogEntry{
		{Index: 1, Term: 1, Command: []byte("cmd1")},
		{Index: 2, Term: 1, Command: []byte("cmd2")},
	}
	if err := s.SaveLog(entries); err != nil {
		t.Fatalf("SaveLog: %v", err)
	}
	loaded, err := s.LoadLog()
	if err != nil {
		t.Fatalf("LoadLog: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("LoadLog returned %d entries, want 2", len(loaded))
	}

	// Test snapshot
	if err := s.SaveSnapshot(10, 3, []byte("snap")); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	idx, tm, data, err := s.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if idx != 10 || tm != 3 || string(data) != "snap" {
		t.Errorf("LoadSnapshot = (%d, %d, %s), want (10, 3, snap)", idx, tm, data)
	}
}
