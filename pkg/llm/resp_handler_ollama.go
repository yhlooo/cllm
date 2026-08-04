package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yhlooo/cllm/pkg/ollama"
)

// OllamaChatResponseHandler Ollama Chat 响应处理器
type OllamaChatResponseHandler struct {
	Formatter    Formatter
	SessionStore SessionStore
}

var _ ResponseHandler = (*OllamaChatResponseHandler)(nil)

// Handle 处理非流式响应
func (h *OllamaChatResponseHandler) Handle(resp *http.Response) error {
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := ollama.ChatResponse{}
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
	if h.SessionStore != nil {
		contentStr := extractMessageContent(data.Message)
		if contentStr != "" {
			if err := h.SessionStore.Add(AssistantMessage(TextPart(contentStr))); err != nil {
				return fmt.Errorf("store assistant message error: %w", err)
			}
		}
	}

	return nil
}

// HandleStream 处理流式响应（NDJSON 格式，逐行解析）
func (h *OllamaChatResponseHandler) HandleStream(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, 100<<20) // 100MB buffer

	msgContent := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		data := ollama.ChatResponse{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			return fmt.Errorf("decode chunk from json error: %w, content: %s", err, line)
		}

		// 格式化输出
		if h.Formatter != nil {
			if err := h.Formatter.Format(data, []byte(line)); err != nil {
				return fmt.Errorf("format output data error: %w, content: %s", err, line)
			}
		}

		// 累积内容
		msgContent += extractMessageContent(data.Message)

		if data.Done {
			break
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	// 记录历史
	if h.SessionStore != nil && msgContent != "" {
		if err := h.SessionStore.Add(AssistantMessage(TextPart(msgContent))); err != nil {
			return fmt.Errorf("store assistant message error: %w", err)
		}
	}

	return nil
}

// extractMessageContent 从 ChatMessage 中提取文本内容
func extractMessageContent(msg ollama.ChatMessage) string {
	return msg.Content
}
