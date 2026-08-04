package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/yhlooo/cllm/pkg/ollama"
)

const (
	ollamaChatURI = "/api/chat"
)

// OllamaChatRequestBuilder Ollama Chat 请求构造器扩展接口
type OllamaChatRequestBuilder interface {
	RequestBuilder

	// WithFormat 设置响应格式（"json" 或 JSON Schema 对象）
	WithFormat(format any) OllamaChatRequestBuilder
	// WithOptions 设置模型推理参数
	WithOptions(opts ollama.Options) OllamaChatRequestBuilder
	// WithKeepAlive 设置模型在内存中保持的时间（如 "5m"、0 立即卸载）
	WithKeepAlive(duration any) OllamaChatRequestBuilder
	// WithThink 设置思考模式（true/false 或 "high"/"medium"/"low"/"max"）
	WithThink(think any) OllamaChatRequestBuilder
	// WithLogprobs 设置是否返回 token 对数概率
	WithLogprobs(enabled bool) OllamaChatRequestBuilder
	// WithTopLogprobs 设置每个 token 位置返回的最可能 token 数量
	WithTopLogprobs(v int) OllamaChatRequestBuilder
}

// NewOllamaChatRequest 创建 Ollama Chat 请求构造器
func NewOllamaChatRequest() OllamaChatRequest {
	return OllamaChatRequest{}
}

// OllamaChatRequest Ollama Chat 请求构造器
type OllamaChatRequest struct {
	ctx         context.Context
	url         string
	apiKey      string
	extraHeader http.Header

	params ollama.ChatRequest
	stream bool

	errors []error
}

var _ OllamaChatRequestBuilder = OllamaChatRequest{}

// WithContext 带上上下文
func (b OllamaChatRequest) WithContext(ctx context.Context) RequestBuilder {
	b.ctx = ctx
	return b
}

// WithURL 带上 URL
func (b OllamaChatRequest) WithURL(url string) RequestBuilder {
	if url == "" {
		return b
	}
	b.url = url
	return b
}

// WithBaseURL 带上 URL 前缀，自动拼接 /api/chat
func (b OllamaChatRequest) WithBaseURL(baseURL string) RequestBuilder {
	if baseURL == "" {
		return b
	}
	b.url = strings.TrimSuffix(baseURL, "/") + ollamaChatURI
	return b
}

// WithAPIKey 带上认证 API Key
func (b OllamaChatRequest) WithAPIKey(apiKey string) RequestBuilder {
	b.apiKey = apiKey
	return b
}

// WithHeader 带上请求头
func (b OllamaChatRequest) WithHeader(key, value string) RequestBuilder {
	b.extraHeader.Add(key, value)
	return b
}

// WithModel 带上模型名
func (b OllamaChatRequest) WithModel(model string) RequestBuilder {
	b.params.Model = model
	return b
}

// WithStream 带上指定流式模式开关
func (b OllamaChatRequest) WithStream(enabled bool) RequestBuilder {
	b.stream = enabled
	return b
}

// WithMessages 带上消息
func (b OllamaChatRequest) WithMessages(messages ...Message) RequestBuilder {
	for _, msg := range messages {
		ollamaMsg, err := newOllamaMessage(msg)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		b.params.Messages = append(b.params.Messages, ollamaMsg)
	}
	return b
}

// BuildBody 构建请求体内容
func (b OllamaChatRequest) BuildBody() (any, error) {
	params := b.params
	params.Stream = b.stream
	return params, nil
}

// Build 构建请求
func (b OllamaChatRequest) Build() (*http.Request, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}

	params, err := b.BuildBody()
	if err != nil {
		return nil, err
	}

	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode params to json error: %w", err)
	}

	if b.ctx == nil {
		b.ctx = context.Background()
	}
	body := bytes.NewReader(paramsRaw)

	req, err := http.NewRequestWithContext(b.ctx, http.MethodPost, b.url, body)
	if err != nil {
		return req, err
	}

	req.Header.Set("Content-Type", "application/json")

	if b.apiKey != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.apiKey))
	}
	for k, v := range b.extraHeader {
		req.Header[k] = v
	}

	return req, nil
}

// WithFormat 设置响应格式
func (b OllamaChatRequest) WithFormat(format any) OllamaChatRequestBuilder {
	b.params.Format = format
	return b
}

// WithOptions 设置模型推理参数
func (b OllamaChatRequest) WithOptions(opts ollama.Options) OllamaChatRequestBuilder {
	b.params.Options = &opts
	return b
}

// WithKeepAlive 设置模型保持时间
func (b OllamaChatRequest) WithKeepAlive(duration any) OllamaChatRequestBuilder {
	b.params.KeepAlive = duration
	return b
}

// WithThink 设置思考模式
func (b OllamaChatRequest) WithThink(think any) OllamaChatRequestBuilder {
	b.params.Think = think
	return b
}

// WithLogprobs 设置是否返回 token 对数概率
func (b OllamaChatRequest) WithLogprobs(enabled bool) OllamaChatRequestBuilder {
	b.params.Logprobs = enabled
	return b
}

// WithTopLogprobs 设置每个 token 位置返回的最可能 token 数量
func (b OllamaChatRequest) WithTopLogprobs(v int) OllamaChatRequestBuilder {
	b.params.TopLogprobs = v
	return b
}

// newOllamaMessage 将 llm.Message 转换为 ollama.ChatMessage
func newOllamaMessage(msg Message) (ollama.ChatMessage, error) {
	switch msg.Role {
	case RoleSystem:
		return ollama.ChatMessage{
			Role:    "system",
			Content: joinTextContent(msg.Content),
		}, nil

	case RoleUser:
		content := joinTextContent(msg.Content)
		// 提取图片
		var images []string
		for _, part := range msg.Content {
			if part.IsBlob() {
				blob := part.Blob
				if strings.HasPrefix(blob.MediaType, "image/") {
					images = append(images, base64.StdEncoding.EncodeToString(blob.Content))
				} else {
					return ollama.ChatMessage{},
						fmt.Errorf("unsupported blob media type: %q (only image/* is supported)", blob.MediaType)
				}
			}
		}
		return ollama.ChatMessage{
			Role:    "user",
			Content: content,
			Images:  images,
		}, nil

	case RoleAssistant:
		var texts []string
		for _, part := range msg.Content {
			if part.IsText() {
				texts = append(texts, part.Text.Content)
			} else if part.IsRefusal() {
				// Ollama 无 refusal 概念，转为普通文本
				texts = append(texts, part.Refusal.Content)
			}
		}
		return ollama.ChatMessage{
			Role:    "assistant",
			Content: strings.Join(texts, ""),
		}, nil

	default:
		return ollama.ChatMessage{}, fmt.Errorf("unsupported role %q", msg.Role)
	}
}

// joinTextContent 将消息内容中的文本部分拼接为单个字符串
func joinTextContent(parts []MessageContentPart) string {
	var texts []string
	for _, part := range parts {
		if part.IsText() {
			texts = append(texts, part.Text.Content)
		}
	}
	return strings.Join(texts, "")
}
