package cmd

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/pkg/prgraph"
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
	var minWeight float64
	var noBots bool
	var excludeDraft bool
	var includeFiles []string
	var depth int
	var groupBy []string
	var excludeGenerated bool
	var keepOrphans bool
	var weightBy string
	var halfLife float64
	var coChange bool
	var coChangeMaxFiles int
	var allowUsers []string
	var userAllowlist string
	var excludeDeleted bool
	var exportFormat string

	cmd := &cobra.Command{
		Use:   "pr-graph [<[HOST/]OWNER/REPO>...]",
		Short: "Generate a relationship graph from pull request activity",
		Long: `Analyze pull request activity and generate a graph showing relationships between users, teams, labels, and code areas.

The graph contains user, team, label, file, directory, and submodule nodes. Edges represent review, approval, comment, review request, team membership, file change, file creation, co-change, directory containment, CODEOWNERS ownership, and labeling relationships. By default, edge weights count occurrences.

Use --weight-by to weight path edges by changed lines instead of occurrences, --half-life to decay contributions by age, --co-change to link paths changed together, and --allow-user or --user-allowlist to restrict the graph to a known set of members.

Specify one or more repositories as arguments, or use --owner to analyze all repositories of an organization. Without arguments, the current repository is used.

Only activity that reached GitHub through a pull request is analyzed: direct pushes are not represented, and the "created" relation is limited to files added within the selected pull requests.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, edgeType := range append(append([]string{}, edgeTypes...), excludeEdgeTypes...) {
				if !slices.Contains(prgraph.RelationValues, edgeType) {
					return fmt.Errorf("invalid edge type %q: expected one of %s", edgeType, strings.Join(prgraph.RelationValues, ", "))
				}
			}
			if halfLife < 0 {
				return fmt.Errorf("invalid half-life %v: expected a non-negative number of days", halfLife)
			}
			if minWeight < 0 || math.IsNaN(minWeight) || math.IsInf(minWeight, 0) {
				return fmt.Errorf("invalid min-weight %v: expected a finite non-negative number", minWeight)
			}
			if coChangeMaxFiles < 0 {
				return fmt.Errorf("invalid co-change file limit %d: expected a non-negative number", coChangeMaxFiles)
			}
			allowedUsers, err := resolveUserAllowlist(allowUsers, userAllowlist)
			if err != nil {
				return err
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
				ExcludeGenerated:    excludeGenerated,
				KeepOrphans:         keepOrphans,
				WeightBy:            weightBy,
				HalfLife:            halfLife,
				CoChange:            coChange,
				CoChangeMaxFiles:    coChangeMaxFiles,
				AllowUsers:          allowedUsers,
				ExcludeDeleted:      excludeDeleted,
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
	cmdutil.StringEnumFlag(cmd, &state, "state", "", "all", []string{"open", "closed", "merged", "all"}, "Filter pull requests by state")
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
	f.Float64Var(&minWeight, "min-weight", 0, "Remove edges with a weight below this threshold from the graph (0 = no filter); --half-life can make weights fractional and --weight-by changes their scale, so choose the threshold accordingly")
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
	f.BoolVar(&excludeGenerated, "exclude-generated", false, "Exclude files marked linguist-generated in the repository's .gitattributes")
	f.BoolVar(&excludeDeleted, "exclude-deleted", false, "Exclude paths that no longer exist on the repository's default branch")
	cmdutil.StringEnumFlag(cmd, &weightBy, "weight-by", "", prgraph.WeightByOccurrences, prgraph.WeightByValues, "Weigh changed paths by (relations without a line count always use occurrences)")
	f.Float64Var(&halfLife, "half-life", 0, "Decay each contribution by the age of its pull request, halving it every N days (0 = no decay)")
	f.BoolVar(&coChange, "co-change", false, "Add co-changed edges between paths changed by the same pull request")
	f.IntVar(&coChangeMaxFiles, "co-change-max-files", 50, "Skip co-change edges for pull requests touching more than this many paths (0 = unlimited)")
	f.StringSliceVar(&allowUsers, "allow-user", nil, "Keep only these users in the graph, dropping others even as pull request authors (repeat or comma-separate)")
	f.StringVar(&userAllowlist, "user-allowlist", "", "Read allowed logins from a file, one per line; blank lines, \"#\" comments and a leading \"@\" are ignored")
	_ = cmdflags.AddFormatFlags(cmd, &opts.Exporter, &exportFormat, "mermaid", []string{"dot", "markdown", "mermaid"})

	return cmd
}

// resolveUserAllowlist merges the --allow-user logins with those read from the
// --user-allowlist file. It returns nil when neither option is set, leaving
// every user in the graph.
func resolveUserAllowlist(logins []string, path string) ([]string, error) {
	if len(logins) == 0 && path == "" {
		return nil, nil
	}
	allowed := append([]string(nil), prgraph.ParseUserAllowlist([]byte(strings.Join(logins, "\n")))...)
	if path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read the user allowlist %q: %w", path, err)
		}
		allowed = append(allowed, prgraph.ParseUserAllowlist(content)...)
	}
	allowed = prgraph.ParseUserAllowlist([]byte(strings.Join(allowed, "\n")))
	if len(allowed) == 0 {
		return nil, fmt.Errorf("the user allowlist is empty: expected at least one login")
	}
	return allowed, nil
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
