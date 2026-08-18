package repository

import (
	"context"
	"testing"

	"github.com/wyw14/cry045/internal/domain"
)

func TestCertificateIDsAreScopedByProjectAndMaterial(t *testing.T) {
	ctx := context.Background()
	registry := NewCertificateRegistry()
	if err := registry.Register(ctx, "project-a", domain.Certificate{ID: "cert-rohs", MaterialID: "mat-a", Number: "ROHS-A"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, "project-b", domain.Certificate{ID: "cert-rohs", MaterialID: "mat-b", Number: "ROHS-B"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Bind(ctx, "project-a", "mat-a", "cert-rohs"); err != nil {
		t.Fatal(err)
	}
	certificate, err := registry.Bound(ctx, "project-a", "mat-a")
	if err != nil {
		t.Fatal(err)
	}
	if certificate.MaterialID != "mat-a" || certificate.Number != "ROHS-A" {
		t.Fatalf("project-a received certificate from another scope: %#v", certificate)
	}
}
