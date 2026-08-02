package llm

import (
	"bytes"
	"encoding/json"
	"io"
)

// Formatter 格式化器
type Formatter interface {
	// Format 格式化响应并输出
	Format(data any, raw []byte) error
}

// RawFormatter 原样输出
type RawFormatter struct {
	Writer io.Writer
}

// Format 格式化响应并输出
func (f RawFormatter) Format(_ any, raw []byte) error {
	_, err := f.Writer.Write(raw)
	return err
}

// JSONFormatter 以格式化 JSON 输出
type JSONFormatter struct {
	Writer io.Writer
}

var _ Formatter = JSONFormatter{}

// Format 格式化响应并输出
func (f JSONFormatter) Format(_ any, raw []byte) error {
	buff := &bytes.Buffer{}
	if err := json.Indent(buff, raw, "", "  "); err != nil {
		return err
	}
	buff.WriteString("\n")
	_, err := io.Copy(f.Writer, buff)
	return err
}
