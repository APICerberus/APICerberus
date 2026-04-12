package raft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockCertificateManager for testing
type MockCertificateManager struct {
	reloadCalled bool
	lastDomain   string
}

func (m *MockCertificateManager) ReloadCertificate(serverName string) error {
	m.reloadCalled = true
	m.lastDomain = serverName
	return nil
}

func TestNewCertFSM(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &MockCertificateManager{}
	fsm := NewCertFSM(tmpDir, mock)

	if fsm == nil {
		t.Fatal("NewCertFSM() returned nil")
	}
	if fsm.Certificates == nil {
		t.Error("Certificates map not initialized")
	}
	if fsm.RenewalLocks == nil {
		t.Error("RenewalLocks map not initialized")
	}
	if fsm.StoragePath != tmpDir {
		t.Errorf("StoragePath = %v, want %v", fsm.StoragePath, tmpDir)
	}
}

func TestCertFSM_SetTLSManager(t *testing.T) {
	fsm := NewCertFSM("/tmp", nil)
	mock := &MockCertificateManager{}

	fsm.SetTLSManager(mock)
	if fsm.tlsManager != mock {
		t.Error("SetTLSManager() did not set the manager")
	}
}

func TestCertFSM_ApplyCertCommand_CertificateUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	mock := &MockCertificateManager{}
	fsm := NewCertFSM(tmpDir, mock)

	update := CertificateUpdateLog{
		Domain:    "example.com",
		CertPEM:   "CERT-PEM-DATA",
		KeyPEM:    "KEY-PEM-DATA",
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		IssuedBy:  "node-1",
	}
	data, _ := json.Marshal(update)

	err := fsm.ApplyCertCommand("certificate_update", data)
	if err != nil {
		t.Errorf("ApplyCertCommand() error = %v", err)
	}

	// Verify certificate is stored
	cert, ok := fsm.GetCertificate("example.com")
	if !ok {
		t.Error("Certificate not found in FSM")
	}
	if cert.Domain != "example.com" {
		t.Errorf("Domain = %v, want example.com", cert.Domain)
	}
	if cert.CertPEM != "CERT-PEM-DATA" {
		t.Errorf("CertPEM = %v, want CERT-PEM-DATA", cert.CertPEM)
	}

	// Verify TLS manager was notified
	if !mock.reloadCalled {
		t.Error("TLS manager ReloadCertificate was not called")
	}
	if mock.lastDomain != "example.com" {
		t.Errorf("TLS manager domain = %v, want example.com", mock.lastDomain)
	}

	// Verify certificate was written to disk
	certPath := filepath.Join(tmpDir, "example.com", "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file not written to disk")
	}
}

func TestCertFSM_ApplyCertCommand_InvalidCommand(t *testing.T) {
	fsm := NewCertFSM("/tmp", nil)

	err := fsm.ApplyCertCommand("unknown_command", []byte("{}"))
	if err == nil {
		t.Error("ApplyCertCommand() should return error for unknown command")
	}
}

func TestCertFSM_ApplyCertCommand_InvalidCertificateData(t *testing.T) {
	fsm := NewCertFSM("/tmp", nil)

	// Missing required fields
	update := CertificateUpdateLog{
		Domain:  "",
		CertPEM: "",
		KeyPEM:  "",
	}
	data, _ := json.Marshal(update)

	err := fsm.ApplyCertCommand("certificate_update", data)
	if err == nil {
		t.Error("ApplyCertCommand() should return error for invalid data")
	}
}

func TestCertFSM_ApplyCertCommand_ACMERenewalLock(t *testing.T) {
	fsm := NewCertFSM("/tmp", nil)

	lock := ACMERenewalLock{
		Domain:   "example.com",
		NodeID:   "node-1",
		Deadline: time.Now().Add(time.Hour),
	}
	data, _ := json.Marshal(lock)

	err := fsm.ApplyCertCommand("acme_renewal_lock", data)
	if err != nil {
		t.Errorf("ApplyCertCommand() error = %v", err)
	}

	// Verify lock is stored
	fsm.lockMu.RLock()
	storedLock, ok := fsm.RenewalLocks["example.com"]
	fsm.lockMu.RUnlock()
	if !ok {
		t.Error("Renewal lock not stored")
	}
	if storedLock.NodeID != "node-1" {
		t.Errorf("Lock NodeID = %v, want node-1", storedLock.NodeID)
	}
}

func TestCertFSM_ApplyCertCommand_ACMERenewalLock_Conflict(t *testing.T) {
	fsm := NewCertFSM("/tmp", nil)

	// First lock
	lock1 := ACMERenewalLock{
		Domain:   "example.com",
		NodeID:   "node-1",
		Deadline: time.Now().Add(time.Hour),
	}
	data1, _ := json.Marshal(lock1)
	_ = fsm.ApplyCertCommand("acme_renewal_lock", data1)

	// Second lock attempt (should fail)
	lock2 := ACMERenewalLock{
		Domain:   "example.com",
		NodeID:   "node-2",
		Deadline: time.Now().Add(2 * time.Hour),
	}
	data2, _ := json.Marshal(lock2)

	err := fsm.ApplyCertCommand("acme_renewal_lock", data2)
	if err == nil {
		t.Error("Should return error when lock already held")
	}
}

