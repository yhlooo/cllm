package llm

import (
	"fmt"
	"io"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// AnthropicMessageHumanReadableFormatter Anthropic Messages API 响应人类可读格式化器
type AnthropicMessageHumanReadableFormatter struct {
	Writer               io.Writer
	ShowReasoningContent bool

	// 内部状态：追踪流式 thinking ANSI 暗色码是否已打开
	thinkingOpened bool
}

var _ Formatter = (*AnthropicMessageHumanReadableFormatter)(nil)

// Format 格式化响应并输出
func (f *AnthropicMessageHumanReadableFormatter) Format(data any, _ []byte) error {
	switch typedData := data.(type) {
	case anthropic.Message:
		return f.formatResp(typedData)
	case anthropic.MessageStreamEventUnion:
		return f.formatStreamEvent(typedData)
	default:
		return fmt.Errorf("unsupported data type %T", data)
	}
}

// formatResp 格式化非流式响应
func (f *AnthropicMessageHumanReadableFormatter) formatResp(resp anthropic.Message) error {
	var hasContent bool
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if _, err := f.Writer.Write([]byte(strings.TrimSuffix(block.Text, "\n") + "\n")); err != nil {
				return err
			}
			hasContent = true
		case "thinking":
			if f.ShowReasoningContent && block.Thinking != "" {
				if _, err := f.Writer.Write([]byte(
					"\x1b[2m" + strings.TrimSuffix(block.Thinking, "\n") + "\x1b[22m\n",
				)); err != nil {
					return err
				}
			}
		case "redacted_thinking":
			// 被编辑的思考内容，不输出
		}
	}

	if !hasContent && resp.StopReason != "" {
		if _, err := f.Writer.Write([]byte("[" + string(resp.StopReason) + "]\n")); err != nil {
			return err
		}
	}

	return nil
}

// formatStreamEvent 格式化流式事件
func (f *AnthropicMessageHumanReadableFormatter) formatStreamEvent(event anthropic.MessageStreamEventUnion) error {
	switch event.Type {
	case "content_block_delta":
		return f.formatContentBlockDelta(event)
	case "content_block_stop":
		return f.formatContentBlockStop(event)
	case "message_delta":
		return f.formatMessageDelta(event)
	}
	return nil
}

// formatContentBlockDelta 格式化内容块增量
func (f *AnthropicMessageHumanReadableFormatter) formatContentBlockDelta(event anthropic.MessageStreamEventUnion) error {
	switch event.Delta.Type {
	case "text_delta":
		// 正文开始时，关闭 thinking 暗色输出
		if f.thinkingOpened {
			if _, err := f.Writer.Write([]byte("\x1b[22m\n")); err != nil {
				return err
			}
			f.thinkingOpened = false
		}
		if _, err := f.Writer.Write([]byte(event.Delta.Text)); err != nil {
			return err
		}
	case "thinking_delta":
		if f.ShowReasoningContent {
			if !f.thinkingOpened {
				if _, err := f.Writer.Write([]byte("\x1b[2m")); err != nil {
					return err
				}
				f.thinkingOpened = true
			}
			if _, err := f.Writer.Write([]byte(event.Delta.Thinking)); err != nil {
				return err
			}
		}
	}
	return nil
}

// formatContentBlockStop 格式化内容块结束
func (f *AnthropicMessageHumanReadableFormatter) formatContentBlockStop(_ anthropic.MessageStreamEventUnion) error {
	// 内容块结束时关闭 thinking 暗色输出
	if f.thinkingOpened {
		if _, err := f.Writer.Write([]byte("\x1b[22m\n")); err != nil {
			return err
		}
		f.thinkingOpened = false
	}
	return nil
}

// formatMessageDelta 格式化消息级别的增量
func (f *AnthropicMessageHumanReadableFormatter) formatMessageDelta(event anthropic.MessageStreamEventUnion) error {
	if f.thinkingOpened {
		if _, err := f.Writer.Write([]byte("\x1b[22m\n")); err != nil {
			return err
		}
		f.thinkingOpened = false
	}

	stopReason := event.Delta.StopReason
	if stopReason != "" && stopReason != "end_turn" {
		if _, err := f.Writer.Write([]byte("\n")); err != nil {
			return err
		}
		if _, err := f.Writer.Write([]byte("[" + string(stopReason) + "]\n")); err != nil {
			return err
		}
	}
	return nil
}
