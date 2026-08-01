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

	MsgVersionDesc = &i18n.Message{
		ID:    "commands.VersionDesc",
		Other: "Print the version information",
	}
)
