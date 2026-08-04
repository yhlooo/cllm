## Context

cllm 现有架构基于三个核心抽象：`RequestBuilder`（构建 HTTP 请求）、`ResponseHandler`（处理响应）、`Formatter`（格式化输出）。OpenAI 的实现（`OpenAIChatCompletionRequest`）展示了 SDK 类型 + 手动 HTTP 的模式。Gemini 的实现需要遵循相同架构模式，使用 `google.golang.org/genai` SDK 的类型定义，但通过标准库 `net/http` 发送请求。

## Goals / Non-Goals

**Goals:**
- 实现 `RequestBuilder` 和 `GeminiGenerateContentRequestBuilder` 接口，与 OpenAI 实现保持一致
- 支持 Gemini generateContent REST API 的非流式和流式（SSE）调用
- 复用内部 `Message` 类型，转换为 Gemini 的 `Content`/`Part` 结构
- 支持 Gemini 特有参数：`topK`、`thinkingConfig`、`responseMimeType` 等
- `ShowReasoningContent` 控制思考内容展示（与 OpenAI 参数名保持一致）

**Non-Goals:**
- 不实现 Vertex AI 端点（仅支持 Gemini Developer API `generativelanguage.googleapis.com`）
- 不实现 Function Calling / Tools（可后续扩展）
- 不实现图片/视频生成（`generateImages`、`generateVideos`）
- 不修改现有 `RequestBuilder`、`Formatter`、`ResponseHandler`、`SessionStore` 公共接口

## Decisions

### 1. 类型策略：自定义 REST 请求体 + 复用 SDK 嵌套类型

Gemini SDK 的 `GenerateContentConfig` 使用 Go 侧的统一抽象，其字段在 REST API 中分布在顶层（`systemInstruction`、`safetySettings`）和 `generationConfig` 嵌套对象中。因此需要自定义顶层请求体结构体 `geminiGenerateContentRequest`，直接映射 REST API JSON 格式。

对于嵌套类型（`Content`、`Part`、`Blob`、`Schema`、`ThinkingConfig`、`SafetySetting` 等），其 SDK 定义的 JSON 标签与 REST API 一致，可直接复用。

**替代方案**：完全自定义所有类型。被拒绝——增加维护成本，且 SDK 的嵌套类型已成熟稳定。

### 2. HTTP 请求构造：标准库 `net/http`

与 OpenAI 实现一致：SDK 类型用于 `json.Marshal` 序列化请求体，通过 `http.NewRequestWithContext` 构造请求，`http.DefaultClient.Do` 发送。

认证方式：API Key 通过 `x-goog-api-key` 请求头传递（Gemini Developer API 标准方式）。

**替代方案**：使用 SDK 的 `client.Models.GenerateContent()`。被拒绝——无法适配 `RequestBuilder` 接口，且无法控制 HTTP 细节（自定义 header、verbose 日志等）。

### 3. 消息转换：system → systemInstruction，连续同角色合并

`Message` → Gemini REST API 的转换规则：
- `RoleSystem` → 顶层 `systemInstruction` 字段（Gemini 不支持 system 角色在 contents 中）
- `RoleUser` → `Content{Role: "user"}`
- `RoleAssistant` → `Content{Role: "model"}`
- `RoleTool` → 暂不支持，记录错误

多个连续同角色消息在 `Build()` 时自动合并，因为 Gemini API 要求 contents 中 user/model 严格交替。

### 4. 流式响应处理：SSE 逐行解析

Gemini 流式 API （URL 通过 `url.Values` 编码添加 `?alt=sse` 参数）返回 SSE 格式 `data: <json>\n\n`，与 OpenAI 的流式格式相似。解析方式：`bufio.Scanner` 逐行读取，`strings.TrimPrefix("data:")` 后 JSON unmarshal 到 `genai.GenerateContentResponse`。

### 5. `GeminiGenerateContentRequestBuilder` 接口设计

在 `RequestBuilder` 基础上扩展 Gemini 专属方法：

```go
type GeminiGenerateContentRequestBuilder interface {
    RequestBuilder

    WithTemperature(v float64) GeminiGenerateContentRequestBuilder
    WithTopP(v float64) GeminiGenerateContentRequestBuilder
    WithTopK(v float64) GeminiGenerateContentRequestBuilder
    WithMaxOutputTokens(v int32) GeminiGenerateContentRequestBuilder
    WithStopSequences(v ...string) GeminiGenerateContentRequestBuilder
    WithSeed(v int32) GeminiGenerateContentRequestBuilder
    WithFrequencyPenalty(v float64) GeminiGenerateContentRequestBuilder
    WithPresencePenalty(v float64) GeminiGenerateContentRequestBuilder
    WithResponseMIMEType(v string) GeminiGenerateContentRequestBuilder
    WithResponseSchema(schema *genai.Schema) GeminiGenerateContentRequestBuilder
    WithThinkingConfig(cfg *genai.ThinkingConfig) GeminiGenerateContentRequestBuilder
    WithCandidateCount(v int32) GeminiGenerateContentRequestBuilder
}
```

### 6. CLI 参数命名

与 OpenAI 保持一致的参数使用相同命名（`--temperature`、`--top-p`、`--seed`、`--frequency-penalty`、`--presence-penalty`）。Gemini 特有的使用 Gemini API 原始命名（`--top-k`、`--max-output-tokens`）。思考功能分为两个独立 flag：`--thinking` 控制 API 参数 `thinkingConfig.includeThoughts`（是否让模型生成思考），`--show-reasoning` 控制是否在输出中展示思考内容（纯显示控制，不影响 API 参数，与 OpenAI 参数名保持一致）。

### 7. 思考内容展示

Gemini 响应中 `Part{Thought: true, Text: "..."}` 为模型思考内容。参考 OpenAI 的 `reasoning_content` 处理方式，用暗色样式（`\x1b[2m`）输出，通过 `--show-reasoning` 开关控制。格式化器通过 `Streaming` 字段区分流式和非流式：流式模式下每个 chunk 直接输出文本不追加换行，仅在 `finishReason` 出现时追加换行；非流式模式下每段文本后追加换行。

## Risks / Trade-offs

- **[风险] Gemini API 突变**: SDK 升级后类型定义可能变化 → 通过 go.mod 锁定版本，升级时手动验证 JSON 序列化兼容性
- **[权衡] systemInstruction 仅支持单条**: Gemini API 的 `systemInstruction` 是一个 `Content` 对象（非数组），多条 system 消息需合并为一条 → 接受限制，常见使用场景仅一条 system prompt
- **[风险] 连续同角色消息合并可能丢失信息**: 合并后的语义可能不完全等价 → 在实际使用场景中（CLI 对话），连续同角色消息不常见
- **[权衡] 不支持 Vertex AI**: 仅支持 Gemini Developer API → 简化实现，后续可按需扩展
