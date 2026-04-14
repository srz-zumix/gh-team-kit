package cmd

import (
	"io/fs"

	"github.com/srz-zumix/gh-team-kit/version"
	"github.com/srz-zumix/go-gh-extension/pkg/skillsmith"
)

// RegisterSkillsCmd registers the skills subcommand with the root command.
// skillsFS must be the embedded filesystem containing the skills directory.
func RegisterSkillsCmd(skillsFS fs.FS) {
	rootCmd.AddCommand(skillsmith.NewSkillsCmd("gh-team-kit", version.Version, skillsFS))
}
