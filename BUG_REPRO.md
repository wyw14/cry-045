# Bug 是什么

两个修改者基于同一旧版本连续提交时，第二个过期写入没有被拒绝，会静默覆盖先提交的材料修订。

# 如何触发

在项目根目录执行：

```bash
go test ./internal/repository -run TestApplyRevisionRejectsStaleSecondWriter -count=20
```

# 错误信息

测试失败并报告：

```text
stale writer was accepted: <nil>
```
