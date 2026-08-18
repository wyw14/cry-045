package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/cry045/internal/domain"
	"github.com/wyw14/cry045/internal/repository"
)

type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

type ComplianceService struct {
	store repository.Store
	clock Clock
}

func NewComplianceService(store repository.Store, clock Clock) *ComplianceService {
	if clock == nil {
		clock = RealClock{}
	}
	return &ComplianceService{store: store, clock: clock}
}

func (s *ComplianceService) SubmitReceipts(ctx context.Context, ledger *ReceiptLedger, requestID, actor string, steps []ReceiptStep) error {
	return ledger.Submit(ctx, requestID, actor, steps)
}

func (s *ComplianceService) ValidateSelection(ctx context.Context, requestID string) (domain.SelectionRequest, error) {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return domain.SelectionRequest{}, err
	}
	material, err := s.store.GetMaterial(ctx, sel.MaterialID)
	if err != nil {
		return domain.SelectionRequest{}, err
	}
	now := s.clock.Now()
	sel.Findings = nil
	certificates, err := s.store.ListCertificates(ctx, material.ID)
	if err != nil {
		return domain.SelectionRequest{}, err
	}
	validByEvidence := make(map[string]bool)
	for _, cert := range certificates {
		if cert.ValidAt(now) {
			validByEvidence[certificateEvidence(cert)] = true
		}
		if cert.Status != "valid" || !now.Before(cert.ExpiresAt) {
			sel.AddFinding(domain.Finding{Code: "CERT_EXPIRED", Severity: "high", Message: "证书已失效或超过有效期", EvidenceID: cert.ID, Blocking: true})
		}
	}
	for _, required := range material.RequiredEvidence {
		if !validByEvidence[required] {
			sel.AddFinding(domain.Finding{Code: "EVIDENCE_MISSING", Severity: "high", Message: "缺少有效证据: " + required, EvidenceID: required, Blocking: true})
		}
	}
	if !contains(material.AllowedProcesses, sel.Process) {
		sel.AddFinding(domain.Finding{Code: "PROCESS_UNSUPPORTED", Severity: "medium", Message: "材料不适用于当前工艺", EvidenceID: sel.Process, Blocking: true})
	}
	if material.RiskClass == "high" && len(sel.SubstituteIDs) == 0 {
		sel.AddFinding(domain.Finding{Code: "SUBSTITUTE_REQUIRED", Severity: "medium", Message: "高风险材料必须提供替代方案", Blocking: true})
	}
	if err := s.store.SaveSelection(ctx, sel); err != nil {
		return domain.SelectionRequest{}, err
	}
	return sel, nil
}

func (s *ComplianceService) Submit(ctx context.Context, requestID, actor string) error {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return err
	}
	if err := s.ValidateCurrentRevision(sel); err != nil {
		return err
	}
	if err := sel.ChangeStatus(domain.StatusSubmitted); err != nil {
		return err
	}
	sel.SubmittedAt = s.clock.Now()
	if err := s.store.SaveSelection(ctx, sel); err != nil {
		return err
	}
	return s.audit(ctx, sel, actor, "submit", domain.StatusDraft, domain.StatusSubmitted)
}

func (s *ComplianceService) ValidateCurrentRevision(sel domain.SelectionRequest) error {
	if sel.Revision < 1 {
		return domain.ErrStaleRevision
	}
	if strings.TrimSpace(sel.Purpose) == "" || strings.TrimSpace(sel.Process) == "" {
		return domain.ErrMissingEvidence
	}
	if sel.HasBlockingFindings() {
		return fmt.Errorf("blocking findings remain")
	}
	return nil
}

func (s *ComplianceService) BeginReview(ctx context.Context, requestID, actor string, second bool) error {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return err
	}
	from, to := domain.StatusSubmitted, domain.StatusFirstReview
	if second {
		from, to = domain.StatusFirstReview, domain.StatusSecondReview
	}
	if sel.Status != from {
		return fmt.Errorf("review requires %s", from)
	}
	if err := sel.ChangeStatus(to); err != nil {
		return err
	}
	if err := s.store.SaveSelection(ctx, sel); err != nil {
		return err
	}
	return s.audit(ctx, sel, actor, "review_started", from, to)
}

