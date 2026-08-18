package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/wyw14/cry045/internal/domain"
)

type ProjectCertificate struct {
	ProjectID   string
	Certificate domain.Certificate
}

type CertificateRegistry struct {
	mu           sync.RWMutex
	certificates map[string]ProjectCertificate
	bindings     map[string]string
}

func NewCertificateRegistry() *CertificateRegistry {
	return &CertificateRegistry{
		certificates: make(map[string]ProjectCertificate),
		bindings:     make(map[string]string),
	}
}

func (r *CertificateRegistry) Register(ctx context.Context, projectID string, certificate domain.Certificate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || strings.TrimSpace(certificate.ID) == "" || strings.TrimSpace(certificate.MaterialID) == "" {
		return errors.New("project, certificate and material ids are required")
	}
	key := certificate.RegistryKey(projectID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.certificates[key]; ok && existing.Certificate.MaterialID != certificate.MaterialID {
		return fmt.Errorf("certificate id already belongs to material %s", existing.Certificate.MaterialID)
	}
	r.certificates[key] = ProjectCertificate{ProjectID: projectID, Certificate: cloneCertificate(certificate)}
	return nil
}

func (r *CertificateRegistry) Bind(ctx context.Context, projectID, materialID, certificateID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	projectID = strings.TrimSpace(projectID)
	materialID = strings.TrimSpace(materialID)
	certificateID = strings.TrimSpace(certificateID)
	if projectID == "" || materialID == "" || certificateID == "" {
		return errors.New("project, material and certificate ids are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.certificates[certificateKey(projectID, certificateID)]
	if !ok {
		return domain.ErrNotFound
	}
	if entry.ProjectID != projectID || entry.Certificate.MaterialID != materialID {
		return errors.New("certificate is outside the requested project/material scope")
	}
	r.bindings[bindingKey(projectID, materialID)] = certificateID
	return nil
}

func (r *CertificateRegistry) Bound(ctx context.Context, projectID, materialID string) (domain.Certificate, error) {
	if err := ctx.Err(); err != nil {
		return domain.Certificate{}, err
	}
	projectID = strings.TrimSpace(projectID)
	materialID = strings.TrimSpace(materialID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.bindings[bindingKey(projectID, materialID)]
	if !ok {
		return domain.Certificate{}, domain.ErrNotFound
	}
	entry, ok := r.certificates[certificateKey(projectID, id)]
	if !ok || entry.ProjectID != projectID || entry.Certificate.MaterialID != materialID {
		return domain.Certificate{}, domain.ErrNotFound
	}
	return cloneCertificate(entry.Certificate), nil
}

func certificateKey(projectID, certificateID string) string {
	return projectID + "\x00" + certificateID
}

func bindingKey(projectID, materialID string) string {
	return projectID + "\x00" + materialID
}

func cloneCertificate(certificate domain.Certificate) domain.Certificate {
	return certificate
}
