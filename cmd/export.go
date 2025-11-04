package cmd

import (
	"fmt"
	"os"

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
	var output string
	var owner string

	var cmd = &cobra.Command{
		Use:   "export",
		Short: "Export team information",
		Long:  `Retrieve and display team information from the specified organization.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
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
				if output == "" || output == "-" {
					err = exporter.Write(organizationConfig, os.Stdout)
				} else {
					err = exporter.WriteFile(organizationConfig, output)
				}
				if err != nil {
					return fmt.Errorf("error writing organization config to file: %w", err)
				}
				logger.Info("Export completed successfully.", "output", output)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&output, "output", "o", "", "Output file for exported team data")
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewExportCmd())
}
