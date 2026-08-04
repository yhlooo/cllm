**[简体中文](README_CN.md)** | [English](README.md)

---

![GitHub License](https://img.shields.io/github/license/yhlooo/cllm)
[![GitHub Release](https://img.shields.io/github/v/release/yhlooo/cllm)](https://github.com/yhlooo/cllm/releases/latest)
[![release](https://github.com/yhlooo/cllm/actions/workflows/release.yaml/badge.svg)](https://github.com/yhlooo/cllm/actions/workflows/release.yaml)

# cllm - CLI Client for LLM

> **🏗️ This project is still in an early stage.**

## Installation

- **Install via script:**

  ```shell
  curl -L https://raw.githubusercontent.com/yhlooo/cllm/refs/heads/main/scripts/install.sh | bash
  ```

  The script installs `cllm` to `~/.local/bin`. If this directory is not in your `PATH`, follow the script prompts to add it.

- **Manual installation:**

  Download the executable binary from the [Releases](https://github.com/yhlooo/cllm/releases) page, extract it, and place the `cllm` file in any `$PATH` directory.

- **Docker**

  Use the Docker image [`ghcr.io/yhlooo/cllm`](https://github.com/yhlooo/cllm/pkgs/container/cllm) directly:

  ```shell
  docker run ghcr.io/yhlooo/cllm:latest --help
  ```

## Usage

### Request OpenAI-compatible API

```shell
cllm openai \
  -b "<base-url>" \
  -k "<api-key>" \
  -m "<model-name>" \
  "<prompt>"
```

For example, with [DeepSeek](https://api-docs.deepseek.com/):

```shell
cllm openai \
  -b "https://api.deepseek.com/ " \
  -k "sk-..." \
  -m "deepseek-v4-pro" \
  "hello"
```

Use `@path/to/file` in user messages or the `-a/--attachment` flag to reference images, audio, and other files:

```shell
# @path/to/file
cllm openai \
  ... \
  "What's in the @example.png ?"

# or -a path/to/file
cllm openai \
  ... \
  -a example.png \
  "What's in that image?"
```

Use `--session-file` to persist session history to a file and resume conversations from it:

```shell
cllm openai \
  ... \
  --session-file history.jsonl \
  "Cllm is a CLI client for LLM."

# This question can be answered based on the previous conversation history
cllm openai \
  ... \
  --session-file history.jsonl \
  "What is cllm?"
```

### Request Ollama Chat API

```shell
cllm ollama \
  -b "<base-url>" \
  -m "<model-name>" \
  "<prompt>"
```

By default, connects to `http://localhost:11434`. For example:

```shell
cllm ollama -m llama3.1 "hello"
```

Stream output with `-s/--stream`:

```shell
cllm ollama -m llama3.1 -s "Write a poem"
```

Attach images with `-a/--attachment` or `@path` inline syntax (supports multimodal models like `llava`):

```shell
cllm ollama -m llava -a photo.jpg "Describe this image"
```

Persist session history with `--session-file`:

```shell
cllm ollama -m llama3.1 --session-file chat.jsonl "Hello"
cllm ollama -m llama3.1 --session-file chat.jsonl "Continue"
```

JSON structured output with `--format`:

```shell
cllm ollama -m llama3.1 --format json "List three colors"
```

Enable thinking mode with `--think` (supports `true`/`false` or `high`/`medium`/`low`/`max`):

```shell
cllm ollama -m gpt-oss --think low --show-reasoning "What is 1+1?"
```

Set model options like `--temperature`, `--seed`, `--num-predict`:

```shell
cllm ollama -m llama3.1 --temperature 0.7 --seed 42 --num-predict 100 "Hello"
```

### Request Gemini API

```shell
cllm gemini \
  -b "<base-url>" \
  -k "<api-key>" \
  -m "<model-name>" \
  "<prompt>"
```

By default, connects to `https://generativelanguage.googleapis.com/v1beta`. For example:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash "hello"
```

Stream output with `-s/--stream`:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash -s "Write a poem"
```

Attach images, audio, and other files with `-a/--attachment` or `@path` inline syntax:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash -a photo.jpg "Describe this image"
```

Persist session history with `--session-file`:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash --session-file chat.jsonl "Hello"
cllm gemini -k "AIza..." -m gemini-2.5-flash --session-file chat.jsonl "Continue"
```

JSON structured output with `--response-mime-type`:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash \
  --response-mime-type application/json \
  "List three colors"
```

Enable model thinking with `--thinking` and `--thinking-budget`:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash \
  --thinking --thinking-budget 1024 --show-reasoning \
  "Solve this math problem: ..."
```

Set generation parameters:

```shell
cllm gemini -k "AIza..." -m gemini-2.5-flash \
  --temperature 0.7 --top-p 0.9 --top-k 40 \
  --max-output-tokens 1024 --seed 42 \
  "Hello"
```
