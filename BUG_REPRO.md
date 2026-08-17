# Bug 是什么

含有循环依赖或会提升合规风险的替代材料方案仍被判定为可用，危险候选没有形成阻断项。

# 如何触发

在项目根目录执行：

```bash
go test ./internal/application -run TestSubstitutePlanRejectsCyclesAndRiskEscalation -count=20
```

# 错误信息

测试失败并报告：

```text
unsafe substitute graph passed
```
