package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/genai"

	"github.com/yhlooo/cllm/pkg/i18n"
	"github.com/yhlooo/cllm/pkg/llm"
)

const (
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// NewGeminiOptions 创建默认 GeminiOptions
func NewGeminiOptions() GeminiOptions {
	return GeminiOptions{
		BaseURL:              defaultGeminiBaseURL,
		ShowReasoningContent: true,
	}
}

// GeminiOptions gemini 子命令选项
type GeminiOptions struct {
	URL     string
	BaseURL string
	APIKey  string
	Headers []string
	Model   string
	Stream  bool

	SessionFile  string
	SystemPrompt string
	Attachments  []string

	Temperature      float64
	TopP             float64
	TopK             float64
	MaxOutputTokens  int64
	Stop             []string
	Seed             int64
	FrequencyPenalty float64
	PresencePenalty  float64
	ResponseMIMEType string
	Thinking         bool
	ThinkingBudget   int64
	CandidateCount   int64

	ShowReasoningContent bool
	DryRun               bool
}

// AddPFlags 将选项绑定到命令行参数
func (o *GeminiOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.URL, "url", "u", o.URL, "Request URL")
	fs.StringVarP(&o.BaseURL, "base-url", "b", o.BaseURL, "Request base URL")
	fs.StringVarP(&o.APIKey, "api-key", "k", o.APIKey, "API Key")
	fs.StringSliceVarP(&o.Headers, "header", "H", o.Headers, "Custom header(s)")
	fs.StringVarP(&o.Model, "model", "m", o.Model, "Model name")
	fs.BoolVarP(&o.Stream, "stream", "s", o.Stream, "Stream output")

	fs.StringVar(&o.SessionFile, "session-file", o.SessionFile, "Session file")
	fs.StringVar(&o.SystemPrompt, "system-prompt", o.SystemPrompt, "System prompt")
	fs.StringSliceVarP(&o.Attachments, "attachment", "a", o.Attachments, "Attachment(s)")

	fs.Float64Var(&o.Temperature, "temperature", o.Temperature, "Temperature")
	fs.Float64Var(&o.TopP, "top-p", o.TopP, "Top P")
	fs.Float64Var(&o.TopK, "top-k", o.TopK, "Top K")
	fs.Int64Var(&o.MaxOutputTokens, "max-output-tokens", o.MaxOutputTokens, "Max output tokens")
	fs.StringSliceVar(&o.Stop, "stop", o.Stop, "Stop sequences")
	fs.Int64Var(&o.Seed, "seed", o.Seed, "Seed")
	fs.Float64Var(&o.FrequencyPenalty, "frequency-penalty", o.FrequencyPenalty, "Frequency penalty")
	fs.Float64Var(&o.PresencePenalty, "presence-penalty", o.PresencePenalty, "Presence penalty")
	fs.StringVar(&o.ResponseMIMEType, "response-mime-type", o.ResponseMIMEType, "Response MIME type (text/plain, application/json)")
	fs.BoolVar(&o.Thinking, "thinking", o.Thinking, "Enable model thinking")
	fs.Int64Var(&o.ThinkingBudget, "thinking-budget", o.ThinkingBudget, "Thinking budget in tokens")
	fs.Int64Var(&o.CandidateCount, "candidate-count", o.CandidateCount, "Candidate count")

	fs.BoolVar(&o.ShowReasoningContent, "show-reasoning", o.ShowReasoningContent, "Show reasoning content")
	fs.BoolVar(&o.DryRun, "dry-run", o.DryRun, "No request sent")
}

// newGeminiCommand 创建 gemini 子命令
func newGeminiCommand() *cobra.Command {
	opts := NewGeminiOptions()
	cmd := &cobra.Command{
		Use:   "gemini [PROMPT...]",
		Short: i18n.T(MsgGeminiDesc),
		Long: i18n.T(MsgGeminiLongDesc) + "\n\n" + `

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

			builder := llm.NewGeminiGenerateContentRequest().
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

			// 添加 Gemini 专属参数
			gBuilder := builder.(llm.GeminiGenerateContentRequestBuilder)
			if opts.Temperature != 0 {
				gBuilder = gBuilder.WithTemperature(opts.Temperature)
			}
			if opts.TopP != 0 {
				gBuilder = gBuilder.WithTopP(opts.TopP)
			}
			if opts.TopK != 0 {
				gBuilder = gBuilder.WithTopK(opts.TopK)
			}
			if opts.MaxOutputTokens != 0 {
				gBuilder = gBuilder.WithMaxOutputTokens(int32(opts.MaxOutputTokens))
			}
			if len(opts.Stop) > 0 {
				gBuilder = gBuilder.WithStopSequences(opts.Stop...)
			}
			if opts.Seed != 0 {
				gBuilder = gBuilder.WithSeed(int32(opts.Seed))
			}
			if opts.FrequencyPenalty != 0 {
				gBuilder = gBuilder.WithFrequencyPenalty(opts.FrequencyPenalty)
			}
			if opts.PresencePenalty != 0 {
				gBuilder = gBuilder.WithPresencePenalty(opts.PresencePenalty)
			}
			if opts.ResponseMIMEType != "" {
				gBuilder = gBuilder.WithResponseMIMEType(opts.ResponseMIMEType)
			}
			if opts.CandidateCount != 0 {
				gBuilder = gBuilder.WithCandidateCount(int32(opts.CandidateCount))
			}

			// 思考配置
			if opts.Thinking || opts.ThinkingBudget > 0 {
				thinkingCfg := &genai.ThinkingConfig{
					IncludeThoughts: true,
				}
				if opts.ThinkingBudget > 0 {
					budget := int32(opts.ThinkingBudget)
					thinkingCfg.ThinkingBudget = &budget
				}
				gBuilder = gBuilder.WithThinkingConfig(thinkingCfg)
			}

			// 构建请求
			req, err := gBuilder.Build()
			if err != nil {
				return fmt.Errorf("build request error: %w", err)
			}
			if globalOpts.Verbose {
				var reqBody any
				if !opts.DryRun {
					reqBody, _ = gBuilder.BuildBody()
				}
				printRequest(os.Stderr, req, reqBody)
			}

			if opts.DryRun {
				reqBody, _ := gBuilder.BuildBody()
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
				formatter = llm.GeminiGenerateContentHumanReadableFormatter{
					Writer:               os.Stdout,
					ShowReasoningContent: opts.ShowReasoningContent,
					Streaming:            opts.Stream,
				}
			}

			handler := &llm.GeminiGenerateContentResponseHandler{
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
