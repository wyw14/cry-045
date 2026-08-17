package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry045/internal/domain"
	"github.com/wyw14/cry045/internal/repository"
)

func TestSubstitutePlanRejectsCyclesAndRiskEscalation(t *testing.T) {
	store := repository.NewDemoStore(time.Now())
	service := NewComplianceService(store, RealClock{})
	candidates := []domain.Material{
		{ID: "mat-safe", RiskClass: "low", AllowedProcesses: []string{"cnc"}, RequiredEvidence: []string{"rohs", "reach"}, SubstituteIDs: []string{"mat-loop"}},
		{ID: "mat-loop", RiskClass: "high", AllowedProcesses: []string{"cnc"}, RequiredEvidence: []string{"rohs", "reach"}, SubstituteIDs: []string{"mat-safe"}},
	}
	plan, err := service.EvaluateSubstitutePlan(context.Background(), "sel-demo", candidates)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, issue := range plan.Issues {
		seen[issue.Code] = true
	}
	if !seen["SUBSTITUTION_CYCLE"] || !seen["RISK_ESCALATION"] || plan.Blocking < 2 {
		t.Fatalf("unsafe substitute graph passed: %#v", plan)
	}
}
