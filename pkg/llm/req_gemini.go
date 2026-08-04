package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/genai"
)

const (
	geminiGenerateContentURI = "/models/%s:generateContent"
)

// geminiGenerateContentRequest Gemini generateContent REST API 请求体
type geminiGenerateContentRequest struct {
	Contents          []*genai.Content        `json:"contents,omitempty"`
	SystemInstruction *genai.Content          `json:"systemInstruction,omitempty"`
	SafetySettings    []*genai.SafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

// geminiGenerationConfig Gemini generationConfig REST API 嵌套对象
type geminiGenerationConfig struct {
	Temperature                *float32                        `json:"temperature,omitempty"`
	TopP                       *float32                        `json:"topP,omitempty"`
	TopK                       *float32                        `json:"topK,omitempty"`
	CandidateCount             int32                           `json:"candidateCount,omitempty"`
	MaxOutputTokens            int32                           `json:"maxOutputTokens,omitempty"`
	StopSequences              []string                        `json:"stopSequences,omitempty"`
	ResponseLogprobs           bool                            `json:"responseLogprobs,omitempty"`
	Logprobs                   *int32                          `json:"logprobs,omitempty"`
	PresencePenalty            *float32                        `json:"presencePenalty,omitempty"`
	FrequencyPenalty           *float32                        `json:"frequencyPenalty,omitempty"`
	Seed                       *int32                          `json:"seed,omitempty"`
	ResponseMIMEType           string                          `json:"responseMimeType,omitempty"`
	ResponseSchema             *genai.Schema                   `json:"responseSchema,omitempty"`
	ResponseJsonSchema         any                             `json:"responseJsonSchema,omitempty"`
	ResponseModalities         []string                        `json:"responseModalities,omitempty"`
	ThinkingConfig             *genai.ThinkingConfig           `json:"thinkingConfig,omitempty"`
	SpeechConfig               *genai.SpeechConfig             `json:"speechConfig,omitempty"`
	MediaResolution            genai.MediaResolution           `json:"mediaResolution,omitempty"`
	ImageConfig                *genai.ImageConfig              `json:"imageConfig,omitempty"`
	EnableEnhancedCivicAnswers *bool                           `json:"enableEnhancedCivicAnswers,omitempty"`
	AudioTranscriptionConfig   *genai.AudioTranscriptionConfig `json:"audioTranscriptionConfig,omitempty"`
}

// GeminiGenerateContentRequestBuilder Gemini 对话内容生成请求构造器接口
type GeminiGenerateContentRequestBuilder interface {
	RequestBuilder

	WithTemperature(v float64) GeminiGenerateContentRequestBuilder
	WithTopP(v float64) GeminiGenerateContentRequestBuilder
	WithTopK(v float64) GeminiGenerateContentRequestBuilder
	WithMaxOutputTokens(v int32) GeminiGenerateContentRequestBuilder
	WithStopSequences(v ...string) GeminiGenerateContentRequestBuilder
	WithSeed(v int32) GeminiGenerateContentRequestBuilder
	WithFrequencyPenalty(v float64) GeminiGenerateContentRequestBuilder
	WithPresencePenalty(v float64) GeminiGenerateContentRequestBuilder
	WithResponseMIMEType(v string) GeminiGenerateContentRequestBuilder
	WithResponseSchema(schema *genai.Schema) GeminiGenerateContentRequestBuilder
	WithThinkingConfig(cfg *genai.ThinkingConfig) GeminiGenerateContentRequestBuilder
	WithCandidateCount(v int32) GeminiGenerateContentRequestBuilder
}

// NewGeminiGenerateContentRequest 创建 Gemini 对话补全请求构造器
func NewGeminiGenerateContentRequest() GeminiGenerateContentRequest {
	return GeminiGenerateContentRequest{}
}

// GeminiGenerateContentRequest Gemini 对话补全请求构造器
type GeminiGenerateContentRequest struct {
	ctx         context.Context
	url         string
	apiKey      string
	extraHeader http.Header

	// 请求体各部分
	contents          []*genai.Content
	systemInstruction *genai.Content
	genConfig         *geminiGenerationConfig
	safetySettings    []*genai.SafetySetting

	stream bool

	errors []error
}

var _ GeminiGenerateContentRequestBuilder = GeminiGenerateContentRequest{}

// initGenConfig 确保 genConfig 已初始化
func (b GeminiGenerateContentRequest) initGenConfig() GeminiGenerateContentRequest {
	if b.genConfig == nil {
		b.genConfig = &geminiGenerationConfig{}
	}
	return b
}

// WithContext 带上上下文
func (b GeminiGenerateContentRequest) WithContext(ctx context.Context) RequestBuilder {
	b.ctx = ctx
	return b
}

// WithURL 带上 URL
func (b GeminiGenerateContentRequest) WithURL(url string) RequestBuilder {
	if url == "" {
		return b
	}
	b.url = url
	return b
}

// WithBaseURL 带上 URL 前缀
func (b GeminiGenerateContentRequest) WithBaseURL(baseURL string) RequestBuilder {
	if baseURL == "" {
		return b
	}
	b.url = strings.TrimSuffix(baseURL, "/") + geminiGenerateContentURI
	return b
}

// WithAPIKey 带上认证 API Key
func (b GeminiGenerateContentRequest) WithAPIKey(apiKey string) RequestBuilder {
	b.apiKey = apiKey
	return b
}

// WithHeader 带上请求头
func (b GeminiGenerateContentRequest) WithHeader(key, value string) RequestBuilder {
	b.extraHeader.Add(key, value)
	return b
}

// WithModel 带上模型名
func (b GeminiGenerateContentRequest) WithModel(model string) RequestBuilder {
	// 将模型名填充到 URL 中
	b.url = fmt.Sprintf(b.url, model)
	return b
}

// WithStream 带上指定流式模式开关
func (b GeminiGenerateContentRequest) WithStream(enabled bool) RequestBuilder {
	b.stream = enabled
	return b
}

// WithMessages 带上消息
func (b GeminiGenerateContentRequest) WithMessages(messages ...Message) RequestBuilder {
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			content := b.convertMessageToContent(msg)
			if content == nil {
				continue
			}
			// 合并到 systemInstruction
			if b.systemInstruction == nil {
				b.systemInstruction = content
			} else {
				b.systemInstruction.Parts = append(b.systemInstruction.Parts, content.Parts...)
			}
		case RoleUser:
			content := b.convertMessageToContent(msg)
			if content == nil {
				continue
			}
			content.Role = "user"
			b = b.appendContent(content)
		case RoleAssistant:
			content := b.convertMessageToContent(msg)
			if content == nil {
				continue
			}
			content.Role = "model"
			b = b.appendContent(content)
		case RoleTool:
			b.errors = append(b.errors, fmt.Errorf("tool role is not supported for Gemini"))
		default:
			b.errors = append(b.errors, fmt.Errorf("unsupported role %q", msg.Role))
		}
	}
	return b
}

