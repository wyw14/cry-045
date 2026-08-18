# Bug

证书注册只以证书 ID 为键，不同项目使用相同 ID 时会相互覆盖，绑定时也未校验材料归属。

# Reproduction

```bash
go test ./internal/repository -run TestCertificateIDsAreScopedByProjectAndMaterial -count=20
```

# Expected failure

测试会显示 project-a 读取到了 project-b 的证书内容。
