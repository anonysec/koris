package grpcclient

import (
	"context"
	"encoding/json"

	"KorisPanel/panel/internal/noderegistry"
)

// CoreEnablerAdapter wraps CoreManager to satisfy the noderegistry.CoreEnabler interface.
// This adapts the return type of AllCoreStatuses from []CoreStatus to []noderegistry.CoreStatusInfo.
type CoreEnablerAdapter struct {
	cm *CoreManager
}

// NewCoreEnablerAdapter creates an adapter that satisfies noderegistry.CoreEnabler.
func NewCoreEnablerAdapter(cm *CoreManager) *CoreEnablerAdapter {
	return &CoreEnablerAdapter{cm: cm}
}

// EnableCore delegates to CoreManager.EnableCore.
func (a *CoreEnablerAdapter) EnableCore(ctx context.Context, nodeID int64, coreType string, listenPort int, extraConfig json.RawMessage) error {
	return a.cm.EnableCore(ctx, nodeID, coreType, listenPort, extraConfig)
}

// AllCoreStatuses delegates to CoreManager.AllCoreStatuses and converts the result
// to []noderegistry.CoreStatusInfo.
func (a *CoreEnablerAdapter) AllCoreStatuses(ctx context.Context, nodeID int64) ([]noderegistry.CoreStatusInfo, error) {
	statuses, err := a.cm.AllCoreStatuses(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	result := make([]noderegistry.CoreStatusInfo, 0, len(statuses))
	for _, cs := range statuses {
		result = append(result, noderegistry.CoreStatusInfo{
			Type:  cs.Type,
			State: cs.State,
		})
	}
	return result, nil
}
