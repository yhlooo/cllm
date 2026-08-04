// Package ollama 定义 Ollama Chat API 的数据类型
package ollama

// ChatRequest 是 POST /api/chat 的请求体
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	Format      any           `json:"format,omitempty"` // "json" 或 JSON Schema 对象
	Options     *Options      `json:"options,omitempty"`
	Stream      bool          `json:"stream"`               // 默认 true
	Think       any           `json:"think,omitempty"`      // bool 或 "high"/"medium"/"low"/"max"
	KeepAlive   any           `json:"keep_alive,omitempty"` // string 或 number（秒）
	Logprobs    bool          `json:"logprobs,omitempty"`
	TopLogprobs int           `json:"top_logprobs,omitempty"`
}

// ChatMessage 是对话中的单条消息
type ChatMessage struct {
	Role      string     `json:"role"`                 // "system" | "user" | "assistant" | "tool"
	Content   string     `json:"content"`              // 消息文本内容
	Images    []string   `json:"images,omitempty"`     // base64 编码的图片
	Thinking  string     `json:"thinking,omitempty"`   // think 模式下的思考过程
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 工具调用
}

// Tool 是可供模型调用的函数工具定义
type Tool struct {
	Type     string       `json:"type"` // 始终为 "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 是工具函数定义
type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"` // JSON Schema 对象
}

// ToolCall 是模型请求的工具调用
type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction 是工具调用的函数信息
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments,omitempty"` // JSON 对象
}

// ChatResponse 是 POST /api/chat 的响应体
type ChatResponse struct {
	Model              string      `json:"model"`
	CreatedAt          string      `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	DoneReason         string      `json:"done_reason,omitempty"`
	TotalDuration      int64       `json:"total_duration,omitempty"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalCount    int64       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalCount          int64       `json:"eval_count,omitempty"`
	EvalDuration       int64       `json:"eval_duration,omitempty"`
}

// Options 是模型推理参数
type Options struct {
	Seed        *int     `json:"seed,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	MinP        *float64 `json:"min_p,omitempty"`
	Stop        any      `json:"stop,omitempty"` // string 或 []string
	NumCtx      *int     `json:"num_ctx,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
}

// 辅助函数：创建基本类型指针

// Float64Ptr 返回 float64 指针
func Float64Ptr(v float64) *float64 {
	return &v
}
