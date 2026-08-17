# 工业材料合规审核与退回追溯平台

这是一个可离线运行的 Go + Vue 材料合规审核示例。设计人员创建选型申请，合规人员执行规则校验、初审和复审，退回意见与修订版本都保留在不可变时间线上。

## 模块职责

- `internal/domain`：材料、证书、申请、问题、审批状态和审计事件。
- `internal/application`：提交、校验、退回修订、批准、导出和影响分析用例。
- `internal/repository`：内存仓储和 PostgreSQL 连接适配器边界。
- `internal/transport/http`：`/api/v1` JSON API、分页、错误码与 request_id。
- `internal/platform`：本地附件、通知和定时复核适配器。
- `web`：Vue 3 + TypeScript + Vite + Pinia 的离线管理界面骨架。
- `migrations`：可重复执行的 PostgreSQL schema；`scripts/seed.go` 提供演示数据。

## 本地启动

复制 `.env.example` 为 `.env`，不需要外部服务即可运行内存演示：

```bash
go run ./cmd/server
curl http://localhost:8080/healthz
```

生产环境把 `DATABASE_URL` 指向 PostgreSQL，并由部署脚本执行 `migrations/001_init.sql`。数据库只保存业务数据，附件默认落在受控的 `LOCAL_FILE_ROOT` 目录。

## 状态规则

申请按 `draft -> submitted -> first_review -> second_review -> returned -> submitted -> approved -> archived` 流转，也可以从草稿或退回版本作废。退回只能基于当前版本修订；批准版本替换时会保留原审查依据。

## 接口示例

```bash
curl -H 'X-Request-ID: demo-1' http://localhost:8080/api/v1/materials?page=1&page_size=20
curl -X POST http://localhost:8080/api/v1/selections/demo-1/submit
```

## 测试与验证

```bash
go test ./...
go test -race ./...
go vet ./...
cd web && npm test && npm run build
```

演示数据通过 `internal/repository.NewDemoRepository` 加载，不覆盖已有数据。项目不提交密钥、依赖缓存、构建产物或运行期数据库文件。
