## 1. Gemini 请求构造器

- [x] 1.1 创建 `pkg/llm/req_gemini.go`，定义 REST API 请求体结构体 `geminiGenerateContentRequest` 和 `geminiGenerationConfig`
- [x] 1.2 定义 `GeminiGenerateContentRequestBuilder` 接口（扩展 `RequestBuilder`，含 Gemini 专属参数方法）
- [x] 1.3 实现 `GeminiGenerateContentRequest` 结构体及其 `RequestBuilder` 接口方法（`WithContext`、`WithURL`、`WithBaseURL`、`WithAPIKey`、`WithHeader`、`WithModel`、`WithStream`）
- [x] 1.4 实现 `WithMessages` 方法：将内部 `Message` 转换为 `[]*genai.Content` 和 `*genai.Content`（systemInstruction），处理连续同角色消息合并
- [x] 1.5 实现 `GeminiGenerateContentRequestBuilder` 专属方法（`WithTemperature`、`WithTopP`、`WithTopK`、`WithMaxOutputTokens`、`WithStopSequences`、`WithSeed`、`WithFrequencyPenalty`、`WithPresencePenalty`、`WithResponseMIMEType`、`WithResponseSchema`、`WithThinkingConfig`、`WithCandidateCount`）
- [x] 1.6 实现 `BuildBody()` 和 `Build()` 方法：序列化请求体为 JSON，构造 `*http.Request`

## 2. Gemini 响应处理

- [x] 2.1 创建 `pkg/llm/resp_handler_gemini.go`，实现 `GeminiGenerateContentResponseHandler`
- [x] 2.2 实现非流式 `Handle` 方法：读取响应体，JSON unmarshal 为 `genai.GenerateContentResponse`，提取文本内容，存入 SessionStore
- [x] 2.3 实现流式 `HandleStream` 方法：SSE 逐行解析，JSON unmarshal 每个 chunk，渐进式输出并累积文本，完成时存入 SessionStore

## 3. Gemini 输出格式化

- [x] 3.1 创建 `pkg/llm/format_gemini.go`，实现 `GeminiGenerateContentHumanReadableFormatter`
- [x] 3.2 实现非流式格式化：输出候选文本，可选展示思考内容（`Part.Thought == true`）
- [x] 3.3 实现流式格式化：渐进式输出文本增量，流结束输出换行
- [x] 3.4 思考内容使用暗色样式（`\x1b[2m`），通过 `ShowReasoningContent` 控制

## 4. Gemini CLI 子命令

- [x] 4.1 创建 `pkg/commands/gemini.go`，定义 `GeminiOptions` 结构体和 `AddPFlags` 方法
- [x] 4.2 实现 `newGeminiCommand()` 函数：组装 request builder → 构建请求 → 发送 HTTP → 处理响应
- [x] 4.3 支持 `--dry-run` 模式：仅打印请求体 JSON
- [x] 4.4 支持 verbose 模式：打印请求和响应详情

## 5. 集成与注册

- [x] 5.1 在 `pkg/commands/root.go` 中注册 `gemini` 子命令
- [x] 5.2 在 `pkg/commands/i18n.go` 中添加 gemini 命令相关的 i18n.Message 定义

## 6. 代码质量检查

- [x] 6.1 运行 `go fmt ./...` 格式化代码
- [x] 6.2 运行 `go vet ./...` 检查语法问题
- [x] 6.3 运行 `go test ./...` 确保现有测试通过
