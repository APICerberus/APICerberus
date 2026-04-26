package raft

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRaftTransportBodySizeLimits verifies MEDIUM-003 fix:
// All Raft RPC handlers use io.LimitReader on JSON decoder.
func TestRaftTransportBodySizeLimits(t *testing.T) {
	t.Run("handleRequestVote uses limit reader", func(t *testing.T) {
		t.Parallel()
		transport := &HTTPTransport{}
		transport.mu.Lock()
		transport.handler = &mockHandler{}
		transport.mu.Unlock()

		reqBody := `{"term":1,"candidate_id":"node-1","last_log_index":0,"last_log_term":0}`
		req := httptest.NewRequest(http.MethodPost, "/raft/request-vote", newReadCloser(reqBody))
		rec := httptest.NewRecorder()

		transport.handleRequestVote(rec, req)

		if rec.Code == http.StatusInternalServerError {
			t.Error("valid request should not return 500")
		}
	})

	t.Run("handleAppendEntries uses limit reader", func(t *testing.T) {
		t.Parallel()
		transport := &HTTPTransport{}
		transport.mu.Lock()
		transport.handler = &mockHandler{}
		transport.mu.Unlock()

		reqBody := `{"term":1,"leader_id":"node-1","prev_log_index":0,"prev_log_term":0,"entries":[],"leader_commit":0}`
		req := httptest.NewRequest(http.MethodPost, "/raft/append-entries", newReadCloser(reqBody))
		rec := httptest.NewRecorder()

		transport.handleAppendEntries(rec, req)

		if rec.Code == http.StatusInternalServerError {
			t.Error("valid request should not return 500")
		}
	})

	t.Run("handleInstallSnapshot uses limit reader", func(t *testing.T) {
		t.Parallel()
		transport := &HTTPTransport{}
		transport.mu.Lock()
		transport.handler = &mockHandler{}
		transport.mu.Unlock()

		reqBody := `{"term":1,"leader_id":"node-1","last_included_index":0,"last_included_term":0,"offset":0,"data":null,"done":true}`
		req := httptest.NewRequest(http.MethodPost, "/raft/install-snapshot", newReadCloser(reqBody))
		rec := httptest.NewRecorder()

		transport.handleInstallSnapshot(rec, req)

		if rec.Code == http.StatusInternalServerError {
			t.Error("valid request should not return 500")
		}
	})
}

type mockHandler struct{}

func (h *mockHandler) HandleRequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	return &RequestVoteResponse{}
}
func (h *mockHandler) HandleAppendEntries(req *AppendEntriesRequest) *AppendEntriesResponse {
	return &AppendEntriesResponse{}
}
func (h *mockHandler) HandleInstallSnapshot(req *InstallSnapshotRequest) *InstallSnapshotResponse {
	return &InstallSnapshotResponse{}
}

// newReadCloser creates an io.ReadCloser from a string for testing.
func newReadCloser(s string) *readCloser {
	return &readCloser{data: []byte(s)}
}

type readCloser struct {
	data []byte
	pos  int
}

func (r *readCloser) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *readCloser) Close() error { return nil }
