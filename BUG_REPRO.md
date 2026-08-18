# Bug

报告导出直接截断最终文件，编码器中途失败时会破坏上一份有效报告。

# Reproduction

```bash
go test ./internal/platform -run TestFailedReportExportPreservesExistingFile -count=20
```

# Expected failure

测试会报告原有报告内容已被 `partial-report` 覆盖。
