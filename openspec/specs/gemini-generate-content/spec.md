# Gemini Generate Content

通过 CLI 命令行向 Gemini generateContent API 发送对话请求，支持多模态内容、流式输出、会话持久化等功能。

## Requirements

### Requirement: 通过 cllm gemini 子命令发送对话请求

系统 SHALL 提供 `cllm gemini` 子命令，允许用户通过命令行向 Gemini generateContent API 发送对话请求并获取回复。

#### Scenario: 基本文本对话

- **WHEN** 用户执行 `cllm gemini --model gemini-2.5-flash --api-key <KEY> "你好"`
- **THEN** 系统向 `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent` 发送 POST 请求
- **AND** 请求体包含 `contents` 数组，其中有一条 `role: "user"` 且 `parts[].text: "你好"` 的消息
- **AND** 将回复文本输出到 stdout

#### Scenario: 带系统提示的对话

- **WHEN** 用户执行 `cllm gemini --system-prompt "你是一个有帮助的助手" "你好"`
- **THEN** 请求体中的 `systemInstruction` 字段包含系统提示内容
- **AND** 系统提示不在 `contents` 数组中

#### Scenario: 带附件的多模态对话

- **WHEN** 用户执行 `cllm gemini --attachment ./photo.jpg "描述这张图片"`
- **THEN** 请求体 `contents[0].parts` 包含 `inlineData` 块（含 byte 数据和 `mimeType: "image/jpeg"`）
- **AND** 同时包含文本块 `text: "描述这张图片"`

### Requirement: 流式输出支持

系统 SHALL 支持通过 `--stream` / `-s` 参数启用 Gemini 流式响应。

#### Scenario: 流式对话

- **WHEN** 用户执行 `cllm gemini --stream "讲一个故事"`
- **THEN** 系统向 URL 添加 `?alt=sse` 参数
- **AND** 逐步将收到的文本块输出到 stdout
- **AND** 在流式响应结束后，将完整的助手消息存入会话文件（如果指定了 `--session-file`）

### Requirement: 会话持久化

系统 SHALL 支持通过 `--session-file` 参数指定会话文件，将对话历史保存到文件，并在后续请求中加载历史消息。

#### Scenario: 多轮对话

- **WHEN** 用户第一次执行 `cllm gemini --session-file /tmp/chat.json "Hello"`
- **AND** 用户第二次执行 `cllm gemini --session-file /tmp/chat.json "How are you?"`
- **THEN** 第二次请求的 `contents` 数组包含历史对话中的所有 user/model 消息

### Requirement: Gemini 专属参数支持

系统 SHALL 支持 Gemini API 特有参数，通过 CLI flags 指定。

#### Scenario: 使用 topK 参数

- **WHEN** 用户执行 `cllm gemini --top-k 40 "hello"`
- **THEN** 请求体 `generationConfig.topK` 值为 40

#### Scenario: 使用思考配置

- **WHEN** 用户执行 `cllm gemini --thinking --thinking-budget 1024 "复杂问题"`
- **THEN** 请求体 `generationConfig.thinkingConfig` 包含 `{"includeThoughts": true, "thinkingBudget": 1024}`
- **AND** 若同时指定 `--show-reasoning`，回复中的思考内容使用暗色样式（`\x1b[2m`）输出

#### Scenario: 指定响应格式

- **WHEN** 用户执行 `cllm gemini --response-mime-type "application/json" "返回JSON"`
- **THEN** 请求体 `generationConfig.responseMimeType` 值为 `"application/json"`

### Requirement: 请求参数映射

系统 SHALL 正确映射以下公共参数到 Gemini API 格式：`--temperature`、`--top-p`、`--seed`、`--frequency-penalty`、`--presence-penalty`。

#### Scenario: 温度参数映射

- **WHEN** 用户执行 `cllm gemini --temperature 0.7 "hello"`
- **THEN** 请求体 `generationConfig.temperature` 值为 0.7

### Requirement: 干运行模式

系统 SHALL 支持 `--dry-run` 参数，仅打印请求体 JSON 而不实际发送请求。

#### Scenario: 干运行输出

- **WHEN** 用户执行 `cllm gemini --dry-run --model gemini-2.5-flash "hello"`
- **THEN** 系统将 JSON 格式的请求体打印到 stdout
- **AND** 不发送任何 HTTP 请求

### Requirement: 输出格式控制

系统 SHALL 支持通过全局 `--output-format` / `-o` 参数控制输出格式：`human-readable`（默认，可读文本）、`json`（格式化 JSON）、`raw`（原始响应）。

#### Scenario: JSON 格式输出

- **WHEN** 用户执行 `cllm gemini -o json "hello"`
- **THEN** 系统将完整的 API 响应以格式化 JSON 输出到 stdout
