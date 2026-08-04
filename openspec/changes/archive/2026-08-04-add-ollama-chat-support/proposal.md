## Why

项目目前仅支持 OpenAI 兼容的 Chat Completions API，无法与 Ollama 本地模型服务交互。Ollama 作为最流行的本地 LLM 运行工具，其 Chat API 与 OpenAI 接口高度相似，添加对它的支持可以让用户用同一套工具链无缝切换云端和本地模型。

## What Changes

- 新增 `pkg/ollama/` 包，定义 Ollama Chat API 的数据类型（ChatRequest、ChatResponse、ChatMessage、Options 等）
- 新增 `cllm ollama` 子命令，支持通过 `POST /api/chat` 向 Ollama 服务发送对话请求
- 新增 `OllamaChatRequestBuilder` 接口和实现，集成到 `pkg/llm/` 的 RequestBuilder 体系中
- 新增 `OllamaChatResponseHandler`，支持流式（NDJSON）和非流式响应的解析
- 新增 `OllamaChatHumanReadableFormatter`，以人类可读格式输出 Ollama 响应
- Ollama 特有参数（`format`、`keep_alive`、`options`）通过扩展的 `OllamaChatRequestBuilder` 接口暴露
- 文件附件（图片等）的处理复用现有 `RequestBuilder` 接口，与 openai 子命令用法一致

## Capabilities

### New Capabilities

- `ollama-chat`: 通过 `cllm ollama` 子命令向 Ollama Chat API 发送对话请求，支持多模态输入、流式/非流式响应、会话持久化和人类可读/JSON/Raw 多种输出格式

### Modified Capabilities

<!-- 无现有 capability 被修改 -->

## Impact

- **新增文件**: `pkg/ollama/types.go`、`pkg/llm/req_ollama.go`、`pkg/llm/resp_handler_ollama.go`、`pkg/llm/format_ollama.go`、`pkg/commands/ollama.go`
- **修改文件**: `pkg/commands/root.go`（注册新子命令）、`pkg/commands/i18n.go`（添加 i18n 消息）
- **依赖**: 不引入新的外部依赖，HTTP 请求使用标准库 `net/http`
- **接口影响**: 无现有接口变更，完全增量添加
