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

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

const (
	anthropicMessagesURI    = "/v1/messages"
	defaultAnthropicVersion = "2023-06-01"

)

// AnthropicMessageRequestBuilder Anthropic Messages API 请求构造器接口
type AnthropicMessageRequestBuilder interface {
	RequestBuilder

	WithMaxTokens(v int64) AnthropicMessageRequestBuilder
	WithTemperature(v float64) AnthropicMessageRequestBuilder
	WithTopP(v float64) AnthropicMessageRequestBuilder
	WithTopK(v int64) AnthropicMessageRequestBuilder
	WithStopSequences(v ...string) AnthropicMessageRequestBuilder
	WithThinking(enabled bool, budgetTokens int64) AnthropicMessageRequestBuilder
}

// NewAnthropicMessageRequest 创建 Anthropic Messages API 请求构造器
func NewAnthropicMessageRequest() AnthropicMessageRequest {
	return AnthropicMessageRequest{}
}

// AnthropicMessageRequest Anthropic Messages API 请求构造器
type AnthropicMessageRequest struct {
	ctx         context.Context
	url         string
	apiKey      string
	extraHeader http.Header

	systemBlocks []anthropic.TextBlockParam
	messages     []anthropic.MessageParam
	model        anthropic.Model
	maxTokens    int64
	temperature  param.Opt[float64]
	topP         param.Opt[float64]
	topK         param.Opt[int64]
	stopSeqs     []string
	thinking     anthropic.ThinkingConfigParamUnion

	stream bool

	errors []error
}

var _ AnthropicMessageRequestBuilder = AnthropicMessageRequest{}

// WithContext 带上上下文
func (b AnthropicMessageRequest) WithContext(ctx context.Context) RequestBuilder {
	b.ctx = ctx
	return b
}

// WithURL 带上 URL
func (b AnthropicMessageRequest) WithURL(url string) RequestBuilder {
	if url == "" {
		return b
	}
	b.url = url
	return b
}

// WithBaseURL 带上 URL 前缀，自动拼接 /v1/messages
func (b AnthropicMessageRequest) WithBaseURL(baseURL string) RequestBuilder {
	if baseURL == "" {
		return b
	}
	b.url = strings.TrimSuffix(baseURL, "/") + anthropicMessagesURI
	return b
}

// WithAPIKey 带上认证 API Key
func (b AnthropicMessageRequest) WithAPIKey(apiKey string) RequestBuilder {
	b.apiKey = apiKey
	return b
}

// WithHeader 带上请求头
func (b AnthropicMessageRequest) WithHeader(key, value string) RequestBuilder {
	b.extraHeader.Add(key, value)
	return b
}

// WithModel 带上模型名
func (b AnthropicMessageRequest) WithModel(model string) RequestBuilder {
	b.model = anthropic.Model(model)
	return b
}

// WithStream 带上指定流式模式开关
func (b AnthropicMessageRequest) WithStream(enabled bool) RequestBuilder {
	b.stream = enabled
	return b
}

// WithMessages 带上消息
func (b AnthropicMessageRequest) WithMessages(messages ...Message) RequestBuilder {
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			// 系统消息提取到顶级 system 字段
			for _, part := range msg.Content {
				if part.IsText() {
					b.systemBlocks = append(b.systemBlocks, anthropic.TextBlockParam{
						Text: part.Text.Content,
					})
				} else if part.IsRefusal() {
					b.systemBlocks = append(b.systemBlocks, anthropic.TextBlockParam{
						Text: part.Refusal.Content,
					})
				}
				// 忽略非文本系统消息（Anthropic 不支持 system blob）
			}
		case RoleUser:
			m, err := newAnthropicUserMessage(msg)
			if err != nil {
				b.errors = append(b.errors, err)
				continue
			}
			b.messages = append(b.messages, m)
		case RoleAssistant:
			m, err := newAnthropicAssistantMessage(msg)
			if err != nil {
				b.errors = append(b.errors, err)
				continue
			}
			b.messages = append(b.messages, m)
		case RoleTool:
			// Anthropic 中 tool result 的 role 是 user
			m, err := newAnthropicToolResultMessage(msg)
			if err != nil {
				b.errors = append(b.errors, err)
				continue
			}
			b.messages = append(b.messages, m)
		default:
			b.errors = append(b.errors, fmt.Errorf("unsupported role %q", msg.Role))
		}
	}
	return b
}

// newAnthropicUserMessage 将内部用户消息转为 Anthropic MessageParam
func newAnthropicUserMessage(msg Message) (anthropic.MessageParam, error) {
	if len(msg.Content) == 0 {
		return anthropic.MessageParam{}, nil
	}
	if len(msg.Content) == 1 && msg.Content[0].IsText() {
		return anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content[0].Text.Content)), nil
	}

	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))
	for _, part := range msg.Content {
		switch {
		case part.IsText():
			blocks = append(blocks, anthropic.NewTextBlock(part.Text.Content))
		case part.IsBlob():
			blob := part.Blob
			if strings.HasPrefix(blob.MediaType, "image/") {
				blocks = append(blocks, anthropic.NewImageBlockBase64(
					blob.MediaType,
					base64.StdEncoding.EncodeToString(blob.Content),
				))
			} else {
				// 不支持的非图片附件
				return anthropic.MessageParam{},
					fmt.Errorf("unsupported blob media type: %q (only image/* is supported)", blob.MediaType)
			}
		default:
			return anthropic.MessageParam{},
				fmt.Errorf("unsupported user message content part")
		}
	}

	return anthropic.NewUserMessage(blocks...), nil
}

