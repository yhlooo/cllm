package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/yhlooo/cllm/pkg/llm"
)

// NewOpenAIOptions 创建默认 OpenAIOptions
func NewOpenAIOptions() OpenAIOptions {
	return OpenAIOptions{
		ShowReasoningContent: true,
	}
}

// OpenAIOptions openai 子命令选项
type OpenAIOptions struct {
	URL     string
	BaseURL string
	APIKey  string
	Headers []string
	Model   string
	Stream  bool

	SessionFile  string
	SystemPrompt string
	Attachments  []string

	ReasoningEffort     string
	MaxTokens           int64
	MaxCompletionTokens int64
	PromptCacheKey      string
	Logprobs            bool
	TopLogprobs         int64
	N                   int64
	SafetyIdentifier    string
	Modalities          []string
	Stop                []string
	ResponseFormat      string
	Seed                int64
	FrequencyPenalty    float64
	PresencePenalty     float64
	Temperature         float64
	TopP                float64
	Store               bool
	ServiceTier         string
	StreamIncludeUsage  bool
	Prediction          []string

	ShowReasoningContent  bool
	ReasoningContentField string
	DryRun                bool
}

// AddPFlags 将选项绑定到命令行参数
func (o *OpenAIOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.URL, "url", "u", o.URL, "Request URL")
	fs.StringVarP(&o.BaseURL, "base-url", "b", o.BaseURL, "Request base URL")
	fs.StringVarP(&o.APIKey, "api-key", "k", o.APIKey, "API Key")
	fs.StringSliceVarP(&o.Headers, "header", "H", o.Headers, "Custom header(s)")
	fs.StringVarP(&o.Model, "model", "m", o.Model, "Model name")
	fs.BoolVarP(&o.Stream, "stream", "s", o.Stream, "Stream output")

	fs.StringVar(&o.SessionFile, "session-file", o.SessionFile, "Session file")
	fs.StringVar(&o.SystemPrompt, "system-prompt", o.SystemPrompt, "System prompt")
	fs.StringSliceVarP(&o.Attachments, "attachment", "a", o.Attachments, "Attachment(s)")

	fs.StringVar(&o.ReasoningEffort, "reasoning-effort", o.ReasoningEffort, "Reasoning Effort")
	fs.Int64Var(&o.MaxTokens, "max-tokens", o.MaxTokens, "Max tokens")
	fs.Int64Var(&o.MaxCompletionTokens, "max-completion-tokens", o.MaxCompletionTokens, "Max completion tokens")
	fs.StringVar(&o.PromptCacheKey, "prompt-cache-key", o.PromptCacheKey, "Prompt cache key")
	fs.BoolVar(&o.Logprobs, "logprobs", o.Logprobs, "Enable logprobs")
	fs.Int64Var(&o.TopLogprobs, "top-logprobs", o.TopLogprobs, "Top logprobs")
	fs.Int64Var(&o.N, "n", o.N, "Number of completions")
	fs.StringVar(&o.SafetyIdentifier, "safety-identifier", o.SafetyIdentifier, "Safety identifier")
	fs.StringSliceVar(&o.Modalities, "modalities", o.Modalities, "Modalities")
	fs.StringSliceVar(&o.Stop, "stop", o.Stop, "Stop sequences")
	fs.StringVar(&o.ResponseFormat, "response-format", o.ResponseFormat, "Response format (text, json)")
	fs.Int64Var(&o.Seed, "seed", o.Seed, "Seed")
	fs.Float64Var(&o.FrequencyPenalty, "frequency-penalty", o.FrequencyPenalty, "Frequency penalty")
	fs.Float64Var(&o.PresencePenalty, "presence-penalty", o.PresencePenalty, "Presence penalty")
	fs.Float64Var(&o.Temperature, "temperature", o.Temperature, "Temperature")
	fs.Float64Var(&o.TopP, "top-p", o.TopP, "Top P")
	fs.BoolVar(&o.Store, "store", o.Store, "Store completion")
	fs.StringVar(&o.ServiceTier, "service-tier", o.ServiceTier, "Service tier")
	fs.BoolVar(&o.StreamIncludeUsage, "stream-include-usage", o.StreamIncludeUsage, "Include usage in stream")
	fs.StringSliceVar(&o.Prediction, "prediction", o.Prediction, "Prediction content")

	fs.BoolVar(&o.ShowReasoningContent, "show-reasoning", o.ShowReasoningContent, "Show reasoning content")
	fs.StringVar(&o.ReasoningContentField, "reasoning-field", o.ReasoningContentField, "Reasoning content field")
	fs.BoolVar(&o.DryRun, "dry-run", o.DryRun, "No request sent")
}

