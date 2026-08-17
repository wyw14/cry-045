package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type SupplierTrace struct { SupplierCode string; BatchCode string; Country string; ManufacturingSite string; ReceivedAt time.Time; LotEntries []LotEntry }
type LotEntry struct { Lot string; Quantity float64; ReceivedAt time.Time; CertificateRefs []string; Quarantine bool }

func (t SupplierTrace) Complete() bool { return t.SupplierCode != "" && t.BatchCode != "" && t.Country != "" && t.ManufacturingSite != "" && len(t.LotEntries) > 0 }
func (t SupplierTrace) QuarantinedLots() []string { result := []string{}; for _, lot := range t.LotEntries { if lot.Quarantine { result = append(result, lot.Lot) } }; sort.Strings(result); return result }
func (t SupplierTrace) CertificateCoverage() map[string]int { result := map[string]int{}; for _, lot := range t.LotEntries { for _, cert := range lot.CertificateRefs { result[cert]++ } }; return result }
func (t SupplierTrace) Summary() string { state := "可追溯"; if !t.Complete() { state = "信息不完整" }; return fmt.Sprintf("%s/%s %s (%s)", t.SupplierCode, t.BatchCode, state, strings.Join(t.QuarantinedLots(), ",")) }
func (t SupplierTrace) LotAt(now time.Time) []LotEntry { result := []LotEntry{}; for _, lot := range t.LotEntries { if !lot.ReceivedAt.After(now) { result = append(result, lot) } }; return result }
