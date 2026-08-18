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
	filtered := make([]domain.Material, 0, len(materials))
	for _, material := range materials {
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
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]domain.Material(nil), filtered[start:end]...)
	return MaterialPage{Items: items, Page: query.Page, PageSize: query.PageSize, Total: len(filtered)}
}
