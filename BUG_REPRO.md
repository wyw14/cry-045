# Bug 是什么

有效期只覆盖评审开始时刻、却无法覆盖整个决策窗口的证书仍被判定为有效，导致不完整的合规证据通过评审。

# 如何触发

在项目根目录执行：

```bash
go test ./internal/domain -run TestCertificateCoverageSpansEntireDecisionWindow -count=20
```

# 错误信息

测试失败并报告：

```text
short-lived certificate passed the review window
```
