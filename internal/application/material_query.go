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
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start > len(materials) {
		start = len(materials)
	}
	if end > len(materials) {
		end = len(materials)
	}
	items := append([]domain.Material(nil), materials[start:end]...)
	filtered := items[:0]
	for _, material := range items {
		if query.Risk != "" && material.RiskClass != query.Risk {
			continue
		}
		if query.Process != "" && !contains(material.AllowedProcesses, query.Process) {
			continue
		}
		filtered = append(filtered, material)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.Compare(filtered[i].Code, filtered[j].Code) < 0
	})
	return MaterialPage{Items: filtered, Page: query.Page, PageSize: query.PageSize, Total: len(items)}
}
