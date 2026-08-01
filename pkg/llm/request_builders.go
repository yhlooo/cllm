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
	// WithSystemPrompt 带上系统提示词
	WithSystemPrompt(content string) RequestBuilder
	// WithUserPrompt 带上用户提示词
	WithUserPrompt(content string) RequestBuilder
	// Build 构建请求
	Build() (*http.Request, error)
}
