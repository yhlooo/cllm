package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/yhlooo/cllm/pkg/i18n"
	"github.com/yhlooo/cllm/pkg/version"
)

const versionTemplate = `Version:   {{ .Version }}
GitCommit: {{ .GitCommit }}
GoVersion: {{ .GoVersion }}
Arch:      {{ .Arch }}
OS:        {{ .OS }}
`

// newVersionCommand 创建 version 子命令
func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: i18n.T(MsgVersionDesc),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			globalOpts := GlobalOptionsFromContext(ctx)

			info := version.GetVersionInfo()

			switch globalOpts.OutputFormat {
			case "json":
				raw, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(raw))
			default:
				tpl, err := template.New("Version").Parse(versionTemplate)
				if err != nil {
					return err
				}
				return tpl.Execute(os.Stdout, info)
			}

			return nil
		},
	}
	return cmd
}
