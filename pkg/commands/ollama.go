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
	"github.com/yhlooo/cllm/pkg/ollama"
)

const (
	defaultOllamaBaseURL = "http://localhost:11434"
)

// NewOllamaOptions 创建默认 OllamaOptions
func NewOllamaOptions() OllamaOptions {
	return OllamaOptions{
		BaseURL:              defaultOllamaBaseURL,
		ShowReasoningContent: true,
	}
}

// OllamaOptions ollama 子命令选项
type OllamaOptions struct {
	URL     string
	BaseURL string
	APIKey  string
	Headers []string
	Model   string
	Stream  bool

	SessionFile  string
	SystemPrompt string
	Attachments  []string

	// Ollama-specific
	Format      string
	KeepAlive   string
	Think       string // "true"/"false" 或 "high"/"medium"/"low"/"max"
	Logprobs    bool
	TopLogprobs int

	// Options fields
	Temperature float64
	TopP        float64
	TopK        int64
	NumPredict  int64
	Stop        []string
	Seed        int64

	ShowReasoningContent bool
	DryRun               bool
}

// AddPFlags 将选项绑定到命令行参数
func (o *OllamaOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.URL, "url", "u", o.URL, "Request URL")
	fs.StringVarP(&o.BaseURL, "base-url", "b", o.BaseURL, "Request base URL")
	fs.StringVarP(&o.APIKey, "api-key", "k", o.APIKey, "API Key")
	fs.StringSliceVarP(&o.Headers, "header", "H", o.Headers, "Custom header(s)")
	fs.StringVarP(&o.Model, "model", "m", o.Model, "Model name")
	fs.BoolVarP(&o.Stream, "stream", "s", o.Stream, "Stream output")

	fs.StringVar(&o.SessionFile, "session-file", o.SessionFile, "Session file")
	fs.StringVar(&o.SystemPrompt, "system-prompt", o.SystemPrompt, "System prompt")
	fs.StringSliceVarP(&o.Attachments, "attachment", "a", o.Attachments, "Attachment(s)")

	fs.StringVar(&o.Format, "format", o.Format, "Response format ('json' or JSON schema)")
	fs.StringVar(&o.KeepAlive, "keep-alive", o.KeepAlive, "Duration to keep model in memory (e.g. 5m, 0)")
	fs.StringVar(&o.Think, "think", o.Think, "Thinking mode (true/false or high/medium/low/max)")
	fs.BoolVar(&o.Logprobs, "logprobs", o.Logprobs, "Return log probabilities of output tokens")
	fs.IntVar(&o.TopLogprobs, "top-logprobs", o.TopLogprobs, "Number of most likely tokens to return at each position")

	fs.Float64Var(&o.Temperature, "temperature", o.Temperature, "Temperature")
	fs.Float64Var(&o.TopP, "top-p", o.TopP, "Top P")
	fs.Int64Var(&o.TopK, "top-k", o.TopK, "Top K")
	fs.Int64Var(&o.NumPredict, "num-predict", o.NumPredict, "Max number of tokens to predict")
	fs.StringSliceVar(&o.Stop, "stop", o.Stop, "Stop sequences")
	fs.Int64Var(&o.Seed, "seed", o.Seed, "Seed")

	fs.BoolVar(&o.ShowReasoningContent, "show-reasoning", o.ShowReasoningContent, "Show reasoning/thinking content")
	fs.BoolVar(&o.DryRun, "dry-run", o.DryRun, "No request sent")
}

// newOllamaCommand 创建 ollama 子命令
func newOllamaCommand() *cobra.Command {
	opts := NewOllamaOptions()
	cmd := &cobra.Command{
		Use:   "ollama [PROMPT...]",
		Short: i18n.T(MsgOllamaCmdShortDesc),
		Long: `Send Ollama chat request (POST /api/chat)

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

			builder := llm.NewOllamaChatRequest().
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

			// 设置 Ollama 特有参数
			ollamaBuilder := builder.(llm.OllamaChatRequestBuilder)
			if opts.Format != "" {
				ollamaBuilder = ollamaBuilder.WithFormat(opts.Format)
			}
			if opts.KeepAlive != "" {
				ollamaBuilder = ollamaBuilder.WithKeepAlive(opts.KeepAlive)
			}
			if opts.Think != "" {
				ollamaBuilder = ollamaBuilder.WithThink(parseThink(opts.Think))
			}
			if opts.Logprobs {
				ollamaBuilder = ollamaBuilder.WithLogprobs(true)
			}
			if opts.TopLogprobs != 0 {
				ollamaBuilder = ollamaBuilder.WithTopLogprobs(opts.TopLogprobs)
			}

			// 组装 Options
			var ollamaOpts ollama.Options
			hasOptions := false
			if opts.Temperature != 0 {
				ollamaOpts.Temperature = ollama.Float64Ptr(opts.Temperature)
				hasOptions = true
			}
			if opts.TopP != 0 {
				ollamaOpts.TopP = ollama.Float64Ptr(opts.TopP)
				hasOptions = true
			}
			if opts.TopK != 0 {
				tk := int(opts.TopK)
				ollamaOpts.TopK = &tk
				hasOptions = true
			}
			if opts.NumPredict != 0 {
				np := int(opts.NumPredict)
				ollamaOpts.NumPredict = &np
				hasOptions = true
			}
			if len(opts.Stop) > 0 {
				ollamaOpts.Stop = opts.Stop
				hasOptions = true
			}
			if opts.Seed != 0 {
				s := int(opts.Seed)
				ollamaOpts.Seed = &s
				hasOptions = true
			}
			if hasOptions {
				ollamaBuilder = ollamaBuilder.WithOptions(ollamaOpts)
			}

			// 构建请求
			req, err := ollamaBuilder.Build()
			if err != nil {
				return fmt.Errorf("build request error: %w", err)
			}
			if globalOpts.Verbose {
				var reqBody any
				if !opts.DryRun {
					reqBody, _ = ollamaBuilder.BuildBody()
				}
				printRequest(os.Stderr, req, reqBody)
			}

			if opts.DryRun {
				reqBody, _ := ollamaBuilder.BuildBody()
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
				formatter = &llm.OllamaChatHumanReadableFormatter{
					Writer:               os.Stdout,
					ShowReasoningContent: opts.ShowReasoningContent,
				}
			}

			handler := llm.OllamaChatResponseHandler{
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

// parseThink 解析 --think 参数值
// 支持 "true"/"false" 返回 bool，或直接返回字符串常量
func parseThink(think string) any {
	switch think {
	case "true":
		return true
	case "false":
		return false
	default:
		return think
	}
}
