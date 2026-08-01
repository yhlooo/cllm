package commands

import "github.com/nicksnyder/go-i18n/v2/i18n"

var (
	MsgGlobalOptsVerboseDesc = &i18n.Message{
		ID:    "commands.GlobalOptsVerboseDesc",
		Other: "Show more logs",
	}

	MsgRootDesc = &i18n.Message{
		ID:    "commands.CmdShortDesc",
		Other: "Example Go application.",
	}
	MsgRootLongDesc = &i18n.Message{
		ID:    "commands.CmdLongDesc",
		Other: `This is an example go application.`,
	}

	MsgVersionDesc = &i18n.Message{
		ID:    "commands.VersionDesc",
		Other: "Print the version information",
	}
	MsgVersionOptsOutputFormatDesc = &i18n.Message{
		ID:    "commands.VersionOptsOutputFormatDesc",
		Other: "Output format. One of (json, yaml)",
	}
)
