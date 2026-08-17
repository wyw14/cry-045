package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyw14/cry045/internal/domain"
)

type ImpactReport struct {
	RequestID        string
	ProjectID        string
	CurrentRevision  int
	ApprovedRevision int
	Vector           domain.ImpactVector
	PassportDelta    domain.PassportDelta
	CanReplace       bool
	Reason           string
	GeneratedAt      time.Time
}

func (s *ComplianceService) BuildImpactReport(ctx context.Context, requestID string, before, after domain.MaterialPassport, changedFields []string) (ImpactReport, error) {
	selection, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return ImpactReport{}, err
	}
	if before.SelectionID != requestID || after.SelectionID != requestID {
		return ImpactReport{}, fmt.Errorf("passport selection mismatch")
	}
	if before.Revision >= after.Revision {
		return ImpactReport{}, fmt.Errorf("replacement revision must increase")
	}
	delta := domain.ComparePassports(before, after)
	vector := domain.BuildImpactVector(selection, changedFields)
	canReplace := selection.Status == domain.StatusDraft || selection.Status == domain.StatusReturned
	reason := "当前版本允许继续修订"
	if !canReplace {
		reason = "只有草稿或退回版本允许替换批准依据"
	}
	if delta.RequiresReapproval {
		reason += "；证据变化需要重新审批"
	}
	return ImpactReport{RequestID: requestID, ProjectID: selection.ProjectID, CurrentRevision: selection.Revision, ApprovedRevision: selection.ApprovedRevision, Vector: vector, PassportDelta: delta, CanReplace: canReplace, Reason: reason, GeneratedAt: s.clock.Now()}, nil
}
