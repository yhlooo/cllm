package llm

import (
	"fmt"
	"io"

	"github.com/yhlooo/cllm/pkg/ollama"
)

// OllamaChatHumanReadableFormatter Ollama Chat 响应人类可读格式化器
type OllamaChatHumanReadableFormatter struct {
	Writer               io.Writer
	ShowReasoningContent bool

	// 内部状态：追踪 thinking ANSI 暗色码是否已打开
	thinkingOpened bool
}

var _ Formatter = (*OllamaChatHumanReadableFormatter)(nil)

// Format 格式化响应并输出
func (f *OllamaChatHumanReadableFormatter) Format(data any, _ []byte) error {
	resp, ok := data.(ollama.ChatResponse)
	if !ok {
		return fmt.Errorf("unsupported data type %T", data)
	}

	// 输出思考内容
	if f.ShowReasoningContent && resp.Message.Thinking != "" {
		if !f.thinkingOpened {
			if _, err := f.Writer.Write([]byte("\x1b[2m")); err != nil {
				return err
			}
			f.thinkingOpened = true
		}
		if _, err := f.Writer.Write([]byte(resp.Message.Thinking)); err != nil {
			return err
		}
	}

	// 正文内容开始时，关闭 thinking 暗色输出
	if resp.Message.Content != "" {
		if f.thinkingOpened {
			if _, err := f.Writer.Write([]byte("\x1b[22m\n")); err != nil {
				return err
			}
			f.thinkingOpened = false
		}
		if _, err := f.Writer.Write([]byte(resp.Message.Content)); err != nil {
			return err
		}
	}

	// 流结束或非流式完成
	if resp.Done {
		if f.thinkingOpened {
			// 只有 thinking 没有 content 的情况（罕见），关闭 ANSI
			if _, err := f.Writer.Write([]byte("\x1b[22m\n")); err != nil {
				return err
			}
			f.thinkingOpened = false
			return nil
		}
		if _, err := f.Writer.Write([]byte("\n")); err != nil {
			return err
		}
	}

	return nil
}
