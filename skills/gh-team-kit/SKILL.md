---
name: gh-team-kit
description: gh-team-kit GitHub CLI extension for managing GitHub Organization teams, members, repositories, org roles, member privileges, IDP/EMU groups, and Copilot metrics. Use when performing team membership operations, syncing teams, managing org roles, comparing repository permissions, exporting/importing team configurations, or handling Enterprise Managed Users (EMU).
license: MIT
compatibility:
  - Requires gh CLI (https://cli.github.com) with gh-team-kit extension installed (`gh extension install srz-zumix/gh-team-kit`)
---

# gh-team-kit

`gh-team-kit` is a GitHub CLI extension for team-related operations in GitHub Organizations.

## Prerequisites

```bash
# Install gh CLI
brew install gh          # macOS
# or: https://cli.github.com/

# Install gh-team-kit extension
gh extension install srz-zumix/gh-team-kit

# Authenticate
gh auth login

# Verify
gh team-kit --version
```

## Global Options

| Flag | Description |
| --- | --- |
| `--owner [HOST/]OWNER` | Organization to operate on (defaults to repo owner) |
| `-R`, `--repo owner/repo` | Filter by or operate on a specific repository |
| `--read-only` | Prevent any write operations |
| `-L`, `--log-level` | Log level (debug, info, warn, error) |
| `--jq <expr>` | Filter JSON output with jq |
| `--json <fields>` | Output JSON with specified fields |
| `--template <string>` | Format output with Go template |

---

## Team Commands

### `list` (alias: `ls`)

```bash
# List all teams in the current organization
gh team-kit list

# List teams for a specific owner
gh team-kit list myorg

# Filter teams by repository
gh team-kit list --repo owner/repo

# Output only team names
gh team-kit list --name-only

# JSON output
gh team-kit list --json name,slug,description
```

### `get` (alias: `view`)

```bash
# Get a team by slug
gh team-kit get <team-slug>

# Get multiple teams
gh team-kit get team-a team-b

# Include parent team info
gh team-kit get <team-slug> --child

# Get recursively
gh team-kit get <team-slug> --recursive

# Specify owner
gh team-kit get <team-slug> --owner myorg
```

### `create`

```bash
# Create a team
gh team-kit create <name>

# Create with description
gh team-kit create <name> --description "My team"

# Create with parent team
gh team-kit create <name> --parent parent-team-slug

# Create with secret privacy
gh team-kit create <name> --privacy secret

# Disable notifications
gh team-kit create <name> --disable-notification
```

### `update`

```bash
# Update team description
gh team-kit update <team-slug> --description "New description"

# Rename via update
gh team-kit update <team-slug> --name "New Name"

# Change parent
gh team-kit update <team-slug> --parent new-parent-slug

# Change privacy
gh team-kit update <team-slug> --privacy secret

# Update notification setting
gh team-kit update <team-slug> --notification disabled
```

### `delete` (alias: `del`)

```bash
# Delete a team
gh team-kit delete <team-slug>

# Delete a team with child teams
gh team-kit delete <team-slug> --with-child

# Skip member/repo count checks
gh team-kit delete <team-slug> --force
```

### `rename` (alias: `rn`)

```bash
gh team-kit rename <team-slug> <new-name>
```

### `move` (alias: `mv`)

```bash
# Move team under a new parent
gh team-kit move <team-slug> <new-parent-slug>

# Move team to root level
gh team-kit move <team-slug>
```

### `diff`

```bash
# Compare repositories between two teams
gh team-kit diff <team-slug1> <team-slug2>

# Filter by specific repositories
gh team-kit diff <team-slug1> <team-slug2> repo1 repo2

# Return exit code 1 if differences exist
gh team-kit diff <team-slug1> <team-slug2> --exit-code

# Control color output
gh team-kit diff <team-slug1> <team-slug2> --color always
```

### `tree`

```bash
# Display full team hierarchy
gh team-kit tree

# Display hierarchy from a specific root team
gh team-kit tree <team-slug>

# Retrieve recursively
gh team-kit tree --recursive
```

### `export`

```bash
# Export team configuration
gh team-kit export

# Export to file
gh team-kit export --output teams.yaml

# Exclude repositories from export
gh team-kit export --no-export-repositories

# Exclude external group connections
gh team-kit export --no-export-group

# Exclude custom org roles
gh team-kit export --no-export-org-roles

# Exclude suspended users
gh team-kit export --no-suspended
```

### `import`

```bash
# Import team configuration from file
gh team-kit import teams.yaml

# Import from stdin
cat teams.yaml | gh team-kit import -

# Dry run (verify without applying)
gh team-kit import teams.yaml --dryrun

# Verify configuration before applying
gh team-kit import teams.yaml --verify

# Use user mapping file
gh team-kit import teams.yaml --usermap usermap.yaml

# Skip removing extra members not in config
gh team-kit import teams.yaml --no-remove-extra-members
```

---

## `member` — Manage Team Members

### `member add`

```bash
# Add a user to a team
gh team-kit member add <team-slug> <username>

# Add multiple users
gh team-kit member add <team-slug> user1 user2

# Add as maintainer
gh team-kit member add <team-slug> <username> --role maintainer

# Allow non-org member
gh team-kit member add <team-slug> <username> --allow-non-organization-member
```

### `member remove` (alias: `rm`)

```bash
# Remove a user from a team
gh team-kit member remove <team-slug> <username>

# Remove multiple users
gh team-kit member remove <team-slug> user1 user2
```

### `member list` (alias: `ls`)

```bash
# List team members
gh team-kit member list <team-slug>

# Include detailed info
gh team-kit member list <team-slug> --details

# Output only names
gh team-kit member list <team-slug> --name-only

# Filter by role
gh team-kit member list <team-slug> --role maintainer

# Exclude suspended users
gh team-kit member list <team-slug> --no-suspended

# List only suspended users
gh team-kit member list <team-slug> --suspended
```

### `member check`

```bash
# Check if a user is a team member
gh team-kit member check <team-slug> <username>

# Return exit code 1 if not a member
gh team-kit member check <team-slug> <username> --exit-code
```

### `member role`

```bash
# Change a user's role in a team
gh team-kit member role <team-slug> <username> member
gh team-kit member role <team-slug> <username> maintainer
```

### `member copy`

```bash
# Copy members from source team to destination (add only)
gh team-kit member copy <src-team-slug> <dst-team-slug>

# Cross-org copy
gh team-kit member copy src-org/src-team dst-org/dst-team
```

### `member sync`

```bash
# Sync members from source to destination (add and remove)
gh team-kit member sync <src-team-slug> <dst-team-slug>

# Cross-org sync
gh team-kit member sync src-org/src-team dst-org/dst-team
```

### `member sets`

```bash
# Union of two teams' members
gh team-kit member sets <team-slug1> '|' <team-slug2>

# Intersection
gh team-kit member sets <team-slug1> '&' <team-slug2>

# Difference (in team1 but not team2)
gh team-kit member sets <team-slug1> '-' <team-slug2>

# Symmetric difference
gh team-kit member sets <team-slug1> '^' <team-slug2>

# Union of all teams
gh team-kit member sets @any '|' <team-slug>

# All org members minus team members
gh team-kit member sets @all '-' <team-slug>
```

### `member only`

```bash
# List members who belong only to this team (not any other)
gh team-kit member only <team-slug>

# With details
gh team-kit member only <team-slug> --details
```

### `member pick`

```bash
# Randomly pick 3 members from a team
gh team-kit member pick <team-slug> 3

# Pick all members (random order)
gh team-kit member pick <team-slug>

# Exclude specific members
gh team-kit member pick <team-slug> 3 --exclude user1,user2

# Pick all except 2
gh team-kit member pick <team-slug> -2
```

---

## `repo` — Manage Team Repository Access

### `repo add`

```bash
# Add a repository to a team with a permission
gh team-kit repo add <team-slug> push --repo owner/repo
gh team-kit repo add <team-slug> pull --repo owner/repo
gh team-kit repo add <team-slug> admin --repo owner/repo
gh team-kit repo add <team-slug> maintain --repo owner/repo
gh team-kit repo add <team-slug> triage --repo owner/repo
```

### `repo remove` (alias: `rm`)

```bash
gh team-kit repo remove <team-slug> --repo owner/repo
```

### `repo list` (alias: `ls`)

```bash
# List repositories for a team
gh team-kit repo list <team-slug>

# Output only names
gh team-kit repo list <team-slug> --name-only

# Filter by permission
gh team-kit repo list <team-slug> --role push

# Disable inherited permissions
gh team-kit repo list <team-slug> --no-inherit
```

### `repo check`

```bash
# Check team's permission for a repository
gh team-kit repo check <team-slug> --repo owner/repo

# Return exit code based on result
gh team-kit repo check <team-slug> --repo owner/repo --exit-code

# Also check submodules
gh team-kit repo check <team-slug> --repo owner/repo --submodules
```

### `repo diff`

```bash
# Compare team permissions between two repositories
gh team-kit repo diff <repo1> <repo2>

# Filter by specific team slugs
gh team-kit repo diff <repo1> <repo2> team-a team-b

# Return exit code 1 if differences exist
gh team-kit repo diff <repo1> <repo2> --exit-code
```

### `repo copy`

```bash
# Copy team permissions from one repo to another
gh team-kit repo copy <dst-repo> --repo owner/src-repo

# Copy to multiple destinations
gh team-kit repo copy dst-repo1 dst-repo2 --repo owner/src-repo

# Force overwrite existing permissions
gh team-kit repo copy <dst-repo> --repo owner/src-repo --force

# Cross-host copy
gh team-kit repo copy <dst-repo> --repo owner/src-repo --dst-host enterprise.internal
```

### `repo sync`

```bash
# Sync team permissions from source to destination
gh team-kit repo sync <dst-repo> --repo owner/src-repo

# Sync to multiple destinations
gh team-kit repo sync dst-repo1 dst-repo2 --repo owner/src-repo
```

### `repo user` — Manage Repository Collaborators

```bash
# Add a collaborator to a repository
gh team-kit repo user add <username> push --repo owner/repo

# Remove a collaborator
gh team-kit repo user remove <username> --repo owner/repo

# List collaborators
gh team-kit repo user list --repo owner/repo

# Filter collaborators by permission
gh team-kit repo user list --repo owner/repo --role push

# Filter by affiliation
gh team-kit repo user list --repo owner/repo --affiliation outside

# Check a user's permission
gh team-kit repo user check <username> --repo owner/repo --exit-code
```

---

## `org-role` — Manage Organization Roles

### `org-role list` (alias: `ls`)

```bash
# List all org roles
gh team-kit org-role list

# Filter by source
gh team-kit org-role list --source Organization

# Output only role names
gh team-kit org-role list --name-only
```

### `org-role import`

```bash
# Import org roles from JSON file
gh team-kit org-role import roles.json

# Dry run
gh team-kit org-role import roles.json --dryrun
```

### `org-role team` — Team-to-Role Assignment

```bash
# Assign a team to an org role
gh team-kit org-role team add <team-slug> <org-role>

# Remove a team from an org role
gh team-kit org-role team remove <team-slug> <org-role>

# List teams assigned to a role
gh team-kit org-role team list <org-role-name>
gh team-kit org-role team list <org-role-name> --name-only
```

### `org-role user` — User-to-Role Assignment

```bash
# Assign a user to an org role
gh team-kit org-role user add <username> <org-role>

# Remove a user from an org role
gh team-kit org-role user remove <username> <org-role>

# List users assigned to a role
gh team-kit org-role user list <org-role-name>
gh team-kit org-role user list <org-role-name> --details
```

---

## `user` — Manage Organization Members

### `user add`

```bash
# Add a user to the organization
gh team-kit user add <username>

# Add as admin
gh team-kit user add <username> --role admin
```

### `user remove` (alias: `rm`)

```bash
gh team-kit user remove <username>
gh team-kit user remove user1 user2
```

### `user list` (alias: `ls`)

```bash
# List org members
gh team-kit user list

# Include details
gh team-kit user list --details

# Filter by role
gh team-kit user list --role admin

# Exclude suspended users
gh team-kit user list --no-suspended
```

### `user check`

```bash
# Check a user's role in the organization
gh team-kit user check <username>

# Return exit code 1 if not a member
gh team-kit user check <username> --exit-code
```

### `user role`

```bash
# Change a user's org role
gh team-kit user role <username> member
gh team-kit user role <username> admin
```

### `user import`

```bash
# Import users from JSON file
gh team-kit user import users.json

# Import from stdin
gh team-kit user list --json login,role_name | gh team-kit user import -

# Dry run
gh team-kit user import users.json --dryrun

# With user mapping
gh team-kit user import users.json --usermap usermap.yaml

# Set default role
gh team-kit user import users.json --role admin
```

### `user map`

```bash
# Generate user mapping between two orgs/hosts
gh team-kit user map <target-org> --owner <source-org>

# Save to file
gh team-kit user map <target-org> --output usermap.yaml

# Include unmatched source users
gh team-kit user map <target-org> --all

# Compact EMU-style regex entries
gh team-kit user map <target-org> --emu
```

### `user search`

```bash
# Search users by query
gh team-kit user search <query>

# Filter by email
gh team-kit user search --email user@example.com
```

### `user teams` (alias: `ls-team`)

```bash
# List teams a user belongs to
gh team-kit user teams <username>

# List my own teams
gh team-kit user teams
```

### `user repos` (alias: `ls-repo`, `repo`)

```bash
# List repositories of a user
gh team-kit user repos <username>

# Filter by permission
gh team-kit user repos <username> --role push

# Filter by visibility
gh team-kit user repos <username> --visibility private

# Exclude archived repos
gh team-kit user repos <username> --no-archived
```

### `user hovercard`

```bash
# Get hovercard for a user in an org context
gh team-kit user hovercard org <username> --owner <org>

# In repo context
gh team-kit user hovercard repo <username> --repo owner/repo

# In issue context
gh team-kit user hovercard issue <username> --repo owner/repo --number 123

# In PR context
gh team-kit user hovercard pr <username> --repo owner/repo --number 456
```

---

## `member-privilege` — Manage Organization Member Privileges

### `member-privilege get` (alias: `view`)

```bash
# Get all member privilege settings
gh team-kit member-privilege get
```

### `member-privilege set`

```bash
# Set default repository permission
gh team-kit member-privilege set --default-repo-permission read

# Allow/disallow members to create repos
gh team-kit member-privilege set --members-can-create-repos
gh team-kit member-privilege set --no-members-can-create-repos

# Allow/disallow creating teams
gh team-kit member-privilege set --members-can-create-teams
gh team-kit member-privilege set --no-members-can-create-teams
```

### `member-privilege base-permissions`

```bash
# Get current base permission
gh team-kit member-privilege base-permissions

# Set base permission
gh team-kit member-privilege base-permissions --set read
gh team-kit member-privilege base-permissions --set write
gh team-kit member-privilege base-permissions --set none
```

### `member-privilege can-create-teams`

```bash
# Get current setting
gh team-kit member-privilege can-create-teams

# Set value
gh team-kit member-privilege can-create-teams --set true
gh team-kit member-privilege can-create-teams --set false
```

### `member-privilege copy`

```bash
# Copy member privilege settings between organizations
gh team-kit member-privilege copy <src-org> <dst-org>
```

---

## `idp` — Manage IDP Group Connections

### `idp list`

```bash
# List all IDP groups in the organization
gh team-kit idp list

# List IDP groups connected to a team
gh team-kit idp list <team-slug>

# Filter by name
gh team-kit idp list --query "my-group"
```

### `idp emu` — Enterprise Managed Users External Groups

```bash
# List all external groups
gh team-kit idp emu list

# List groups connected to a team
gh team-kit idp emu list <team-slug>

# Filter by name
gh team-kit idp emu list --query "my-group"

# Include detailed info
gh team-kit idp emu list --details

# Get a specific external group
gh team-kit idp emu get <group-name>

# Find the group connected to a team
gh team-kit idp emu find <team-slug>

# Connect a group to a team
gh team-kit idp emu set <group-name> <team-slug>

# Disconnect a group from a team
gh team-kit idp emu unset <team-slug>

# List teams connected to a group
gh team-kit idp emu teams <group-name>
```

---

## `copilot` — Copilot Metrics

```bash
# Show Copilot metrics for a team
gh team-kit copilot metrics <team-slug>

# Filter by date range
gh team-kit copilot metrics <team-slug> --since 2025-01-01T00:00:00Z --until 2025-03-31T23:59:59Z

# JSON output
gh team-kit copilot metrics <team-slug> --json
```

---

## `mannequin` — Manage Mannequins

```bash
# List mannequins in the organization
gh team-kit mannequin list

# Output only login names
gh team-kit mannequin list --name-only
```

---

## Common Workflows

### Copy team configuration across organizations

```bash
# Export from source org
gh team-kit export --owner src-org --output teams.yaml

# Generate user mapping
gh team-kit user map dst-org --owner src-org --output usermap.yaml

# Import to destination org
gh team-kit import teams.yaml --owner dst-org --usermap usermap.yaml --dryrun
gh team-kit import teams.yaml --owner dst-org --usermap usermap.yaml
```

### Sync members between teams

```bash
# One-way sync (add only)
gh team-kit member copy src-team dst-team

# Full sync (add and remove)
gh team-kit member sync src-team dst-team
```

### Audit team membership

```bash
# Who is in team-a but not team-b?
gh team-kit member sets team-a '-' team-b

# Who is in the org but not in any team?
gh team-kit member sets @all '-' @any
```

### Replicate repository permissions

```bash
# Copy all team permissions from one repo to another
gh team-kit repo copy owner/new-repo --repo owner/template-repo

# Verify differences before sync
gh team-kit repo diff owner/repo1 owner/repo2 --exit-code
```

## Getting Help

```bash
gh team-kit --help
gh team-kit member --help
gh team-kit member add --help
gh team-kit repo --help
```

## References

- Extension: https://github.com/srz-zumix/gh-team-kit
- GitHub Teams API: https://docs.github.com/en/rest/teams
- GitHub Orgs API: https://docs.github.com/en/rest/orgs
