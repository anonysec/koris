package grpcclient

import (
	"context"
	"fmt"
	"log"

	"github.com/anonysec/koris/internal/knodepb"
)

// ClientCertResult holds the generated client certificate, key, and CA PEM strings.
type ClientCertResult struct {
	CertPEM string
	KeyPEM  string
	CAPEM   string
}

// ClientCertService wraps the gRPC connection pool for client certificate operations.
type ClientCertService struct {
	pool Pool
}

// NewClientCertService creates a new ClientCertService.
func NewClientCertService(pool Pool) *ClientCertService {
	return &ClientCertService{pool: pool}
}

// GenerateClientCert calls the knode's GenerateClientCert gRPC to issue an OpenVPN
// client certificate for the given username on the specified node.
func (s *ClientCertService) GenerateClientCert(ctx context.Context, nodeID int64, username string) (*ClientCertResult, error) {
	node, err := s.pool.Get(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node %d not found in pool: %w", nodeID, err)
	}

	if node.Status == StatusOffline {
		return nil, fmt.Errorf("node %q is offline, cannot generate client cert", node.NodeName)
	}

	client := knodepb.NewKnodeServiceClient(node.Conn)
	resp, err := client.GenerateClientCert(ctx, &knodepb.GenerateClientCertRequest{
		Username: username,
	})
	if err != nil {
		log.Printf("[knode] GenerateClientCert RPC failed for %q on node %d: %v", username, nodeID, err)
		return nil, fmt.Errorf("GenerateClientCert RPC failed: %w", err)
	}

	if !resp.GetSuccess() {
		return nil, fmt.Errorf("cert generation failed: %s", resp.GetMessage())
	}

	return &ClientCertResult{
		CertPEM: resp.GetCertPem(),
		KeyPEM:  resp.GetKeyPem(),
		CAPEM:   resp.GetCaPem(),
	}, nil
}
