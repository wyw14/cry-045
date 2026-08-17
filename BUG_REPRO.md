# Bug 是什么

已批准的项目材料被新版本替换后，原批准版本及其审核依据没有保留在历史中，审计时只能看到覆盖后的记录。

# 如何触发

在项目根目录执行：

```bash
go test ./internal/repository -run TestApprovedReplacementPreservesOriginalReviewBasis -count=20
```

# 错误信息

测试失败并报告：

```text
approved basis was overwritten
```
