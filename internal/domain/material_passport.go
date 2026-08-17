package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MaterialPassport is the signed, human-readable compliance snapshot attached
// to an approved selection. It is intentionally immutable once issued.
type MaterialPassport struct {
	PassportID       string
	MaterialID       string
	MaterialCode     string
	SelectionID      string
	Revision         int
	ProjectID        string
	RiskClass        string
	DeclaredUses     []string
	SubstanceEntries []SubstanceEntry
	CertificateRefs  []string
	RulebookID       string
	IssuedAt         time.Time
	Issuer           string
	Digest           string
}

type SubstanceEntry struct {
	Name       string
	CAS        string
	MassPPM    int
	Threshold  int
	Restricted bool
	Source     string
}

type PassportDelta struct {
	AddedSubstances     []string
	RemovedSubstances   []string
	ChangedThresholds   []string
	ChangedCertificates []string
	ChangedUses         []string
	RequiresReapproval  bool
	Explanation         string
}

type ImpactVector struct {
	ProjectID       string
	SelectionID     string
	MaterialID      string
	Revision        int
	AffectedParts   []string
	AffectedStages  []string
	AffectedReports []string
	Severity        string
	Rationale       string
}

func (p MaterialPassport) CanonicalDigest() string {
	parts := []string{p.PassportID, p.MaterialID, p.SelectionID, fmt.Sprint(p.Revision), p.ProjectID, p.RulebookID, p.IssuedAt.UTC().Format(time.RFC3339), p.Issuer}
	for _, entry := range p.SubstanceEntries {
		parts = append(parts, strings.Join([]string{entry.Name, entry.CAS, fmt.Sprint(entry.MassPPM), fmt.Sprint(entry.Threshold), fmt.Sprint(entry.Restricted), entry.Source}, ":"))
	}
	sort.Strings(parts)
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}

func (p *MaterialPassport) Seal() error {
	if strings.TrimSpace(p.PassportID) == "" || strings.TrimSpace(p.MaterialID) == "" || p.Revision < 1 {
		return fmt.Errorf("passport identity is incomplete")
	}
	if p.IssuedAt.IsZero() {
		return fmt.Errorf("passport issue time is required")
	}
	p.Digest = p.CanonicalDigest()
	return nil
}

func (p MaterialPassport) VerifySeal() bool { return p.Digest != "" && p.Digest == p.CanonicalDigest() }

func (p MaterialPassport) RestrictedNames() []string {
	names := make([]string, 0)
	for _, entry := range p.SubstanceEntries {
		if entry.Restricted || entry.MassPPM > entry.Threshold {
			names = append(names, entry.Name)
		}
	}
	sort.Strings(names)
	return names
}

func ComparePassports(before, after MaterialPassport) PassportDelta {
	oldSubstances, newSubstances := map[string]SubstanceEntry{}, map[string]SubstanceEntry{}
	for _, entry := range before.SubstanceEntries {
		oldSubstances[entry.Name] = entry
	}
	for _, entry := range after.SubstanceEntries {
		newSubstances[entry.Name] = entry
	}
	delta := PassportDelta{}
	for name, entry := range newSubstances {
		old, exists := oldSubstances[name]
		if !exists {
			delta.AddedSubstances = append(delta.AddedSubstances, name)
			continue
		}
		if old.Threshold != entry.Threshold || old.MassPPM != entry.MassPPM || old.Restricted != entry.Restricted {
			delta.ChangedThresholds = append(delta.ChangedThresholds, name)
		}
	}
	for name := range oldSubstances {
		if _, exists := newSubstances[name]; !exists {
			delta.RemovedSubstances = append(delta.RemovedSubstances, name)
		}
	}
	oldCerts, newCerts := stringSet(before.CertificateRefs), stringSet(after.CertificateRefs)
	for cert := range newCerts {
		if !oldCerts[cert] {
			delta.ChangedCertificates = append(delta.ChangedCertificates, "added:"+cert)
		}
	}
	for cert := range oldCerts {
		if !newCerts[cert] {
			delta.ChangedCertificates = append(delta.ChangedCertificates, "removed:"+cert)
		}
	}
	oldUses, newUses := stringSet(before.DeclaredUses), stringSet(after.DeclaredUses)
	for use := range newUses {
		if !oldUses[use] {
			delta.ChangedUses = append(delta.ChangedUses, "added:"+use)
		}
	}
	for use := range oldUses {
		if !newUses[use] {
			delta.ChangedUses = append(delta.ChangedUses, "removed:"+use)
		}
	}
	delta.RequiresReapproval = len(delta.AddedSubstances)+len(delta.RemovedSubstances)+len(delta.ChangedThresholds)+len(delta.ChangedCertificates) > 0
	delta.Explanation = fmt.Sprintf("新增 %d、移除 %d、阈值变化 %d、证书变化 %d", len(delta.AddedSubstances), len(delta.RemovedSubstances), len(delta.ChangedThresholds), len(delta.ChangedCertificates))
	sort.Strings(delta.AddedSubstances)
	sort.Strings(delta.RemovedSubstances)
	sort.Strings(delta.ChangedThresholds)
	sort.Strings(delta.ChangedCertificates)
	sort.Strings(delta.ChangedUses)
	return delta
}

func BuildImpactVector(selection SelectionRequest, changedFields []string) ImpactVector {
	vector := ImpactVector{ProjectID: selection.ProjectID, SelectionID: selection.ID, MaterialID: selection.MaterialID, Revision: selection.Revision, Severity: "low", Rationale: "版本变更尚未评估"}
	for _, field := range changedFields {
		switch field {
		case "material", "certificate", "substance":
			vector.Severity = "high"
			vector.AffectedParts = append(vector.AffectedParts, "产品物料清单")
			vector.AffectedReports = append(vector.AffectedReports, "合规放行报告")
		case "process":
			vector.Severity = "medium"
			vector.AffectedStages = append(vector.AffectedStages, selection.Process)
		case "purpose":
			vector.AffectedParts = append(vector.AffectedParts, "设计用途说明")
		case "quantity":
			vector.AffectedReports = append(vector.AffectedReports, "用量核算表")
		default:
			vector.AffectedReports = append(vector.AffectedReports, "版本审计记录")
		}
	}
	sort.Strings(vector.AffectedParts)
	sort.Strings(vector.AffectedStages)
	sort.Strings(vector.AffectedReports)
	if len(changedFields) > 0 {
		vector.Rationale = "字段变化已映射到受影响的项目材料清单和报告"
	}
	return vector
}

func stringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}
