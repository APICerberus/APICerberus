package raft

import (
	"fmt"
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
		_ = n.Stop()
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

func TestLogCompaction(t *testing.T) {
	nodes, _ := setupCluster(t)

	// Set a low snapshot threshold so compaction triggers quickly
	for _, n := range nodes {
		n.config.SnapshotThreshold = 5
	}

	startNodes(t, nodes)
	defer stopNodes(nodes)

	leader := waitForLeader(t, nodes, 3*time.Second)

	// Append enough entries to trigger compaction (threshold = 5)
	for i := 0; i < 10; i++ {
		cmd := FSMCommand{
			Type:    CmdIncrementCounter,
			Payload: []byte(fmt.Sprintf(`{"key":"counter-%d","count":1}`, i)),
		}

		index, err := leader.AppendEntry(cmd)
		if err != nil {
			t.Fatalf("AppendEntry %d failed: %v", i, err)
		}

		if err := leader.WaitForCommit(index, 3*time.Second); err != nil {
			t.Fatalf("WaitForCommit %d failed: %v", i, err)
		}
	}

	// Allow replication and compaction to complete
	time.Sleep(500 * time.Millisecond)

	// Verify the leader's log was compacted
	leader.mu.RLock()
	logLen := len(leader.Log)
	baseIndex := leader.Log[0].Index
	snapIdx := leader.lastSnapshotIndex
	leader.mu.RUnlock()

	// After compaction, the log should be shorter than 1 (dummy) + 10 (entries) = 11
	if logLen >= 11 {
		t.Errorf("Expected compacted log length < 11, got %d", logLen)
	}

	// The base index should have advanced past 0
	if baseIndex == 0 {
		t.Error("Expected log base index to advance after compaction")
	}

	// Snapshot index should be set
	if snapIdx == 0 {
		t.Error("Expected lastSnapshotIndex to be set after compaction")
	}

	// Verify FSM state is still correct (all counters applied)
	leaderFSM := leader.fsm.(*GatewayFSM)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("counter-%d", i)
		count := leaderFSM.GetRequestCount(key)
		if count != 1 {
			t.Errorf("Expected counter %s = 1, got %d", key, count)
		}
	}
}

func TestSnapshotTransfer(t *testing.T) {
	// Create a 3-node cluster
	nodes, transports := setupCluster(t)

	// Set a low snapshot threshold
	for _, n := range nodes {
		n.config.SnapshotThreshold = 5
	}

	startNodes(t, nodes)
	defer func() {
		// Stop all nodes including the late joiner
		stopNodes(nodes)
	}()

	leader := waitForLeader(t, nodes, 3*time.Second)

	// Add many entries to trigger compaction
	for i := 0; i < 15; i++ {
		cmd := FSMCommand{
			Type:    CmdIncrementCounter,
			Payload: []byte(fmt.Sprintf(`{"key":"snap-counter-%d","count":1}`, i)),
		}

		index, err := leader.AppendEntry(cmd)
		if err != nil {
			t.Fatalf("AppendEntry %d failed: %v", i, err)
		}

		if err := leader.WaitForCommit(index, 3*time.Second); err != nil {
			t.Fatalf("WaitForCommit %d failed: %v", i, err)
		}
	}

	// Wait for compaction to happen
	time.Sleep(500 * time.Millisecond)

	// Verify the leader has compacted
	leader.mu.RLock()
	snapIdx := leader.lastSnapshotIndex
	leader.mu.RUnlock()

	if snapIdx == 0 {
		t.Fatal("Expected leader to have a snapshot after compaction")
	}

	// Create a new node that will join late and need a snapshot transfer
	newNodeID := "node-4"
	newNodeAddr := "127.0.0.1:20004"
	newTransport := NewInmemTransport()

	newCfg := DefaultConfig()
	newCfg.NodeID = newNodeID
	newCfg.BindAddress = newNodeAddr
	newCfg.ElectionTimeoutMin = 50 * time.Millisecond
	newCfg.ElectionTimeoutMax = 150 * time.Millisecond
	newCfg.HeartbeatInterval = 20 * time.Millisecond
	newCfg.SnapshotThreshold = 5

	newFSM := NewGatewayFSM()
	newNode, err := NewNode(newCfg, newFSM, newTransport)
	if err != nil {
		t.Fatalf("Failed to create new node: %v", err)
	}

	// Connect the new node to all existing nodes and vice versa
	for i, n := range nodes {
		newTransport.Connect(n.ID, transports[i])
		transports[i].Connect(newNodeID, newTransport)
		newNode.Peers[n.ID] = n.Address
	}

	// Add the new node as a peer on the leader
	leader.AddPeer(newNodeID, newNodeAddr)

	// Start the new node
	if err := newNode.Start(); err != nil {
		t.Fatalf("Failed to start new node: %v", err)
	}
	// Extend nodes for cleanup
	nodes = append(nodes, newNode)

	// Wait for the new node to receive the snapshot and catch up
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	caughtUp := false
	for !caughtUp {
		select {
		case <-deadline:
			newNode.mu.RLock()
			lastApplied := newNode.LastApplied
			commitIdx := newNode.CommitIndex
			newNode.mu.RUnlock()
			t.Fatalf("New node did not catch up via snapshot. LastApplied=%d, CommitIndex=%d, expected >= %d",
				lastApplied, commitIdx, snapIdx)
		case <-ticker.C:
			newNode.mu.RLock()
			lastApplied := newNode.LastApplied
			newNode.mu.RUnlock()
			if lastApplied >= snapIdx {
				caughtUp = true
			}
		}
	}

	// Verify the new node's FSM has the correct state
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("snap-counter-%d", i)
		count := newFSM.GetRequestCount(key)
		if count != 1 {
			t.Errorf("New node: expected counter %s = 1, got %d", key, count)
		}
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
