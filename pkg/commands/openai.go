package commands

import (
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
		URL:                   "",
		BaseURL:               "",
		APIKey:                "",
		SystemPrompt:          "",
		Headers:               nil,
		Model:                 "",
		Stream:                false,
		ShowReasoningContent:  true,
		ReasoningContentField: "",
	}
}

// OpenAIOptions openai 子命令选项
type OpenAIOptions struct {
	URL          string
	BaseURL      string
	APIKey       string
	SystemPrompt string
	Headers      []string
	Model        string
	Stream       bool

	ShowReasoningContent  bool
	ReasoningContentField string
}

// AddPFlags 将选项绑定到命令行参数
func (o *OpenAIOptions) AddPFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.URL, "url", "u", o.URL, "Request URL")
	fs.StringVarP(&o.BaseURL, "base-url", "b", o.BaseURL, "Request base URL")
	fs.StringVarP(&o.APIKey, "api-key", "k", o.APIKey, "API Key")
	fs.StringVar(&o.SystemPrompt, "system-prompt", o.SystemPrompt, "System prompt")
	fs.StringSliceVarP(&o.Headers, "header", "H", o.Headers, "Custom header(s)")
	fs.StringVarP(&o.Model, "model", "m", o.Model, "Model name")
	fs.BoolVarP(&o.Stream, "stream", "s", o.Stream, "Stream output")
	fs.BoolVar(&o.ShowReasoningContent, "show-reasoning", o.ShowReasoningContent, "Show reasoning content")
	fs.StringVar(&o.ReasoningContentField, "reasoning-field", o.ReasoningContentField, "Reasoning content field")
}

// newOpenAICommand 创建 openai 子命令
func newOpenAICommand() *cobra.Command {
	opts := NewOpenAIOptions()
	cmd := &cobra.Command{
		Use:   "openai PROMPT",
		Short: "Send OpenAI create chat completion request (POST /chat/completions)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			globalOpts := GlobalOptionsFromContext(ctx)

			builder := llm.NewOpenAIChatCompletionRequest().
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

			if resp.StatusCode != http.StatusOK {
				if globalOpts.OutputFormat == "raw" {
					_, _ = io.Copy(os.Stdout, resp.Body)
				}
				return fmt.Errorf("unexpected status code: %d (!= 200)", resp.StatusCode)
			}

			// 输出结果
			switch globalOpts.OutputFormat {
			case "json":
				err = llm.OpenAIChatCompletionJSONFormatter{
					Writer: os.Stdout,
				}.Format(resp.Body)
			case "raw":
				_, err = io.Copy(os.Stdout, resp.Body)
				if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
					return fmt.Errorf("read from response error: %w", err)
				}
			default: // "human-readable"
				err = llm.OpenAIChatCompletionHumanReadableFormatter{
					Writer:                os.Stdout,
					ShowReasoningContent:  opts.ShowReasoningContent,
					ReasoningContentField: opts.ReasoningContentField,
				}.Format(resp.Body)
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
