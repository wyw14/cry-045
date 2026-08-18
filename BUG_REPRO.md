# Bug

退回问题答案只按问题 ID 保存，并在整批校验完成前逐项写入，造成跨修订覆盖和失败批次部分提交。

# Reproduction

```bash
go test ./internal/application -run TestRevisionAnswersAreIsolatedAndCommittedAtomically -count=20
```

# Expected failure

测试会显示第一版答案被第二版覆盖，或无效批次留下了部分答案。
