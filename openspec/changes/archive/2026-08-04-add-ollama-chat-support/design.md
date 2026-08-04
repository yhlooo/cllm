## Context

项目当前通过 `cllm openai` 子命令支持 OpenAI Chat Completions API，拥有成熟的接口抽象层（RequestBuilder、ResponseHandler、Formatter、SessionStore）。需要新增对 Ollama Chat API 的支持，遵循相同的架构模式。

Ollama Chat API (`POST /api/chat`) 与 OpenAI Chat Completions API 高度相似：都使用 `messages` 数组表示对话、都支持流式/非流式、都支持多模态输入。关键差异：
- 模型参数嵌套在 `options` 对象中，而非请求体顶层
- 流式格式使用 NDJSON（每行一个完整 JSON），而非 SSE
- 默认无需认证

## Goals / Non-Goals

**Goals:**
- 支持通过 `cllm ollama` 子命令向 Ollama 服务发送 Chat API 请求
- 支持流式（NDJSON 逐行）和非流式响应
- 复用现有 RequestBuilder / ResponseHandler / Formatter / SessionStore 接口体系
- 支持多模态输入（图片附件），用法与 openai 子命令一致
- 支持会话持久化（`--session-file`）
- 支持多种输出格式（human-readable / JSON / raw）

**Non-Goals:**
- Ollama 管理类 API（模型拉取/列表/删除等）
- Ollama Generate API（使用 Chat API 代替）
- Tool calling / Function calling
- 文本转语音等多模态输出

## Decisions

### 1. 不引入 Ollama SDK，使用标准库

**决策**：直接用 `net/http` + `encoding/json` 构建请求和解析响应，不依赖第三方 Ollama Go SDK。

**理由**：Ollama Chat API 的请求/响应结构简单，第三方 SDK 带来的收益有限，反而增加依赖维护负担。参考现有 OpenAI 实现，`openai-go` SDK 主要用于处理复杂 union types，Ollama 无此需求。

### 2. 类型定义独立为 pkg/ollama/types.go

**决策**：Ollama 专属数据结构（ChatRequest、ChatResponse、ChatMessage、Tool、ToolFunction、ToolCall、Options 等）放在 `pkg/ollama/types.go`，与 `pkg/llm/` 的集成逻辑分离。

**理由**：
- 关注点分离：类型定义是纯数据，不依赖任何接口
- 可被独立引用：理论上其他项目可以只引用 `pkg/ollama/` 而无需引入 `pkg/llm/` 的接口体系
- 文件清爽：`types.go` 只包含 struct 定义，零逻辑

**备选方案**：放在 `pkg/llm/` —— 会使 llm 包变臃肿，且混合了通用接口和特定 API 的类型。

### 3. OllamaChatRequestBuilder 扩展 RequestBuilder

**决策**：创建 `OllamaChatRequestBuilder` 接口，内嵌 `RequestBuilder`，添加 Ollama 特有方法：

```go
type OllamaChatRequestBuilder interface {
    RequestBuilder
    WithFormat(format any) OllamaChatRequestBuilder
    WithOptions(opts ollama.Options) OllamaChatRequestBuilder
    WithKeepAlive(duration any) OllamaChatRequestBuilder
    WithThink(think any) OllamaChatRequestBuilder
    WithLogprobs(enabled bool) OllamaChatRequestBuilder
    WithTopLogprobs(v int) OllamaChatRequestBuilder
}
```

**理由**：与 `OpenAIChatCompletionRequestBuilder` 的模式完全一致 —— 通用参数通过 `RequestBuilder` 设置，平台特有参数通过扩展接口设置。CLI 层通过类型断言访问扩展方法。

### 4. 消息转换对标 OpenAI

**决策**：在 `req_ollama.go` 中实现 `llm.Message → ollama.ChatMessage` 的转换，处理逻辑对标 `newOpenAIMessage()`：

```
llm.Message                         ollama.ChatMessage
──────────────────────────          ──────────────────────────
system + Text              →        {role: "system", content: "文本"}
user + Text                →        {role: "user", content: "文本"}
user + Text + Blob(image)  →        {role: "user", content: "文本",
                                       images: ["base64..."]}
assistant + Text           →        {role: "assistant", content: "文本"}
assistant + Text + Refusal →        {role: "assistant", content: "文本拒绝原因"}
```
Ollama Chat API 的图片通过 `images` 字段传递（base64 字符串数组），而非 OpenAI 风格的 content array with `image_url` 类型。

### 5. 流式解析：NDJSON 逐行

**决策**：`HandleStream` 使用 `bufio.Scanner` 逐行读取，每行直接 JSON 反序列化为 `ollama.ChatResponse`，以 `done: true` 作为流结束标志。

与 OpenAI 的 SSE 格式对比：
```
OpenAI (SSE):                     Ollama (NDJSON):
data: {"choices":[...]}\n         {"message":{...},"done":false}\n
data: {"choices":[...]}\n         {"message":{...},"done":false}\n
data: [DONE]\n                    {"message":{...},"done":true}\n
```

无需 `data:` 前缀剥离和 `[DONE]` 判断，解析更简单。

### 6. CLI 命令结构对标 openai

**决策**：`ollama` 子命令的 RunE 流程与 `openai` 几乎一致：

```
NewOllamaChatRequest()
  → WithBaseURL / WithModel / WithStream
  → SessionStore.Load() → WithMessages(history...)
  → WithMessages(systemMsg)
  → WithMessages(userMsg)
  → Builder.(OllamaChatRequestBuilder).WithFormat(...)
  → Build() → http.Do() → Handler.Handle[Stream]()
```

**共用参数名**：`--model`、`--stream`、`--session-file`、`--system-prompt`、`--attachment`、`--url`、`--api-key`、`--header` 等与 openai 子命令保持一致。

**Ollama 默认 URL**：`http://localhost:11434`，符合 Ollama 默认配置。

**Ollama 特有参数**：

| 参数 | 说明 |
|------|------|
| `--format` | 设为 `"json"` 或 JSON Schema 对象 |
| `--keep-alive` | 模型在内存中保持的时间，如 `"5m"`、`0`（立即卸载） |
| `--think` | 思考模式：`true`/`false` 或 `high`/`medium`/`low`/`max` |
| `--logprobs` | 是否返回输出 token 的对数概率 |
| `--top-logprobs` | 每个 token 位置返回的最可能 token 数量 |
| `--show-reasoning` | 是否显示思考过程（暗色输出） |
| `--temperature` 等 | 进入 `options` 对象而非顶层字段 |

## Risks / Trade-offs

- **Ollama API 版本兼容**：Ollama API 仍在演进，`pkg/ollama/types.go` 作为独立类型定义层起到缓冲作用，API 变化只需更新类型和转换逻辑
- **Options 字段不完整**：Ollama 支持的 model options 很多（mirostat、num_ctx 等），初期只暴露常用参数。后续可按需添加，不影响架构
- **无认证支持**：Ollama 默认无认证，但保留 `--api-key` 和 `--header` 参数供代理认证场景使用
