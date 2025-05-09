package client

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
)

func (g *GitHubClient) Write(exporter cmdutil.Exporter, data any) error {
	return exporter.Write(g.IO, data)
}
