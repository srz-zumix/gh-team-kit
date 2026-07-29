package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/prgraph"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type PrGraphOptions struct {
	Exporter cmdutil.Exporter
}

func NewPrGraphCmd() *cobra.Command {
	opts := &PrGraphOptions{}
	var owner string
	var state string
	var since string
	var until string
	var limit int
	var exportFormat string

	cmd := &cobra.Command{
		Use:   "pr-graph [<[HOST/]OWNER/REPO>...]",
		Short: "Generate a relationship graph from pull request activity",
		Long: `Analyze pull request activity and generate a graph showing relationships between users, teams, labels, and code areas.

The graph contains user, team, label, file, and directory nodes. Edges represent review, approval, comment, review request, team membership, file change, directory containment, CODEOWNERS ownership, and labeling relationships, weighted by the number of occurrences.

Specify one or more repositories as arguments, or use --owner to analyze all repositories of an organization. Without arguments, the current repository is used.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if state != "open" && state != "closed" && state != "all" {
				return fmt.Errorf("invalid state %q: expected open, closed, or all", state)
			}

			collectOpts := prgraph.Options{State: state, Limit: limit}
			if since != "" {
				t, err := parseDateTime(since)
				if err != nil {
					return fmt.Errorf("failed to parse --since: %w", err)
				}
				collectOpts.Since = &t
			}
			if until != "" {
				t, err := parseUntil(until)
				if err != nil {
					return fmt.Errorf("failed to parse --until: %w", err)
				}
				collectOpts.Until = &t
			}

			ctx := cmd.Context()
			var repos []repository.Repository
			var client *gh.GitHubClient

			if len(args) > 0 {
				if owner != "" {
					return fmt.Errorf("cannot use --owner with repository arguments")
				}
				for _, arg := range args {
					repo, err := parser.Repository(parser.RepositoryInput(arg))
					if err != nil {
						return fmt.Errorf("failed to parse repository %q: %w", arg, err)
					}
					repos = append(repos, repo)
				}
				for _, repo := range repos[1:] {
					if !strings.EqualFold(repo.Host, repos[0].Host) {
						return fmt.Errorf("cannot analyze repositories across multiple hosts: %q and %q", repos[0].Host, repo.Host)
					}
				}
				c, err := gh.NewGitHubClientWithRepo(repos[0])
				if err != nil {
					return fmt.Errorf("failed to create GitHub client: %w", err)
				}
				client = c
			} else if owner != "" {
				ownerRepo, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
				if err != nil {
					return fmt.Errorf("failed to parse owner: %w", err)
				}
				c, err := gh.NewGitHubClientWithRepo(ownerRepo)
				if err != nil {
					return fmt.Errorf("failed to create GitHub client: %w", err)
				}
				client = c
				ownerRepos, err := gh.ListOwnerRepositories(ctx, c, ownerRepo.Owner)
				if err != nil {
					return fmt.Errorf("failed to list repositories for owner '%s': %w", ownerRepo.Owner, err)
				}
				for _, r := range ownerRepos {
					repos = append(repos, repository.Repository{
						Host:  ownerRepo.Host,
						Owner: r.GetOwner().GetLogin(),
						Name:  r.GetName(),
					})
				}
			} else {
				repo, err := parser.Repository()
				if err != nil {
					return fmt.Errorf("failed to parse repository: %w", err)
				}
				repos = append(repos, repo)
				c, err := gh.NewGitHubClientWithRepo(repo)
				if err != nil {
					return fmt.Errorf("failed to create GitHub client: %w", err)
				}
				client = c
			}

			graph, err := prgraph.Collect(ctx, client, repos, collectOpts)
			if err != nil {
				return fmt.Errorf("failed to analyze pull request activity: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			return prgraph.Render(renderer, exportFormat, graph)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Analyze all repositories of the organization ([HOST/]OWNER)")
	f.StringVar(&state, "state", "all", "Filter pull requests by state: {open|closed|all}")
	f.StringVar(&since, "since", "", "Only include pull requests created on or after the given date (YYYY-MM-DD or RFC 3339)")
	f.StringVar(&until, "until", "", "Only include pull requests created on or before the given date (YYYY-MM-DD or RFC 3339)")
	f.IntVar(&limit, "limit", 30, "Maximum number of pull requests to analyze per repository (0 = unlimited)")
	_ = cmdflags.AddFormatFlags(cmd, &opts.Exporter, &exportFormat, "mermaid", []string{"dot", "markdown", "mermaid"})

	return cmd
}

// parseDateTime parses a date string in RFC 3339 or YYYY-MM-DD format.
func parseDateTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// parseUntil parses the --until value. RFC 3339 values are used as an exact
// instant, while a date-only value (YYYY-MM-DD) is treated as inclusive through
// the end of that UTC day so PRs created later on that date are not excluded.
func parseUntil(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
}

func init() {
	rootCmd.AddCommand(NewPrGraphCmd())
}
