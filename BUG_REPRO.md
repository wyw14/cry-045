# Bug

材料查询在筛选和排序之前截取分页窗口，导致跨页漏项，并把窗口长度误当作筛选后的总数。

# Reproduction

```bash
go test ./internal/application -run TestMaterialQueryFiltersAndSortsBeforePagination -count=20
```

# Expected failure

测试会报告 `Total` 不等于筛选结果总量，或第二页不是预期材料。
