# gh-team-kit

A GitHub CLI extension that makes GitHub Organization teams, members, and permissions manageable from the terminal.

Everything the web UI makes you click through one team at a time — creating teams, syncing members, replicating repository
permissions, migrating an organization — becomes a single scriptable command.

## Highlights

- **Bulk and scriptable**: Operate on many teams, members, and repositories at once, straight from the shell or CI.
- **Safe by default**: Run any command with `--read-only` to see what would happen before it happens, and use `--dryrun`
  on import-style commands.
- **Set operations on teams**: Answer "who is in team A but not team B?" with union, intersection, and difference
  operations on team members.
- **Diff and sync**: Compare teams, repositories, and permissions, then copy or synchronize them across repositories and
  organizations.
- **Organization migration**: Export an entire team structure (members, repositories, external groups, org roles, code
  review settings) to YAML, map users between organizations, and import it elsewhere — mannequins included.
- **Enterprise ready**: Works with GHES and Enterprise Managed Users (EMU), including IDP/external group connections and
  custom organization roles.
- **Insight, not just administration**: `pr-graph` turns pull request activity into a relationship graph of users, teams,
  labels, and code areas for reviewer and code-ownership analysis.
- **Agent friendly**: Ships embedded agent skills so AI assistants can drive the CLI correctly.

## Installation

```sh
gh extension install srz-zumix/gh-team-kit
```

## Quick Start

```sh
# List teams in the organization
gh team-kit list

# Show the team hierarchy
gh team-kit tree

# Add members to a team
gh team-kit member add my-team alice bob

# Who is in team-a but not in team-b?
gh team-kit member sets team-a '-' team-b

# Compare team permissions between two repositories
gh team-kit repo diff owner/repo1 owner/repo2

# Back up the whole organization structure
gh team-kit export --output teams.yaml

# Preview changes without touching anything
gh team-kit import teams.yaml --dryrun
```

Add `--read-only` to any command to block every write API call:

```sh
gh team-kit member add my-team alice --read-only
```

## Documentation

- [Command Reference](docs/commands.md) — usage, flags, and defaults for every command
- [Team Migration Guide](docs/migrate.md) — migrating teams between organizations with `export` / `import`
- [Shell Completion Guide](https://github.com/srz-zumix/go-gh-extension/blob/main/docs/shell-completion.md) — enable
  completion for the extension (`gh team-kit completion -s <shell>`); gh CLI completion must be configured first
- [Contributing](CONTRIBUTING.md) — development setup and pull request workflow

## Copilot CLI Canvas Extension

This repository ships a [GitHub Copilot CLI](https://github.com/github/copilot-cli) canvas extension in
[`.github/extensions/pr-graph-dashboard`](.github/extensions/pr-graph-dashboard) that renders `pr-graph` output as an
interactive graph in the Copilot app side panel, with filtering, search, and focus controls. It is discovered
automatically when the repository is opened in the Copilot app and requires a local Graphviz `dot` binary. See the
[extension README](.github/extensions/pr-graph-dashboard/README.md) for details.

## License

[MIT](LICENSE)
