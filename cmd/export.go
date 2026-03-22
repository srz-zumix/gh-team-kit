package cmd

import (
	"fmt"
	"os"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/config"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
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
	var host string
	var owner string
	var noExportRepositories bool
	var noExportGroup bool
	var noExportOrgRoles bool
	var noSuspended bool
	var format string

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

			if host != "" {
				repository.Host = host
			}

			exporter, err := config.NewExporter(cmd.Context(), repository)
			if err != nil {
				return fmt.Errorf("error creating exporter: %w", err)
			}
			organizationConfig, err := exporter.Export(&config.ExportOptions{
				IsExportRepositories: !noExportRepositories,
				IsExportGroup:        !noExportGroup,
				IsExportOrgRoles:     !noExportOrgRoles,
				ExcludeSuspended:     noSuspended,
			})
			if err != nil {
				return fmt.Errorf("error exporting teams: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				return renderer.RenderExportedData(organizationConfig)
			}

			if output == "" || output == "-" {
				output = "stdout"
				err = organizationConfig.Write(os.Stdout)
			} else {
				err = organizationConfig.WriteFile(output)
			}
			if err != nil {
				return fmt.Errorf("error writing organization config to file: %w", err)
			}
			logger.Info("Export completed successfully.", "output", output)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&output, "output", "o", "", "Output file for exported team data")
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.StringVarP(&host, "host", "H", "", "Specify the GitHub host")
	f.BoolVar(&noExportRepositories, "no-export-repositories", false, "Specify whether to export repositories")
	f.BoolVar(&noExportGroup, "no-export-group", false, "Specify whether to export external group connections")
	f.BoolVar(&noExportOrgRoles, "no-export-org-roles", false, "Specify whether to export custom organization roles")
	f.BoolVar(&noSuspended, "no-suspended", false, "Exclude suspended users from export")

	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	cmdflags.SetupFormatFlagWithNonJSONFormats(cmd, &opts.Exporter, &format, "", []string{"yaml"})

	return cmd
}

func init() {
	rootCmd.AddCommand(NewExportCmd())
}
