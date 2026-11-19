package user

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/user/collaborator"
)

// NewCollaboratorCmd creates a new `user collaborator` command
func NewCollaboratorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collaborator",
		Short: "Manage outside collaborators of the organization",
		Long:  `Manage outside collaborators of the organization.`,
	}

	cmd.AddCommand(collaborator.NewConvertCmd())
	cmd.AddCommand(collaborator.NewListCmd())
	cmd.AddCommand(collaborator.NewRemoveCmd())

	return cmd
}
