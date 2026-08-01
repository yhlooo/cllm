package llm

import (
	"bytes"
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

// Format 格式化响应并输出
func (f OpenAIChatCompletionHumanReadableFormatter) Format(r io.Reader) error {
	ret, err := decodeOpenAIChatCompletion(r)
	if err != nil {
		return err
	}

	if len(ret.Choices) == 0 {
		return nil
	}
	msg := ret.Choices[0].Message

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
				_, err = f.Writer.Write([]byte("\x1b[2m" + strings.TrimSuffix(contentStr, "\n") + "\x1b[22m\n"))
				if err != nil {
					return err
				}
			}
		}
	}

	_, err = f.Writer.Write([]byte(strings.TrimSuffix(msg.Content, "\n") + "\n"))

	return nil
}

// decodeOpenAIChatCompletion 解码 OpenAI 对话补全响应
func decodeOpenAIChatCompletion(r io.Reader) (*openai.ChatCompletion, error) {
	d := json.NewDecoder(r)
	ret := openai.ChatCompletion{}
	if err := d.Decode(&ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

// OpenAIChatCompletionJSONFormatter OpenAI 对话补全响应 JSON 格式化器
type OpenAIChatCompletionJSONFormatter struct {
	Writer io.Writer
}

// Format 格式化响应并输出
func (f OpenAIChatCompletionJSONFormatter) Format(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	buff := &bytes.Buffer{}
	if err := json.Indent(buff, raw, "", "  "); err != nil {
		return err
	}
	buff.WriteString("\n")
	_, err = io.Copy(f.Writer, buff)
	return err
}
