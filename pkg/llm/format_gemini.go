package llm

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/genai"
)

// GeminiGenerateContentHumanReadableFormatter Gemini 对话补全响应人类可读格式化器
type GeminiGenerateContentHumanReadableFormatter struct {
	Writer               io.Writer
	ShowReasoningContent bool
	Streaming            bool
}

var _ Formatter = GeminiGenerateContentHumanReadableFormatter{}

// Format 格式化响应并输出
func (f GeminiGenerateContentHumanReadableFormatter) Format(data any, _ []byte) error {
	resp, ok := data.(genai.GenerateContentResponse)
	if !ok {
		return fmt.Errorf("unsupported data type %T", data)
	}
	if f.Streaming {
		return f.formatChunk(resp)
	}
	return f.formatResp(resp)
}

// formatResp 格式化非流式响应并输出
func (f GeminiGenerateContentHumanReadableFormatter) formatResp(resp genai.GenerateContentResponse) error {
	if len(resp.Candidates) == 0 {
		return nil
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return nil
	}

	for _, part := range candidate.Content.Parts {
		if part.Thought {
			if f.ShowReasoningContent && part.Text != "" {
				if _, err := f.Writer.Write([]byte(
					"\x1b[2m" + strings.TrimSuffix(part.Text, "\n") + "\x1b[22m\n",
				)); err != nil {
					return err
				}
			}
		} else if part.Text != "" {
			if _, err := f.Writer.Write([]byte(
				strings.TrimSuffix(part.Text, "\n") + "\n",
			)); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatChunk 格式化流式消息块并输出
func (f GeminiGenerateContentHumanReadableFormatter) formatChunk(chunk genai.GenerateContentResponse) error {
	if len(chunk.Candidates) == 0 {
		return nil
	}

	candidate := chunk.Candidates[0]
	if candidate.Content == nil {
		return nil
	}

	for _, part := range candidate.Content.Parts {
		if part.Thought {
			if f.ShowReasoningContent && part.Text != "" {
				if _, err := f.Writer.Write([]byte(
					"\x1b[2m" + part.Text + "\x1b[22m",
				)); err != nil {
					return err
				}
			}
		} else if part.Text != "" {
			if _, err := f.Writer.Write([]byte(part.Text)); err != nil {
				return err
			}
		}
	}

	if candidate.FinishReason != "" {
		if _, err := f.Writer.Write([]byte("\n")); err != nil {
			return err
		}
		if candidate.FinishReason != genai.FinishReasonStop {
			if _, err := f.Writer.Write([]byte(
				"[" + string(candidate.FinishReason) + "]\n",
			)); err != nil {
				return err
			}
		}
	}

	return nil
}
