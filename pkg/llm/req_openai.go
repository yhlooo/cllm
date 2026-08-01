package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
)

const (
	oaiChatCompletionURI = "/chat/completions"
)

// NewOpenAIChatCompletionRequest 创建 OpenAI 对话补全请求构造器
func NewOpenAIChatCompletionRequest() OpenAIChatCompletionRequest {
	return OpenAIChatCompletionRequest{}
}

// OpenAIChatCompletionRequest OpenAI 兼容请求构造器
type OpenAIChatCompletionRequest struct {
	ctx         context.Context
	url         string
	apiKey      string
	extraHeader http.Header

	messages []openai.ChatCompletionMessageParamUnion
	model    string
	stream   bool
}

var _ RequestBuilder = OpenAIChatCompletionRequest{}

// WithContext 带上上下文
func (b OpenAIChatCompletionRequest) WithContext(ctx context.Context) RequestBuilder {
	b.ctx = ctx
	return b
}

// WithURL 带上 URL
func (b OpenAIChatCompletionRequest) WithURL(url string) RequestBuilder {
	if url == "" {
		return b
	}
	b.url = url
	return b
}

// WithBaseURL 带上 URL 前缀
func (b OpenAIChatCompletionRequest) WithBaseURL(baseURL string) RequestBuilder {
	if baseURL == "" {
		return b
	}
	b.url = strings.TrimSuffix(baseURL, "/") + oaiChatCompletionURI
	return b
}

// WithAPIKey 带上认证 API Key
func (b OpenAIChatCompletionRequest) WithAPIKey(apiKey string) RequestBuilder {
	b.apiKey = apiKey
	return b
}

// WithHeader 带上请求头
func (b OpenAIChatCompletionRequest) WithHeader(key, value string) RequestBuilder {
	b.extraHeader.Add(key, value)
	return b
}

// WithModel 带上模型名
func (b OpenAIChatCompletionRequest) WithModel(model string) RequestBuilder {
	b.model = model
	return b
}

// WithStream 带上指定流式模式开关
func (b OpenAIChatCompletionRequest) WithStream(enabled bool) RequestBuilder {
	b.stream = enabled
	return b
}

// WithSystemPrompt 带上系统提示词
func (b OpenAIChatCompletionRequest) WithSystemPrompt(content string) RequestBuilder {
	if content == "" {
		return b
	}
	b.messages = append(b.messages, openai.SystemMessage(content))
	return b
}

// WithUserPrompt 带上用户提示词
func (b OpenAIChatCompletionRequest) WithUserPrompt(content string) RequestBuilder {
	if content == "" {
		return b
	}
	b.messages = append(b.messages, openai.UserMessage(content))
	return b
}

// Build 构建请求
func (b OpenAIChatCompletionRequest) Build() (*http.Request, error) {
	params := &openai.ChatCompletionNewParams{
		Model:    b.model,
		Messages: b.messages,
	}
	params.SetExtraFields(map[string]any{
		"stream": b.stream,
	})

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
