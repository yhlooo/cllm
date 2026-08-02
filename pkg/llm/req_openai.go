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

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
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

	model  string
	stream bool

	oaiMessages []openai.ChatCompletionMessageParamUnion

	errors []error
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

// WithMessages 带上消息
func (b OpenAIChatCompletionRequest) WithMessages(messages ...Message) RequestBuilder {
	for _, msg := range messages {
		oaiMsg, err := newOpenAIMessage(msg)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		b.oaiMessages = append(b.oaiMessages, oaiMsg)
	}
	return b
}

// BuildBody 构建请求体内容
func (b OpenAIChatCompletionRequest) BuildBody() (any, error) {
	params := &openai.ChatCompletionNewParams{
		Model:    b.model,
		Messages: b.oaiMessages,
	}
	params.SetExtraFields(map[string]any{
		"stream": b.stream,
	})

	return params, nil
}

// Build 构建请求
func (b OpenAIChatCompletionRequest) Build() (*http.Request, error) {
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

func newOpenAIMessage(msg Message) (openai.ChatCompletionMessageParamUnion, error) {
	switch msg.Role {
	case RoleSystem:
		switch len(msg.Content) {
		case 0:
			return openai.SystemMessage(""), nil
		case 1:
			if !msg.Content[0].IsText() {
				return openai.ChatCompletionMessageParamUnion{},
					fmt.Errorf("system message only support text content")
			}
			return openai.SystemMessage(msg.Content[0].Text.Content), nil
		}

		parts := make([]openai.ChatCompletionContentPartTextParam, 0, len(msg.Content))
		for _, part := range msg.Content {
			if !part.IsText() {
				return openai.ChatCompletionMessageParamUnion{},
					fmt.Errorf("system message only support text content")
			}
			parts = append(parts, openai.ChatCompletionContentPartTextParam{
				Text: part.Text.Content,
			})
		}
		return openai.SystemMessage(parts), nil

	case RoleUser:
		if len(msg.Content) == 0 {
			return openai.UserMessage(""), nil
		}
		if len(msg.Content) == 1 && msg.Content[0].IsText() {
			return openai.UserMessage(msg.Content[0].Text.Content), nil
		}

		parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(msg.Content))
		for _, part := range msg.Content {
			if part.IsText() {
				parts = append(parts, openai.TextContentPart(part.Text.Content))
				continue
			}
			if !part.IsBlob() {
				return openai.ChatCompletionMessageParamUnion{},
					fmt.Errorf("unsupported user message content")
			}

			blob := part.Blob

			if strings.HasPrefix(blob.MediaType, "image/") {
				// 图片
				parts = append(parts, openai.ImageContentPart(
					openai.ChatCompletionContentPartImageImageURLParam{
						URL: "data:" + blob.MediaType + ";base64," + base64.StdEncoding.EncodeToString(blob.Content),
					},
				))
			} else if strings.HasPrefix(blob.MediaType, "audio/") {
				// 音频
				format := openai.ChatCompletionAudioParamFormat("")
				switch blob.MediaType {
				case "audio/mpeg":
					format = openai.ChatCompletionAudioParamFormatMP3
				case "audio/wav", "audio/vnd.wave":
					format = openai.ChatCompletionAudioParamFormatWAV
				default:
					// TODO: 转为 mp3
					return openai.ChatCompletionMessageParamUnion{},
						fmt.Errorf(
							"unsupported audio type: %q (must be 'audio/mpeg' or 'audio/wav')",
							blob.MediaType,
						)
				}
				parts = append(parts, openai.InputAudioContentPart(
					openai.ChatCompletionContentPartInputAudioInputAudioParam{
						Format: string(format),
						Data:   base64.StdEncoding.EncodeToString(blob.Content),
					},
				))
			} else {
				// 文件
				parts = append(parts, openai.FileContentPart(
					openai.ChatCompletionContentPartFileFileParam{
						Filename: param.NewOpt(blob.Filename),
						FileData: param.NewOpt(
							"data:" + blob.MediaType + ";base64," + base64.StdEncoding.EncodeToString(blob.Content),
						),
					},
				))
			}
		}
		return openai.UserMessage(parts), nil

	case RoleAssistant:
		if len(msg.Content) == 0 {
			return openai.AssistantMessage(""), nil
		}
		if len(msg.Content) == 1 && msg.Content[0].IsText() {
			return openai.AssistantMessage(msg.Content[0].Text.Content), nil
		}

		parts := make([]openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion, 0, len(msg.Content))
		for _, part := range msg.Content {
			if part.IsRefusal() {
				parts = append(parts, openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
					OfRefusal: &openai.ChatCompletionContentPartRefusalParam{
						Refusal: part.Refusal.Content,
					},
				})
			} else if part.IsText() {
				parts = append(parts, openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
					OfText: &openai.ChatCompletionContentPartTextParam{
						Text: part.Text.Content,
					},
				})
			} else {
				return openai.ChatCompletionMessageParamUnion{},
					fmt.Errorf("assistant message only support text content")
			}
		}
		return openai.AssistantMessage(parts), nil

	case RoleTool:
		switch len(msg.Content) {
		case 0:
			return openai.ToolMessage("", msg.ToolCallID), nil
		case 1:
			if !msg.Content[0].IsText() {
				return openai.ChatCompletionMessageParamUnion{},
					fmt.Errorf("tool message only support text content")
			}
			return openai.ToolMessage(msg.Content[0].Text.Content, msg.ToolCallID), nil
		}

		parts := make([]openai.ChatCompletionContentPartTextParam, 0, len(msg.Content))
		for _, part := range msg.Content {
			if !part.IsText() {
				return openai.ChatCompletionMessageParamUnion{},
					fmt.Errorf("tool message only support text content")
			}
			parts = append(parts, openai.ChatCompletionContentPartTextParam{
				Text: part.Text.Content,
			})
		}
		return openai.ToolMessage(parts, msg.ToolCallID), nil
	}

	return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("unsupported role %q", msg.Role)
}
