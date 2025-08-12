package audit

import (
	"context"
	"os/user"
	"time"

	"github.com/ryanjwong/Atlas/atlas-cli/pkg/state"
)

type AuditService interface {
	StartOperation(ctx context.Context, clusterName string, opType state.OperationType, details map[string]interface{}) (*OperationContext, error)
	CompleteOperation(ctx context.Context, opCtx *OperationContext, status state.OperationStatus, errorMsg string) error
}

type OperationContext struct {
	ID           int
	ClusterName  string
	OpType       state.OperationType
	StartTime    time.Time
	UserID       string
	stateManager state.StateManager
}

type auditService struct {
	stateManager state.StateManager
}

func NewAuditService(sm state.StateManager) AuditService {
	return &auditService{stateManager: sm}
}

func (a *auditService) StartOperation(ctx context.Context, clusterName string, opType state.OperationType, details map[string]interface{}) (*OperationContext, error) {
	currentUser, _ := user.Current()
	userID := "atlas-cli"
	if currentUser != nil {
		userID = currentUser.Username
	}

	op := &state.OperationHistory{
		ClusterName:      clusterName,
		OperationType:    opType,
		OperationStatus:  state.OpStatusStarted,
		StartedAt:        time.Now(),
		UserID:           userID,
		OperationDetails: details,
		Metadata:         make(map[string]string),
	}

	if details != nil {
		op.OperationDetails = details
	} else {
		op.OperationDetails = make(map[string]interface{})
	}

	err := a.stateManager.StartOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	return &OperationContext{
		ID:           op.ID,
		ClusterName:  clusterName,
		OpType:       opType,
		StartTime:    op.StartedAt,
		UserID:       userID,
		stateManager: a.stateManager,
	}, nil
}

func (a *auditService) CompleteOperation(ctx context.Context, opCtx *OperationContext, status state.OperationStatus, errorMsg string) error {
	if errorMsg != "" {
		err := a.stateManager.UpdateOperation(ctx, opCtx.ID, status, errorMsg)
		if err != nil {
			return err
		}
	}
	
	return a.stateManager.CompleteOperation(ctx, opCtx.ID, status)
}