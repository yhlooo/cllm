package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/yhlooo/cllm/pkg/i18n"
	"github.com/yhlooo/cllm/pkg/llm"
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
)

// NewAnthropicOptions 创建默认 AnthropicOptions
func NewAnthropicOptions() AnthropicOptions {
	return AnthropicOptions{
		BaseURL:              defaultAnthropicBaseURL,
		ShowReasoningContent: true,
		MaxTokens:            64000,
	}
}

// AnthropicOptions anthropic 子命令选项
type AnthropicOptions struct {
	URL     string
	BaseURL string
	APIKey  string
	Headers []string
	Model   string
	Stream  bool

	SessionFile  string
	SystemPrompt string
	Attachments  []string

	// Anthropic 特有参数
	MaxTokens      int64
	Temperature    float64
	TopP           float64
	TopK           int64
	Stop           []string
	Thinking       bool
	ThinkingBudget int64

	ShowReasoningContent bool
	DryRun               bool
}

// AddPFlags 将选项绑定到命令行参数
func (o *AnthropicOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.URL, "url", "u", o.URL, "Request URL")
	fs.StringVarP(&o.BaseURL, "base-url", "b", o.BaseURL, "Request base URL")
	fs.StringVarP(&o.APIKey, "api-key", "k", o.APIKey, "API Key")
	fs.StringSliceVarP(&o.Headers, "header", "H", o.Headers, "Custom header(s)")
	fs.StringVarP(&o.Model, "model", "m", o.Model, "Model name")
	fs.BoolVarP(&o.Stream, "stream", "s", o.Stream, "Stream output")

	fs.StringVar(&o.SessionFile, "session-file", o.SessionFile, "Session file")
	fs.StringVar(&o.SystemPrompt, "system-prompt", o.SystemPrompt, "System prompt")
	fs.StringSliceVarP(&o.Attachments, "attachment", "a", o.Attachments, "Attachment(s)")

	fs.Int64Var(&o.MaxTokens, "max-tokens", o.MaxTokens, "Max tokens")
	fs.Float64Var(&o.Temperature, "temperature", o.Temperature, "Temperature")
	fs.Float64Var(&o.TopP, "top-p", o.TopP, "Top P")
	fs.Int64Var(&o.TopK, "top-k", o.TopK, "Top K")
	fs.StringSliceVar(&o.Stop, "stop", o.Stop, "Stop sequences")
	fs.BoolVar(&o.Thinking, "thinking", o.Thinking, "Enable extended thinking")
	fs.Int64Var(&o.ThinkingBudget, "thinking-budget", o.ThinkingBudget, "Thinking budget in tokens (min 1024)")

	fs.BoolVar(&o.ShowReasoningContent, "show-reasoning", o.ShowReasoningContent, "Show reasoning/thinking content")
	fs.BoolVar(&o.DryRun, "dry-run", o.DryRun, "No request sent")
}

