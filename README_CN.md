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

