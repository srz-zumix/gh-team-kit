# gh-team-kit

**Manage GitHub teams, memberships, and repository access from your terminal.**

A [GitHub CLI](https://cli.github.com/) extension that turns everyday organization administration into commands
you can inspect, repeat, and automate.

## Why gh-team-kit?

- **Understand who has access.** Browse team hierarchies, inspect membership, and compare repository permissions.
- **Cut down on repetitive work.** Copy or synchronize team members and repository access instead of updating them
  one by one.
- **Back up and migrate teams.** Export and import team configurations, including hierarchy, members, repositories,
  and organization role assignments.
- **Go beyond team maintenance.** Manage organization roles and IDP/EMU group connections, or explore collaboration
  through pull request relationship graphs.

## Installation

Install [GitHub CLI](https://cli.github.com/) first, then authenticate with an account that has the permissions needed
for your organization:

```sh
gh auth login
gh extension install srz-zumix/gh-team-kit
```

## Quick Start

Replace `my-org`, `engineering`, and `platform` with your organization and existing team slugs.
These examples only read GitHub data.
The optional `--owner` flag defaults to the current repository's owner when omitted.

### Team Management

#### Display the team hierarchy

```sh
gh team-kit tree --owner my-org --recursive
```

Show the organization's team hierarchy. An optional team slug limits the starting point to that team;
`--recursive` retrieves nested teams (default: `false`).

### Member Management

#### List team members

```sh
gh team-kit member list engineering --owner my-org
```

List the members of a team. The team slug is required.

### Repository Management

#### Compare repository access between teams

```sh
gh team-kit diff engineering platform --owner my-org
```

Compare repositories and permissions for two teams. Both team slugs are required; optional trailing repository names
limit the comparison (default: all repositories associated with either team).

## Documentation

- [Command reference](docs/commands.md) — detailed usage, flags, examples, and read-only mode, grouped by command.
- [Team migration guide](docs/migrate.md) — export/import workflows and cross-organization migration.
- [Agent skills](docs/commands.md#skills-management) — install the bundled
  [gh-team-kit skill](skills/gh-team-kit/SKILL.md) for AI assistants.
- [Copilot CLI canvas dashboard](docs/commands.md#copilot-cli-canvas-extension) — explore pull request graphs interactively.
- [Contributing](CONTRIBUTING.md) — development setup and contribution guidelines.
