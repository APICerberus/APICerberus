package raft

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func TestNewSQLiteStorage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	storage, err := NewSQLiteStorage(db)
	if err != nil {
		t.Fatalf("NewSQLiteStorage() error = %v", err)
	}
	if storage == nil {
		t.Fatal("NewSQLiteStorage() returned nil")
	}
	if storage.db != db {
		t.Error("storage.db mismatch")
	}
}

func TestSQLiteStorage_SaveStateAndLoadState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	storage, err := NewSQLiteStorage(db)
	if err != nil {
		t.Fatalf("NewSQLiteStorage() error = %v", err)
	}

	// Test SaveState
	t.Run("SaveState", func(t *testing.T) {
		if err := storage.SaveState(5, "node-1"); err != nil {
			t.Errorf("SaveState() error = %v", err)
		}
	})

	// Test LoadState
	t.Run("LoadState", func(t *testing.T) {
		term, votedFor, err := storage.LoadState()
		if err != nil {
			t.Errorf("LoadState() error = %v", err)
		}
		if term != 5 {
			t.Errorf("LoadState() term = %v, want %v", term, 5)
		}
		if votedFor != "node-1" {
			t.Errorf("LoadState() votedFor = %v, want %v", votedFor, "node-1")
		}
	})

	// Test updating state
	t.Run("UpdateState", func(t *testing.T) {
		if err := storage.SaveState(10, "node-2"); err != nil {
			t.Errorf("SaveState() update error = %v", err)
		}

		term, votedFor, err := storage.LoadState()
		if err != nil {
			t.Errorf("LoadState() after update error = %v", err)
		}
		if term != 10 {
			t.Errorf("LoadState() term after update = %v, want %v", term, 10)
		}
		if votedFor != "node-2" {
			t.Errorf("LoadState() votedFor after update = %v, want %v", votedFor, "node-2")
		}
	})
}

func TestSQLiteStorage_SaveLogAndLoadLog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	storage, err := NewSQLiteStorage(db)
	if err != nil {
		t.Fatalf("NewSQLiteStorage() error = %v", err)
	}

	t.Run("EmptyLog", func(t *testing.T) {
		entries, err := storage.LoadLog()
		if err != nil {
			t.Errorf("LoadLog() empty error = %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("LoadLog() empty length = %v, want 0", len(entries))
		}
	})

	t.Run("SaveAndLoadLog", func(t *testing.T) {
		entries := []LogEntry{
			{Index: 1, Term: 1, Command: []byte(`{"action":"create"}`)},
			{Index: 2, Term: 1, Command: []byte(`{"action":"update"}`)},
			{Index: 3, Term: 2, Command: []byte(`{"action":"delete"}`)},
		}

		if err := storage.SaveLog(entries); err != nil {
			t.Errorf("SaveLog() error = %v", err)
		}

		loaded, err := storage.LoadLog()
		if err != nil {
			t.Errorf("LoadLog() error = %v", err)
		}
		if len(loaded) != 3 {
			t.Errorf("LoadLog() length = %v, want 3", len(loaded))
		}

		for i, e := range loaded {
			if e.Index != entries[i].Index {
				t.Errorf("LoadLog()[%d].Index = %v, want %v", i, e.Index, entries[i].Index)
			}
			if e.Term != entries[i].Term {
				t.Errorf("LoadLog()[%d].Term = %v, want %v", i, e.Term, entries[i].Term)
			}
		}
	})

	t.Run("ReplaceLogEntry", func(t *testing.T) {
		// Replace existing entry
		replacement := []LogEntry{
			{Index: 2, Term: 3, Command: []byte(`{"action":"replaced"}`)},
		}

		if err := storage.SaveLog(replacement); err != nil {
			t.Errorf("SaveLog() replace error = %v", err)
		}

		loaded, err := storage.LoadLog()
		if err != nil {
			t.Errorf("LoadLog() after replace error = %v", err)
		}

		// Find index 2
		var found bool
		for _, e := range loaded {
			if e.Index == 2 {
				found = true
				if e.Term != 3 {
					t.Errorf("Replaced entry term = %v, want 3", e.Term)
				}
				break
			}
		}
		if !found {
			t.Error("Replaced entry not found")
		}
	})
}

