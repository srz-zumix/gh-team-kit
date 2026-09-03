# gh-team-kit

gh extension of github team api

## Installation

To install the tool, you can use the following command:

```sh
gh extension install srz-zumix/gh-team-kit
```

## Shell Completion

**Workaround Available!** While gh CLI doesn't natively support extension completion, we provide a patch script that enables it.

**Prerequisites:** Before setting up gh-team-kit completion, ensure gh CLI completion is configured for your shell. See [gh completion documentation](https://cli.github.com/manual/gh_completion) for setup instructions.

For detailed installation instructions and setup for each shell, see the [Shell Completion Guide](https://github.com/srz-zumix/go-gh-extension/blob/main/docs/shell-completion.md).

## Commands Overview

The following commands are available in `gh-team-kit`. Each command is designed to help manage GitHub teams, repositories, and users efficiently.

- **Team Management**: Create, update, delete, move, and display team hierarchies.
- **Configuration Management**: Export and import team information for backup and bulk operations.
- **Member Management**: Add, remove, check, and modify roles of team members.
- **Repository Management**: Add, remove, list, and compare repositories associated with teams.
- **Skills Management**: Install, update, and manage embedded agent skills for AI assistants.
- **User Management**: Add, remove, list, and check users in the organization or repositories.
- **Organization-Role Management**: Manage roles within the organization, including listing available roles.
- **Code Review Management**: Get and set code review assignment settings for teams.
- **Pull Request Analysis**: Generate relationship graphs from pull request activity, including users, teams, labels, and code areas.
- **IDP Management**: Manage IDP groups (SAML team sync) and external groups (Enterprise Managed Users).
- **Mannequin Management**: List and reattribute mannequins (placeholder accounts for unclaimed users), individually or in bulk via a user mapping file.
- **Permission Management**: Check and synchronize permissions for teams and users across repositories.
- **Comparison Tools**: Compare teams, repositories, and permissions to identify differences.

Refer to the specific command sections below for detailed usage and examples.

## Usage

### Global Options

#### Read-Only Mode

```sh
gh team-kit <command> --read-only
```

Run any command in read-only mode to prevent write operations. This option is useful for safely testing commands or verifying what changes would be made without actually applying them. When enabled, all API calls that would modify data (create, update, delete operations) will be blocked.

Example:

```sh
# Test member addition without actually adding
gh team-kit member add my-team username --read-only

# Verify import operations without applying changes
gh team-kit --read-only import config.yaml
```

### Shell Completion Commands

#### Generate shell completion script

```sh
gh team-kit completion -s <shell>
```

Generate shell completion patch for gh-team-kit. This patches gh's existing completion system to support the extension.

**Flags:**

- `-s, --shell string`: Shell type: {bash|zsh|fish|powershell}

**Prerequisites:** gh completion must be configured first. See [gh completion documentation](https://cli.github.com/manual/gh_completion).

**Examples:**

```sh
# Bash
eval "$(gh team-kit completion -s bash)"

# PowerShell (load gh completion first)
Invoke-Expression $(gh completion -s powershell | Out-String)
Invoke-Expression $(gh team-kit completion -s powershell | Out-String)
```

For detailed setup instructions, see the [Shell Completion guide in go-gh-extension](https://github.com/srz-zumix/go-gh-extension/blob/main/docs/shell-completion.md).

### Team Management

#### Create a new team

```sh
gh team-kit create <name> --description <description> --parent <parent-team-slug>
```

Create a new team in the specified organization with various options such as description, privacy, and notification settings.

#### Delete a team

```sh
gh team-kit delete <team-slug>
```

Delete a specified team from the organization. Ensure that the team is no longer needed as this action is irreversible.

#### Display a team hierarchy in a tree structure

```sh
gh team-kit tree [team-slug]
```

Display a team hierarchy in a tree structure based on the team's slug.

#### Get a team by its slug

```sh
gh team-kit get [team-slug...]
```

Retrieve details of a team using the team's slug.

#### List all teams in the organization

```sh
gh team-kit list [owner]
```

Retrieve and display a list of all teams in the specified organization. You can optionally filter the results by repository.

#### Move a team to a new parent

```sh
gh team-kit move <team-slug> [new-parent-slug]
```

Change the parent of an existing team in the specified organization to a new parent team. If no new parent is specified, the team will be moved to the root level.

#### Rename an existing team

```sh
gh team-kit rename <team-slug> <new-name>
```

Rename an existing team in the specified organization to a new name.

#### Update a team

```sh
gh team-kit update <team-slug> --description <new-description> --parent <parent-team-slug>
```

Update the details of an existing team in the specified organization, such as its description or settings.

### Member Management

#### Add a member to a team

```sh
gh team-kit member add <team-slug> <username...> [role]
```

Add a specified user to the specified team in the organization. Optionally specify the role (default: member).

#### Check if a user is a member of a team

```sh
gh team-kit member check <team-slug> <username>
```

Check if a user is a member of a team.

#### Change the role of a user in a team

```sh
gh team-kit member role <team-slug> <username> <role>
```

Change the role of a specified user in the specified team. Valid roles are: `member`, `maintainer`.

#### List members of a team

```sh
gh team-kit member list <team-slug>
```

List all members of the specified team in the organization.

#### Randomly pick members from a team

```sh
gh team-kit member pick <team-slug> [count]
```

Randomly select a specified number of members from the team. If count is 0 (default), all members are returned. If count is negative, it picks (total members - |count|) members.

#### Perform set operations on two teams members

```sh
gh team-kit member sets <[[HOST/]OWNER/]team-slug1> <|,&,-,^> <[[HOST/]OWNER/]team-slug2>
```

Perform set operations on the members of two teams. The operation can be union (`|`), intersection (`&`), difference (`-`), or symmetric difference (`^`).

#### Sync members from one team to another

```sh
gh team-kit member sync <[[HOST/]OWNER/]src-team-slug> <[[HOST/]OWNER/]dst-team-slug>
```

Sync members from the source team to the destination team. Members in the source team will be added to the destination team, and members not in the source team will be removed from the destination team.

#### Copy members from one team to another

```sh
gh team-kit member copy <[[HOST/]OWNER/]src-team-slug> <[[HOST/]OWNER/]dst-team-slug>
```

Copy members from the source team to the destination team. Members in the source team will be added to the destination team, but no members will be removed from the destination team.

#### Remove a member from a team

```sh
gh team-kit member remove <team-slug> <username...>
```

Remove a specified user from the specified team in the organization.

### Repository Management

#### Add a repository to a team

```sh
gh team-kit repo add <team-slug> <permission>
```

Add a specified repository to the specified team in the organization.

#### Check team permissions for a repository

```sh
gh team-kit repo check <team-slug>
```

Checks whether a team has admin, push, maintain, triage, pull, or none permission for a repository.

#### Compare repositories between two teams

```sh
gh team-kit diff <team-slug1> <team-slug2> [repository...]
```

Compare the repositories associated with two teams and display the differences.

#### Compare team permissions between two repositories

```sh
gh team-kit repo diff <repo1> <repo2> [team-slug...]
```

Compare the team permissions between two repositories and display the differences.

#### Copy teams and permissions to multiple destination repos

```sh
gh team-kit repo copy <dst-repository...>
```

Copy teams and permissions from a source repository to multiple destination repositories.

#### List repositories for a team

```sh
gh team-kit repo list <team-slug>
```

List all repositories for the specified team in the organization.

#### Remove a repository from a team

```sh
gh team-kit repo remove <team-slug>
```

Remove a specified repository from the specified team in the organization.

#### Sync teams and permissions to multiple destination repos

```sh
gh team-kit repo sync <dst-repository...>
```

Synchronize teams and permissions from a source repository to multiple destination repositories.

### User Management

#### Add a user to the organization

```sh
gh team-kit user add <username>
```

Add a specified user to the organization.

#### Check the role of a user in the organization

```sh
gh team-kit user check <username>
```

Check the role of a specified user in the organization.

#### Import users into the organization

```sh
gh team-kit user import <input> [--owner <[HOST/]OWNER>] [--role <member|admin>] [--usermap <file>] [--dryrun] [--ignore-errors]
```

Read a JSON list of users (as produced by `user list --format json`) and add each user to the organization.
Each entry must have a `login` field. The role is taken from the `role_name` field if present; otherwise `--role` is used as the default (`member`).
When `--usermap` is specified, source logins are resolved using the mapping file (as produced by `user map`). The `src` field supports regular expressions and `dst` may contain `$N` or `${name}` capture-group references.
Specify `-` as `<input>` to read from stdin.

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |
| `--role <member\|admin>` | `member` | Default role when not specified in input |
| `--usermap <file>` | — | User mapping file for login conversion during import |
| `--dryrun`, `-n` | `false` | Dry run: show count without applying changes |
| `--ignore-errors` | `false` | Continue without exiting on error during import |

#### Create a user mapping file between source and target organizations

```sh
gh team-kit user map <target> [--owner <[HOST/]OWNER>] [--output <file>] [--all] [--no-suspended] [--format <json|yaml>]
```

Generate a YAML mapping file that correlates users by their public email between a source organization (`--owner`) and a target organization (positional argument).
Both `--owner` and `<target>` accept the `[HOST/]OWNER` format to specify a host.
This mapping can be used with `user import --usermap` or `import --usermap` to automatically convert source logins to target logins.
The `src` field in the output supports regular expressions and `dst` may contain `$N` or `${name}` capture-group references when hand-edited.
`--output` and `--format` are mutually exclusive.

Mapping file format:

```yaml
users:
  - src: user1
    dst: target_user1
    email: user1@example.com
```

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Source organization |
| `--output <file>`, `-o` | — | Write mapping YAML to file; if omitted, output goes to stdout using `--format` (default: table) |
| `--all`, `-a` | `false` | Include source users with no email match (dst will be empty) |
| `--emu` | `false` | Compact matched pairs sharing the same base login into a single regex entry. Supports: both logins have a slug (`alice_corp`→`alice_new` ⇒ `(.+)_corp`→`$1_new`), src-only slug (`alice_corp`→`alice` ⇒ `(.+)_corp`→`$1`), dst-only slug (`alice`→`alice_new` ⇒ `(.+)`→`$1_new`). Pairs whose base names differ or have an empty dst are kept as exact entries. |
| `--no-suspended` | `false` | Exclude suspended users from source and target before matching |
| `--quiet` | `false` | Suppress warnings for source users with no matching target user |
| `--format <json\|yaml>` | — | Output format: `json` or `yaml` (mutually exclusive with `--output`) |

#### List all users in the organization

```sh
gh team-kit user list
```

Retrieve and display a list of all users in the organization.

#### List user repositories

```sh
gh team-kit user repos [username]
```

Retrieve and display a list of repositories that a specified user has access to, including their roles and permissions.

#### Remove a user from the organization

```sh
gh team-kit user remove <username>
```

Remove a specified user from the organization.

#### Change the role of a user in a orgnization

```sh
gh team-kit user role <username> <role>
```

Change the role of a specified user in the organization. Valid roles include `member` and `admin`.

#### Search GitHub users

```sh
gh team-kit user search <query>
```

Search for GitHub users by query string.

#### Get contextual hovercard information for a user

```sh
# Basic hovercard
gh team-kit user hovercard get <username>

# Issue hovercard
gh team-kit user hovercard issue <username> <issue-number>

# Organization hovercard
gh team-kit user hovercard org <username> <org>

# Pull request hovercard
gh team-kit user hovercard pr <username> <pr-number>

# Repository hovercard
gh team-kit user hovercard repo <username> <repo>
```

Retrieve contextual hovercard information for a user. Use each subcommand for different contexts (basic, issue, organization, pull request, repository).

#### Add a user as a collaborator to a repository

```sh
gh team-kit repo user add <username> <permission>
```

Add a specified user as a collaborator to a repository with a given permission (`admin`, `maintain`, `push`, `triage`, `pull`).

#### Check user permissions for a repository

```sh
gh team-kit repo user check <username>
```

Check the permissions of a specified user for a repository.

#### Copy direct user permissions to multiple destination repos

```sh
gh team-kit repo user copy [-R <[HOST/]OWNER/REPO>] [--dst-host <host>] [-f] <dst-repository...>
```

Copy direct user collaborator permissions from the source repository to multiple destination repositories. Only direct collaborators (not team-inherited) are copied. If a user already has a different permission on the destination, the command fails unless `--force` (or `-f`) is specified.

| Flag | Default | Description |
| --- | --- | --- |
| `-R, --repo <[HOST/]OWNER/REPO>` | (current repo) | Source repository |
| `--dst-host <host>` | (same as source) | Destination host for cross-host copy |
| `-f, --force` | `false` | Overwrite existing permissions if they differ |

#### List users with access to a repository

```sh
gh team-kit repo user list
```

List all collaborators for the specified repository. You can filter the results by affiliation and role.

#### Remove a user's access to a repository

```sh
gh team-kit repo user remove <username>
```

Remove a specified user's access to a repository.

#### Sync direct user permissions to multiple destination repos

```sh
gh team-kit repo user sync [-R <[HOST/]OWNER/REPO>] [--dst-host <host>] <dst-repository...>
```

Synchronize direct user collaborator permissions from the source repository to multiple destination repositories. Users are added or updated to match the source, and users not present in the source are removed from the destination.

| Flag | Default | Description |
| --- | --- | --- |
| `-R, --repo <[HOST/]OWNER/REPO>` | (current repo) | Source repository |
| `--dst-host <host>` | (same as source) | Destination host for cross-host sync |

### Member Privileges Management

#### Get or set the default repository permission (base permissions)

```sh
gh team-kit member-privilege base-permissions [--set <read|write|admin|none>] [--owner <[HOST/]OWNER>]
```

Get or set the default repository permission for organization members. When `--set` is specified, the setting is updated and the result is displayed; otherwise the current value is displayed.

| Flag | Default | Description |
| --- | --- | --- |
| `--set <read\|write\|admin\|none>` | — | New value to set. Omit to get the current value. |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |

#### Get or set whether members can create teams

```sh
gh team-kit member-privilege can-create-teams [--set[=false]] [--owner <[HOST/]OWNER>]
```

Get or set whether organization members can create teams. When `--set` is specified, the setting is updated and the result is displayed; otherwise the current value is displayed.

| Flag | Default | Description |
| --- | --- | --- |
| `--set` | — | Allow members to create teams. Use `--set=false` to disallow. Omit to get the current value. |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |

#### Copy member privileges from one organization to another

```sh
gh team-kit member-privilege copy <src-owner> <dst-owner>
```

Copy all member privileges settings from the source organization to the destination organization.

#### Get member privileges of an organization

```sh
gh team-kit member-privilege get [--owner <[HOST/]OWNER>] [--field <fields>]
```

Get the member privileges settings of the specified organization. Displays a table with fields such as default repository permission, repository creation, forking, pages, team creation, and web commit signoff settings.

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |
| `--field <fields>` | (all fields) | Comma-separated list of fields to display |

#### Set member privileges of an organization

```sh
gh team-kit member-privilege set [flags]
```

Update one or more member privileges settings of the specified organization. Only the flags explicitly specified will be changed.

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |
| `--default-repo-permission <read\|write\|admin\|none>` | — | Default repository permission for organization members |
| `--members-can-create-repos` / `--no-members-can-create-repos` | — | Allow or disallow members to create repositories |
| `--members-can-create-public-repos` / `--no-members-can-create-public-repos` | — | Allow or disallow members to create public repositories |
| `--members-can-create-private-repos` / `--no-members-can-create-private-repos` | — | Allow or disallow members to create private repositories |
| `--members-can-create-internal-repos` / `--no-members-can-create-internal-repos` | — | Allow or disallow members to create internal repositories |
| `--members-can-fork-private-repos` / `--no-members-can-fork-private-repos` | — | Allow or disallow members to fork private repositories |
| `--members-can-create-pages` / `--no-members-can-create-pages` | — | Allow or disallow members to create GitHub Pages sites |
| `--members-can-create-public-pages` / `--no-members-can-create-public-pages` | — | Allow or disallow members to create public GitHub Pages sites |
| `--members-can-create-private-pages` / `--no-members-can-create-private-pages` | — | Allow or disallow members to create private GitHub Pages sites |
| `--members-can-create-teams` / `--no-members-can-create-teams` | — | Allow or disallow members to create teams |
| `--web-commit-signoff-required` / `--no-web-commit-signoff-required` | — | Require or disable web commit signoff |

### Organization-Role Management

#### List teams assigned to an organization role

```sh
gh team-kit org-role team list <org-role>
```

Retrieve and display a list of all teams assigned to a specific role in the organization.

#### Add a team to an organization role

```sh
gh team-kit org-role team add <team-slug> <org-role>
```

Add a specified team to the specified role in the organization.

#### Remove a team from an organization role

```sh
gh team-kit org-role team remove <team-slug> <org-role>
```

Remove a specified team from the specified role in the organization.

#### Import custom organization roles

```sh
gh team-kit org-role import <input> [--owner <[HOST/]OWNER>] [--dryrun]
```

Read a JSON list of custom organization roles (as produced by `org-role list --format json`) and create or update each role in the organization.
Only `Organization`-sourced (user-defined) roles are applied; `Predefined` and `System` roles are skipped.
Specify `-` as `<input>` to read from stdin.

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |
| `--dryrun`, `-n` | `false` | Dry run: show count without applying changes |

#### List organization roles

```sh
gh team-kit org-role list [owner] [--source <Organization|Predefined|System>] [--field <field>] [--name-only] [--format json]
```

List all roles available in the organization. Use `--source` to filter by role source (can be specified multiple times). Use `--field` to select columns to display (valid fields: `ID`, `NAME`, `DESCRIPTION`, `BASE_ROLE`, `SOURCE`, `PERMISSIONS`, `CREATED_AT`, `UPDATED_AT`). Use `--name-only` to output only role names.

#### List users assigned to an organization role

```sh
gh team-kit org-role user list [org-role-name]
```

Retrieve and display a list of all users assigned to a specific role in the organization. Supports options for detailed information, suspended users, and filtering by owner.

#### Add a user to an organization role

```sh
gh team-kit org-role user add <username> <org-role>
```

Assign a specified user to the specified role in the organization.

#### Remove a user from an organization role

```sh
gh team-kit org-role user remove <username> <org-role>
```

Remove a specified user from the specified role in the organization.

### Mannequin Management

#### List mannequins in the organization

```sh
gh team-kit mannequin list [owner] [--name-only] [--format json] [--jq <expression>] [--template <string>]
```

List all mannequins (placeholder accounts for unclaimed users) in the specified organization. Use `--name-only` to output only login names. Use `--format json` with `--jq` or `--template` to customize JSON output.

#### Bulk-migrate mannequins using a user mapping file

```sh
gh team-kit mannequin migrate --usermap <file> [--owner <[HOST/]OWNER>] [--skip-invitation] [--force] [--dryrun] [--src <[HOST/]OWNER>] [--no-suspended]
```

List all mannequins in the organization and reattribute each one to its mapped target user.
The mapping file (`--usermap`) must be a YAML file as produced by `user map`. Each mannequin is matched first by src login (supports regular expressions), then by email.
Mannequins already claimed are skipped unless `--force` is specified. Entries whose dst is empty are skipped. Bot mannequins (login ending with `[bot]`) are skipped because they cannot be reclaimed.
With `--src`, mannequins that are not members of the source organization are skipped without error when their mapped target user does not exist.
With `--no-suspended`, `--src` is required; mannequins whose login is a suspended member of the source organization are skipped.

| Flag | Default | Description |
| --- | --- | --- |
| `--usermap <file>` | — | User mapping file (required) |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Target organization |
| `--skip-invitation` | `false` | Skip invitation and directly reclaim (requires GitHub Support enablement) |
| `--force` | `false` | Process mannequins that are already claimed |
| `--dryrun`, `-n` | `false` | Show what would be done without making changes |
| `--no-suspended` | `false` | Skip mannequins whose login is a suspended member of `--src` |
| `--src <[HOST/]OWNER>` | — | Source organization used for membership and suspension checks; required with `--no-suspended` |

#### Reattribute a mannequin by email

```sh
gh team-kit mannequin reattribute-by-email <email> [--owner <[HOST/]OWNER>] [--repo [HOST/]OWNER/REPO] [--usermap <file>] [--skip-invitation] [--force]
```

Find the mannequin and target user by email, then send an attribution invitation.
Without `--usermap`, both are resolved by searching their email within the target organization.
With `--usermap`, the mannequin login (`src`) and target user login (`dst`) are read directly from the mapping file using the email as key.

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Target organization |
| `--repo [HOST/]OWNER/REPO`, `-R` | — | Target repository (alternative to `--owner`) |
| `--usermap <file>` | — | Resolve logins from mapping file instead of searching by email |
| `--skip-invitation` | `false` | Skip invitation and directly reclaim (requires GitHub Support enablement) |
| `--force` | `false` | Proceed even if mannequin is already claimed |

#### Reattribute a mannequin to a user

```sh
gh team-kit mannequin reattribute <mannequin-login> <target-user-login> [--owner <[HOST/]OWNER>] [--skip-invitation] [--force]
```

Send an attribution invitation to a user to claim the specified mannequin. The target user must be a member of the organization.
Use `--skip-invitation` to skip the invitation step and directly reclaim the mannequin (requires the feature to be enabled by GitHub Support).

| Flag | Default | Description |
| --- | --- | --- |
| `--owner <[HOST/]OWNER>` | (current repo owner) | Organization ([HOST/]OWNER) |
| `--skip-invitation` | `false` | Skip invitation and directly reclaim (requires GitHub Support enablement) |
| `--force` | `false` | Proceed even if mannequin is already claimed |

### IDP Management

#### Connect an external group to a team

```sh
gh team-kit idp emu set <group-name> <team-slug> [--owner <[HOST/]OWNER>] [--field <field>] [--format <json|table>]
```

Connect an external group to a team in the organization (Enterprise Managed Users). Resolves the group by name and connects it to the specified team. Use `--field` to display specific fields (`ID`, `NAME`, `UPDATED_AT`, `TEAM_COUNT`, `MEMBER_COUNT`). On success, prints a confirmation message unless `--field` or `--format` is used.

#### Find the external group connected to a team

```sh
gh team-kit idp emu find <team-slug> [--owner <[HOST/]OWNER>] [--field <field>] [--format <json|table>]
```

Find the external group connected to a team in the organization (Enterprise Managed Users). Exits with no output if no external group is connected (e.g. the team has explicit members). Available fields: `ID`, `NAME`, `UPDATED_AT`, `TEAM_COUNT`, `MEMBER_COUNT`.

#### Get an external group

```sh
gh team-kit idp emu get <group-name> [--owner <[HOST/]OWNER>] [--field <field>] [--format <json|table>]
```

Get details of a single external group by name in the organization (Enterprise Managed Users). Available fields: `ID`, `NAME`, `UPDATED_AT`, `TEAM_COUNT`, `MEMBER_COUNT`.

#### List external groups in the organization or connected to a team

```sh
gh team-kit idp emu list [team-slug] [--owner <[HOST/]OWNER>] [--query <name-filter>] [--details] [--field <field>] [--format <json|table>]
```

List all external groups available in the organization, or list external groups connected to the specified team (Enterprise Managed Users). Use `--query` to filter by name. Use `--details` to fetch full details (teams list) for each group by calling the detail API per entry. Available fields: `ID`, `NAME`, `UPDATED_AT`, `TEAM_COUNT`, `TEAMS`.

#### List IDP groups in the organization or connected to a team

```sh
gh team-kit idp list [team-slug] [--owner <[HOST/]OWNER>] [--query <name-filter>] [--field <field>] [--format <json|table>]
```

List all IDP groups available in the organization (SAML team sync), or list IDP groups connected to the specified team. Use `--query` to filter by name when listing all groups (not available when a team slug is provided). Available fields: `ID`, `NAME`, `DESCRIPTION`.

#### List teams connected to an external group

```sh
gh team-kit idp emu teams <group-name> [--owner <[HOST/]OWNER>] [--field <field>] [--format <json|table>]
```

List the teams connected to an external group, with detailed team info fetched from the organization (Enterprise Managed Users). Available fields: `TEAM_ID`, `TEAM_NAME`, `SLUG`, `DESCRIPTION`, `PRIVACY`, `HTML_URL`.

#### Remove the connection between an external group and a team

```sh
gh team-kit idp emu unset <team-slug> [--owner <[HOST/]OWNER>]
```

Remove the connection between an external group and a team in the organization (Enterprise Managed Users). Only the team slug is required.

### Copilot Management

#### Show Copilot metrics for a team

```sh
gh team-kit copilot metrics <team-slug> [--owner <[HOST/]OWNER>] [--since <RFC3339>] [--until <RFC3339>]
```

Display GitHub Copilot usage metrics for the specified team. You can optionally specify the owner with `--owner` and limit the date range with `--since` and `--until`.

### Code Review Management

#### Get code review settings

```sh
gh team-kit code-review get <team-slug> [--owner <[HOST/]OWNER>] [--format <json|table>]
```

Retrieve code review assignment settings for the specified team, including auto-assignment status, algorithm, member count, and notification preferences.

#### Set code review settings

```sh
gh team-kit code-review set <team-slug> [--owner <[HOST/]OWNER>] [--enabled|--disable-enabled] [--algorithm <ROUND_ROBIN|LOAD_BALANCE>] [--member-count <int>] [--notify-team|--disable-notify-team]
```

Update code review assignment settings for the specified team. You can enable/disable auto-assignment, set the assignment algorithm, configure the number of members to assign, and control team notifications.

### Pull Request Analysis

#### Generate a relationship graph from pull request activity

```sh
gh team-kit pr-graph [<[HOST/]OWNER/REPO>...] [--owner <[HOST/]OWNER>] [--state <open|closed|merged|all>] [--since <date>] [--until <date>] [--limit <int>] [--label <name>[,<name>...]] [--exclude-label <name>[,<name>...]] [--base <pattern>] [--head <pattern>] [--exclude-head-branch <pattern>[,<pattern>...]] [--exclude-draft] [--no-bots] [--edge-type <relation>[,<relation>...]] [--exclude-edge-type <relation>[,<relation>...]] [--min-weight <number>] [--keep-orphans] [--depth <int>] [--group-by <pattern>[,<pattern>...]] [--weight-by <occurrences|lines|additions|deletions>] [--half-life <days>] [--co-change] [--co-change-max-files <int>] [--exclude-author <login>[,<login>...]] [--hide-user <login>[,<login>...]] [--exclude-user <login>[,<login>...]] [--allow-user <login>[,<login>...]] [--user-allowlist <file>] [--exclude-file <pattern>[,<pattern>...]] [--exclude-generated] [--exclude-deleted] [--include-file <pattern>[,<pattern>...]] [--format <json|dot|markdown|mermaid>]
```

Analyze pull request activity and generate a graph showing relationships between users, teams, labels, and code areas. The graph contains user, team, label, file, directory, and submodule nodes. Edges represent review, approval, comment, review request, team membership, file change, file creation, co-change, directory containment, CODEOWNERS ownership, and labeling relationships. By default, edge weights count occurrences. Repository arguments are optional: specify one or more repositories, use `--owner` to analyze all repositories of an organization (mutually exclusive with repository arguments), or omit both to use the current repository. When multiple repositories are given they must all be on the same host, and team nodes are namespaced by owner (`owner/slug`) to avoid collisions across organizations. Use `--state` to filter pull requests by state (default: `all`); `merged` selects closed pull requests that were actually merged. Use `--since` and `--until` to limit the analysis to pull requests created in a date range (`YYYY-MM-DD` or RFC 3339); a date-only `--until` is inclusive through the end of that day. Use `--limit` to cap the number of pull requests analyzed per repository, counted after state/date/label/branch/`--exclude-author` filtering (default: `30`, `0` = unlimited). Use `--label` to only include pull requests having at least one of the given labels, and `--exclude-label` to skip pull requests having any of the given labels (both may be repeated or accept comma-separated names, default: none; exclusion takes precedence when both match). Use `--base` and `--head` to only include pull requests whose base or head branch matches a glob pattern, and `--exclude-head-branch` to skip pull requests whose head branch matches any of the given glob patterns (repeatable or comma-separated). Use `--exclude-draft` to skip draft pull requests, and `--no-bots` to automatically exclude and hide users whose login has a `[bot]` suffix (default: `false` for both). Use `--edge-type` to keep only the given edge relation types in the graph, and `--exclude-edge-type` to remove the given relation types (both may be repeated or accept comma-separated relation names, default: none); valid relation names are `approved`, `changes-requested`, `reviewed`, `commented`, `review-commented`, `review-requested`, `member-of`, `changed`, `created`, `co-changed`, `in`, `owned-by`, and `labeled`. The `created` relation is a subset of `changed` recorded when a pull request adds or copies a path, which identifies who introduced a code area rather than who merely touched it; renames are not treated as creation because the content already existed under its previous name. Use `--min-weight` to remove edges below a given weight from the graph (default: `0` = no filter); it accepts fractions because `--half-life` can produce fractional weights, while `--weight-by` changes their scale, so choose the threshold accordingly. Nodes left without any edge after edge filtering are removed from the graph; use `--keep-orphans` to retain them (default: `false`). Use `--group-by` to fold changed file paths using glob-style prefix patterns such as `LocalPackages/*,Assets/*/*` (repeatable or comma-separated, default: none); patterns are evaluated in order and the first match wins. Use `--depth` to fold the remaining paths into their ancestor directory truncated to the given number of path segments (default: `0` = no folding); the two options can be combined, with `--depth` acting as the fallback for paths that match no `--group-by` pattern. A folded path is emitted as a directory node rather than a file node. A changed path registered in the repository's `.gitmodules` is emitted as a submodule node, distinguishing submodule pointer updates from ordinary file changes. Use `--exclude-author` to drop pull requests authored by the given users from analysis entirely, and `--hide-user` to omit a user's non-author activity and CODEOWNERS relationships from the graph without excluding their pull requests. Both options may be repeated or accept comma-separated logins (default: none), and login matching is case-insensitive. `--exclude-user` is deprecated and equivalent to setting both `--exclude-author` and `--hide-user` for the given logins. Use `--exclude-file` to omit files (and their directory/CODEOWNERS relationships) whose path matches a `.gitignore`-style pattern, and `--include-file` to only include files matching such a pattern (default for both: none, meaning no restriction); both options may be repeated or accept comma-separated patterns. Use `--exclude-generated` to additionally omit files marked with the `linguist-generated` attribute in the repository's `.gitattributes` (default: `false`), and `--exclude-deleted` to omit paths that no longer exist on the repository's default branch, so that ownership reflects the code that is still there (default: `false`); the option disables itself with a warning when the repository tree is too large for GitHub to return in full.

Use `--weight-by` to select how a changed path contributes to path-derived edge weights: `occurrences` (default, one per changed path), `lines` (additions plus deletions), `additions`, or `deletions`. This affects the `changed`, `created`, `in`, `owned-by`, and `co-changed` relations only; reviews, comments, labels, and team membership have no line count and always count occurrences. Use `--half-life` to decay every contribution by the age of its pull request, halving it every given number of days (default: `0` = no decay), so that recent work outweighs old work; ages are measured against `--until` when set, which keeps repeated runs reproducible, and against the current time otherwise. Use `--co-change` to add `co-changed` edges between paths modified by the same pull request, revealing code areas that move together (default: `false`); the relation is undirected and each pair is emitted once, with the pair weight being the smaller of the two contributions so that a single sweeping change does not dominate every pair it touches. Because the pair count grows quadratically, `--co-change-max-files` skips pull requests touching more than the given number of distinct paths (default: `50`, `0` = unlimited). Use `--allow-user` and `--user-allowlist` to restrict the graph to a known set of members, such as the current roster of a team: unlike `--hide-user`, a user outside the allowlist is dropped even as a pull request author, while their pull requests still contribute file, directory, CODEOWNERS, and co-change edges. `--allow-user` may be repeated or accept comma-separated logins, `--user-allowlist` reads logins from a file with one login per line (blank lines, `#` comments, and a leading `@` are ignored), and the two may be combined; matching is case-insensitive and an allowlist that resolves to no login is an error. Use the global `--http-timeout` flag to raise the timeout of each GitHub API request (default: `30s`) when analyzing repositories with a very large pull request history. The output format defaults to `mermaid`; `dot`, `markdown` (Mermaid in a code fence), and `json` (node/edge data) are also supported.

##### Code ownership analysis

Combining `--weight-by lines`, `--half-life`, `created` edges (via `--edge-type created`), `--exclude-deleted`, and an allowlist makes `pr-graph`
usable as an ownership signal: who introduced a code area, who keeps changing it recently, and how large those changes
were. Two limits are worth knowing before relying on it.

The graph is built entirely from pull request activity, so anything that never went through a pull request — direct
pushes to the default branch, imported history, or a repository migration — is invisible. The `created` relation is
likewise scoped to the analyzed pull requests: a file added before `--since` has no `created` edge at all, so widen the
range when you need first authorship rather than recent activity.

Edge weights measure the volume of change, not the amount of surviving code. A file rewritten by someone else after
the analyzed window still credits its original authors, and a large but long-reverted change still carries its full
weight. `pr-graph` cannot answer "who wrote the lines that are in the file today" — that requires line-level blame,
which is not available through the pull request APIs.

### Configuration Management

#### Export team information

```sh
gh team-kit export [--output <file>] [--owner <[HOST/]OWNER>] [--host <host>] [--no-export-repositories] [--no-export-group] [--no-export-org-roles] [--no-suspended] [--format <json|table>]
```

Retrieve and display team information from the specified organization. Exports team structure, members, maintainers, repositories, external group connections (EMU), custom organization role assignments, and code review settings to a file or stdout. Use `--output` to specify the output file (default: stdout). Use `--no-export-repositories` to skip repository permissions, `--no-export-group` to skip external group connections, `--no-export-org-roles` to skip custom org role assignments, and `--no-suspended` to exclude suspended users.

#### Import team information

```sh
gh team-kit import <input> [--dryrun] [--verify] [--owner <[HOST/]OWNER>] [--host <host>] [--usermap <file>] [--no-remove-extra-members] [--ignore-errors] [--format <json|yaml>]
```

Read and apply team information to the specified organization from a file or stdin.
`<input>`: Path to a YAML file containing team configuration (as produced by `export`), or `-` to read from standard input.
Use `--dryrun` to preview changes without applying them. When `--usermap` is specified, source logins are resolved using the mapping file (as produced by `user map`). The `src` field supports regular expressions and `dst` may contain `$N` or `${name}` capture-group references. If the input contains a `group` field for a team, the corresponding external group is connected automatically (EMU only; only applicable to leaf teams without parent/child teams). When the organization supports external groups and a team has no `group` specified, any existing external group connection is removed. If a team has an `org_roles` field, the listed custom organization roles are assigned to that team on import. By default, team members not present in the imported configuration are removed; use `--no-remove-extra-members` to skip this removal. Use `--ignore-errors` to continue without exiting on error during import. See [docs/migrate.md](docs/migrate.md) for migration examples.

### Skills Management

> **Note:** The `skills` subcommands use `--dry-run` (with a hyphen), which differs from other `gh team-kit` commands that use `--dryrun` (no hyphen). This is intentional: `skills` is powered by the upstream [skillsmith](https://github.com/Songmu/skillsmith) library, which defines its own flag names.

#### Install agent skills

```sh
gh team-kit skills install [--scope <user|repo>] [--prefix <dir>] [--dry-run] [--force]
```

Install embedded agent skills into the skills directory. By default installs to `~/.agents/skills` (user scope). Use `--scope repo` to install to the current repository's `.agents/skills` directory. Use `--prefix` to specify a custom directory. Use `--dry-run` to preview changes without applying them. Use `--force` to overwrite unmanaged skills.

| Flag | Default | Description |
| --- | --- | --- |
| `--scope <user\|repo>` | `user` | Installation scope: `user` (`~/.agents/skills`) or `repo` (repository root) |
| `--prefix <dir>` | — | Override the installation directory (ignores `--scope`) |
| `--dry-run` | `false` | Preview changes without applying them (note: uses `--dry-run`, not `--dryrun`) |
| `--force` | `false` | Overwrite unmanaged skills |

#### List embedded agent skills

```sh
gh team-kit skills list
```

List all agent skills bundled with this tool.

#### Reinstall agent skills

```sh
gh team-kit skills reinstall [--scope <user|repo>] [--prefix <dir>] [--dry-run] [--force]
```

Reinstall all managed skills regardless of version.

#### Show agent skills status

```sh
gh team-kit skills status [--scope <user|repo>] [--prefix <dir>]
```

Show the installation status of each skill, including installed and available versions.

#### Uninstall agent skills

```sh
gh team-kit skills uninstall [--scope <user|repo>] [--prefix <dir>] [--dry-run]
```

Remove all managed skills installed by this tool.

#### Update agent skills

```sh
gh team-kit skills update [--scope <user|repo>] [--prefix <dir>] [--dry-run] [--force]
```

Update installed skills when a newer version is available.

## Copilot CLI Canvas Extension

This repository ships a [GitHub Copilot CLI](https://github.com/github/copilot-cli) canvas extension in
[`.github/extensions/pr-graph-dashboard`](.github/extensions/pr-graph-dashboard) that renders `pr-graph` DOT output as an
interactive graph in the Copilot app side panel.

The extension is discovered automatically when the repository is opened in the Copilot app, so no installation step is
required. Rendering uses the local Graphviz `dot` binary, which must be installed separately (for example
`brew install graphviz`).

The dashboard can load an existing `.dot` file or generate one by running `gh team-kit pr-graph <args> --format dot`, and it
supports filtering by node type, edge relation, edge weight, free-text search, and a focus node with a configurable hop
radius. It can also hand the current graph context back to the agent, either through preset prompts or a free-text box, and
exposes the same operations as agent-callable canvas actions. See the
[extension README](.github/extensions/pr-graph-dashboard/README.md) for details.
