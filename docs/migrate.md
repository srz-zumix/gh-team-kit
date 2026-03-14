# Team Migration Guide

This guide explains how to migrate teams from one GitHub organization to another using the `export` and `import` commands.

## Overview

1. Export team configuration (members, repositories, external groups, etc.) from the source organization to a YAML file
2. Review and edit the file as needed
3. Import the configuration into the destination organization

## Step 1: Export from source organization

```sh
gh team-kit export --owner <source-org> --output teams.yaml
```

This generates a YAML file containing team structure, members, maintainers, repositories, external group connections (EMU), and code review settings.

### Export options

| Flag | Default | Description |
| ------ | --------- | ----------- |
| `--owner <org>` | (current repo owner) | Source organization name |
| `--output <file>`, `-o <file>` | stdout | Output file path (`-` for stdout) |
| `--host <host>`, `-H <host>` | (current host) | GitHub host (for GHES) |
| `--no-export-repositories` | false | Skip repository permissions |
| `--no-export-group` | false | Skip external group connections (EMU) |
| `--no-suspended` | false | Exclude suspended users |

### Example output (`teams.yaml`)

```yaml
teams:
  - name: my-team
    slug: my-team
    description: My team description
    privacy: closed
    notification_setting: notifications_enabled
    maintainers:
      - alice
    members:
      - bob
      - carol
    group: "My External Group"   # EMU only
    repositories:
      - name: my-repo
        permission: push
hierarchy:
  - slug: my-team
```

> **Note:** The `group` field is only exported when the organization uses Enterprise Managed Users (EMU) and has external groups configured.

## Step 2: Review and edit the file

Before importing, update the YAML file as needed:

- Change `slug` / `name` if renaming teams
- Add or remove members
- Update `group` to the correct external group name in the destination organization
- Adjust repository permissions

## Step 3: Import into destination organization

```sh
gh team-kit import teams.yaml --owner <dest-org>
```

### Dry run (preview without applying)

```sh
gh team-kit import teams.yaml --owner <dest-org> --dryrun
```

### Import options

| Flag | Default | Description |
| ------ | --------- | ----------- |
| `--owner <org>` | (current repo owner) | Destination organization name |
| `--host <host>`, `-H <host>` | (current host) | GitHub host (for GHES) |
| `--dryrun`, `-n` | false | Preview changes without applying |

### Reading from stdin

```sh
gh team-kit export --owner <source-org> | gh team-kit import - --owner <dest-org>
```

## Migrating between GitHub instances (GHES to GHEC or vice versa)

When migrating across different GitHub hosts, specify `--host` explicitly:

```sh
# Export from GHES
gh team-kit export --owner <source-org> --host github.example.com --output teams.yaml

# Import into GHEC
gh team-kit import teams.yaml --owner <dest-org>
```

## EMU external group migration

If the destination organization uses Enterprise Managed Users (EMU), set the `group` field in the YAML to the name of the external group in the destination organization before importing.

```yaml
teams:
  - name: engineering
    slug: engineering
    group: "Engineering-Prod"   # external group name in the destination org
```

The `import` command will resolve the group by name and connect it to the team automatically.

## Advanced: transforming with --jq on the fly

Use `--format json --jq <expr>` on export to transform the configuration before importing.
The output is JSON, which is accepted directly by `import`.

### Add a suffix to all member and maintainer names

Useful when the destination organization uses a different login format (e.g. adding `_corp`).

```sh
gh team-kit export --owner <source-org> --format json \
  --jq '.teams |= map(
      (.members   |= map(. + "_suffix")) |
      (.maintainers |= map(. + "_suffix"))
    )' \
  | gh team-kit import - --owner <dest-org>
```

```sh
gh team-kit export --owner <source-org> --format json --jq '.teams |= map((.members |= map(. + "_suffix")) | (.maintainers |= map(. + "_suffix")))' | gh team-kit import - --owner <dest-org>
```

### Set an external group on a specific team

Connect external group `"Engineering-Prod"` to the team with slug `engineering`:

```sh
gh team-kit export --owner <source-org> --format json \
  --jq '.teams |= map(
      if .slug == "engineering" then .group = "Engineering-Prod" else . end
    )' \
  | gh team-kit import - --owner <dest-org>
```

```sh
gh team-kit export --owner <source-org> --format json --jq '.teams |= map(if .slug == "engineering" then .group = "Engineering-Prod" else . end)' | gh team-kit import - --owner <dest-org>
```

When a team is managed by an external group (EMU), membership is controlled by the IdP and `members`/`maintainers` should be left empty. You can clear them at the same time:

```sh
gh team-kit export --owner <source-org> --format json \
  --jq '.teams |= map(
      if .slug == "engineering" then
        .group = "Engineering-Prod" | .members = [] | .maintainers = []
      else . end
    )' \
  | gh team-kit import - --owner <dest-org>
```

```sh
gh team-kit export --owner <source-org> --format json --jq '.teams |= map(if .slug == "engineering" then .group = "Engineering-Prod" | .members = [] | .maintainers = [] else . end)' | gh team-kit import - --owner <dest-org>
```

### Combine both transformations in one pipeline

Add a suffix to all logins **and** assign an external group to a specific team:

```sh
gh team-kit export --owner <source-org> --format json \
  --jq '.teams |= map(
      (.members   |= map(. + "_suffix")) |
      (.maintainers |= map(. + "_suffix")) |
      if .slug == "engineering" then .group = "Engineering-Prod" else . end
    )' \
  | gh team-kit import - --owner <dest-org>
```

```sh
gh team-kit export --owner <source-org> --format json --jq '.teams |= map((.members |= map(. + "_suffix")) | (.maintainers |= map(. + "_suffix")) | if .slug == "engineering" then .group = "Engineering-Prod" else . end)' | gh team-kit import - --owner <dest-org>
```

## Notes

- Teams are created or updated idempotently; existing teams are updated rather than duplicated.
- Members and maintainers are added, and users not listed are removed from the team.
- Repository permissions are applied per team; repositories must already exist in the destination organization.
- External group connections (`group`) require EMU to be enabled in the destination organization.
