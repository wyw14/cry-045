package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Material struct {
	ID               string
	Code             string
	Name             string
	RiskClass        string
	AllowedProcesses []string
	RequiredEvidence []string
	SubstituteIDs    []string
}

type Certificate struct {
	ID          string
	MaterialID  string
	Number      string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Status      string
	Issuer      string
	EvidenceRef string
}

type Attachment struct {
	ID        string
	Name      string
	MediaType string
	Size      int64
	SHA256    string
	LocalPath string
}

type SelectionStatus string

const (
	StatusDraft        SelectionStatus = "draft"
	StatusSubmitted    SelectionStatus = "submitted"
	StatusFirstReview  SelectionStatus = "first_review"
	StatusSecondReview SelectionStatus = "second_review"
	StatusReturned     SelectionStatus = "returned"
	StatusApproved     SelectionStatus = "approved"
	StatusVoided       SelectionStatus = "voided"
	StatusArchived     SelectionStatus = "archived"
)

type SelectionRequest struct {
	ID               string
	ProjectID        string
	MaterialID       string
	Quantity         float64
	Purpose          string
	Process          string
	Revision         int
	Status           SelectionStatus
	SubstituteIDs    []string
	Attachments      []Attachment
	Findings         []Finding
	Opinions         []Opinion
	SubmittedAt      time.Time
	ApprovedAt       *time.Time
	ApprovedRevision int
}

type Finding struct {
	Code       string
	Severity   string
	Message    string
	EvidenceID string
	Blocking   bool
}

type Opinion struct {
	ID         string
	RequestID  string
	Revision   int
	ReviewerID string
	Decision   string
	Comment    string
	CreatedAt  time.Time
}

type AuditEvent struct {
	ID        string
	RequestID string
	Revision  int
	Action    string
	ActorID   string
	Before    string
	After     string
	Hash      string
	CreatedAt time.Time
}

type ProjectMaterial struct {
	ProjectID        string
	RequestID        string
	MaterialID       string
	ApprovedRevision int
	ImpactSummary    string
}

var (
	ErrNotFound        = errors.New("domain: not found")
	ErrInvalidState    = errors.New("domain: invalid state transition")
	ErrStaleRevision   = errors.New("domain: stale revision")
	ErrExpiredEvidence = errors.New("domain: expired certificate")
	ErrMissingEvidence = errors.New("domain: missing evidence")
)

func (s *SelectionRequest) ValidateShape() error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.ProjectID) == "" || strings.TrimSpace(s.MaterialID) == "" {
		return errors.New("selection id, project_id and material_id are required")
	}
	if s.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if strings.TrimSpace(s.Purpose) == "" || strings.TrimSpace(s.Process) == "" {
		return errors.New("purpose and process are required")
	}
	if s.Revision < 1 {
		s.Revision = 1
	}
	return nil
}

func (s SelectionStatus) CanMoveTo(next SelectionStatus) bool {
	allowed := map[SelectionStatus][]SelectionStatus{
		StatusDraft:        {StatusSubmitted, StatusVoided},
		StatusReturned:     {StatusSubmitted, StatusVoided},
		StatusSubmitted:    {StatusFirstReview, StatusVoided},
		StatusFirstReview:  {StatusSecondReview, StatusReturned, StatusVoided},
		StatusSecondReview: {StatusApproved, StatusReturned, StatusVoided},
		StatusApproved:     {StatusArchived},
		StatusArchived:     {},
		StatusVoided:       {},
	}
	for _, candidate := range allowed[s] {
		if candidate == next {
			return true
		}
	}
	return false
}

func (s *SelectionRequest) ChangeStatus(next SelectionStatus) error {
	if !s.Status.CanMoveTo(next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidState, s.Status, next)
	}
	s.Status = next
	return nil
}

func (s *SelectionRequest) AddFinding(f Finding) {
	for _, current := range s.Findings {
		if current.Code == f.Code && current.EvidenceID == f.EvidenceID {
			return
		}
	}
	s.Findings = append(s.Findings, f)
	sort.SliceStable(s.Findings, func(i, j int) bool {
		if s.Findings[i].Blocking != s.Findings[j].Blocking {
			return s.Findings[i].Blocking
		}
		return s.Findings[i].Code < s.Findings[j].Code
	})
}

func (s *SelectionRequest) HasBlockingFindings() bool {
	for _, finding := range s.Findings {
		if finding.Blocking {
			return true
		}
	}
	return false
}

func (c Certificate) ValidAt(now time.Time) bool {
	return c.Status == "valid" && now.Before(c.ExpiresAt)
}

type CertificateAssessment struct {
	EvidenceKind  string
	CertificateID string
	Code          string
	Blocking      bool
	ValidUntil    time.Time
}

func EvaluateCertificateCoverage(certificates []Certificate, required []string, reviewAt, decisionDue time.Time) []CertificateAssessment {
	normalizedRequired := uniqueEvidenceKinds(required)
	grouped := make(map[string][]Certificate)
	for _, certificate := range certificates {
		kind := certificateEvidenceKind(certificate)
		grouped[kind] = append(grouped[kind], certificate)
	}

	result := make([]CertificateAssessment, 0, len(normalizedRequired))
	for _, kind := range normalizedRequired {
		candidates := append([]Certificate(nil), grouped[kind]...)
		sort.SliceStable(candidates, func(i, j int) bool {
			if !candidates[i].ExpiresAt.Equal(candidates[j].ExpiresAt) {
				return candidates[i].ExpiresAt.After(candidates[j].ExpiresAt)
			}
			return candidates[i].IssuedAt.After(candidates[j].IssuedAt)
		})
		result = append(result, assessEvidenceWindow(kind, candidates, reviewAt, decisionDue))
	}
	return result
}

func assessEvidenceWindow(kind string, candidates []Certificate, reviewAt, decisionDue time.Time) CertificateAssessment {
	assessment := CertificateAssessment{EvidenceKind: kind, Code: "missing", Blocking: true}
	for _, certificate := range candidates {
		if certificate.Status == "revoked" || certificate.Status == "void" {
			continue
		}
		if certificate.IssuedAt.After(reviewAt) {
			continue
		}
		if !reviewAt.Before(certificate.ExpiresAt) {
			assessment = CertificateAssessment{EvidenceKind: kind, CertificateID: certificate.ID, Code: "expired", Blocking: true, ValidUntil: certificate.ExpiresAt}
			continue
		}
		if !decisionDue.IsZero() && certificate.ExpiresAt.Before(decisionDue) {
			return CertificateAssessment{EvidenceKind: kind, CertificateID: certificate.ID, Code: "expires_before_decision", Blocking: true, ValidUntil: certificate.ExpiresAt}
		}
		return CertificateAssessment{EvidenceKind: kind, CertificateID: certificate.ID, Code: "valid_for_window", Blocking: false, ValidUntil: certificate.ExpiresAt}
	}
	return assessment
}

func uniqueEvidenceKinds(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func certificateEvidenceKind(c Certificate) string {
	value := strings.ToLower(c.Number)
	if strings.Contains(value, "rohs") {
		return "rohs"
	}
	if strings.Contains(value, "reach") {
		return "reach"
	}
	return "certificate"
}
