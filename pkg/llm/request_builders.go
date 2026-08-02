package llm

import (
	"context"
	"net/http"
)

// RequestBuilder 请求构造器
type RequestBuilder interface {
	// WithContext 带上上下文
	WithContext(ctx context.Context) RequestBuilder
	// WithURL 带上 URL
	WithURL(url string) RequestBuilder
	// WithBaseURL 带上 URL 前缀
	WithBaseURL(baseURL string) RequestBuilder
	// WithAPIKey 带上认证 API Key
	WithAPIKey(apiKey string) RequestBuilder
	// WithHeader 带上请求头
	WithHeader(key, value string) RequestBuilder
	// WithModel 带上模型名
	WithModel(model string) RequestBuilder
	// WithStream 带上指定流式模式开关
	WithStream(enabled bool) RequestBuilder

	// WithSessionFile 带上会话历史消息
	WithSessionFile(path string) RequestBuilder
	// WithSystemText 带上系统文本消息
	WithSystemText(content string) RequestBuilder
	// WithUserAttachment 带上用户附件消息
	WithUserAttachment(path string) RequestBuilder
	// WithUserText 带上用户文本消息
	WithUserText(content string) RequestBuilder

	// BuildBody 构建请求体内容
	BuildBody() (any, error)
	// Build 构建请求
	Build() (*http.Request, error)
}