// newAnthropicCommand 创建 anthropic 子命令
func newAnthropicCommand() *cobra.Command {
	opts := NewAnthropicOptions()
	cmd := &cobra.Command{
		Use:   "anthropic [PROMPT...]",
		Short: i18n.T(MsgAnthropicCmdShortDesc),
		Long: `Send Anthropic create message request (POST /v1/messages)

## Input Messages

  ┌────────────────────────────┐
  │     [History Messages]     │
  │     --session-file ...     │
  ├────────────────────────────┤
  │      [System Message]      │
  │    --system-prompt ...     │
  ├────────────────────────────┤
  │       [User Message]       │
  │  --attachment ...  (blob)  │
  │            ...             │
  │       args...  (text)      │
  └────────────────────────────┘
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			globalOpts := GlobalOptionsFromContext(ctx)

			builder := llm.NewAnthropicMessageRequest().
				WithContext(ctx).
				WithBaseURL(opts.BaseURL).
				WithURL(opts.URL).
				WithAPIKey(opts.APIKey).
				WithModel(opts.Model).
				WithStream(opts.Stream)

			for _, h := range opts.Headers {
				key, value, err := parseHeader(h)
				if err != nil {
					return fmt.Errorf("invalid header %q: %w (must be in format 'Key: value')", h, err)
				}
				builder = builder.WithHeader(key, value)
			}

			sessionStore := llm.SessionStore(llm.DiscardSessionStore{})
			if opts.SessionFile != "" {
				sessionStore = &llm.FileSessionStore{Path: opts.SessionFile}
			}
			defer func() { _ = sessionStore.Close() }()

			// 加载历史消息
			history, err := sessionStore.Load()
			if err != nil {
				return fmt.Errorf("load session history error: %w", err)
			}
			builder = builder.WithMessages(history...)

			// 添加系统消息
			if opts.SystemPrompt != "" {
				spMsg := llm.SystemMessage(llm.TextPart(opts.SystemPrompt))
				builder = builder.WithMessages(spMsg)
				if err := sessionStore.Add(spMsg); err != nil {
					return fmt.Errorf("store system message error: %w", err)
				}
			}

			// 组装用户消息
			var userParts []llm.MessageContentPart
			for _, path := range opts.Attachments {
				p, err := llm.ReadBlobFile(path)
				if err != nil {
					return fmt.Errorf("read attachment %q error: %w", path, err)
				}
				userParts = append(userParts, p)
			}
			for _, arg := range args {
				attachmentParts, err := llm.ParseTextAttachments(arg)
				if err != nil {
					return err
				}
				userParts = append(userParts, attachmentParts...)
				userParts = append(userParts, llm.TextPart(arg))
			}
			if len(userParts) > 0 {
				usrMsg := llm.UserMessage(userParts...)
				builder = builder.WithMessages(usrMsg)
				if err := sessionStore.Add(usrMsg); err != nil {
					return fmt.Errorf("store user message error: %w", err)
				}
			}

			// 添加 Anthropic 专属参数
			antBuilder := builder.(llm.AnthropicMessageRequestBuilder)
			if opts.MaxTokens != 0 {
				antBuilder = antBuilder.WithMaxTokens(opts.MaxTokens)
			}
			if opts.Temperature != 0 {
				antBuilder = antBuilder.WithTemperature(opts.Temperature)
			}
			if opts.TopP != 0 {
				antBuilder = antBuilder.WithTopP(opts.TopP)
			}
			if opts.TopK != 0 {
				antBuilder = antBuilder.WithTopK(opts.TopK)
			}
			if len(opts.Stop) > 0 {
				antBuilder = antBuilder.WithStopSequences(opts.Stop...)
			}
			if opts.Thinking || opts.ThinkingBudget > 0 {
				antBuilder = antBuilder.WithThinking(true, opts.ThinkingBudget)
			}

			// 构建请求
			req, err := antBuilder.Build()
			if err != nil {
				return fmt.Errorf("build request error: %w", err)
			}
			if globalOpts.Verbose {
				var reqBody any
				if !opts.DryRun {
					reqBody, _ = antBuilder.BuildBody()
				}
				printRequest(os.Stderr, req, reqBody)
			}

			if opts.DryRun {
				reqBody, _ := antBuilder.BuildBody()
				reqBodyRaw, _ := json.MarshalIndent(reqBody, "", "  ")
				fmt.Println(string(reqBodyRaw))
				return nil
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

			if resp.StatusCode != http.StatusOK {
				_, _ = io.Copy(os.Stdout, resp.Body)
				return fmt.Errorf("unexpected status code: %d (!= 200)", resp.StatusCode)
			}

			// 输出结果
			var formatter llm.Formatter
			switch globalOpts.OutputFormat {
			case "json":
				formatter = llm.JSONFormatter{Writer: os.Stdout}
			case "raw":
				formatter = llm.RawFormatter{Writer: os.Stdout}
			default: // "human-readable"
				formatter = &llm.AnthropicMessageHumanReadableFormatter{
					Writer:               os.Stdout,
					ShowReasoningContent: opts.ShowReasoningContent,
				}
			}

			handler := &llm.AnthropicMessageResponseHandler{
				Formatter:    formatter,
				SessionStore: sessionStore,
			}
			if opts.Stream {
				err = handler.HandleStream(resp)
			} else {
				err = handler.Handle(resp)
			}
			if err != nil {
				return fmt.Errorf("read from response error: %w", err)
			}

			return nil
		},
	}

	opts.AddPFlags(cmd.Flags())

	return cmd
}
