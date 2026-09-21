package copilot

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/version"
	"github.com/srz-zumix/go-gh-extension/pkg/copilotext"
)

// NewExtensionCmd creates the "extension" command for managing Copilot CLI canvas
// extensions bundled with gh-team-kit.
func NewExtensionCmd() *cobra.Command {
	cfg := copilotext.Config{
		ToolName:    "gh-team-kit",
		ToolVersion: version.Version,
		Extensions: []copilotext.Extension{
			{
				Name: "pr-graph-dashboard",
				URL:  fmt.Sprintf("https://github.com/srz-zumix/gh-team-kit/tree/v%s/.github/extensions/pr-graph-dashboard", version.Version),
			},
		},
	}
	return copilotext.NewExtensionCmd(cfg)
}
