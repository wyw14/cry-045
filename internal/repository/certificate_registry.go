package repository

import (
	"context"
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
	return &CertificateRegistry{certificates: map[string]ProjectCertificate{}, bindings: map[string]string{}}
}

func (r *CertificateRegistry) Register(ctx context.Context, projectID string, certificate domain.Certificate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.certificates[certificate.RegistryKey(projectID)] = ProjectCertificate{ProjectID: projectID, Certificate: certificate}
	return nil
}

func (r *CertificateRegistry) Bind(ctx context.Context, projectID, materialID, certificateID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bindings[projectID+"/"+materialID] = certificateID
	return nil
}

func (r *CertificateRegistry) Bound(ctx context.Context, projectID, materialID string) (domain.Certificate, error) {
	if err := ctx.Err(); err != nil {
		return domain.Certificate{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	certificateID := r.bindings[projectID+"/"+materialID]
	key := domain.Certificate{ID: certificateID, MaterialID: materialID}.RegistryKey(projectID)
	entry, ok := r.certificates[key]
	if !ok {
		return domain.Certificate{}, domain.ErrNotFound
	}
	return entry.Certificate, nil
}
