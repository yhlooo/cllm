**[简体中文](README_CN.md)** | [English](README.md)

---

![GitHub License](https://img.shields.io/github/license/yhlooo/cllm)
[![GitHub Release](https://img.shields.io/github/v/release/yhlooo/cllm)](https://github.com/yhlooo/cllm/releases/latest)
[![release](https://github.com/yhlooo/cllm/actions/workflows/release.yaml/badge.svg)](https://github.com/yhlooo/cllm/actions/workflows/release.yaml)

# cllm - LLM 的 CLI 客户端

> **🏗️ 该项目还处于较早期阶段。**

## 安装

- **脚本安装**

  ```shell
  curl -L https://raw.githubusercontent.com/yhlooo/cllm/refs/heads/main/scripts/install.sh | bash
  ```

  脚本将 `cllm` 安装到 `~/.local/bin` 目录。若该目录不在 `PATH` 变量中，需按照脚本提示添加。

- **手动安装**

  通过 [Releases](https://github.com/yhlooo/cllm/releases) 页面下载可执行二进制，解压并将其中 `cllm` 文件放置到任意 `$PATH` 目录下。

- **Docker**

  直接使用 Docker 镜像 [`ghcr.io/yhlooo/cllm`](https://github.com/yhlooo/cllm/pkgs/container/cllm)

  ```shell
  docker run ghcr.io/yhlooo/cllm:latest --help
  ```

## 使用

### 请求 OpenAI 兼容接口

```shell
cllm openai \
  -b "<base-url>" \
  -k "<api-key>" \
  -m "<model-name>" \
  "<prompt>"
```

比如 [DeepSeek](https://api-docs.deepseek.com/zh-cn/) ：

```shell
cllm openai \
  -b "https://api.deepseek.com/ " \
  -k "sk-..." \
  -m "deepseek-v4-pro" \
  "hello"
```

在用户消息中使用 `@path/to/file` 或使用 `-a/--attachment` 参数可以引用图片、音频等文件：

```shell
# @path/to/file
cllm openai \
  ... \
  "What's in the @example.png ?"

# or -a path/to/file
cllm openai \
  ... \
  -a example.png
  "What's in that image?"
```

指定参数 `--session-file` 可将会话历史持久化到指定文件，并从该文件恢复历史对话记录：

```shell
cllm openai \
  ... \
  --session-file history.jsonl \
  "Cllm is a CLI client for LLM."

# 该问题能基于上一轮对话历史回答
cllm openai \
  ... \
  --session-file history.jsonl \
  "What is cllm?"
```

### 请求 Ollama Chat API

```shell
cllm ollama \
  -b "<base-url>" \
  -m "<model-name>" \
  "<prompt>"
```

默认连接 `http://localhost:11434`，例如：

```shell
cllm ollama -m llama3.1 "你好"
```

通过 `-s/--stream` 启用流式输出：

```shell
cllm ollama -m llama3.1 -s "写一首诗"
```

使用 `-a/--attachment` 或 `@path` 内联语法附加图片（需多模态模型如 `llava`）：

```shell
cllm ollama -m llava -a photo.jpg "描述这张图片"
```

通过 `--session-file` 持久化会话历史：

```shell
cllm ollama -m llama3.1 --session-file chat.jsonl "你好"
cllm ollama -m llama3.1 --session-file chat.jsonl "继续"
```

通过 `--format` 启用 JSON 结构化输出：

```shell
cllm ollama -m llama3.1 --format json "列出三个颜色"
```

通过 `--think` 启用思考模式（支持 `true`/`false` 或 `high`/`medium`/`low`/`max`）：

```shell
cllm ollama -m gpt-oss --think low --show-reasoning "1+1 等于几？"
```

设置模型参数如 `--temperature`、`--seed`、`--num-predict`：

```shell
cllm ollama -m llama3.1 --temperature 0.7 --seed 42 --num-predict 100 "你好"
```

### 请求 Gemini API

```shell
cllm gemini \
  -b "<base-url>" \
  -k "<api-key>" \
  -m "<model-name>" \
  "<prompt>"
```

默认连接 `https://generativelanguage.googleapis.com/v1beta`，例如：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash "你好"
```

通过 `-s/--stream` 启用流式输出：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash -s "写一首诗"
```

使用 `-a/--attachment` 或 `@path` 内联语法附加图片、音频等文件：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash -a photo.jpg "描述这张图片"
```

通过 `--session-file` 持久化会话历史：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash --session-file chat.jsonl "你好"
cllm gemini -k "AIza..." -m gemini-2.5-flash --session-file chat.jsonl "继续"
```

通过 `--response-mime-type` 启用 JSON 结构化输出：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash \
  --response-mime-type application/json \
  "列出三个颜色"
```

通过 `--thinking` 和 `--thinking-budget` 启用模型思考：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash \
  --thinking --thinking-budget 1024 --show-reasoning \
  "求解这道数学题：..."
```

设置生成参数：

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash \
  --temperature 0.7 --top-p 0.9 --top-k 40 \
  --max-output-tokens 1024 --seed 42 \
  "你好"
```

