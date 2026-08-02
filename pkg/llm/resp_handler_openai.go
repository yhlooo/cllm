package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
)

// OpenAIChatCompletionResponseHandler OpenAI 对话补全响应处理器
type OpenAIChatCompletionResponseHandler struct {
	Formatter    Formatter
	SessionStore SessionStore
}

var _ ResponseHandler = (*OpenAIChatCompletionResponseHandler)(nil)

// Handle 处理响应
func (h *OpenAIChatCompletionResponseHandler) Handle(resp *http.Response) error {
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := openai.ChatCompletion{}
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("decode response from json error: %w, content: %s", err, string(content))
	}

	// 格式化输出
	if h.Formatter != nil {
		if err := h.Formatter.Format(data, content); err != nil {
			return fmt.Errorf("format output data error: %w, content: %s", err, string(content))
		}
	}

	// 记录历史
	if h.SessionStore != nil && len(data.Choices) > 0 {
		oaiMsg := data.Choices[0].Message
		var parts []MessageContentPart
		if oaiMsg.Content != "" {
			parts = append(parts, TextPart(oaiMsg.Content))
		}
		if oaiMsg.Refusal != "" {
			parts = append(parts, RefusalPart(oaiMsg.Refusal))
		}
		if err := h.SessionStore.Add(AssistantMessage(parts...)); err != nil {
			return fmt.Errorf("store assistant message error: %w", err)
		}
	}

	return nil
}

// HandleStream 处理流式响应
func (h *OpenAIChatCompletionResponseHandler) HandleStream(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, 100<<20)

	msgContent := ""
	msgRefusal := ""
	for scanner.Scan() {
		content := scanner.Text()
		if strings.HasPrefix(content, "data:") {
			content = content[5:]
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		if content == "[DONE]" {
			break
		}

		data := openai.ChatCompletionChunk{}
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			return fmt.Errorf("decode chunk from json error: %w, content: %s", err, content)
		}

		// 格式化输出
		if err := h.Formatter.Format(data, []byte(content)); err != nil {
			return fmt.Errorf("format output data error: %w, content: %s", err, content)
		}

		if len(data.Choices) == 0 {
			continue
		}
		oaiMsg := data.Choices[0].Delta
		if oaiMsg.Content != "" {
			msgContent += oaiMsg.Content
		}
		if oaiMsg.Refusal != "" {
			msgRefusal += oaiMsg.Refusal
		}
	}

	// 记录历史
	var parts []MessageContentPart
	if msgContent != "" {
		parts = append(parts, TextPart(msgContent))
	}
	if msgRefusal != "" {
		parts = append(parts, RefusalPart(msgRefusal))
	}
	if err := h.SessionStore.Add(AssistantMessage(parts...)); err != nil {
		return fmt.Errorf("store assistant message error: %w", err)
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	return nil
}
