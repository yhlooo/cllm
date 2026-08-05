package commands

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	"github.com/yhlooo/cllm/pkg/version"
)

// NewGlobalOptions 创建默认 GlobalOptions
func NewGlobalOptions() GlobalOptions {
	return GlobalOptions{
		Verbose:      false,
		OutputFormat: "human-readable",
	}
}

// GlobalOptions 全局选项
type GlobalOptions struct {
	// 更多日志
	Verbose bool
	// 输出格式
	OutputFormat string
}

// AddPFlags 将选项绑定到命令行参数
func (o *GlobalOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&o.Verbose, "verbose", "v", o.Verbose, i18n.T(MsgGlobalOptsVerboseDesc))
	fs.StringVarP(&o.OutputFormat, "output-format", "o", o.OutputFormat, i18n.T(MsgGlobalOptsOutputFormatDesc))
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
type Options struct{}

// AddPFlags 将选项绑定到命令行参数
func (o *Options) AddPFlags(_ *pflag.FlagSet) {}

// NewCommand 创建根命令
func NewCommand(name string) *cobra.Command {
	globalOpts := NewGlobalOptions()
	opts := NewOptions()

	var keylog *os.File
	cmd := &cobra.Command{
		Use:           name,
		Short:         i18n.T(MsgRootDesc),
		Long:          i18n.T(MsgRootLongDesc),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,

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
		newOpenAICommand(),
		newGeminiCommand(),
		newOllamaCommand(),
		newAnthropicCommand(),
		newVersionCommand(),
	)

	return cmd
}

// printRequest 打印请求
func printRequest(w io.StringWriter, req *http.Request, reqBody any) {
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

	if reqBody != nil {
		reqBodyRaw, _ := json.MarshalIndent(reqBody, "> ", "  ")
		_, _ = w.WriteString("> " + string(reqBodyRaw) + "\n")
	}
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
