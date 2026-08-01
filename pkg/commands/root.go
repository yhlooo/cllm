package commands

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/yhlooo/cllm/pkg/i18n"
	"github.com/yhlooo/cllm/pkg/llm"
	"github.com/yhlooo/cllm/pkg/version"
)

// NewGlobalOptions 创建默认 GlobalOptions
func NewGlobalOptions() GlobalOptions {
	return GlobalOptions{
		Verbose: false,
	}
}

// GlobalOptions 全局选项
type GlobalOptions struct {
	// 更多日志
	Verbose bool
}

// AddPFlags 将选项绑定到命令行参数
func (o *GlobalOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&o.Verbose, "verbose", "v", o.Verbose, i18n.T(MsgGlobalOptsVerboseDesc))
}

type globalOptsContextKey struct{}

// ContextWithGlobalOptions 创建带全局选项的 context.Context
func ContextWithGlobalOptions(parent context.Context, opts GlobalOptions) context.Context {
	return context.WithValue(parent, globalOptsContextKey{}, opts)
}

// GlobalOptionsFromContext 从 ctx 获取全局选项
func GlobalOptionsFromContext(ctx context.Context) GlobalOptions {
	opts, _ := ctx.Value(globalOptsContextKey{}).(GlobalOptions)
	return opts
}

// NewOptions 创建默认 Options
func NewOptions() Options {
	return Options{}
}

// Options 运行选项
type Options struct {
	URL          string
	BaseURL      string
	APIKey       string
	SystemPrompt string
	Headers      []string
	Model        string
	Stream       bool
}

// AddPFlags 将选项绑定到命令行参数
func (o *Options) AddPFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.URL, "url", "u", o.URL, "Request URL")
	fs.StringVarP(&o.BaseURL, "base-url", "b", o.BaseURL, "Request base URL")
	fs.StringVarP(&o.APIKey, "api-key", "k", o.APIKey, "API Key")
	fs.StringVar(&o.SystemPrompt, "system-prompt", o.SystemPrompt, "System prompt")
	fs.StringSliceVarP(&o.Headers, "header", "H", o.Headers, "Custom header(s)")
	fs.StringVarP(&o.Model, "model", "m", o.Model, "Model name")
	fs.BoolVarP(&o.Stream, "stream", "s", o.Stream, "Stream output")
}

// NewCommand 创建根命令
func NewCommand(name string) *cobra.Command {
	globalOpts := NewGlobalOptions()
	opts := NewOptions()

	var keylog *os.File
	cmd := &cobra.Command{
		Use:           fmt.Sprintf("%s [OPTIONS] PROMPT", name),
		Short:         i18n.T(MsgRootDesc),
		Long:          i18n.T(MsgRootLongDesc),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
		Args:          cobra.MinimumNArgs(1),

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			ctx = ContextWithGlobalOptions(ctx, globalOpts)

			// 初始化 logger
			logrusLogger := logrus.New()
			if globalOpts.Verbose {
				logrusLogger.SetLevel(logrus.DebugLevel)
			}
			logger := logrusr.New(logrusLogger)
			ctx = logr.NewContext(ctx, logger)

			// 设置本地化器
			ctx = i18n.ContextWithLocalizer(ctx, i18n.NewLocalizer(i18n.GetEnvLanguage()))

			cmd.SetContext(ctx)

			// 输出 TLS 握手密钥
			var err error
			keylog, err = setKeyLog()
			if err != nil {
				return fmt.Errorf("set tls key log error: %w", err)
			}

			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			builder := llm.NewOpenAIChatCompletionRequestBuilder().
				WithContext(ctx).
				WithBaseURL(opts.BaseURL).
				WithURL(opts.URL).
				WithAPIKey(opts.APIKey).
				WithModel(opts.Model).
				WithStream(opts.Stream).
				WithSystemPrompt(opts.SystemPrompt)

			for _, h := range opts.Headers {
				key, value, err := parseHeader(h)
				if err != nil {
					return fmt.Errorf("invalid header %q: %w (must be in format 'Key: value')", h, err)
				}
				builder = builder.WithHeader(key, value)
			}

			for _, arg := range args {
				builder = builder.WithUserPrompt(arg)
			}

			// 构建请求
			req, err := builder.Build()
			if err != nil {
				return fmt.Errorf("build request error: %w", err)
			}
			if globalOpts.Verbose {
				printRequest(os.Stderr, req)
			}

			// 发送请求
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("send request error: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if globalOpts.Verbose {
				printResponse(os.Stderr, resp)
			}

			if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
				return fmt.Errorf("read from response error: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d (!= 200)", resp.StatusCode)
			}

			return nil
		},

		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if keylog != nil {
				_ = keylog.Close()
			}
			return nil
		},
	}

	globalOpts.AddPFlags(cmd.PersistentFlags())
	opts.AddPFlags(cmd.Flags())

	cmd.AddCommand(
		newVersionCommand(),
	)

	return cmd
}

// printRequest 打印请求
func printRequest(w io.StringWriter, req *http.Request) {
	// 请求行
	_, _ = w.WriteString(fmt.Sprintf("> %s %s %s\n", req.Method, req.URL.RequestURI(), req.Proto))

	// 请求头
	_, _ = w.WriteString(fmt.Sprintf("> Host: %s\n", req.Host))
	headerKeys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		for _, v := range req.Header[k] {
			_, _ = w.WriteString(fmt.Sprintf("> %s: %s\n", k, v))
		}
	}
	_, _ = w.WriteString(fmt.Sprintf("> Content-Length: %d\n", req.ContentLength))
	_, _ = w.WriteString(">\n")
}

// printResponse 打印响应
func printResponse(w io.StringWriter, resp *http.Response) {
	_, _ = w.WriteString(fmt.Sprintf("< %s %s\n", resp.Proto, resp.Status))

	// 响应头
	headerKeys := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		for _, v := range resp.Header[k] {
			_, _ = w.WriteString(fmt.Sprintf("< %s: %s\n", k, v))
		}
	}
	_, _ = w.WriteString("<\n")
}

// parseHeader 解析请求头
func parseHeader(content string) (key, value string, err error) {
	divided := strings.SplitN(content, ":", 2)
	if len(divided) != 2 {
		return "", "", fmt.Errorf("missing ':'")
	}

	key = strings.TrimSpace(divided[0])
	if key == "" {
		return "", "", fmt.Errorf("missing key")
	}
	value = strings.TrimSpace(divided[1])

	return key, value, nil
}

// setKeyLog 设置 TLS keylog
func setKeyLog() (*os.File, error) {
	keylog := os.Getenv("SSLKEYLOGFILE")
	if keylog == "" {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(keylog), 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(keylog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	// 设置输出 keylog 文件
	http.DefaultClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,

			TLSClientConfig: &tls.Config{KeyLogWriter: f},
		},
	}

	return f, nil
}
