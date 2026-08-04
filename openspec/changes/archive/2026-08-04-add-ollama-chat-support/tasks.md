## 1. Ollama 类型定义

- [x] 1.1 创建 `pkg/ollama/types.go`，定义 ChatRequest、ChatResponse、ChatMessage、Tool、ToolFunction、ToolCall、Options 等结构体

## 2. LLM 集成层 —— 请求构造

- [x] 2.1 创建 `pkg/llm/req_ollama.go`，定义 `OllamaChatRequestBuilder` 扩展接口（WithFormat、WithOptions、WithKeepAlive）
- [x] 2.2 实现 `OllamaChatRequest` 结构体，实现 `RequestBuilder` 接口（WithContext、WithURL、WithBaseURL、WithAPIKey、WithModel、WithStream、WithHeader、WithMessages、BuildBody、Build）
- [x] 2.3 实现 `llm.Message` → `ollama.ChatMessage` 的转换函数，支持 system/user/assistant 角色及文本/图片内容块

## 3. LLM 集成层 —— 响应处理

- [x] 3.1 创建 `pkg/llm/resp_handler_ollama.go`，实现 `OllamaChatResponseHandler.Handle()` 非流式响应处理方法
- [x] 3.2 实现 `OllamaChatResponseHandler.HandleStream()` 流式响应处理方法（NDJSON 逐行解析）

## 4. LLM 集成层 —— 输出格式化

- [x] 4.1 创建 `pkg/llm/format_ollama.go`，实现 `OllamaChatHumanReadableFormatter`，支持非流式和流式两种数据类型的格式化输出

## 5. CLI 命令

- [x] 5.1 创建 `pkg/commands/ollama.go`，定义 `OllamaOptions` 结构体和 `newOllamaCommand()` 函数，构建 ollama 子命令
- [x] 5.2 在 `pkg/commands/root.go` 中注册 ollama 子命令
- [x] 5.3 在 `pkg/commands/i18n.go` 中添加 ollama 子命令相关的 i18n 消息

## 6. 代码质量与翻译

- [x] 6.1 运行 `go fmt ./...` 格式化代码
- [x] 6.2 运行 `go vet ./...` 检查语法问题
- [x] 6.3 运行 `go test ./...` 确保测试通过
- [x] 6.4 运行 `i18n-translate` skill 更新翻译文件
