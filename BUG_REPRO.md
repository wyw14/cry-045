# Bug

提交处理在 context 取消后仍会继续执行，并把已经完成的步骤立即写入回执，导致取消请求留下部分业务记录。

# Reproduction

```bash
go test ./internal/application -run TestCancelledSubmissionLeavesNoPartialReceipts -count=20
```

# Expected failure

测试会报告提交没有返回 `context.Canceled`，或取消后仍存在回执。