// newAnthropicAssistantMessage 将内部助手消息转为 Anthropic MessageParam
func newAnthropicAssistantMessage(msg Message) (anthropic.MessageParam, error) {
	if len(msg.Content) == 0 {
		return anthropic.MessageParam{}, nil
	}

	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))
	for _, part := range msg.Content {
		switch {
		case part.IsText():
			blocks = append(blocks, anthropic.NewTextBlock(part.Text.Content))
		case part.IsRefusal():
			// Anthropic 无 refusal 概念，转为普通文本
			blocks = append(blocks, anthropic.NewTextBlock(part.Refusal.Content))
		case part.IsReasoning():
			// 推理内容暂时跳过，因为内部 MessageContentPart 不存储 thinking signature
			// Anthropic API 允许历史消息中不包含 thinking block
		default:
			return anthropic.MessageParam{},
				fmt.Errorf("unsupported assistant message content part")
		}
	}

	return anthropic.NewAssistantMessage(blocks...), nil
}

// newAnthropicToolResultMessage 将内部工具消息转为 Anthropic MessageParam
// Anthropic 中 tool_result 的 role 是 user
func newAnthropicToolResultMessage(msg Message) (anthropic.MessageParam, error) {
	return anthropic.MessageParam{}, fmt.Errorf("tool role is not supported for Anthropic")
}

// BuildBody 构建请求体内容
func (b AnthropicMessageRequest) BuildBody() (any, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}

	params := anthropic.MessageNewParams{
		Model:     b.model,
		Messages:  b.messages,
		MaxTokens: b.maxTokens,
	}

	// 系统消息
	if len(b.systemBlocks) > 0 {
		params.System = b.systemBlocks
	}

	// 采样参数（omitzero 自动处理零值省略）
	params.Temperature = b.temperature
	params.TopP = b.topP
	params.TopK = b.topK

	// 停止序列
	if len(b.stopSeqs) > 0 {
		params.StopSequences = b.stopSeqs
	}

	// 思考配置（omitzero 自动处理零值省略）
	params.Thinking = b.thinking

	// MessageNewParams 有自定义 MarshalJSON，无法通过 struct 嵌入注入额外字段，
	// 因此先 marshal 再注入 "stream": true
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode params to json error: %w", err)
	}

	if b.stream {
		paramsRaw = bytes.TrimRight(paramsRaw, " \t\r\n}")
		paramsRaw = append(paramsRaw, []byte(`, "stream": true}`)...)
	}

	return json.RawMessage(paramsRaw), nil
}

// Build 构建请求
func (b AnthropicMessageRequest) Build() (*http.Request, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}

	bodyData, err := b.BuildBody()
	if err != nil {
		return nil, err
	}

	// BuildBody 返回 json.RawMessage，直接 marshal 就是原始字节
	paramsRaw, err := json.Marshal(bodyData)
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

	// Anthropic 使用 x-api-key header 而非 Authorization: Bearer
	if b.apiKey != "" {
		req.Header.Set("x-api-key", b.apiKey)
	}

	// anthropic-version header 默认值，允许通过 --header 覆盖
	req.Header.Set("anthropic-version", defaultAnthropicVersion)

	for k, v := range b.extraHeader {
		req.Header[k] = v
	}

	return req, nil
}

// WithMaxTokens 带上最大输出 token 数
func (b AnthropicMessageRequest) WithMaxTokens(v int64) AnthropicMessageRequestBuilder {
	b.maxTokens = v
	return b
}

// WithTemperature 带上温度参数
func (b AnthropicMessageRequest) WithTemperature(v float64) AnthropicMessageRequestBuilder {
	b.temperature = param.NewOpt(v)
	return b
}

// WithTopP 带上 TopP 参数
func (b AnthropicMessageRequest) WithTopP(v float64) AnthropicMessageRequestBuilder {
	b.topP = param.NewOpt(v)
	return b
}

// WithTopK 带上 TopK 参数
func (b AnthropicMessageRequest) WithTopK(v int64) AnthropicMessageRequestBuilder {
	b.topK = param.NewOpt(v)
	return b
}

// WithStopSequences 带上停止序列
func (b AnthropicMessageRequest) WithStopSequences(v ...string) AnthropicMessageRequestBuilder {
	b.stopSeqs = v
	return b
}

// WithThinking 带上思考配置
func (b AnthropicMessageRequest) WithThinking(enabled bool, budgetTokens int64) AnthropicMessageRequestBuilder {
	if !enabled {
		b.thinking = anthropic.ThinkingConfigParamUnion{
			OfDisabled: &anthropic.ThinkingConfigDisabledParam{},
		}
		return b
	}

	enabledCfg := &anthropic.ThinkingConfigEnabledParam{
		BudgetTokens: budgetTokens,
	}
	b.thinking = anthropic.ThinkingConfigParamUnion{
		OfEnabled: enabledCfg,
	}
	return b
}
