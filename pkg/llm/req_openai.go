package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
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

	history     []openai.ChatCompletionMessageParamUnion
	systemText  string
	userContent []openai.ChatCompletionContentPartUnionParam

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

// WithSessionFile 带上会话历史消息
func (b OpenAIChatCompletionRequest) WithSessionFile(_ string) RequestBuilder {
	// TODO: ...
	return b
}

// WithSystemText 带上系统文本消息
func (b OpenAIChatCompletionRequest) WithSystemText(content string) RequestBuilder {
	b.systemText = content
	return b
}

// WithUserAttachment 带上用户附件消息
func (b OpenAIChatCompletionRequest) WithUserAttachment(path string) RequestBuilder {
	mediaType, content, err := readAttachment(path)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("read attachment %q error: %w", path, err))
		return b
	}

	if strings.HasPrefix(mediaType, "image/") {
		// 图片
		b.userContent = append(b.userContent, openai.ImageContentPart(
			openai.ChatCompletionContentPartImageImageURLParam{
				URL: "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(content),
			},
		))
	} else if strings.HasPrefix(mediaType, "audio/") {
		// 音频
		format := openai.ChatCompletionAudioParamFormat("")
		switch mediaType {
		case "audio/mpeg":
			format = openai.ChatCompletionAudioParamFormatMP3
		case "audio/wav", "audio/vnd.wave":
			format = openai.ChatCompletionAudioParamFormatWAV
		default:
			// TODO: 转为 mp3
			b.errors = append(b.errors, fmt.Errorf(
				"unsupported audio type: %q (must be 'audio/mpeg' or 'audio/wav')",
				mediaType,
			))
			return b
		}
		b.userContent = append(b.userContent, openai.InputAudioContentPart(
			openai.ChatCompletionContentPartInputAudioInputAudioParam{
				Format: string(format),
				Data:   base64.StdEncoding.EncodeToString(content),
			},
		))
	} else {
		// 文件
		b.userContent = append(b.userContent, openai.FileContentPart(
			openai.ChatCompletionContentPartFileFileParam{
				Filename: param.NewOpt(filepath.Base(path)),
				FileData: param.NewOpt("data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(content)),
			},
		))
	}

	return b
}

// WithUserText 带上用户文本消息
func (b OpenAIChatCompletionRequest) WithUserText(content string) RequestBuilder {
	if content == "" {
		return b
	}
	// TODO: 解析其中的 @path/to/file
	b.userContent = append(b.userContent, openai.TextContentPart(content))
	return b
}

// Build 构建请求
func (b OpenAIChatCompletionRequest) Build() (*http.Request, error) {
	if len(b.errors) > 0 {
		return nil, errors.Join(b.errors...)
	}

	var messages []openai.ChatCompletionMessageParamUnion
	messages = append(messages, b.history...)
	if b.systemText != "" {
		messages = append(messages, openai.SystemMessage(b.systemText))
	}
	if len(b.userContent) > 0 {
		messages = append(messages, openai.UserMessage(b.userContent))
	}

	params := &openai.ChatCompletionNewParams{
		Model:    b.model,
		Messages: messages,
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

// readAttachment
func readAttachment(path string) (mediaType string, content []byte, err error) {
	content, err = os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}

	// 通过文件扩展名判断 media type
	mediaType = mime.TypeByExtension(filepath.Ext(path))

	if mediaType == "" {
		// 通过 Magic Number 判断 media type
		t, err := filetype.Match(content)
		if err == nil {
			mediaType = t.MIME.Value
		}
	}

	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	return mediaType, content, nil
}
