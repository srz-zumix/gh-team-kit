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

type RepoOptions struct {
	Exporter cmdutil.Exporter
}

func NewRepoCmd() *cobra.Command {
	opts := &RepoOptions{}
	var nameOnly bool
	var owner string

	cmd := &cobra.Command{
		Use:   "list <username>",
		Short: "List repositories of a user",
		Long:  `List all repositories owned by the specified user`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to parse repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			repos, err := gh.ListUserAccessableRepositories(ctx, client, repository, username, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to list repositories for user '%s': %w", username, err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, repos); err != nil {
					return fmt.Errorf("failed to export repositories: %w", err)
				}
				return nil
			}

			if nameOnly {
				for _, repo := range repos {
					fmt.Println(repo.GetName())
				}
				return nil
			}

			headers := []string{"NAME", "DESCRIPTION"}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, repo := range repos {
				table.Append([]string{
					repo.GetName(),
					repo.GetDescription(),
				})
			}
			table.Render()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&nameOnly, "name-only", "", false, "Output only repository names")
	cmd.Flags().StringVar(&owner, "owner", "", "Specify the owner of the repository")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
