package cmd

import (
	"fmt"
	"slices"
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
	var excludeUsers []string
	var excludeAuthors []string
	var hideUsers []string
	var excludeFiles []string
	var labels []string
	var excludeLabels []string
	var base string
	var head string
	var excludeHeadBranches []string
	var edgeTypes []string
	var excludeEdgeTypes []string
	var minWeight int
	var noBots bool
	var excludeDraft bool
	var includeFiles []string
	var depth int
	var groupBy []string
	var keepOrphans bool
	var exportFormat string

	cmd := &cobra.Command{
		Use:   "pr-graph [<[HOST/]OWNER/REPO>...]",
		Short: "Generate a relationship graph from pull request activity",
		Long: `Analyze pull request activity and generate a graph showing relationships between users, teams, labels, and code areas.

The graph contains user, team, label, file, directory, and submodule nodes. Edges represent review, approval, comment, review request, team membership, file change, directory containment, CODEOWNERS ownership, and labeling relationships, weighted by the number of occurrences.

Specify one or more repositories as arguments, or use --owner to analyze all repositories of an organization. Without arguments, the current repository is used.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if state != "open" && state != "closed" && state != "merged" && state != "all" {
				return fmt.Errorf("invalid state %q: expected open, closed, merged, or all", state)
			}
			for _, edgeType := range append(append([]string{}, edgeTypes...), excludeEdgeTypes...) {
				if !slices.Contains(prgraph.RelationValues, edgeType) {
					return fmt.Errorf("invalid edge type %q: expected one of %s", edgeType, strings.Join(prgraph.RelationValues, ", "))
				}
			}

			collectOpts := prgraph.Options{
				State:               state,
				Limit:               limit,
				ExcludeUsers:        excludeUsers,
				ExcludeAuthors:      excludeAuthors,
				HideUsers:           hideUsers,
				ExcludeFiles:        excludeFiles,
				IncludeFiles:        includeFiles,
				Labels:              labels,
				ExcludeLabels:       excludeLabels,
				Base:                base,
				Head:                head,
				ExcludeHeadBranches: excludeHeadBranches,
				IncludeEdgeTypes:    edgeTypes,
				ExcludeEdgeTypes:    excludeEdgeTypes,
				MinWeight:           minWeight,
				NoBots:              noBots,
				ExcludeDraft:        excludeDraft,
				Depth:               depth,
				GroupBy:             groupBy,
				KeepOrphans:         keepOrphans,
			}
			if since != "" {
				t, err := parseDateTime(since)
				if err != nil {
					return fmt.Errorf("failed to parse --since: %w", err)
				}
				collectOpts.Since = &t
			}
			if until != "" {
				t, err := parseDateTime(until)
				if err != nil {
					return fmt.Errorf("failed to parse --until: %w", err)
				}
				if _, err := time.Parse(time.DateOnly, until); err == nil {
					t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
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
				ownerRepos, err := gh.ListOwnerRepositories(ctx, c, ownerRepo)
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
	f.StringVar(&state, "state", "all", "Filter pull requests by state: {open|closed|merged|all}")
	f.StringVar(&since, "since", "", "Only include pull requests created on or after the given date (YYYY-MM-DD or RFC 3339)")
	f.StringVar(&until, "until", "", "Only include pull requests created on or before the given date (YYYY-MM-DD or RFC 3339)")
	f.IntVar(&limit, "limit", 30, "Maximum number of pull requests to analyze per repository, counted after state/date/--exclude-author/--label/--base/--head filtering (0 = unlimited)")
	f.StringSliceVar(&labels, "label", nil, "Only include pull requests having at least one of these labels (repeat or comma-separate)")
	f.StringSliceVar(&excludeLabels, "exclude-label", nil, "Skip pull requests having any of these labels (repeat or comma-separate)")
	f.StringVar(&base, "base", "", "Only include pull requests whose base branch matches this glob pattern")
	f.StringVar(&head, "head", "", "Only include pull requests whose head branch matches this glob pattern")
	f.StringSliceVar(&excludeHeadBranches, "exclude-head-branch", nil, "Skip pull requests whose head branch matches any of these glob patterns (repeat or comma-separate)")
	f.StringSliceVar(&edgeTypes, "edge-type", nil, "Only include these edge relation types in the graph (repeat or comma-separate); default: all")
	f.StringSliceVar(&excludeEdgeTypes, "exclude-edge-type", nil, "Exclude these edge relation types from the graph (repeat or comma-separate)")
	f.IntVar(&minWeight, "min-weight", 0, "Remove edges with a weight below this threshold from the graph (0 = no filter)")
	f.BoolVar(&keepOrphans, "keep-orphans", false, "Keep nodes left without any edge after edge filtering instead of removing them")
	f.BoolVar(&noBots, "no-bots", false, "Automatically exclude and hide users whose login has a \"[bot]\" suffix")
	f.BoolVar(&excludeDraft, "exclude-draft", false, "Skip draft pull requests")
	f.StringSliceVar(&includeFiles, "include-file", nil, "Only include files matching these .gitignore-style patterns (repeat or comma-separate); default: all")
	f.IntVar(&depth, "depth", 0, "Fold changed file paths into their ancestor directory truncated to this many path segments, applied to paths matching no --group-by pattern (0 = no folding)")
	f.StringSliceVar(&groupBy, "group-by", nil, "Fold changed file paths using glob-style prefix patterns, e.g. \"LocalPackages/*,Assets/*/*\" (repeat or comma-separate); the first matching pattern wins and unmatched paths fall back to --depth")
	f.StringSliceVar(&excludeAuthors, "exclude-author", nil, "Skip pull requests authored by these users, removing them from analysis (repeat or comma-separate)")
	f.StringSliceVar(&hideUsers, "hide-user", nil, "Omit these users' non-author nodes and edges from the graph without excluding their pull requests (repeat or comma-separate)")
	f.StringSliceVar(&excludeUsers, "exclude-user", nil, "Deprecated: equivalent to setting both --exclude-author and --hide-user for these users")
	_ = f.MarkDeprecated("exclude-user", "use --exclude-author and/or --hide-user instead")
	f.StringSliceVar(&excludeFiles, "exclude-file", nil, "Exclude files from the graph using .gitignore-style patterns (repeat or comma-separate)")
	_ = cmdflags.AddFormatFlags(cmd, &opts.Exporter, &exportFormat, "mermaid", []string{"dot", "markdown", "mermaid"})

	return cmd
}

// parseDateTime parses a date string in RFC 3339 or YYYY-MM-DD format.
func parseDateTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse(time.DateOnly, s)
}

func init() {
	rootCmd.AddCommand(NewPrGraphCmd())
}
