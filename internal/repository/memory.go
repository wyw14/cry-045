package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry045/internal/domain"
)

type Store interface {
	GetMaterial(context.Context, string) (domain.Material, error)
	GetCertificate(context.Context, string) (domain.Certificate, error)
	ListCertificates(context.Context, string) ([]domain.Certificate, error)
	GetSelection(context.Context, string) (domain.SelectionRequest, error)
	SaveSelection(context.Context, domain.SelectionRequest) error
	AppendOpinion(context.Context, domain.Opinion) error
	AppendAudit(context.Context, domain.AuditEvent) error
	ListAudit(context.Context, string) ([]domain.AuditEvent, error)
	GetProjectMaterial(context.Context, string, string) (domain.ProjectMaterial, error)
	SaveProjectMaterial(context.Context, domain.ProjectMaterial) error
}

type MemoryStore struct {
	mu               sync.RWMutex
	materials        map[string]domain.Material
	certificates     map[string]domain.Certificate
	selections       map[string]domain.SelectionRequest
	opinions         []domain.Opinion
	audits           []domain.AuditEvent
	projectMaterials map[string]domain.ProjectMaterial
	projectHistory   map[string][]ProjectMaterialVersion
	projectReceipts  map[string]ProjectMaterialVersion
}

type ProjectMaterialVersion struct {
	Item              domain.ProjectMaterial
	ApprovalBasisHash string
	ReplacesRevision  int
	RecordedAt        time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		materials:        make(map[string]domain.Material),
		certificates:     make(map[string]domain.Certificate),
		selections:       make(map[string]domain.SelectionRequest),
		projectMaterials: make(map[string]domain.ProjectMaterial),
		projectHistory:   make(map[string][]ProjectMaterialVersion),
		projectReceipts:  make(map[string]ProjectMaterialVersion),
	}
}

func NewDemoStore(now time.Time) *MemoryStore {
	s := NewMemoryStore()
	s.materials["mat-al"] = domain.Material{ID: "mat-al", Code: "AL-6061", Name: "铝合金 6061", RiskClass: "medium", AllowedProcesses: []string{"cnc", "die_cast"}, RequiredEvidence: []string{"rohs", "reach"}, SubstituteIDs: []string{"mat-steel"}}
	s.materials["mat-steel"] = domain.Material{ID: "mat-steel", Code: "ST-304", Name: "不锈钢 304", RiskClass: "low", AllowedProcesses: []string{"cnc", "laser"}, RequiredEvidence: []string{"rohs"}}
	s.certificates["cert-al-rohs"] = domain.Certificate{ID: "cert-al-rohs", MaterialID: "mat-al", Number: "ROHS-AL-2026", IssuedAt: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(0, 6, 0), Status: "valid", Issuer: "Internal Lab"}
	s.certificates["cert-al-reach"] = domain.Certificate{ID: "cert-al-reach", MaterialID: "mat-al", Number: "REACH-AL-2026", IssuedAt: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(0, 3, 0), Status: "valid", Issuer: "Internal Lab"}
	s.certificates["cert-steel-rohs"] = domain.Certificate{ID: "cert-steel-rohs", MaterialID: "mat-steel", Number: "ROHS-ST-2026", IssuedAt: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(1, 0, 0), Status: "valid", Issuer: "Internal Lab"}
	s.selections["sel-demo"] = domain.SelectionRequest{ID: "sel-demo", ProjectID: "project-atlas", MaterialID: "mat-al", Quantity: 120, Purpose: "机箱支架", Process: "cnc", Revision: 1, Status: domain.StatusDraft}
	return s
}

