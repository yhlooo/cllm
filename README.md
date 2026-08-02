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
