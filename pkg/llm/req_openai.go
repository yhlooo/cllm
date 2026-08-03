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
	"github.com/openai/openai-go/shared"
)

const (
	oaiChatCompletionURI = "/chat/completions"
)

// OpenAIChatCompletionRequestBuilder OpenAI 对话补全请求构造器接口
type OpenAIChatCompletionRequestBuilder interface {
	RequestBuilder

	WithReasoningEffort(effort string) OpenAIChatCompletionRequestBuilder
	WithMaxTokens(v int64) OpenAIChatCompletionRequestBuilder
	WithMaxCompletionTokens(v int64) OpenAIChatCompletionRequestBuilder
	WithPromptCacheKey(v string) OpenAIChatCompletionRequestBuilder
	WithLogprobs(enabled bool) OpenAIChatCompletionRequestBuilder
	WithTopLogprobs(v int64) OpenAIChatCompletionRequestBuilder
	WithN(v int64) OpenAIChatCompletionRequestBuilder
	WithSafetyIdentifier(id string) OpenAIChatCompletionRequestBuilder
	WithModalities(modalities []string) OpenAIChatCompletionRequestBuilder
	WithStop(flag ...string) OpenAIChatCompletionRequestBuilder
	WithResponseFormatText() OpenAIChatCompletionRequestBuilder
	WithResponseFormatJSONSchema(name string, strict *bool, desc *string, schema any) OpenAIChatCompletionRequestBuilder
	WithResponseFormatJSON() OpenAIChatCompletionRequestBuilder
	WithPrediction(content ...string) OpenAIChatCompletionRequestBuilder
	WithSeed(seed int64) OpenAIChatCompletionRequestBuilder
	WithFrequencyPenalty(v float64) OpenAIChatCompletionRequestBuilder
	WithPresencePenalty(v float64) OpenAIChatCompletionRequestBuilder
	WithTemperature(v float64) OpenAIChatCompletionRequestBuilder
	WithTopP(v float64) OpenAIChatCompletionRequestBuilder
	WithLogitBias(v map[string]int64) OpenAIChatCompletionRequestBuilder
	WithStore(enabled bool) OpenAIChatCompletionRequestBuilder
	WithServiceTier(v string) OpenAIChatCompletionRequestBuilder
	WithStreamIncludeUsage(enabled bool) OpenAIChatCompletionRequestBuilder
	WithMetadata(meta map[string]string) OpenAIChatCompletionRequestBuilder

	// TODO: FunctionCall Functions ToolChoice Tools WebSearchOptions
}

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

	params openai.ChatCompletionNewParams
	stream bool

	errors []error
}

var _ OpenAIChatCompletionRequestBuilder = OpenAIChatCompletionRequest{}

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
	b.params.Model = model
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
		b.params.Messages = append(b.params.Messages, oaiMsg)
	}
	return b
}

