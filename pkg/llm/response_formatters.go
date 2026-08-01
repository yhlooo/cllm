package llm

import "io"

// ResponseFormatters 响应格式化器
type ResponseFormatters interface {
	// Format 格式化响应并输出
	Format(r io.Reader) error
}
