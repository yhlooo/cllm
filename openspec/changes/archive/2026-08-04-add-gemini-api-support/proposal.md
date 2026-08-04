## Why

当前 cllm 仅支持 OpenAI 兼容的 Chat Completions API，需要扩展支持 Google Gemini 的 generateContent API，使用户能够通过统一的 CLI 工具访问多个 LLM 提供商。

## What Changes

- 新增 `cllm gemini` 子命令，支持通过 Gemini generateContent API 与 Gemini 模型对话
- 实现 `RequestBuilder` 公共接口，与 OpenAI 子命令共享统一的消息构建模式
- 新增 `GeminiGenerateContentRequestBuilder` 专属接口，提供 Gemini 特有参数（如 `topK`、`thinkingConfig` 等）
- 新增 Gemini 响应处理，支持流式（SSE）和非流式两种模式
- 新增 Gemini 人类可读格式化器，支持展示模型思考内容（通过 `--show-reasoning` 控制）
- 复用现有的 `Message`、`SessionStore`、`Formatter` 等公共抽象

## Capabilities

### New Capabilities

- `gemini-generate-content`: 通过 Gemini generateContent API 发送对话请求，支持多模态内容（文本、图片、音频等）、流式输出、会话持久化、思考内容展示等功能

### Modified Capabilities

（无，此为新功能，不修改现有能力）

## Impact

- **新增依赖**: `google.golang.org/genai`（已添加至 go.mod）
- **新增文件**: `pkg/llm/req_gemini.go`、`pkg/llm/resp_handler_gemini.go`、`pkg/llm/format_gemini.go`、`pkg/commands/gemini.go`
- **修改文件**: `pkg/commands/root.go`（注册子命令）、`pkg/commands/i18n.go`（新增 i18n 消息）
- **公共接口**: 不影响现有 `RequestBuilder`、`Formatter`、`ResponseHandler`、`SessionStore` 接口
