package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EvidenceProfile is the versioned rulebook used by a compliance reviewer. It
// deliberately keeps policy data separate from HTTP handlers and repositories.
type EvidenceProfile struct {
	MaterialCode         string
	Revision             int
	RestrictedSubstances []string
	ProcessLimits        map[string]float64
	RequiredKinds        []string
	RiskThreshold        int
	EffectiveFrom        time.Time
	EffectiveUntil       *time.Time
}

type MaterialConflict struct {
	MaterialID   string
	SubstituteID string
	Reason       string
	Severity     string
}

type RuleObservation struct {
	RuleID   string
	Subject  string
	Outcome  string
	Message  string
	Blocking bool
	Evidence []string
}

type ReviewWindow struct {
	OpenedAt     time.Time
	DueAt        time.Time
	ReviewerRole string
	Stage        SelectionStatus
}

func (p EvidenceProfile) ActiveAt(now time.Time) bool {
	if now.Before(p.EffectiveFrom) {
		return false
	}
	return p.EffectiveUntil == nil || now.Before(*p.EffectiveUntil)
}

func (p EvidenceProfile) Requires(kind string) bool {
	for _, required := range p.RequiredKinds {
		if strings.EqualFold(required, kind) {
			return true
		}
	}
	return false
}

func (p EvidenceProfile) ExceedsProcessLimit(process string, quantity float64) bool {
	limit, ok := p.ProcessLimits[strings.ToLower(strings.TrimSpace(process))]
	return ok && quantity > limit
}

func (p EvidenceProfile) StableID() string {
	parts := append([]string{p.MaterialCode, fmt.Sprint(p.Revision), p.EffectiveFrom.UTC().Format(time.RFC3339)}, p.RestrictedSubstances...)
	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}

func BuildObservations(material Material, selection SelectionRequest, certificates []Certificate, now time.Time) []RuleObservation {
	observations := make([]RuleObservation, 0, 8)
	if material.RiskClass == "high" {
		observations = append(observations, RuleObservation{RuleID: "risk-high", Subject: material.Code, Outcome: "review", Message: "高风险材料需要复审", Blocking: true})
	}
	if !contains(material.AllowedProcesses, selection.Process) {
		observations = append(observations, RuleObservation{RuleID: "process-matrix", Subject: selection.Process, Outcome: "reject", Message: "工艺不在材料适用矩阵", Blocking: true})
	}
	valid := 0
	for _, cert := range certificates {
		if cert.ValidAt(now) {
			valid++
		}
	}
	if valid == 0 {
		observations = append(observations, RuleObservation{RuleID: "certificate-presence", Subject: material.Code, Outcome: "reject", Message: "没有可用证书", Blocking: true})
	}
	if selection.Quantity > 10000 {
		observations = append(observations, RuleObservation{RuleID: "quantity-cap", Subject: selection.ID, Outcome: "review", Message: "数量超过人工复核阈值", Blocking: true})
	}
	if len(selection.SubstituteIDs) > 0 {
		observations = append(observations, RuleObservation{RuleID: "substitute-declared", Subject: selection.ID, Outcome: "observe", Message: "申请包含替代方案", Evidence: append([]string(nil), selection.SubstituteIDs...)})
	}
	return observations
}

func DetectConflicts(primary Material, substitutes []Material, process string) []MaterialConflict {
	result := make([]MaterialConflict, 0)
	for _, candidate := range substitutes {
		if candidate.ID == primary.ID {
			result = append(result, MaterialConflict{MaterialID: primary.ID, SubstituteID: candidate.ID, Reason: "替代物不能与主材料相同", Severity: "high"})
			continue
		}
		if !contains(candidate.AllowedProcesses, process) {
			result = append(result, MaterialConflict{MaterialID: primary.ID, SubstituteID: candidate.ID, Reason: "替代物不适用于当前工艺", Severity: "medium"})
		}
		if primary.RiskClass == "low" && candidate.RiskClass == "high" {
			result = append(result, MaterialConflict{MaterialID: primary.ID, SubstituteID: candidate.ID, Reason: "替代物风险等级升高", Severity: "high"})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Severity != result[j].Severity {
			return result[i].Severity > result[j].Severity
		}
		return result[i].SubstituteID < result[j].SubstituteID
	})
	return result
}

func NewReviewWindow(now time.Time, stage SelectionStatus, role string, duration time.Duration) ReviewWindow {
	if duration <= 0 {
		duration = 24 * time.Hour
	}
	return ReviewWindow{OpenedAt: now, DueAt: now.Add(duration), ReviewerRole: role, Stage: stage}
}

func (w ReviewWindow) Overdue(now time.Time) bool {
	return now.After(w.DueAt) && w.Stage != StatusApproved && w.Stage != StatusArchived
}

func SummarizeObservations(observations []RuleObservation) (blocking int, notes []string) {
	for _, observation := range observations {
		if observation.Blocking {
			blocking++
		}
		notes = append(notes, observation.RuleID+": "+observation.Message)
	}
	sort.Strings(notes)
	return blocking, notes
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}
