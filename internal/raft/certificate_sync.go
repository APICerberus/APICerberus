package raft

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CertificateManager interface for TLS certificate operations
type CertificateManager interface {
	ReloadCertificate(serverName string) error
}

// CertificateUpdateLog represents a certificate update in Raft log
type CertificateUpdateLog struct {
	Domain    string    `json:"domain"`
	CertPEM   string    `json:"cert_pem"`
	KeyPEM    string    `json:"key_pem"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedBy  string    `json:"issued_by"` // Node ID that issued the cert
}

// ACMERenewalLock represents a lock for ACME renewal
type ACMERenewalLock struct {
	Domain   string    `json:"domain"`
	NodeID   string    `json:"node_id"`
	Deadline time.Time `json:"deadline"` // Lock expires after this time
}

// CertificateState holds certificate data in FSM
type CertificateState struct {
	Domain    string    `json:"domain"`
	CertPEM   string    `json:"cert_pem"`
	KeyPEM    string    `json:"key_pem"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedBy  string    `json:"issued_by"`
}

// CertFSM extends GatewayFSM with certificate-specific state
type CertFSM struct {
	// Certificates map (domain -> certificate state)
	Certificates map[string]*CertificateState `json:"certificates"`

	// Storage path for certificates
	StoragePath string `json:"-"`

	// TLS Manager for hot reload
	tlsManager CertificateManager

	// Logger
	logger *log.Logger
}

// NewCertFSM creates a new certificate FSM
func NewCertFSM(storagePath string, tlsManager CertificateManager) *CertFSM {
	return &CertFSM{
		Certificates: make(map[string]*CertificateState),
		StoragePath:  storagePath,
		tlsManager:   tlsManager,
		logger:       log.New(log.Writer(), "[cert-fsm] ", log.LstdFlags),
	}
}

// SetTLSManager sets the TLS manager for certificate reload
func (c *CertFSM) SetTLSManager(tm CertificateManager) {
	c.tlsManager = tm
}

// GetCertificate returns certificate from FSM state
func (c *CertFSM) GetCertificate(domain string) (*CertificateState, bool) {
	cert, ok := c.Certificates[domain]
	return cert, ok
}

// ProposeCertificateUpdate proposes a certificate update to the Raft cluster
func (n *Node) ProposeCertificateUpdate(domain, certPEM, keyPEM string, expiresAt time.Time) error {
	// Must be leader to propose
	if !n.IsLeader() {
		return fmt.Errorf("not the leader, cannot propose certificate update")
	}

	update := CertificateUpdateLog{
		Domain:    domain,
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: expiresAt,
		IssuedBy:  n.ID,
	}

	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate update: %w", err)
	}

	// Create FSM command
	cmd := FSMCommand{
		Type:    "certificate_update",
		Payload: data,
	}

	// Append to Raft log
	_, err = n.AppendEntry(cmd)
	if err != nil {
		return fmt.Errorf("failed to append certificate update to log: %w", err)
	}

	return nil
}

// AcquireACMERenewalLock tries to acquire a lock for ACME renewal
func (n *Node) AcquireACMERenewalLock(domain string, timeout time.Duration) (bool, error) {
	// Must be leader to propose lock
	if !n.IsLeader() {
		return false, fmt.Errorf("not the leader, cannot acquire renewal lock")
	}

	lock := ACMERenewalLock{
		Domain:   domain,
		NodeID:   n.ID,
		Deadline: time.Now().Add(timeout),
	}

	data, err := json.Marshal(lock)
	if err != nil {
		return false, err
	}

	// Create FSM command
	cmd := FSMCommand{
		Type:    "acme_renewal_lock",
		Payload: data,
	}

	// Append to Raft log
	_, err = n.AppendEntry(cmd)
	if err != nil {
		return false, err // Lock already held or other error
	}

	return true, nil
}

// RaftNode interface for certificate manager
type RaftNode interface {
	ProposeCertificateUpdate(domain, certPEM, keyPEM string, expiresAt time.Time) error
	AcquireACMERenewalLock(domain string, timeout time.Duration) (bool, error)
	IsLeader() bool
	GetNodeID() string
}

// GetNodeID returns the node ID
func (n *Node) GetNodeID() string {
	return n.ID
}

// Ensure Node implements RaftNode interface
var _ RaftNode = (*Node)(nil)
