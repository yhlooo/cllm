package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/openai/openai-go"
)

const (
	openaiDefaultReasoningContentField = "reasoning_content"
)

// OpenAIChatCompletionHumanReadableFormatter OpenAI 对话补全响应人类可读格式化器
type OpenAIChatCompletionHumanReadableFormatter struct {
	Writer                io.Writer
	ShowReasoningContent  bool
	ReasoningContentField string
}

var _ Formatter = OpenAIChatCompletionHumanReadableFormatter{}

// Format 格式化响应并输出
func (f OpenAIChatCompletionHumanReadableFormatter) Format(data any, _ []byte) error {
	switch typedData := data.(type) {
	case openai.ChatCompletion:
		return f.formatResp(typedData)
	case openai.ChatCompletionChunk:
		return f.formatChunk(typedData)
	default:
		return fmt.Errorf("unsupported data type %T", data)
	}
}

// formatResp 格式化响应并输出
func (f OpenAIChatCompletionHumanReadableFormatter) formatResp(resp openai.ChatCompletion) error {
	if len(resp.Choices) == 0 {
		return nil
	}
	msg := resp.Choices[0].Message

	if f.ShowReasoningContent {
		field := f.ReasoningContentField
		if field == "" {
			field = openaiDefaultReasoningContentField
		}
		content, ok := msg.JSON.ExtraFields[field]
		if ok {
			var contentStr string
			if err := json.Unmarshal([]byte(content.Raw()), &contentStr); err != nil {
				return fmt.Errorf("invalid reasoning content: %w, content: %q (must be string)", err, content.Raw())
			}
			if contentStr != "" {
				_, err := f.Writer.Write([]byte("\x1b[2m" + strings.TrimSuffix(contentStr, "\n") + "\x1b[22m\n"))
				if err != nil {
					return err
				}
			}
		}
	}

	_, err := f.Writer.Write([]byte(strings.TrimSuffix(msg.Content, "\n") + "\n"))
	return err
}

// formatChunk 格式化流式消息块并输出
func (f OpenAIChatCompletionHumanReadableFormatter) formatChunk(chunk openai.ChatCompletionChunk) error {
	if len(chunk.Choices) == 0 {
		return nil
	}
	choice := chunk.Choices[0]

	// TODO: 思考过程

	if _, err := f.Writer.Write([]byte(choice.Delta.Content)); err != nil {
		return err
	}
	if choice.FinishReason != "" {
		if _, err := f.Writer.Write([]byte("\n")); err != nil {
			return err
		}
		if choice.FinishReason != string(openai.CompletionChoiceFinishReasonStop) {
			if _, err := f.Writer.Write([]byte("[" + choice.FinishReason + "]\n")); err != nil {
				return err
			}
		}
	}
	return nil
}
