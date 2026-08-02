package llm

import "net/http"

// ResponseHandler 响应处理器
type ResponseHandler interface {
	// Handle 处理响应
	Handle(resp *http.Response) error
	// HandleStream 处理流式响应
	HandleStream(resp *http.Response) error
}
