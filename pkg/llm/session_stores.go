package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// SessionStore 会话存储器
type SessionStore interface {
	// Load 加载历史消息
	Load() ([]Message, error)
	// Add 记录指定消息
	Add(msg Message) error
	// Close 关闭
	Close() error
}

// FileSessionStore 基于文件的会话存储器
type FileSessionStore struct {
	Path string

	f *os.File
}

var _ SessionStore = (*FileSessionStore)(nil)

// Load 加载历史消息
func (s *FileSessionStore) Load() ([]Message, error) {
	if err := s.ensureFile(); err != nil {
		return nil, err
	}

	if _, err := s.f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek session file to start error: %w", err)
	}

	var history []Message
	d := json.NewDecoder(s.f)
	for d.More() {
		var msg Message
		if err := d.Decode(&msg); err != nil {
			return history, fmt.Errorf("decoding session file message: %w", err)
		}
		history = append(history, msg)
	}

	return history, nil
}

// Add 记录指定消息
func (s *FileSessionStore) Add(msg Message) error {
	if err := s.ensureFile(); err != nil {
		return err
	}

	if err := json.NewEncoder(s.f).Encode(msg); err != nil {
		return fmt.Errorf("encode message error: %w", err)
	}

	return nil
}

// ensureFile 确保文件打开
func (s *FileSessionStore) ensureFile() error {
	if s.f != nil {
		return nil
	}

	var err error
	s.f, err = os.OpenFile(s.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Close 关闭
func (s *FileSessionStore) Close() error {
	if s.f == nil {
		return nil
	}
	return s.f.Close()
}

// DiscardSessionStore 忽略所有消息的会话存储器
type DiscardSessionStore struct{}

var _ SessionStore = DiscardSessionStore{}

// Load 加载历史消息
func (DiscardSessionStore) Load() ([]Message, error) {
	return nil, nil
}

// Add 记录指定消息
func (s DiscardSessionStore) Add(_ Message) error {
	return nil
}

// Close 关闭
func (s DiscardSessionStore) Close() error {
	return nil
}
