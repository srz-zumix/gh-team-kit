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

type ImportOptions struct {
	Exporter cmdutil.Exporter
}

func NewImportCmd() *cobra.Command {
	opts := &ImportOptions{}
	var input string
	var dryrun bool

	var cmd = &cobra.Command{
		Use:   "import [owner]",
		Short: "Import team information",
		Long:  `Read and apply team information to the specified organization.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := ""
			if len(args) > 0 {
				owner = args[0]
			}
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			importer, err := config.NewImporter(repository)
			if err != nil {
				return fmt.Errorf("error creating importer: %w", err)
			}
			organizationConfig, err := importer.ReadFile(input)
			if err != nil {
				return fmt.Errorf("error importing teams: %w", err)
			}
			if !dryrun {
				err = importer.Import(organizationConfig)
				if err != nil {
					return fmt.Errorf("error applying organization config: %w", err)
				}
			}
			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				renderer.RenderExportedData(organizationConfig)
				return nil
			}

			if dryrun {
				logger.Info("Dry run completed. No changes were made.")
			} else {
				logger.Info("Teams imported successfully.")
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: do not actually apply team changes")
	f.StringVarP(&input, "input", "i", "teams.yml", "Input file for imported team data")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewImportCmd())
}
