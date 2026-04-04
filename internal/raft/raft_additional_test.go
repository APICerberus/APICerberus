package raft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test CertFSM LoadCertificatesFromDisk error paths
func TestCertFSM_LoadCertificatesFromDisk_Errors(t *testing.T) {
	t.Run("non-existent directory", func(t *testing.T) {
		fsm := NewCertFSM("/nonexistent/path", nil)
		err := fsm.LoadCertificatesFromDisk()
		// Should not error, just skip loading
		if err != nil {
			t.Errorf("LoadCertificatesFromDisk() error = %v", err)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		fsm := NewCertFSM(tmpDir, nil)

		err := fsm.LoadCertificatesFromDisk()
		if err != nil {
			t.Errorf("LoadCertificatesFromDisk() error = %v", err)
		}

		// Should have no certificates
		if len(fsm.Certificates) != 0 {
			t.Errorf("Expected 0 certificates, got %d", len(fsm.Certificates))
		}
	})
}

// Test CertFSM GetCertificate not found
func TestCertFSM_GetCertificate_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	cert, ok := fsm.GetCertificate("nonexistent.example.com")
	if ok {
		t.Error("GetCertificate should return false for non-existent certificate")
	}
	if cert != nil {
		t.Error("GetCertificate should return nil for non-existent certificate")
	}
}

// Test CertFSM GetCertificate found
func TestCertFSM_GetCertificate_Found_Additional(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// Add a certificate directly to the map
	fsm.Certificates["test.example.com"] = &CertificateState{
		Domain:    "test.example.com",
		CertPEM:   "cert",
		KeyPEM:    "key",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	cert, ok := fsm.GetCertificate("test.example.com")
	if !ok {
		t.Error("GetCertificate should return true for existing certificate")
	}
	if cert == nil {
		t.Fatal("GetCertificate should return certificate")
	}
	if cert.Domain != "test.example.com" {
		t.Errorf("Expected domain test.example.com, got %s", cert.Domain)
	}
}

// Test CertFSM ListCertificates
func TestCertFSM_ListCertificates(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// Initially empty
	if len(fsm.Certificates) != 0 {
		t.Errorf("Expected 0 certificates, got %d", len(fsm.Certificates))
	}

	// Add a certificate directly to the map
	fsm.Certificates["test.example.com"] = &CertificateState{
		Domain:    "test.example.com",
		CertPEM:   "cert",
		KeyPEM:    "key",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if len(fsm.Certificates) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(fsm.Certificates))
	}
}

// Test CertFSM ApplyCertCommand error paths
func TestCertFSM_ApplyCertCommand_Error(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// Test with unknown command type
	err := fsm.ApplyCertCommand("unknown_operation", []byte("{}"))
	if err == nil {
		t.Error("ApplyCertCommand should return error for unknown command type")
	}

	// Test with invalid JSON for certificate_update
	err = fsm.ApplyCertCommand("certificate_update", []byte("not valid json"))
	if err == nil {
		t.Error("ApplyCertCommand should return error for invalid JSON")
	}

	// Test with missing required fields
	invalidCert := `{"domain": "", "cert_pem": "", "key_pem": ""}`
	err = fsm.ApplyCertCommand("certificate_update", []byte(invalidCert))
	if err == nil {
		t.Error("ApplyCertCommand should return error for missing required fields")
	}

	// Test with invalid JSON for acme_renewal_lock
	err = fsm.ApplyCertCommand("acme_renewal_lock", []byte("not valid json"))
	if err == nil {
		t.Error("ApplyCertCommand should return error for invalid renewal lock JSON")
	}
}

// Test CertFSM ApplyCertCommand certificate_update success
func TestCertFSM_ApplyCertCommand_CertificateUpdate_Additional(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	update := &CertificateUpdateLog{
		Domain:    "test.example.com",
		CertPEM:   "cert content",
		KeyPEM:    "key content",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		IssuedBy:  "node1",
	}

	data, _ := update.MarshalJSON()
	err := fsm.ApplyCertCommand("certificate_update", data)
	if err != nil {
		t.Errorf("ApplyCertCommand() error = %v", err)
	}

	// Verify certificate was stored
	cert, ok := fsm.GetCertificate("test.example.com")
	if !ok {
		t.Error("Certificate should be stored in FSM")
	}
	if cert.CertPEM != "cert content" {
		t.Errorf("Expected cert content 'cert content', got %s", cert.CertPEM)
	}

	// Verify certificate was written to disk
	certPath := filepath.Join(tmpDir, "test.example.com", "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file should exist on disk")
	}
}

// Test CertFSM ApplyCertCommand acme_renewal_lock success
func TestCertFSM_ApplyCertCommand_ACMERenewalLock_Additional(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	lock := &ACMERenewalLock{
		Domain:   "test.example.com",
		NodeID:   "node1",
		Deadline: time.Now().Add(time.Minute),
	}

	data, _ := lock.MarshalJSON()
	err := fsm.ApplyCertCommand("acme_renewal_lock", data)
	if err != nil {
		t.Errorf("ApplyCertCommand() error = %v", err)
	}

	// Verify lock was stored
	fsm.lockMu.RLock()
	storedLock, ok := fsm.RenewalLocks["test.example.com"]
	fsm.lockMu.RUnlock()

	if !ok {
		t.Error("Renewal lock should be stored in FSM")
	}
	if storedLock.NodeID != "node1" {
		t.Errorf("Expected node ID 'node1', got %s", storedLock.NodeID)
	}
}

