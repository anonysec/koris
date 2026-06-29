package dbstore

import "time"

// Domain represents an independent VPN domain entity.
type Domain struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	IPAddress string    `json:"ip_address"`
	Status    string    `json:"status"` // active, blocked, retired
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Computed fields (from JOINs in ListDomains)
	BindingCount int    `json:"binding_count,omitempty"`
	CertStatus   string `json:"cert_status,omitempty"` // valid, expiring_soon, expired, none
}

// DomainIPHistory records one IP rotation event.
type DomainIPHistory struct {
	ID            int64     `json:"id"`
	DomainID      int64     `json:"domain_id"`
	PreviousIP    string    `json:"previous_ip"`
	NewIP         string    `json:"new_ip"`
	AdminUsername string    `json:"admin_username"`
	RotatedAt     time.Time `json:"rotated_at"`
}

// ProtocolBinding ties a domain to a protocol on a node at a specific failover position.
type ProtocolBinding struct {
	ID        int64     `json:"id"`
	NodeID    int64     `json:"node_id"`
	Protocol  string    `json:"protocol"`
	DomainID  int64     `json:"domain_id"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	// Joined fields (populated by ListProtocolBindings)
	DomainName   string `json:"domain_name,omitempty"`
	DomainIP     string `json:"domain_ip,omitempty"`
	DomainStatus string `json:"domain_status,omitempty"`
}

// VpnCertificate represents an IKEv2 certificate record for a domain on a node.
type VpnCertificate struct {
	ID          int64      `json:"id"`
	NodeID      int64      `json:"node_id"`
	DomainID    *int64     `json:"domain_id,omitempty"`
	CertType    string     `json:"cert_type"`
	Status      string     `json:"status"` // pending, active, expired
	Certificate *string    `json:"certificate,omitempty"`
	PrivateKey  *string    `json:"private_key,omitempty"`
	CAChain     *string    `json:"ca_chain,omitempty"`
	IssuedAt    *time.Time `json:"issued_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RetryCount  int        `json:"retry_count"`
	LastError   *string    `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