func TestCertFSM_GetCertificate(t *testing.T) {
	fsm := NewCertFSM("/tmp", nil)

	// Non-existent
	_, ok := fsm.GetCertificate("nonexistent.com")
	if ok {
		t.Error("GetCertificate() should return false for non-existent cert")
	}

	// Add certificate
	fsm.Certificates["example.com"] = &CertificateState{
		Domain:  "example.com",
		CertPEM: "test-cert",
	}

	cert, ok := fsm.GetCertificate("example.com")
	if !ok {
		t.Error("GetCertificate() should return true for existing cert")
	}
	if cert.Domain != "example.com" {
		t.Errorf("Domain = %v, want example.com", cert.Domain)
	}
}

func TestCertFSM_GetCertificateFromDisk(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// Create certificate files
	domainDir := filepath.Join(tmpDir, "example.com")
	_ = os.MkdirAll(domainDir, 0755)
	_ = os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("CERT-DATA"), 0600)
	_ = os.WriteFile(filepath.Join(domainDir, "key.pem"), []byte("KEY-DATA"), 0600)

	// Create metadata
	meta := map[string]interface{}{
		"issued_at":  time.Now().Format(time.RFC3339),
		"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"issued_by":  "node-1",
	}
	metaData, _ := json.Marshal(meta)
	_ = os.WriteFile(filepath.Join(domainDir, "meta.json"), metaData, 0600)

	cert, err := fsm.GetCertificateFromDisk("example.com")
	if err != nil {
		t.Errorf("GetCertificateFromDisk() error = %v", err)
	}
	if cert.Domain != "example.com" {
		t.Errorf("Domain = %v, want example.com", cert.Domain)
	}
	if cert.CertPEM != "CERT-DATA" {
		t.Errorf("CertPEM = %v, want CERT-DATA", cert.CertPEM)
	}
	if cert.KeyPEM != "KEY-DATA" {
		t.Errorf("KeyPEM = %v, want KEY-DATA", cert.KeyPEM)
	}
}

func TestCertFSM_GetCertificateFromDisk_NoStoragePath(t *testing.T) {
	fsm := NewCertFSM("", nil)

	_, err := fsm.GetCertificateFromDisk("example.com")
	if err == nil {
		t.Error("Should return error when storage path not configured")
	}
}

func TestCertFSM_LoadCertificatesFromDisk(t *testing.T) {
	tmpDir := t.TempDir()
	fsm := NewCertFSM(tmpDir, nil)

	// Create multiple certificate directories
	domains := []string{"example.com", "test.com"}
	for _, domain := range domains {
		domainDir := filepath.Join(tmpDir, domain)
		_ = os.MkdirAll(domainDir, 0755)
		_ = os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("CERT-"+domain), 0600)
		_ = os.WriteFile(filepath.Join(domainDir, "key.pem"), []byte("KEY-"+domain), 0600)
	}

	err := fsm.LoadCertificatesFromDisk()
	if err != nil {
		t.Errorf("LoadCertificatesFromDisk() error = %v", err)
	}

	for _, domain := range domains {
		cert, ok := fsm.GetCertificate(domain)
		if !ok {
			t.Errorf("Certificate %s not loaded", domain)
		}
		if cert.Domain != domain {
			t.Errorf("Domain = %v, want %v", cert.Domain, domain)
		}
	}
}

func TestCertFSM_LoadCertificatesFromDisk_NoStoragePath(t *testing.T) {
	fsm := NewCertFSM("", nil)

	err := fsm.LoadCertificatesFromDisk()
	if err != nil {
		t.Errorf("Should return nil when storage path not configured, got %v", err)
	}
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")

	data := []byte("test data for atomic write")
	err := atomicWriteFile(testPath, data)
	if err != nil {
		t.Errorf("atomicWriteFile() error = %v", err)
	}

	// Verify file exists and content is correct
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("Content = %v, want %v", string(content), string(data))
	}

	// Verify permissions (skip on Windows)
	if os.PathSeparator == '/' {
		info, err := os.Stat(testPath)
		if err != nil {
			t.Errorf("Failed to stat file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("Permissions = %v, want 0600", info.Mode().Perm())
		}
	}
}

func TestCertLogEntryType_Constants(t *testing.T) {
	if CertLogEntryCertificateUpdate != 0 {
		t.Errorf("CertLogEntryCertificateUpdate = %v, want 0", CertLogEntryCertificateUpdate)
	}
	if CertLogEntryACMERenewalLock != 1 {
		t.Errorf("CertLogEntryACMERenewalLock = %v, want 1", CertLogEntryACMERenewalLock)
	}
}
