package cmd

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/config"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type ExportOptions struct {
	Exporter cmdutil.Exporter
}

func NewExportCmd() *cobra.Command {
	opts := &ExportOptions{}
	var repo string
	var output string

	var cmd = &cobra.Command{
		Use:   "export [owner]",
		Short: "Export team information",
		Long:  `Retrieve and display team information from the specified organization.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := ""
			if len(args) > 0 {
				owner = args[0]
			}
			repository, err := parser.Repository(parser.RepositoryOwner(owner), parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			exporter, err := config.NewExporter(repository)
			if err != nil {
				return fmt.Errorf("error creating exporter: %w", err)
			}
			organizationConfig, err := exporter.Export()
			if err != nil {
				return fmt.Errorf("error exporting teams: %w", err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				renderer.RenderExportedData(organizationConfig)
			} else {
				err = exporter.WriteFile(organizationConfig, output)
				if err != nil {
					return fmt.Errorf("error writing organization config to file: %w", err)
				}
				logger.Info("Exported team data to %s\n", output)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "Specify a repository to filter teams")
	f.StringVarP(&output, "output", "o", "teams.yml", "Output file for exported team data")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewExportCmd())
}
