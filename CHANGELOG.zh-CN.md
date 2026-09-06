# 变更日志

[English](CHANGELOG.md) | 中文

这里记录 Goark Core 的重要变更。

## [未发布]

暂无未发布变更。

## [0.0.1] - 2026-09-06

### 新增

- 显式 Bean 定义、Scope、别名、依赖图、有序启动和生命周期安全的应用上下文。
- 配置环境、属性源、类型转换、占位符、配置属性契约和安全 GaEL 表达式求值。
- 同步有序事件和结构化框架错误。
- Go 原生 Web 与 MVC 契约，覆盖路由、绑定、校验、Advice、Filter、Interceptor、
  静态资源、视图、流式响应、WebSocket 和 HTTP Client。
- 基于 Go 1.26 的跨平台 CI、vet 和 race 门禁。

### 变更

- 将所有实际使用的 `golang.org/x` 模块对齐到最新稳定版本。

### 修复

- 空生命周期转换不再失败。
- 启动遍历遵守 Bean 顺序。
- 保持传输层中立 Servlet 请求和 Cookie 缺失语义。
- 条件路由选择会保留协商后的响应媒体类型。

[未发布]: https://github.com/goark-projects/goark/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/goark-projects/goark/releases/tag/v0.0.1
