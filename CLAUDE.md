# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 提案规范

- 所有 OpenSpec artifacts 都必须使用中文编写

## 代码质量检查

编辑代码后，必须按以下顺序执行检查：

```bash
# 1. 代码格式化（必须在编辑后立即执行）
go fmt ./...
# 2. 静态分析检查语法问题（使用 go vet 而非 go build）
go vet ./...
# 3. 运行单元测试确认功能正常
go test ./...
```

**说明**：
- `go fmt ./...` - 自动格式化所有 Go 代码，确保代码风格一致
- `go vet ./...` - 检查代码中的常见错误，而不是使用 `go build`
- `go test ./...` - 运行所有单元测试，确保修改没有破坏现有功能
- **禁止** 使用 `go build` ，构建没有必要，而且产生预期之外的产物，且 vet 能发现 build 无法检测的问题

## 国际化

原则上对用户展示的文本（不含日志）都需要支持中文和英文两种语言。这些需要支持国际化的文本需要以 `i18n.Message` 结构体的形式定义在同一个包中的 `i18n.go` 文件中，比如 CLI 命令相关的描述文本定义在 `pkg/commands/i18n.go` 中。

翻译文件在 `pkg/i18n/active.en.yaml` 和 `pkg/i18n/active.zh.yaml` 中，但是这两个文件 **不能直接修改** ，需要通过 Skill `i18n-translate` 从代码中提取 `i18n.Message` 生成。

## 日志

日志统一使用 `github.com/go-logr/logr` 输出。在程序入口已经初始化了 logr.Logger 并注入到了上下文中，一般通过上下文 `ctx context.Context` 传递 logger 。通过 `logger := logr.FromContextOrDiscard(ctx)` 可以获取 logger 。

只有在某些不适合传递 ctx 的特殊模块中（比如特别简单、基础的纯计算方法不适合传递 ctx ），可以考虑在初始化结构体时传入 logger ，但是这种模块一般也不需要输出日志。

## 关注架构设计

架构设计需要尤为慎重，坏的架构将导致可修改性破坏，脆弱的逻辑往往诞生于不合理的架构中。不要急于确定一个模块的架构设计，从多个不同角度推敲，自然地推导出合理的设计。

每当设计一个功能模块尝试考虑这样几个问题：

1. 这个模块的功能是什么，边界如何定义？
2. 这个模块与其它关联模块通过什么方式交互？
3. 这个模块具有什么样的接口？

抛开具体的问题，从更高的视角单独审视上述问题的回答，这些回答是否合理（合理指它们清爽、优雅、易于理解）？如果不合理应该如何修改？重复这些思考直到自己觉得满意。