// newOpenAICommand 创建 openai 子命令
func newOpenAICommand() *cobra.Command {
	opts := NewOpenAIOptions()
	cmd := &cobra.Command{
		Use:   "openai [PROMPT...]",
		Short: "Send OpenAI create chat completion request (POST /chat/completions)",
		Long: `Send OpenAI create chat completion request (POST /chat/completions)

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

			builder := llm.NewOpenAIChatCompletionRequest().
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

			// 添加其他 OpenAI 参数
			oaiBuilder := builder.(llm.OpenAIChatCompletionRequestBuilder)
			if opts.ReasoningEffort != "" {
				oaiBuilder = oaiBuilder.WithReasoningEffort(opts.ReasoningEffort)
			}
			if opts.MaxTokens != 0 {
				oaiBuilder = oaiBuilder.WithMaxTokens(opts.MaxTokens)
			}
			if opts.MaxCompletionTokens != 0 {
				oaiBuilder = oaiBuilder.WithMaxCompletionTokens(opts.MaxCompletionTokens)
			}
			if opts.PromptCacheKey != "" {
				oaiBuilder = oaiBuilder.WithPromptCacheKey(opts.PromptCacheKey)
			}
			if opts.Logprobs {
				oaiBuilder = oaiBuilder.WithLogprobs(true)
			}
			if opts.TopLogprobs != 0 {
				oaiBuilder = oaiBuilder.WithTopLogprobs(opts.TopLogprobs)
			}
			if opts.N != 0 {
				oaiBuilder = oaiBuilder.WithN(opts.N)
			}
			if opts.SafetyIdentifier != "" {
				oaiBuilder = oaiBuilder.WithSafetyIdentifier(opts.SafetyIdentifier)
			}
			if len(opts.Modalities) > 0 {
				oaiBuilder = oaiBuilder.WithModalities(opts.Modalities)
			}
			if len(opts.Stop) > 0 {
				oaiBuilder = oaiBuilder.WithStop(opts.Stop...)
			}
			switch opts.ResponseFormat {
			case "text":
				oaiBuilder = oaiBuilder.WithResponseFormatText()
			case "json":
				oaiBuilder = oaiBuilder.WithResponseFormatJSON()
			}
			if opts.Seed != 0 {
				oaiBuilder = oaiBuilder.WithSeed(opts.Seed)
			}
			if opts.FrequencyPenalty != 0 {
				oaiBuilder = oaiBuilder.WithFrequencyPenalty(opts.FrequencyPenalty)
			}
			if opts.PresencePenalty != 0 {
				oaiBuilder = oaiBuilder.WithPresencePenalty(opts.PresencePenalty)
			}
			if opts.Temperature != 0 {
				oaiBuilder = oaiBuilder.WithTemperature(opts.Temperature)
			}
			if opts.TopP != 0 {
				oaiBuilder = oaiBuilder.WithTopP(opts.TopP)
			}
			if opts.Store {
				oaiBuilder = oaiBuilder.WithStore(true)
			}
			if opts.ServiceTier != "" {
				oaiBuilder = oaiBuilder.WithServiceTier(opts.ServiceTier)
			}
			if opts.StreamIncludeUsage {
				oaiBuilder = oaiBuilder.WithStreamIncludeUsage(true)
			}
			if len(opts.Prediction) > 0 {
				oaiBuilder = oaiBuilder.WithPrediction(opts.Prediction...)
			}

			// 构建请求
			req, err := oaiBuilder.Build()
			if err != nil {
				return fmt.Errorf("build request error: %w", err)
			}
			if globalOpts.Verbose {
				var reqBody any
				if !opts.DryRun {
					reqBody, _ = oaiBuilder.BuildBody()
				}
				printRequest(os.Stderr, req, reqBody)
			}

			if opts.DryRun {
				reqBody, _ := oaiBuilder.BuildBody()
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
				formatter = llm.OpenAIChatCompletionHumanReadableFormatter{
					Writer:                os.Stdout,
					ShowReasoningContent:  opts.ShowReasoningContent,
					ReasoningContentField: opts.ReasoningContentField,
				}
			}

			handler := llm.OpenAIChatCompletionResponseHandler{
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
