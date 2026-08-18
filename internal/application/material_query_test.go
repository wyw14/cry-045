package application

import (
	"testing"

	"github.com/wyw14/cry045/internal/domain"
)

func TestMaterialQueryFiltersAndSortsBeforePagination(t *testing.T) {
	materials := []domain.Material{
		{ID: "m-z", Code: "Z-9", RiskClass: "high", AllowedProcesses: []string{"laser"}},
		{ID: "m-b", Code: "B-2", RiskClass: "medium", AllowedProcesses: []string{"cnc"}},
		{ID: "m-x", Code: "X-8", RiskClass: "low", AllowedProcesses: []string{"laser"}},
		{ID: "m-a", Code: "A-1", RiskClass: "medium", AllowedProcesses: []string{"cnc"}},
	}
	page := QueryMaterials(materials, MaterialQuery{Page: 2, PageSize: 1, Sort: "code", Process: "cnc"})
	if page.Total != 2 {
		t.Fatalf("Total = %d, want 2 filtered materials", page.Total)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "m-b" {
		t.Fatalf("page items = %#v, want second sorted CNC material m-b", page.Items)
	}
}
