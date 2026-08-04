# Ollama Chat

## Purpose

通过 `cllm ollama` 子命令向 Ollama Chat API (`POST /api/chat`) 发送对话请求，支持流式/非流式、多模态输入、会话持久化、思考模式、日志概率等多种功能。

## Requirements

### Requirement: 用户可以向 Ollama 发送对话请求

系统 SHALL 提供 `cllm ollama` 子命令，允许用户通过 `POST /api/chat` 端点向 Ollama 服务发送对话请求并输出响应。

#### Scenario: 基本对话

- **WHEN** 用户执行 `cllm ollama --model llama3.1 "你好"`
- **THEN** 系统向 Ollama 服务的 `/api/chat` 端点发送 POST 请求
- **THEN** 请求体中包含 `{"model": "llama3.1", "messages": [{"role": "user", "content": "你好"}]}`
- **THEN** 系统以人类可读格式输出 assistant 的回复文本

#### Scenario: 指定服务地址

- **WHEN** 用户执行 `cllm ollama --url http://ollama.example.com:11434 --model llama3.1 "你好"`
- **THEN** 系统向 `http://ollama.example.com:11434/api/chat` 发送请求

#### Scenario: 使用 base URL

- **WHEN** 用户执行 `cllm ollama --base-url http://ollama.example.com:11434 --model llama3.1 "你好"`
- **THEN** 系统自动拼接 `/api/chat` 路径，向 `http://ollama.example.com:11434/api/chat` 发送请求

#### Scenario: 默认服务地址

- **WHEN** 用户执行 `cllm ollama --model llama3.1 "你好"` 且未指定 `--url` 或 `--base-url`
- **THEN** 系统默认向 `http://localhost:11434/api/chat` 发送请求

### Requirement: 用户可以向 Ollama 发送流式对话请求

系统 SHALL 支持通过 `--stream` 参数启用 Ollama 流式响应，逐 token 输出内容。

#### Scenario: 流式输出

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --stream "你好"`
- **THEN** 请求体中 `"stream": true`
- **THEN** 系统以 NDJSON 格式接收流式响应，逐行解析 `ollama.ChatResponse`
- **THEN** 系统逐 token 输出 assistant content，直到收到 `"done": true`

#### Scenario: 非流式输出

- **WHEN** 用户执行 `cllm ollama --model llama3.1 "你好"` 且未指定 `--stream`
- **THEN** 系统发送非流式请求，读取完整响应后一次性输出

### Requirement: 用户可以通过系统提示词设定模型行为

系统 SHALL 支持通过 `--system-prompt` 参数设置系统消息。

#### Scenario: 设置系统提示词

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --system-prompt "你是一个翻译助手" "hello"`
- **THEN** 请求体的 messages 数组中以 `{"role": "system", "content": "你是一个翻译助手"}` 作为首条消息

### Requirement: 用户可以在消息中附加图片文件

系统 SHALL 支持通过 `--attachment` 参数或 `@path` 内联语法附加图片文件到用户消息中。

#### Scenario: 通过参数附加图片

- **WHEN** 用户执行 `cllm ollama --model llava --attachment ./photo.jpg "描述这张图"`
- **THEN** 请求体中用户消息的 `content` 字段为 `"描述这张图"`，`images` 字段包含以 base64 编码的图片数据

#### Scenario: 通过内联语法附加图片

- **WHEN** 用户执行 `cllm ollama --model llava "描述这张图 @./photo.jpg"`
- **THEN** 系统自动将 `@./photo.jpg` 解析为图片附件，效果与 `--attachment` 一致

### Requirement: 用户可以指定 JSON 格式的输出

系统 SHALL 支持通过 `--format` 参数设置 Ollama 的 JSON 结构化输出模式。

#### Scenario: JSON 输出模式

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --format json "列出三个颜色"`
- **THEN** 请求体中包含 `"format": "json"`
- **THEN** Ollama 返回结构化的 JSON 响应

### Requirement: 用户可以持久化对话历史

系统 SHALL 支持通过 `--session-file` 参数保存和恢复对话历史。

#### Scenario: 保存对话历史

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --session-file ./chat.jsonl "你好"`
- **THEN** 系统将用户消息和 assistant 回复追加写入 `./chat.jsonl` 文件

#### Scenario: 恢复对话历史

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --session-file ./chat.jsonl "继续"`
- **THEN** 系统从 `./chat.jsonl` 加载历史消息，与当前消息合并一起发送

### Requirement: 用户可以选择不同的输出格式

系统 SHALL 支持通过 `--output-format` 全局参数选择输出格式（human-readable / json / raw）。

#### Scenario: JSON 格式输出

- **WHEN** 用户执行 `cllm ollama --model llama3.1 -o json "你好"`
- **THEN** 系统以格式化 JSON 输出完整的 API 响应

#### Scenario: Raw 格式输出

- **WHEN** 用户执行 `cllm ollama --model llama3.1 -o raw "你好"`
- **THEN** 系统原样输出 API 响应的原始字节流

### Requirement: 用户可以设置 Ollama 特有的模型参数

系统 SHALL 支持通过命令行参数设置 Ollama 的模型推理选项（Options），这些参数将被序列化到请求体的 `options` 对象中。

#### Scenario: 设置温度和随机种子

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --temperature 0.7 --seed 42 "你好"`
- **THEN** 请求体中包含 `"options": {"temperature": 0.7, "seed": 42}`

#### Scenario: 设置停止序列

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --stop "\n\n" "你好"`
- **THEN** 请求体中包含 `"options": {"stop": ["\n\n"]}`

#### Scenario: 设置模型保持时间

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --keep-alive "5m" "你好"`
- **THEN** 请求体中包含 `"keep_alive": "5m"`

### Requirement: 用户可以使用思考模式

系统 SHALL 支持通过 `--think` 参数启用 Ollama 思考模式，在输出响应前返回思考过程。

#### Scenario: 启用思考模式

- **WHEN** 用户执行 `cllm ollama --model gpt-oss --think low "1+1 等于几？"`
- **THEN** 请求体中包含 `"think": "low"`
- **THEN** 系统在输出正文前以暗色文本显示思考过程

#### Scenario: 禁用思考模式

- **WHEN** 用户执行 `cllm ollama --model gpt-oss --think false "1+1 等于几？"`
- **THEN** 请求体中包含 `"think": false`

### Requirement: 用户可以获取 token 对数概率

系统 SHALL 支持通过 `--logprobs` 和 `--top-logprobs` 参数获取输出 token 的对数概率信息。

#### Scenario: 启用对数概率

- **WHEN** 用户执行 `cllm ollama --model llama3.1 --logprobs --top-logprobs 5 "你好"`
- **THEN** 请求体中包含 `"logprobs": true, "top_logprobs": 5`
- **THEN** 响应中包含每个生成 token 的 logprob 信息

### Requirement: 用户可以显示思考过程

系统 SHALL 支持通过 `--show-reasoning` 参数控制是否在人类可读输出中显示思考过程。

#### Scenario: 显示思考过程

- **WHEN** 用户执行 `cllm ollama --model gpt-oss --think low --show-reasoning "你好"`
- **THEN** 流式输出时思考 token 以 ANSI 暗色连续显示，正文开始后转回正常颜色

#### Scenario: 隐藏思考过程

- **WHEN** 用户执行 `cllm ollama --model gpt-oss --think low --show-reasoning=false "你好"`
- **THEN** 思考内容不在输出中显示，仅显示正文
