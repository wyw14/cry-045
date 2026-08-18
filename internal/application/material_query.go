package application

import (
	"sort"
	"strings"

	"github.com/wyw14/cry045/internal/domain"
)

type MaterialQuery struct {
	Page     int
	PageSize int
	Sort     string
	Risk     string
	Process  string
}

type MaterialPage struct {
	Items    []domain.Material
	Page     int
	PageSize int
	Total    int
}

func QueryMaterials(materials []domain.Material, query MaterialQuery) MaterialPage {
	normalized := normalizeMaterialQuery(query)
	filtered := filterMaterials(materials, normalized)
	sortMaterials(filtered, normalized.Sort)
	items := materialPageSlice(filtered, normalized.Page, normalized.PageSize)
	return MaterialPage{
		Items:    cloneMaterials(items),
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
		Total:    len(filtered),
	}
}

func normalizeMaterialQuery(query MaterialQuery) MaterialQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.Sort = strings.ToLower(strings.TrimSpace(query.Sort))
	if query.Sort == "" {
		query.Sort = "code"
	}
	query.Risk = strings.ToLower(strings.TrimSpace(query.Risk))
	query.Process = strings.ToLower(strings.TrimSpace(query.Process))
	return query
}

func filterMaterials(materials []domain.Material, query MaterialQuery) []domain.Material {
	result := make([]domain.Material, 0, len(materials))
	for _, material := range materials {
		if query.Risk != "" && !strings.EqualFold(material.RiskClass, query.Risk) {
			continue
		}
		if query.Process != "" && !hasProcess(material.AllowedProcesses, query.Process) {
			continue
		}
		result = append(result, material)
	}
	return result
}

func hasProcess(processes []string, wanted string) bool {
	for _, process := range processes {
		if strings.EqualFold(strings.TrimSpace(process), wanted) {
			return true
		}
	}
	return false
}

func sortMaterials(materials []domain.Material, key string) {
	descending := strings.HasPrefix(key, "-")
	field := strings.TrimPrefix(key, "-")
	sort.SliceStable(materials, func(i, j int) bool {
		left, right := materialSortValue(materials[i], field), materialSortValue(materials[j], field)
		if descending {
			return left > right
		}
		return left < right
	})
}

func materialSortValue(material domain.Material, field string) string {
	switch field {
	case "name":
		return strings.ToLower(material.Name)
	case "risk":
		return strings.ToLower(material.RiskClass)
	default:
		return strings.ToLower(material.Code)
	}
}

func materialPageSlice(materials []domain.Material, page, size int) []domain.Material {
	start := (page - 1) * size
	if start >= len(materials) {
		return []domain.Material{}
	}
	end := start + size
	if end > len(materials) {
		end = len(materials)
	}
	return materials[start:end]
}

func cloneMaterials(materials []domain.Material) []domain.Material {
	out := make([]domain.Material, len(materials))
	for index, material := range materials {
		out[index] = material
		out[index].AllowedProcesses = append([]string(nil), material.AllowedProcesses...)
		out[index].RequiredEvidence = append([]string(nil), material.RequiredEvidence...)
		out[index].SubstituteIDs = append([]string(nil), material.SubstituteIDs...)
	}
	return out
}