// convertMessageToContent 将内部 Message 转换为 genai.Content
func (b GeminiGenerateContentRequest) convertMessageToContent(msg Message) *genai.Content {
	if len(msg.Content) == 0 {
		return nil
	}

	var parts []*genai.Part
	for _, part := range msg.Content {
		switch {
		case part.IsText():
			parts = append(parts, &genai.Part{Text: part.Text.Content})
		case part.IsBlob():
			parts = append(parts, &genai.Part{
				InlineData: &genai.Blob{
					Data:     part.Blob.Content,
					MIMEType: part.Blob.MediaType,
				},
			})
		case part.IsRefusal():
			// Gemini 无 refusal 对应概念，转为普通文本
			parts = append(parts, &genai.Part{Text: part.Refusal.Content})
		default:
			b.errors = append(b.errors, fmt.Errorf("unsupported message content part"))
		}
	}

	if len(parts) == 0 {
		return nil
	}

	return &genai.Content{Parts: parts}
}

// appendContent 添加 Content 到列表，处理连续同角色合并
func (b GeminiGenerateContentRequest) appendContent(content *genai.Content) GeminiGenerateContentRequest {
	if len(b.contents) > 0 && b.contents[len(b.contents)-1].Role == content.Role {
		// 同角色合并
		b.contents[len(b.contents)-1].Parts = append(
			b.contents[len(b.contents)-1].Parts, content.Parts...)
	} else {
		b.contents = append(b.contents, content)
	}
	return b
}

