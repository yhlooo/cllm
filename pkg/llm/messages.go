package llm

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/h2non/filetype"
)

// Message 消息
type Message struct {
	Role       Role                 `json:"role"`
	Content    []MessageContentPart `json:"content,omitempty"`
	ToolCallID string               `json:"toolCallID,omitempty"`
}

// Role 角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// MessageContentPart 消息内容块
type MessageContentPart struct {
	Reasoning *MessageContentTextPart `json:"reasoning,omitempty"`
	Text      *MessageContentTextPart `json:"text,omitempty"`
	Blob      *MessageContentBlobPart `json:"blob,omitempty"`
	Refusal   *MessageContentTextPart `json:"refusal,omitempty"`
}

// IsReasoning 判断是否是思考内容
func (part MessageContentPart) IsReasoning() bool {
	return part.Reasoning != nil
}

// IsText 判断是否是文本内容
func (part MessageContentPart) IsText() bool {
	return part.Text != nil
}

// IsBlob 判断是否是二进制内容
func (part MessageContentPart) IsBlob() bool {
	return part.Blob != nil
}

// IsRefusal 判断是否是拒绝内容
func (part MessageContentPart) IsRefusal() bool {
	return part.Refusal != nil
}

// MessageContentTextPart 消息内容文本块
type MessageContentTextPart struct {
	Content string `json:"content"`
}

// MessageContentBlobPart 消息内容二进制块
type MessageContentBlobPart struct {
	Filename  string `json:"filename,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	Content   []byte `json:"content,omitempty"`
}

// NewMessage 创建消息
func NewMessage(role Role, content ...MessageContentPart) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

// SystemMessage 创建系统消息
func SystemMessage(content ...MessageContentPart) Message {
	return NewMessage(RoleSystem, content...)
}

// UserMessage 创建用户消息
func UserMessage(content ...MessageContentPart) Message {
	return NewMessage(RoleUser, content...)
}

// AssistantMessage 创建助手消息
func AssistantMessage(content ...MessageContentPart) Message {
	return NewMessage(RoleAssistant, content...)
}

// TextPart 创建文本块
func TextPart(content string) MessageContentPart {
	return MessageContentPart{
		Text: &MessageContentTextPart{
			Content: content,
		},
	}
}

// RefusalPart 创建拒绝块
func RefusalPart(content string) MessageContentPart {
	return MessageContentPart{
		Refusal: &MessageContentTextPart{
			Content: content,
		},
	}
}

var attachmentPathRegexp = regexp.MustCompile(`@((?:\S|\\ )+)(?:\s|$)`)

// ParseTextAttachments 解析文本中提及的附件
func ParseTextAttachments(content string) ([]MessageContentPart, error) {
	homeDir, _ := os.UserHomeDir()

	// 解析其中的 @path/to/file
	attachmentPaths := attachmentPathRegexp.FindAllStringSubmatch(content, -1)

	parts := make([]MessageContentPart, 0, len(attachmentPaths))
	for _, groups := range attachmentPaths {
		path := groups[1]
		if strings.HasPrefix(path, "~/") {
			path = filepath.Join(homeDir, path[2:])
		}
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			continue
		}

		part, err := ReadBlobFile(path)
		if err != nil {
			return nil, fmt.Errorf("read attachment %q error: %w", path, err)
		}

		parts = append(parts, part)
	}

	return parts, nil
}

// ReadBlobFile 读二进制文件
func ReadBlobFile(path string) (MessageContentPart, error) {
	part := MessageContentPart{
		Blob: &MessageContentBlobPart{
			Filename: filepath.Base(path),
		},
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return part, err
	}
	part.Blob.Content = content

	// 通过文件扩展名判断 media type
	mediaType := mime.TypeByExtension(filepath.Ext(path))

	if mediaType == "" {
		// 通过 Magic Number 判断 media type
		t, err := filetype.Match(content)
		if err == nil {
			mediaType = t.MIME.Value
		}
	}

	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	part.Blob.MediaType = mediaType

	return part, nil
}
