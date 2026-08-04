package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/genai"
)

// GeminiGenerateContentResponseHandler Gemini 对话内容生成响应处理器
type GeminiGenerateContentResponseHandler struct {
	Formatter    Formatter
	SessionStore SessionStore
}

// Handle 处理非流式响应
func (h *GeminiGenerateContentResponseHandler) Handle(resp *http.Response) error {
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := genai.GenerateContentResponse{}
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
	if h.SessionStore != nil && len(data.Candidates) > 0 {
		msg := h.candidateToMessage(data.Candidates[0])
		if err := h.SessionStore.Add(msg); err != nil {
			return fmt.Errorf("store assistant message error: %w", err)
		}
	}

	return nil
}

// HandleStream 处理流式响应
func (h *GeminiGenerateContentResponseHandler) HandleStream(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(nil, 100<<20)

	var msgParts []MessageContentPart
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			line = line[5:]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		data := genai.GenerateContentResponse{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			return fmt.Errorf("decode chunk from json error: %w, content: %s", err, line)
		}

		// 格式化输出
		if err := h.Formatter.Format(data, []byte(line)); err != nil {
			return fmt.Errorf("format output data error: %w, content: %s", err, line)
		}

		// 累积消息内容
		if len(data.Candidates) > 0 && data.Candidates[0].Content != nil {
			for _, part := range data.Candidates[0].Content.Parts {
				if part.Text != "" && !part.Thought {
					msgParts = append(msgParts, TextPart(part.Text))
				}
			}
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	// 记录历史（流式结束后保存累积的文本）
	if h.SessionStore != nil {
		msg := AssistantMessage(msgParts...)
		if err := h.SessionStore.Add(msg); err != nil {
			return fmt.Errorf("store assistant message error: %w", err)
		}
	}

	return nil
}

// candidateToMessage 将 Gemini Candidate 转为内部 Message
func (h *GeminiGenerateContentResponseHandler) candidateToMessage(candidate *genai.Candidate) Message {
	var parts []MessageContentPart
	if candidate.Content != nil {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" && !part.Thought {
				parts = append(parts, TextPart(part.Text))
			}
		}
	}
	return AssistantMessage(parts...)
}
