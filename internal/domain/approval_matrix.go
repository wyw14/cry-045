package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ApprovalMatrix struct {
	Rules       []ApprovalRule
	Version     string
	PublishedAt time.Time
	RetiredAt   *time.Time
}
type ApprovalRule struct {
	ID              string
	RiskClass       string
	Process         string
	MinimumRole     string
	RequiredStages  []SelectionStatus
	EvidenceKinds   []string
	MaxQuantity     float64
	EscalationHours int
}
type ApprovalDecision struct {
	RuleID          string
	Allowed         bool
	Escalate        bool
	MissingRole     bool
	MissingStage    bool
	MissingEvidence bool
	Message         string
}

var roleRank = map[string]int{"designer": 1, "reviewer": 2, "senior_reviewer": 3, "compliance_owner": 4}

func (m ApprovalMatrix) ActiveAt(now time.Time) bool {
	return !m.PublishedAt.IsZero() && !now.Before(m.PublishedAt) && (m.RetiredAt == nil || now.Before(*m.RetiredAt))
}

func (m ApprovalMatrix) Evaluate(selection SelectionRequest, material Material, actorRole string, evidence map[string]bool) []ApprovalDecision {
	decisions := make([]ApprovalDecision, 0)
	for _, rule := range m.Rules {
		if rule.RiskClass != "" && rule.RiskClass != material.RiskClass {
			continue
		}
		if rule.Process != "" && !strings.EqualFold(rule.Process, selection.Process) {
			continue
		}
		decision := ApprovalDecision{RuleID: rule.ID, Allowed: true}
		if roleRank[actorRole] < roleRank[rule.MinimumRole] {
			decision.Allowed = false
			decision.MissingRole = true
			decision.Message = "审核角色级别不足"
		}
		if rule.MaxQuantity > 0 && selection.Quantity > rule.MaxQuantity {
			decision.Allowed = false
			decision.Escalate = true
			decision.Message = "数量超过该风险规则阈值"
		}
		if !containsStage(rule.RequiredStages, selection.Status) {
			decision.Allowed = false
			decision.MissingStage = true
			decision.Message = "申请状态未达到规则要求的审核阶段"
		}
		for _, kind := range rule.EvidenceKinds {
			if !evidence[kind] {
				decision.Allowed = false
				decision.MissingEvidence = true
				decision.Message = "缺少规则要求的证据: " + kind
			}
		}
		decisions = append(decisions, decision)
	}
	sort.SliceStable(decisions, func(i, j int) bool { return decisions[i].RuleID < decisions[j].RuleID })
	return decisions
}

func (m ApprovalMatrix) Explain(selection SelectionRequest, material Material, actorRole string, evidence map[string]bool) string {
	decisions := m.Evaluate(selection, material, actorRole, evidence)
	parts := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		state := "允许"
		if !decision.Allowed {
			state = "阻止"
		}
		if decision.Escalate {
			state += "/升级"
		}
		parts = append(parts, fmt.Sprintf("%s=%s(%s)", decision.RuleID, state, decision.Message))
	}
	return strings.Join(parts, "; ")
}

func DefaultApprovalMatrix(now time.Time) ApprovalMatrix {
	return ApprovalMatrix{Version: "industrial-2026.08", PublishedAt: now, Rules: []ApprovalRule{
		{ID: "low-standard", RiskClass: "low", MinimumRole: "reviewer", RequiredStages: []SelectionStatus{StatusFirstReview, StatusSecondReview}, EvidenceKinds: []string{"rohs"}, MaxQuantity: 50000, EscalationHours: 72},
		{ID: "medium-double-check", RiskClass: "medium", MinimumRole: "senior_reviewer", RequiredStages: []SelectionStatus{StatusSecondReview}, EvidenceKinds: []string{"rohs", "reach"}, MaxQuantity: 10000, EscalationHours: 48},
		{ID: "process-change", Process: "laser", MinimumRole: "compliance_owner", RequiredStages: []SelectionStatus{StatusSecondReview}, EvidenceKinds: []string{"rohs"}, MaxQuantity: 5000, EscalationHours: 24},
	}}
}

func containsStage(values []SelectionStatus, wanted SelectionStatus) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func CountBlocking(decisions []ApprovalDecision) int {
	count := 0
	for _, decision := range decisions {
		if !decision.Allowed {
			count++
		}
	}
	return count
}

func EscalationDeadline(rule ApprovalRule, opened time.Time) time.Time {
	hours := rule.EscalationHours
	if hours < 1 {
		hours = 24
	}
	return opened.Add(time.Duration(hours) * time.Hour)
}

func NeedsEscalation(rule ApprovalRule, opened, now time.Time) bool {
	return now.After(EscalationDeadline(rule, opened))
}

func MatrixSummary(matrix ApprovalMatrix) map[string]int {
	result := map[string]int{}
	for _, rule := range matrix.Rules {
		result[rule.RiskClass]++
	}
	return result
}