// WithTemperature 带上温度参数
func (b GeminiGenerateContentRequest) WithTemperature(v float64) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	f := float32(v)
	b.genConfig.Temperature = &f
	return b
}

// WithTopP 带上 TopP 参数
func (b GeminiGenerateContentRequest) WithTopP(v float64) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	f := float32(v)
	b.genConfig.TopP = &f
	return b
}

// WithTopK 带上 TopK 参数
func (b GeminiGenerateContentRequest) WithTopK(v float64) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	f := float32(v)
	b.genConfig.TopK = &f
	return b
}

// WithMaxOutputTokens 带上最大输出 token 数
func (b GeminiGenerateContentRequest) WithMaxOutputTokens(v int32) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.MaxOutputTokens = v
	return b
}

// WithStopSequences 带上停止序列
func (b GeminiGenerateContentRequest) WithStopSequences(v ...string) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.StopSequences = v
	return b
}

// WithSeed 带上随机种子
func (b GeminiGenerateContentRequest) WithSeed(v int32) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.Seed = &v
	return b
}

// WithFrequencyPenalty 带上频率惩罚
func (b GeminiGenerateContentRequest) WithFrequencyPenalty(v float64) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	f := float32(v)
	b.genConfig.FrequencyPenalty = &f
	return b
}

// WithPresencePenalty 带上存在惩罚
func (b GeminiGenerateContentRequest) WithPresencePenalty(v float64) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	f := float32(v)
	b.genConfig.PresencePenalty = &f
	return b
}

// WithResponseMIMEType 带上响应 MIME 类型
func (b GeminiGenerateContentRequest) WithResponseMIMEType(v string) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.ResponseMIMEType = v
	return b
}

// WithResponseSchema 带上响应 Schema
func (b GeminiGenerateContentRequest) WithResponseSchema(schema *genai.Schema) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.ResponseSchema = schema
	return b
}

// WithThinkingConfig 带上思考配置
func (b GeminiGenerateContentRequest) WithThinkingConfig(cfg *genai.ThinkingConfig) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.ThinkingConfig = cfg
	return b
}

// WithCandidateCount 带上候选数量
func (b GeminiGenerateContentRequest) WithCandidateCount(v int32) GeminiGenerateContentRequestBuilder {
	b = b.initGenConfig()
	b.genConfig.CandidateCount = v
	return b
}

// BuildBody 构建请求体内容
func (b GeminiGenerateContentRequest) BuildBody() (any, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}

	return geminiGenerateContentRequest{
		Contents:          b.contents,
		SystemInstruction: b.systemInstruction,
		GenerationConfig:  b.genConfig,
		SafetySettings:    b.safetySettings,
	}, nil
}

// Build 构建请求
func (b GeminiGenerateContentRequest) Build() (*http.Request, error) {
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

	// 构造 URL，流式参数通过 query string 传递
	reqURL := b.url
	if b.stream {
		parsedURL, err := url.Parse(reqURL)
		if err != nil {
			return nil, fmt.Errorf("parse request url error: %w", err)
		}
		query := parsedURL.Query()
		query.Set("alt", "sse")
		parsedURL.RawQuery = query.Encode()
		reqURL = parsedURL.String()
	}

	req, err := http.NewRequestWithContext(b.ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return req, err
	}

	req.Header.Set("Content-Type", "application/json")

	if b.apiKey != "" {
		req.Header.Set("x-goog-api-key", b.apiKey)
	}
	for k, v := range b.extraHeader {
		req.Header[k] = v
	}

	return req, nil
}
