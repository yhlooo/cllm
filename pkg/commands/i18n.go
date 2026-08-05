package commands

import "github.com/nicksnyder/go-i18n/v2/i18n"

var (
	MsgGlobalOptsVerboseDesc = &i18n.Message{
		ID:    "commands.GlobalOptsVerboseDesc",
		Other: "Show more logs",
	}
	MsgGlobalOptsOutputFormatDesc = &i18n.Message{
		ID:    "commands.GlobalOptsOutputFormatDesc",
		Other: "Output format. One of (human-readable, json, raw)",
	}

	MsgRootDesc = &i18n.Message{
		ID:    "commands.CmdShortDesc",
		Other: "cllm - CLI Client for LLM",
	}
	MsgRootLongDesc = &i18n.Message{
		ID:    "commands.CmdLongDesc",
		Other: `cllm - CLI Client for LLM`,
	}

	MsgGeminiDesc = &i18n.Message{
		ID:    "commands.GeminiDesc",
		Other: "Send Gemini generate content request (POST /models/{model}:generateContent)",
	}
	MsgGeminiLongDesc = &i18n.Message{
		ID:    "commands.GeminiLongDesc",
		Other: `Send Gemini generate content request (POST /models/{model}:generateContent)`,
	}

	MsgVersionDesc = &i18n.Message{
		ID:    "commands.VersionDesc",
		Other: "Print the version information",
	}

	MsgOllamaCmdShortDesc = &i18n.Message{
		ID:    "commands.OllamaCmdShortDesc",
		Other: "Send Ollama chat request (POST /api/chat)",
	}

	MsgAnthropicCmdShortDesc = &i18n.Message{
		ID:    "commands.AnthropicCmdShortDesc",
		Other: "Send Anthropic create message request (POST /v1/messages)",
	}
)