func TestSQLiteStorage_SaveSnapshotAndLoadSnapshot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	storage, err := NewSQLiteStorage(db)
	if err != nil {
		t.Fatalf("NewSQLiteStorage() error = %v", err)
	}

	t.Run("NoSnapshot", func(t *testing.T) {
		index, term, data, err := storage.LoadSnapshot()
		if err != nil {
			t.Errorf("LoadSnapshot() no snapshot error = %v", err)
		}
		if index != 0 || term != 0 || data != nil {
			t.Errorf("LoadSnapshot() no snapshot values = (%v, %v, %v), want (0, 0, nil)", index, term, data)
		}
	})

	t.Run("SaveAndLoadSnapshot", func(t *testing.T) {
		data := []byte(`{"state":"active","nodes":["a","b","c"]}`)
		if err := storage.SaveSnapshot(100, 5, data); err != nil {
			t.Errorf("SaveSnapshot() error = %v", err)
		}

		index, term, loaded, err := storage.LoadSnapshot()
		if err != nil {
			t.Errorf("LoadSnapshot() error = %v", err)
		}
		if index != 100 {
			t.Errorf("LoadSnapshot() index = %v, want 100", index)
		}
		if term != 5 {
			t.Errorf("LoadSnapshot() term = %v, want 5", term)
		}
		if string(loaded) != string(data) {
			t.Errorf("LoadSnapshot() data = %v, want %v", string(loaded), string(data))
		}
	})

	t.Run("UpdateSnapshot", func(t *testing.T) {
		newData := []byte(`{"state":"updated"}`)
		if err := storage.SaveSnapshot(200, 10, newData); err != nil {
			t.Errorf("SaveSnapshot() update error = %v", err)
		}

		index, term, loaded, err := storage.LoadSnapshot()
		if err != nil {
			t.Errorf("LoadSnapshot() after update error = %v", err)
		}
		if index != 200 {
			t.Errorf("LoadSnapshot() index after update = %v, want 200", index)
		}
		if term != 10 {
			t.Errorf("LoadSnapshot() term after update = %v, want 10", term)
		}
		if string(loaded) != string(newData) {
			t.Errorf("LoadSnapshot() data after update = %v, want %v", string(loaded), string(newData))
		}
	})
}

func TestInmemStorage_All(t *testing.T) {
	t.Run("NewInmemStorage", func(t *testing.T) {
		s := NewInmemStorage()
		if s == nil {
			t.Fatal("NewInmemStorage() returned nil")
		}
	})

	t.Run("SaveAndLoadState", func(t *testing.T) {
		s := NewInmemStorage()

		if err := s.SaveState(42, "candidate-1"); err != nil {
			t.Errorf("SaveState() error = %v", err)
		}

		term, votedFor, err := s.LoadState()
		if err != nil {
			t.Errorf("LoadState() error = %v", err)
		}
		if term != 42 {
			t.Errorf("LoadState() term = %v, want 42", term)
		}
		if votedFor != "candidate-1" {
			t.Errorf("LoadState() votedFor = %v, want candidate-1", votedFor)
		}
	})

	t.Run("SaveAndLoadLog", func(t *testing.T) {
		s := NewInmemStorage()

		entries := []LogEntry{
			{Index: 1, Term: 1, Command: []byte("cmd1")},
			{Index: 2, Term: 1, Command: []byte("cmd2")},
		}

		if err := s.SaveLog(entries); err != nil {
			t.Errorf("SaveLog() error = %v", err)
		}

		loaded, err := s.LoadLog()
		if err != nil {
			t.Errorf("LoadLog() error = %v", err)
		}
		if len(loaded) != 2 {
			t.Errorf("LoadLog() length = %v, want 2", len(loaded))
		}

		// Append more entries
		moreEntries := []LogEntry{
			{Index: 3, Term: 2, Command: []byte("cmd3")},
		}

		if err := s.SaveLog(moreEntries); err != nil {
			t.Errorf("SaveLog() append error = %v", err)
		}

		loaded, err = s.LoadLog()
		if err != nil {
			t.Errorf("LoadLog() after append error = %v", err)
		}
		if len(loaded) != 3 {
			t.Errorf("LoadLog() length after append = %v, want 3", len(loaded))
		}
	})

	t.Run("SaveAndLoadSnapshot", func(t *testing.T) {
		s := NewInmemStorage()

		data := []byte("snapshot-data")
		if err := s.SaveSnapshot(50, 3, data); err != nil {
			t.Errorf("SaveSnapshot() error = %v", err)
		}

		index, term, loaded, err := s.LoadSnapshot()
		if err != nil {
			t.Errorf("LoadSnapshot() error = %v", err)
		}
		if index != 50 {
			t.Errorf("LoadSnapshot() index = %v, want 50", index)
		}
		if term != 3 {
			t.Errorf("LoadSnapshot() term = %v, want 3", term)
		}
		if string(loaded) != string(data) {
			t.Errorf("LoadSnapshot() data = %v, want %v", string(loaded), string(data))
		}
	})
}