// BuildBody 构建请求体内容
func (b OpenAIChatCompletionRequest) BuildBody() (any, error) {
	params := b.params
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

func (b OpenAIChatCompletionRequest) WithReasoningEffort(effort string) OpenAIChatCompletionRequestBuilder {
	b.params.ReasoningEffort = openai.ReasoningEffort(effort)
	return b
}

func (b OpenAIChatCompletionRequest) WithMaxTokens(v int64) OpenAIChatCompletionRequestBuilder {
	b.params.MaxTokens = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithMaxCompletionTokens(v int64) OpenAIChatCompletionRequestBuilder {
	b.params.MaxCompletionTokens = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithPromptCacheKey(v string) OpenAIChatCompletionRequestBuilder {
	b.params.PromptCacheKey = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithLogprobs(enabled bool) OpenAIChatCompletionRequestBuilder {
	b.params.Logprobs = param.NewOpt(enabled)
	return b
}

func (b OpenAIChatCompletionRequest) WithTopLogprobs(v int64) OpenAIChatCompletionRequestBuilder {
	b.params.TopLogprobs = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithN(v int64) OpenAIChatCompletionRequestBuilder {
	b.params.N = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithSafetyIdentifier(id string) OpenAIChatCompletionRequestBuilder {
	b.params.SafetyIdentifier = param.NewOpt(id)
	return b
}

func (b OpenAIChatCompletionRequest) WithModalities(modalities []string) OpenAIChatCompletionRequestBuilder {
	b.params.Modalities = modalities
	return b
}

func (b OpenAIChatCompletionRequest) WithStop(flag ...string) OpenAIChatCompletionRequestBuilder {
	switch len(flag) {
	case 0:
		return b
	case 1:
		b.params.Stop.OfString = param.NewOpt(flag[0])
	default:
		b.params.Stop.OfStringArray = flag
	}
	return b
}

func (b OpenAIChatCompletionRequest) WithResponseFormatText() OpenAIChatCompletionRequestBuilder {
	b.params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
		OfText: &openai.ResponseFormatTextParam{},
	}
	return b
}

func (b OpenAIChatCompletionRequest) WithResponseFormatJSONSchema(
	name string,
	strict *bool,
	desc *string,
	schema any,
) OpenAIChatCompletionRequestBuilder {
	var paramStrict param.Opt[bool]
	if strict != nil {
		paramStrict = param.NewOpt(*strict)
	}
	var paramDesc param.Opt[string]
	if desc != nil {
		paramDesc = param.NewOpt[string](*desc)
	}
	b.params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        name,
				Strict:      paramStrict,
				Description: paramDesc,
				Schema:      schema,
			},
		},
	}
	return b
}

func (b OpenAIChatCompletionRequest) WithResponseFormatJSON() OpenAIChatCompletionRequestBuilder {
	b.params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
	}
	return b
}

func (b OpenAIChatCompletionRequest) WithPrediction(content ...string) OpenAIChatCompletionRequestBuilder {
	switch len(content) {
	case 0:
		return b
	case 1:
		b.params.Prediction.Content = openai.ChatCompletionPredictionContentContentUnionParam{
			OfString: param.NewOpt(content[0]),
		}
	default:
		parts := make([]openai.ChatCompletionContentPartTextParam, 0, len(content))
		for _, part := range content {
			parts = append(parts, openai.ChatCompletionContentPartTextParam{
				Text: part,
			})
		}
		b.params.Prediction.Content = openai.ChatCompletionPredictionContentContentUnionParam{
			OfArrayOfContentParts: parts,
		}
	}
	return b
}

func (b OpenAIChatCompletionRequest) WithSeed(seed int64) OpenAIChatCompletionRequestBuilder {
	b.params.Seed = param.NewOpt(seed)
	return b
}

func (b OpenAIChatCompletionRequest) WithFrequencyPenalty(v float64) OpenAIChatCompletionRequestBuilder {
	b.params.FrequencyPenalty = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithPresencePenalty(v float64) OpenAIChatCompletionRequestBuilder {
	b.params.PresencePenalty = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithTemperature(v float64) OpenAIChatCompletionRequestBuilder {
	b.params.Temperature = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithTopP(v float64) OpenAIChatCompletionRequestBuilder {
	b.params.TopP = param.NewOpt(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithLogitBias(v map[string]int64) OpenAIChatCompletionRequestBuilder {
	b.params.LogitBias = v
	return b
}

func (b OpenAIChatCompletionRequest) WithStore(enabled bool) OpenAIChatCompletionRequestBuilder {
	b.params.Store = param.NewOpt(enabled)
	return b
}

func (b OpenAIChatCompletionRequest) WithServiceTier(v string) OpenAIChatCompletionRequestBuilder {
	b.params.ServiceTier = openai.ChatCompletionNewParamsServiceTier(v)
	return b
}

func (b OpenAIChatCompletionRequest) WithStreamIncludeUsage(enabled bool) OpenAIChatCompletionRequestBuilder {
	b.params.StreamOptions.IncludeUsage = param.NewOpt(enabled)
	return b
}

func (b OpenAIChatCompletionRequest) WithMetadata(meta map[string]string) OpenAIChatCompletionRequestBuilder {
	b.params.Metadata = meta
	return b
}
