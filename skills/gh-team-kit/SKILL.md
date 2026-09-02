---
name: gh-team-kit
description: gh-team-kit GitHub CLI extension for managing GitHub Organization teams, members, repositories, org roles, member privileges, IDP/EMU groups, and Copilot metrics. Use when performing team membership operations, syncing teams, managing org roles, comparing repository permissions, exporting/importing team configurations, handling Enterprise Managed Users (EMU), or generating pull request activity relationship graphs.
license: MIT
compatibility:
  - Requires gh CLI (https://cli.github.com) with gh-team-kit extension installed (`gh extension install srz-zumix/gh-team-kit`)
---

# gh-team-kit

`gh-team-kit` is a GitHub CLI extension for team-related operations in GitHub Organizations.

## CLI Structure

```
gh team-kit
├── list                        # List teams in the organization
├── get                         # Get a team by slug
├── create                      # Create a new team
├── update                      # Update an existing team
├── delete                      # Delete a team
├── rename                      # Rename a team
├── move                        # Move a team to a new parent
├── diff                        # Compare repositories between two teams
├── tree                        # Display team hierarchy as a tree
├── export                      # Export team configuration to file
├── import                      # Import team configuration from file
├── member                      # Manage team members
│   ├── add
│   ├── check
│   ├── copy
│   ├── list
│   ├── only
│   ├── pick
│   ├── remove
│   ├── role
│   ├── sets
│   └── sync
├── repo                        # Manage team repository access
│   ├── add
│   ├── check
│   ├── copy
│   ├── diff
│   ├── list
│   ├── remove
│   ├── sync
│   └── user
│       ├── add
│       ├── check
│       ├── list
│       └── remove
├── org-role                    # Manage organization roles
│   ├── import
│   ├── list
│   ├── team
│   │   ├── add
│   │   ├── list
│   │   └── remove
│   └── user
│       ├── add
│       ├── list
│       └── remove
├── user                        # Manage organization members
│   ├── add
│   ├── check
│   ├── hovercard
│   │   ├── get
│   │   ├── issue
│   │   ├── org
│   │   ├── pr
│   │   └── repo
│   ├── import
│   ├── list
│   ├── map
│   ├── remove
│   ├── repos
│   ├── role
│   ├── search
│   └── teams
├── member-privilege            # Manage organization member privileges
│   ├── base-permissions
│   ├── can-create-teams
│   ├── copy
│   ├── get
│   └── set
├── idp                         # Manage IDP group connections
│   ├── list
│   └── emu                     # Enterprise Managed Users external groups
│       ├── find
│       ├── get
│       ├── list
│       ├── set
│       ├── teams
│       └── unset
├── copilot                     # Copilot metrics
│   └── metrics
├── mannequin                   # Manage mannequins
│   ├── list
│   ├── migrate
│   ├── reattribute
│   └── reattribute-by-email
├── code-review                 # Code review assignment settings
│   ├── get
│   └── set
├── pr-graph                    # Generate PR activity relationship graph
└── skills                      # Manage embedded agent skills
    ├── install
    ├── list
    ├── reinstall
    ├── status
    ├── uninstall
    └── update
```

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

## Persistent Global Flags

| Flag | Description |
| --- | --- |
| `--read-only` | Prevent any write operations |
| `-L`, `--log-level` | Log level (debug, info, warn, error) |

Common flags such as `--owner`, `--repo`, `--jq`, `--format`, and `--template` are available only on specific subcommands. Check each subcommand's help for supported options.
JSON output uses `--format json`; `--jq` and `--template` require `--format json`.
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
gh team-kit list --format json
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

`<input>`: Path to a YAML file containing team configuration (as produced by `export`), or `-` to read from standard input.

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

# Continue on errors without exiting
gh team-kit import teams.yaml --ignore-errors
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

# Specify organization explicitly
gh team-kit member list <team-slug> --owner myorg

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

# Copy direct user permissions to another repository
gh team-kit repo user copy owner/dst-repo --repo owner/src-repo

# Copy and overwrite existing permissions
gh team-kit repo user copy owner/dst-repo --repo owner/src-repo --force

# Copy across hosts
gh team-kit repo user copy owner/dst-repo --repo owner/src-repo --dst-host ghe.example.com

# Sync direct user permissions to another repository (adds, updates, and removes)
gh team-kit repo user sync owner/dst-repo --repo owner/src-repo

# Sync across hosts
gh team-kit repo user sync owner/dst-repo --repo owner/src-repo --dst-host ghe.example.com
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
gh team-kit user list --format json | gh team-kit user import -

# Dry run
gh team-kit user import users.json --dryrun

# With user mapping
gh team-kit user import users.json --usermap usermap.yaml

# Set default role
gh team-kit user import users.json --role admin

# Continue on errors without exiting
gh team-kit user import users.json --ignore-errors
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
# Get hovercard (basic, with optional subject type/id)
gh team-kit user hovercard get [username]
gh team-kit user hovercard get [username] --subject-type organization --subject-id 12345

# In org context (username is optional positional arg)
gh team-kit user hovercard org [username] --owner <org>

# In repo context (username is optional positional arg)
gh team-kit user hovercard repo [username] --repo owner/repo

# In issue context (issue-number is first positional arg, username is optional second)
gh team-kit user hovercard issue <issue-number> [username] --repo owner/repo

# In PR context (pr-number is first positional arg, username is optional second)
gh team-kit user hovercard pr <pr-number> [username] --repo owner/repo
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
gh team-kit copilot metrics <team-slug> --format json
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

## `code-review` — Code Review Assignment Settings

### `code-review get` (alias: `view`)

```bash
# Get code review assignment settings for a team
gh team-kit code-review get <team-slug>
```

### `code-review set`

```bash
# Enable code review assignment
gh team-kit code-review set <team-slug> --enable

# Disable code review assignment
gh team-kit code-review set <team-slug> --disable

# Set number of reviewers to assign
gh team-kit code-review set <team-slug> --member-count 3

# Set assignment algorithm (ROUND_ROBIN or LOAD_BALANCE)
gh team-kit code-review set <team-slug> --algorithm ROUND_ROBIN
gh team-kit code-review set <team-slug> --algorithm LOAD_BALANCE

# Notify the entire team when a review is requested
gh team-kit code-review set <team-slug> --notify-team

# Disable team notification
gh team-kit code-review set <team-slug> --no-notify-team

# Include child team members in review pool
gh team-kit code-review set <team-slug> --include-child-team-members

# Count members who have already been requested
gh team-kit code-review set <team-slug> --count-members-already-requested

# Remove the team from the review request when assigning individuals
gh team-kit code-review set <team-slug> --remove-team-request

# Exclude specific members from code review assignment
gh team-kit code-review set <team-slug> --exclude-members user1,user2

# Combined example
gh team-kit code-review set <team-slug> --enable --member-count 3 --algorithm ROUND_ROBIN --notify-team --exclude-members alice
```

---

## `pr-graph` — Pull Request Activity Relationship Graph

```bash
# Generate a Mermaid relationship graph for the current repository
gh team-kit pr-graph

# Analyze specific repositories
gh team-kit pr-graph owner/repo1 owner/repo2

# Analyze all repositories of an organization
gh team-kit pr-graph --owner my-org

# Filter pull requests by state and creation date range
gh team-kit pr-graph --state closed --since 2025-01-01 --until 2025-03-31

# Only include merged pull requests (excludes closed-without-merge)
gh team-kit pr-graph --state merged

# Limit the number of pull requests analyzed per repository, counted after
# state/date/label/branch/--exclude-author filtering (default: 30, 0 = unlimited)
gh team-kit pr-graph --limit 100

# Only include pull requests with at least one of these labels
gh team-kit pr-graph --label bug,feature

# Skip pull requests with any of these labels (e.g. merge-relay bots)
gh team-kit pr-graph --exclude-label auto-merge,merge-relay

# Only include pull requests targeting a base branch matching this glob pattern
gh team-kit pr-graph --base main

# Only include pull requests from a head branch matching this glob pattern
gh team-kit pr-graph --head 'feature/*'

# Skip pull requests whose head branch matches any of these glob patterns
gh team-kit pr-graph --exclude-head-branch 'relay/*,cherry-pick/*'

# Skip draft pull requests
gh team-kit pr-graph --exclude-draft

# Automatically exclude and hide users whose login has a "[bot]" suffix
gh team-kit pr-graph --no-bots

# Keep only specific edge relation types in the graph
gh team-kit pr-graph --edge-type changed,reviewed,approved

# Exclude specific edge relation types from the graph (e.g. directory containment)
gh team-kit pr-graph --exclude-edge-type in

# Remove edges with a weight below this threshold from the graph
gh team-kit pr-graph --min-weight 3

# Keep nodes that lost all their edges to edge filtering
gh team-kit pr-graph --exclude-edge-type in,member-of --keep-orphans

# Fold changed file paths into their ancestor directory truncated to 2 segments
gh team-kit pr-graph --depth 2

# Fold changed file paths using glob-style prefix patterns (first match wins)
gh team-kit pr-graph --group-by 'LocalPackages/*,Assets/*/*'

# Combine both: unmatched paths fall back to --depth
gh team-kit pr-graph --group-by 'LocalPackages/*' --depth 1

# Drop pull requests authored by these users from analysis entirely
gh team-kit pr-graph --exclude-author dependabot --exclude-author alice,bob

# Omit a user's non-author activity and CODEOWNERS relationships without
# excluding their pull requests
gh team-kit pr-graph --hide-user dependabot

# Deprecated: equivalent to --exclude-author and --hide-user for these logins
gh team-kit pr-graph --exclude-user dependabot --exclude-user alice,bob

# Exclude files using .gitignore-style patterns
gh team-kit pr-graph --exclude-file "*.md" --exclude-file "vendor/**"

# Exclude files marked linguist-generated in .gitattributes
gh team-kit pr-graph --exclude-generated

# Only include files matching these .gitignore-style patterns
gh team-kit pr-graph --include-file "src/**"

# Exclude paths that no longer exist on the default branch
gh team-kit pr-graph --exclude-deleted

# Weigh changed paths by changed lines instead of one per file
gh team-kit pr-graph --weight-by lines

# Halve each contribution every 90 days so recent work dominates
gh team-kit pr-graph --half-life 90

# Link paths changed together, skipping pull requests above 30 paths
gh team-kit pr-graph --co-change --co-change-max-files 30

# Keep only the current team roster in the graph
gh team-kit pr-graph --allow-user alice,bob
gh team-kit pr-graph --user-allowlist ./roster.txt

# Ownership profile: recent, line-weighted, surviving code, roster only
gh team-kit pr-graph --since 2025-01-01 --limit 0 --weight-by lines \
  --half-life 90 --exclude-deleted --exclude-generated \
  --user-allowlist ./roster.txt --edge-type changed,created,owned-by \
  --format json

# Raise the per-request API timeout for repositories with a huge PR history
gh team-kit pr-graph --since 2026-01-01 --http-timeout 5m

# Output as Graphviz DOT, Markdown (fenced Mermaid), or JSON node/edge data
gh team-kit pr-graph --format dot
gh team-kit pr-graph --format markdown
gh team-kit pr-graph --format json
```

Nodes: users, teams, labels, files, directories, and submodules. Edges (weighted by occurrence count unless `--weight-by` says otherwise): `approved`, `changes-requested`, `reviewed`, `commented`, `review-commented`, `review-requested`, `member-of`, `changed`, `created` (the path was added by the pull request), `co-changed` (two paths changed together), `in` (directory containment), `owned-by` (CODEOWNERS), and `labeled`.

When multiple repositories are given they must all be on the same host, and team nodes are namespaced by owner (`owner/slug`) to avoid collisions across organizations. A date-only `--until` is inclusive through the end of that day. `--state merged` selects closed pull requests that were actually merged. Use `--label`/`--exclude-label` to filter by pull request labels, and `--base`/`--head`/`--exclude-head-branch` to filter by branch name glob patterns; exclusion takes precedence when both an include and exclude filter match. Use `--exclude-draft` to skip draft pull requests, and `--no-bots` to automatically exclude and hide users whose login has a `[bot]` suffix. Use `--edge-type`/`--exclude-edge-type` to keep or remove specific edge relation types (`approved`, `changes-requested`, `reviewed`, `commented`, `review-commented`, `review-requested`, `member-of`, `changed`, `created`, `co-changed`, `in`, `owned-by`, `labeled`) from the rendered graph, and `--min-weight` to remove edges below a given weight (it accepts fractions, so scale it with `--weight-by` and `--half-life`). Nodes left without any edge after edge filtering are removed from the graph; use `--keep-orphans` to retain them. Use `--group-by` to fold changed file paths into package-level nodes using glob-style prefix patterns such as `LocalPackages/*,Assets/*/*` (repeatable or comma-separated; evaluated in order, first match wins), and `--depth` to fold the remaining paths into their ancestor directory truncated to the given number of path segments; the two can be combined, with `--depth` acting as the fallback. A folded path is emitted as a directory node rather than a file node. A changed path registered in the repository's `.gitmodules` is emitted as a submodule node, distinguishing submodule pointer updates from ordinary file changes. Use `--exclude-author` to drop pull requests authored by the given users from analysis entirely, and `--hide-user` to omit a user's non-author activity and CODEOWNERS relationships from the graph without excluding their pull requests. `--exclude-user` is deprecated and equivalent to setting both `--exclude-author` and `--hide-user`. Login matching is case-insensitive. Use `--exclude-file`/`--include-file` to omit or restrict files (and their directory/CODEOWNERS relationships) by `.gitignore`-style pattern, and `--exclude-generated` to additionally omit files marked with the `linguist-generated` attribute in the repository's `.gitattributes`. Use `--exclude-deleted` to omit paths that no longer exist on the repository's default branch, so that ownership reflects the code that is still there; the option disables itself with a warning when the repository tree is too large for GitHub to return in full. Use `--weight-by` to weigh a changed path by `occurrences` (default), `lines`, `additions`, or `deletions`; it affects the `changed`, `created`, `in`, `owned-by`, and `co-changed` relations only, because reviews, comments, labels, and team membership have no line count. Use `--half-life` to halve each contribution every given number of days so recent work outweighs old work; ages are measured against `--until` when set and the current time otherwise. Use `--co-change` to link paths changed by the same pull request, with `--co-change-max-files` (default: `50`) skipping wide pull requests whose pair count would explode; each pair is emitted once with the smaller of the two contributions as its weight. Use `--allow-user`/`--user-allowlist` to restrict the graph to a known roster: unlike `--hide-user`, a user outside the allowlist is dropped even as a pull request author, while their pull requests still contribute file, directory, CODEOWNERS, and co-change edges. Use the global `--http-timeout` flag to raise the timeout of each GitHub API request (default: `30s`) when analyzing repositories with a very large pull request history. When `--since` is set, pagination stops as soon as a fetched page is entirely older than the given date, and transient request failures such as timeouts are retried automatically.

**Ownership analysis limits.** The graph is built entirely from pull request activity, so direct pushes, imported history, and repository migrations are invisible, and a `created` edge only exists when the analyzed pull requests added the path — widen `--since` when you need first authorship rather than recent activity. Edge weights measure the volume of change, not the amount of surviving code: a file later rewritten by someone else still credits its original authors. `pr-graph` cannot answer "who wrote the lines in the file today", which requires line-level blame that the pull request APIs do not expose.

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
