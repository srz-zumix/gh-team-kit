# gh-team-kit

A [GitHub CLI](https://cli.github.com) extension that makes managing GitHub Organization teams fast, scriptable, and safe.

Managing teams, members, and repository permissions through the GitHub web UI is tedious and error-prone at scale. `gh-team-kit` brings all of it to your terminal — from everyday member management to organization-wide migrations.

## Why gh-team-kit?

- **Complete team operations from the CLI**: Create, update, move, and visualize team hierarchies. Manage members, roles, repository permissions, organization roles, and member privileges without leaving your terminal.
- **Bulk operations & migration**: Export an entire organization's team structure to YAML and import it into another — including members, repository permissions, external groups (EMU), and org roles. User mapping files with regex support make cross-organization and cross-host migrations practical. Mannequin reattribution is automated too.
- **Sync & compare**: Sync members between teams, copy team/user permissions across repositories, diff teams and repositories, and perform set operations (union, intersection, difference) on team members.
- **Safe by design**: The global `--read-only` flag blocks all write API calls, and destructive commands support dry-run modes — preview every change before applying it.
- **Insight into collaboration**: `pr-graph` analyzes pull request activity and generates relationship graphs (Mermaid, DOT, JSON) between users, teams, labels, and code areas — usable as a code ownership signal.
- **Enterprise-ready**: Works with GitHub Enterprise hosts (`[HOST/]OWNER` format), Enterprise Managed Users (EMU) external groups, SAML IDP group sync, and Copilot metrics.

## Installation

```sh
gh extension install srz-zumix/gh-team-kit
```

## Quick Start

```sh
# Display your organization's team hierarchy as a tree
gh team-kit tree

# List members of a team
gh team-kit member list my-team

# Add a member to a team (preview safely first with --read-only)
gh team-kit member add my-team username --read-only
gh team-kit member add my-team username

# Export team configuration for backup or migration
gh team-kit export --owner my-org --output teams.yaml

# Preview an import without applying changes
gh team-kit import teams.yaml --dryrun
```

## Documentation

- **[Command Reference](docs/commands.md)** — Detailed usage, flags, and examples for all commands.
- **[Migration Guide](docs/migrate.md)** — Examples for migrating teams between organizations.
- **[Shell Completion Guide](https://github.com/srz-zumix/go-gh-extension/blob/main/docs/shell-completion.md)** — Enable shell completion for the extension (see also [`completion` command](docs/commands.md#shell-completion-commands)).

## Copilot CLI Canvas Extension

This repository ships a [GitHub Copilot CLI](https://github.com/github/copilot-cli) canvas extension in
[`.github/extensions/pr-graph-dashboard`](.github/extensions/pr-graph-dashboard) that renders `pr-graph` DOT output as an
interactive graph in the Copilot app side panel. See the
[extension README](.github/extensions/pr-graph-dashboard/README.md) for details.

## License

[MIT](LICENSE)
