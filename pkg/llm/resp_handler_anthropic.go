package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// AnthropicMessageResponseHandler Anthropic Messages API 响应处理器
type AnthropicMessageResponseHandler struct {
	Formatter    Formatter
	SessionStore SessionStore
}

var _ ResponseHandler = (*AnthropicMessageResponseHandler)(nil)

// Handle 处理非流式响应
func (h *AnthropicMessageResponseHandler) Handle(resp *http.Response) error {
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := anthropic.Message{}
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
		msg := messageToInternal(data)
		if err := h.SessionStore.Add(msg); err != nil {
			return fmt.Errorf("store assistant message error: %w", err)
		}
	}

	return nil
}

// HandleStream 处理流式 SSE 响应
func (h *AnthropicMessageResponseHandler) HandleStream(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, 100<<20) // 100MB buffer

	var (
		eventType string
		dataLine  string
		textBuf   strings.Builder
	)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimSpace(line[5:])
		} else if line == "" {
			// 空行表示 SSE 事件结束
			if dataLine == "" {
				continue
			}

			// 忽略 ping 事件
			if eventType == "ping" {
				eventType, dataLine = "", ""
				continue
			}

			// 解析事件
			event := anthropic.MessageStreamEventUnion{}
			if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
				return fmt.Errorf("decode sse event from json error: %w, content: %s", err, dataLine)
			}

			// 格式化输出
			if h.Formatter != nil && eventType != "message_start" && eventType != "message_stop" {
				if err := h.Formatter.Format(event, []byte(dataLine)); err != nil {
					return fmt.Errorf("format output data error: %w, content: %s", err, dataLine)
				}
			}

			// 累积文本内容
			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					textBuf.WriteString(event.Delta.Text)
				}
			}

			eventType, dataLine = "", ""
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	// 记录历史
	if h.SessionStore != nil {
		textContent := textBuf.String()
		var parts []MessageContentPart
		if textContent != "" {
			parts = append(parts, TextPart(textContent))
		}
		if err := h.SessionStore.Add(AssistantMessage(parts...)); err != nil {
			return fmt.Errorf("store assistant message error: %w", err)
		}
	}

	return nil
}

// messageToInternal 将 Anthropic Message 转为内部 Message
func messageToInternal(msg anthropic.Message) Message {
	var parts []MessageContentPart
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				parts = append(parts, TextPart(block.Text))
			}
		case "thinking":
			if block.Thinking != "" {
				parts = append(parts, MessageContentPart{
					Reasoning: &MessageContentTextPart{
						Content: block.Thinking,
					},
				})
			}
		}
	}
	return AssistantMessage(parts...)
}
