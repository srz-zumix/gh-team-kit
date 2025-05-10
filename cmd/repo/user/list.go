package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var details bool
	var nameOnly bool
	var roles []string
	var repo string
	var suspended bool
	var affiliations []string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List direct repository collaborators",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			collaborators, err := gh.ListRepositoryCollaborators(ctx, client, repository, affiliations, roles)
			if err != nil {
				return fmt.Errorf("failed to list collaborators for repository %s: %w", repo, err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, collaborators); err != nil {
					return fmt.Errorf("error exporting collaborators: %w", err)
				}
				return nil
			}

			if nameOnly {
				for _, collaborator := range collaborators {
					fmt.Println(collaborator.Login)
				}
				return nil
			}

			if details {
				collaborators, err = gh.UpdateUsers(ctx, client, collaborators)
				if err != nil {
					return fmt.Errorf("failed to update collaborators: %w", err)
				}
				if suspended {
					collaborators = gh.CollectSuspendedUsers(collaborators)
				}
			}

			headers := []string{"USERNAME", "PERMISSION"}
			if details {
				headers = append(headers, "EMAIL", "SUSPENDED")
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, collaborator := range collaborators {
				row := []string{
					*collaborator.Login,
					gh.GetPermissionName(collaborator.Permissions),
				}
				if details {
					if collaborator.Email != nil {
						row = append(row, *collaborator.Email)
					} else {
						row = append(row, "")
					}
					if collaborator.SuspendedAt != nil {
						row = append(row, "Yes")
					} else {
						row = append(row, "No")
					}
				}
				table.Append(row)
			}
			table.Render()

			return nil
		},
	}

	cmdutil.StringSliceEnumFlag(cmd, &affiliations, "affiliation", "a", nil, gh.CollaboratorAffiliationList, "List of affiliations to filter users")
	cmd.Flags().BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "Repository in the format 'owner/name'")
	cmd.Flags().BoolVarP(&nameOnly, "name-only", "", false, "Output only collaborator names")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.PermissionsList, "List of permissions to filter users")
	cmd.Flags().BoolVarP(&suspended, "suspended", "", false, "Output only suspended members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
