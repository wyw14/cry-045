package domain

import (
	"testing"
	"time"
)

func TestCertificateCoverageSpansEntireDecisionWindow(t *testing.T) {
	now := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	due := now.Add(72 * time.Hour)
	certificates := []Certificate{
		{ID: "rohs-short", Number: "ROHS-2026", Status: "valid", IssuedAt: now.AddDate(-1, 0, 0), ExpiresAt: now.Add(24 * time.Hour)},
		{ID: "reach-long", Number: "REACH-2026", Status: "valid", IssuedAt: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(0, 3, 0)},
	}
	assessments := EvaluateCertificateCoverage(certificates, []string{"rohs", "reach"}, now, due)
	if len(assessments) != 2 {
		t.Fatalf("unexpected assessments: %#v", assessments)
	}
	if assessments[1].EvidenceKind != "rohs" || assessments[1].Code != "expires_before_decision" || !assessments[1].Blocking {
		t.Fatalf("short-lived certificate passed the review window: %#v", assessments)
	}
}
