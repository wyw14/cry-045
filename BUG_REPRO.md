# Bug 是什么

审计时间线没有遵守请求的时间窗口，并且把本应隐藏的审核者身份与变更详情返回给无权限视图。

# 如何触发

在项目根目录执行：

```bash
go test ./internal/transport/http -run TestTimelineViewFiltersWindowAndRedactsReviewerDetails -count=20
```

# 错误信息

测试失败并报告：

```text
timeline ignored requested window
```