// Test CertFSM ApplyCertCommand acme_renewal_lock conflict
func TestCertFSM_ApplyCertCommand_ACMERenewalLock_Conflict_Additional(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// First lock
	lock1 := &ACMERenewalLock{
		Domain:   "test.example.com",
		NodeID:   "node1",
		Deadline: time.Now().Add(time.Minute),
	}
	data1, _ := lock1.MarshalJSON()
	err := fsm.ApplyCertCommand("acme_renewal_lock", data1)
	if err != nil {
		t.Fatalf("First lock should succeed: %v", err)
	}

	// Second lock should fail
	lock2 := &ACMERenewalLock{
		Domain:   "test.example.com",
		NodeID:   "node2",
		Deadline: time.Now().Add(time.Minute),
	}
	data2, _ := lock2.MarshalJSON()
	err = fsm.ApplyCertCommand("acme_renewal_lock", data2)
	if err == nil {
		t.Error("Second lock should fail when first is still valid")
	}
}

// Test CertFSM GetCertificateFromDisk error paths
func TestCertFSM_GetCertificateFromDisk_Errors(t *testing.T) {
	t.Run("empty storage path", func(t *testing.T) {
		fsm := NewCertFSM("", nil)
		_, err := fsm.GetCertificateFromDisk("test.example.com")
		if err == nil {
			t.Error("GetCertificateFromDisk should return error for empty storage path")
		}
	})

	t.Run("non-existent domain", func(t *testing.T) {
		tmpDir := t.TempDir()
		fsm := NewCertFSM(tmpDir, nil)
		_, err := fsm.GetCertificateFromDisk("nonexistent.example.com")
		if err == nil {
			t.Error("GetCertificateFromDisk should return error for non-existent domain")
		}
	})

	t.Run("missing key file", func(t *testing.T) {
		tmpDir := t.TempDir()
		fsm := NewCertFSM(tmpDir, nil)

		// Create domain directory with only cert file
		domainDir := filepath.Join(tmpDir, "test.example.com")
		os.MkdirAll(domainDir, 0755)
		os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("cert"), 0644)

		_, err := fsm.GetCertificateFromDisk("test.example.com")
		if err == nil {
			t.Error("GetCertificateFromDisk should return error for missing key file")
		}
	})
}

// Test CertFSM GetCertificateFromDisk success
func TestCertFSM_GetCertificateFromDisk_Success(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// Create domain directory with cert and key
	domainDir := filepath.Join(tmpDir, "test.example.com")
	os.MkdirAll(domainDir, 0755)
	os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("cert content"), 0644)
	os.WriteFile(filepath.Join(domainDir, "key.pem"), []byte("key content"), 0644)

	// Write metadata
	meta := `{"issued_at": "2024-01-01T00:00:00Z", "expires_at": "2025-01-01T00:00:00Z", "issued_by": "node1"}`
	os.WriteFile(filepath.Join(domainDir, "meta.json"), []byte(meta), 0644)

	cert, err := fsm.GetCertificateFromDisk("test.example.com")
	if err != nil {
		t.Errorf("GetCertificateFromDisk() error = %v", err)
	}
	if cert == nil {
		t.Fatal("GetCertificateFromDisk should return certificate")
	}
	if cert.CertPEM != "cert content" {
		t.Errorf("Expected cert content 'cert content', got %s", cert.CertPEM)
	}
	if cert.KeyPEM != "key content" {
		t.Errorf("Expected key content 'key content', got %s", cert.KeyPEM)
	}
	if cert.IssuedBy != "node1" {
		t.Errorf("Expected issued_by 'node1', got %s", cert.IssuedBy)
	}
}

// Test CertFSM writeCertificateToDisk error paths
func TestCertFSM_WriteCertificateToDisk_Errors(t *testing.T) {
	t.Run("empty storage path", func(t *testing.T) {
		fsm := NewCertFSM("", nil)
		update := &CertificateUpdateLog{
			Domain:  "test.example.com",
			CertPEM: "cert",
			KeyPEM:  "key",
		}
		err := fsm.writeCertificateToDisk(update)
		if err == nil {
			t.Error("writeCertificateToDisk should return error for empty storage path")
		}
	})

	t.Run("invalid directory", func(t *testing.T) {
		// Use a path with invalid characters on Windows
		fsm := NewCertFSM("/::invalid/path::/test", nil)
		update := &CertificateUpdateLog{
			Domain:  "test.example.com",
			CertPEM: "cert",
			KeyPEM:  "key",
		}
		err := fsm.writeCertificateToDisk(update)
		if err == nil {
			t.Error("writeCertificateToDisk should return error for invalid directory")
		}
	})
}

// Helper method for CertificateUpdateLog to marshal JSON
func (c *CertificateUpdateLog) MarshalJSON() ([]byte, error) {
	type Alias CertificateUpdateLog
	return json.Marshal((*Alias)(c))
}

// Helper method for ACMERenewalLock to marshal JSON
func (a *ACMERenewalLock) MarshalJSON() ([]byte, error) {
	type Alias ACMERenewalLock
	return json.Marshal((*Alias)(a))
}