func (s *MemoryStore) GetMaterial(ctx context.Context, id string) (domain.Material, error) {
	if err := ctx.Err(); err != nil {
		return domain.Material{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.materials[id]
	if !ok {
		return domain.Material{}, domain.ErrNotFound
	}
	return m, nil
}

func (s *MemoryStore) GetCertificate(ctx context.Context, id string) (domain.Certificate, error) {
	if err := ctx.Err(); err != nil {
		return domain.Certificate{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.certificates[id]
	if !ok {
		return domain.Certificate{}, domain.ErrNotFound
	}
	return c, nil
}

func (s *MemoryStore) ListCertificates(ctx context.Context, materialID string) ([]domain.Certificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Certificate, 0)
	for _, cert := range s.certificates {
		if cert.MaterialID == materialID {
			result = append(result, cert)
		}
	}
	return result, nil
}

func (s *MemoryStore) GetSelection(ctx context.Context, id string) (domain.SelectionRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.SelectionRequest{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sel, ok := s.selections[id]
	if !ok {
		return domain.SelectionRequest{}, domain.ErrNotFound
	}
	return cloneSelection(sel), nil
}

func (s *MemoryStore) SaveSelection(ctx context.Context, selection domain.SelectionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.selections[selection.ID]; ok && old.Revision > selection.Revision {
		return domain.ErrStaleRevision
	}
	s.selections[selection.ID] = cloneSelection(selection)
	return nil
}

func (s *MemoryStore) AppendOpinion(ctx context.Context, opinion domain.Opinion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opinions = append(s.opinions, opinion)
	return nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, event)
	return nil
}

func (s *MemoryStore) ListAudit(ctx context.Context, requestID string) ([]domain.AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.AuditEvent, 0)
	for _, event := range s.audits {
		if event.RequestID == requestID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetProjectMaterial(ctx context.Context, projectID, materialID string) (domain.ProjectMaterial, error) {
	if err := ctx.Err(); err != nil {
		return domain.ProjectMaterial{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.projectMaterials[projectID+"/"+materialID]
	if !ok {
		return domain.ProjectMaterial{}, domain.ErrNotFound
	}
	return v, nil
}

func (s *MemoryStore) SaveProjectMaterial(ctx context.Context, item domain.ProjectMaterial) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projectMaterials[item.ProjectID+"/"+item.MaterialID] = item
	return nil
}

func cloneSelection(in domain.SelectionRequest) domain.SelectionRequest {
	out := in
	out.SubstituteIDs = append([]string(nil), in.SubstituteIDs...)
	out.Attachments = append([]domain.Attachment(nil), in.Attachments...)
	out.Findings = append([]domain.Finding(nil), in.Findings...)
	out.Opinions = append([]domain.Opinion(nil), in.Opinions...)
	return out
}

func (s *MemoryStore) RecordProjectMaterialVersion(ctx context.Context, version ProjectMaterialVersion, idempotencyKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if version.Item.ProjectID == "" || version.Item.MaterialID == "" || version.Item.RequestID == "" {
		return fmt.Errorf("project, material and request identity are required")
	}
	if version.Item.ApprovedRevision < 1 || version.ApprovalBasisHash == "" || idempotencyKey == "" {
		return fmt.Errorf("approved revision, basis hash and idempotency key are required")
	}
	key := version.Item.ProjectID + "/" + version.Item.MaterialID
	receiptKey := key + ":" + idempotencyKey
	s.mu.Lock()
	defer s.mu.Unlock()
	if receipt, exists := s.projectReceipts[receiptKey]; exists {
		if receipt.Item.ApprovedRevision != version.Item.ApprovedRevision || receipt.ApprovalBasisHash != version.ApprovalBasisHash {
			return fmt.Errorf("idempotency key reused for another approved basis")
		}
		return nil
	}
	history := s.projectHistory[key]
	if len(history) > 0 {
		latest := history[len(history)-1]
		if version.Item.ApprovedRevision <= latest.Item.ApprovedRevision {
			return domain.ErrStaleRevision
		}
		if version.ReplacesRevision != latest.Item.ApprovedRevision {
			return fmt.Errorf("replacement must reference the latest approved revision")
		}
	} else if version.ReplacesRevision != 0 {
		return fmt.Errorf("initial approval cannot replace another revision")
	}
	if version.RecordedAt.IsZero() {
		version.RecordedAt = time.Now().UTC()
	}
	s.projectHistory[key] = append(history, cloneProjectVersion(version))
	s.projectMaterials[key] = version.Item
	s.projectReceipts[receiptKey] = cloneProjectVersion(version)
	return nil
}

func (s *MemoryStore) ListProjectMaterialVersions(ctx context.Context, projectID, materialID string) ([]ProjectMaterialVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := projectID + "/" + materialID
	s.mu.RLock()
	defer s.mu.RUnlock()
	history, exists := s.projectHistory[key]
	if !exists {
		return nil, domain.ErrNotFound
	}
	result := make([]ProjectMaterialVersion, len(history))
	for index, version := range history {
		result[index] = cloneProjectVersion(version)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Item.ApprovedRevision < result[j].Item.ApprovedRevision })
	return result, nil
}

func cloneProjectVersion(in ProjectMaterialVersion) ProjectMaterialVersion { out := in; return out }