func (s *ComplianceService) ReturnForRevision(ctx context.Context, requestID, actor, comment string) error {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return err
	}
	if sel.Status != domain.StatusFirstReview && sel.Status != domain.StatusSecondReview {
		return domain.ErrInvalidState
	}
	from := sel.Status
	if err := sel.ChangeStatus(domain.StatusReturned); err != nil {
		return err
	}
	sel.Revision++
	sel.Opinions = append(sel.Opinions, domain.Opinion{ID: fmt.Sprintf("op-%s-%d", requestID, sel.Revision), RequestID: requestID, Revision: sel.Revision, ReviewerID: actor, Decision: "return", Comment: comment, CreatedAt: s.clock.Now()})
	if err := s.store.SaveSelection(ctx, sel); err != nil {
		return err
	}
	return s.audit(ctx, sel, actor, "return", from, domain.StatusReturned)
}

func (s *ComplianceService) Approve(ctx context.Context, requestID, actor string) error {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return err
	}
	if sel.Status != domain.StatusSecondReview {
		return domain.ErrInvalidState
	}
	if sel.HasBlockingFindings() {
		return fmt.Errorf("cannot approve with blocking findings")
	}
	if err := sel.ChangeStatus(domain.StatusApproved); err != nil {
		return err
	}
	now := s.clock.Now()
	sel.ApprovedAt = &now
	sel.ApprovedRevision = sel.Revision
	if err := s.store.SaveSelection(ctx, sel); err != nil {
		return err
	}
	if err := s.store.SaveProjectMaterial(ctx, domain.ProjectMaterial{ProjectID: sel.ProjectID, RequestID: sel.ID, MaterialID: sel.MaterialID, ApprovedRevision: sel.Revision, ImpactSummary: "initial approval"}); err != nil {
		return err
	}
	return s.audit(ctx, sel, actor, "approve", domain.StatusSecondReview, domain.StatusApproved)
}

func (s *ComplianceService) Revise(ctx context.Context, requestID, actor, purpose, process string, quantity float64) (domain.SelectionRequest, error) {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return domain.SelectionRequest{}, err
	}
	if sel.Status != domain.StatusReturned {
		return domain.SelectionRequest{}, domain.ErrInvalidState
	}
	if sel.Revision < 2 {
		return domain.SelectionRequest{}, domain.ErrStaleRevision
	}
	sel.Purpose, sel.Process, sel.Quantity = purpose, process, quantity
	sel.Status = domain.StatusDraft
	sel.Findings = nil
	if err := sel.ValidateShape(); err != nil {
		return domain.SelectionRequest{}, err
	}
	if err := s.store.SaveSelection(ctx, sel); err != nil {
		return domain.SelectionRequest{}, err
	}
	if err := s.audit(ctx, sel, actor, "revise", domain.StatusReturned, domain.StatusDraft); err != nil {
		return domain.SelectionRequest{}, err
	}
	return sel, nil
}

func (s *ComplianceService) Timeline(ctx context.Context, requestID string) ([]domain.AuditEvent, error) {
	events, err := s.store.ListAudit(ctx, requestID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].CreatedAt.Before(events[j].CreatedAt) })
	return events, nil
}

func (s *ComplianceService) ExportReport(ctx context.Context, requestID string) ([]byte, error) {
	sel, err := s.store.GetSelection(ctx, requestID)
	if err != nil {
		return nil, err
	}
	timeline, err := s.Timeline(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Selection domain.SelectionRequest `json:"selection"`
		Timeline  []domain.AuditEvent     `json:"timeline"`
	}{sel, timeline})
}

func (s *ComplianceService) audit(ctx context.Context, sel domain.SelectionRequest, actor, action string, before, after domain.SelectionStatus) error {
	b, _ := json.Marshal(struct {
		ID       string
		Revision int
		Status   domain.SelectionStatus
	}{sel.ID, sel.Revision, before})
	a, _ := json.Marshal(struct {
		ID       string
		Revision int
		Status   domain.SelectionStatus
	}{sel.ID, sel.Revision, after})
	h := sha256.Sum256(append(append([]byte{}, b...), a...))
	event := domain.AuditEvent{ID: fmt.Sprintf("audit-%s-%d-%d", sel.ID, sel.Revision, len(action)), RequestID: sel.ID, Revision: sel.Revision, Action: action, ActorID: actor, Before: string(b), After: string(a), Hash: hex.EncodeToString(h[:]), CreatedAt: s.clock.Now()}
	return s.store.AppendAudit(ctx, event)
}

func certificateEvidence(c domain.Certificate) string {
	no := strings.ToLower(c.Number)
	if strings.Contains(no, "rohs") {
		return "rohs"
	}
	if strings.Contains(no, "reach") {
		return "reach"
	}
	return "certificate"
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
