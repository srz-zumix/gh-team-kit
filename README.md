# gh-team-kit

gh extension of github team api

## Installation

To install the tool, you can use the following command:

```sh
gh extension install srz-zumix/gh-team-kit
```

## Usage

### Add a member to a team

```sh
gh team-kit member add <team-slug> <username> [role]
```

Add a specified user to the specified team in the organization. Optionally specify the role (default: member).

### Add a repository to a team

```sh
gh team-kit repo add <team-slug> <permission>
```

Add a specified repository to the specified team in the organization.

### Check if a user is a member of a team

```sh
gh team-kit member check <team-slug> <username>
```

Check whether a user is a member of the specified team in the organization.

### Check team permissions for a repository

```sh
gh team-kit repo check <team-slug>
```

Check whether a team has admin, push, maintain, triage, pull, or none permission for a repository.

### Compare repositories between two teams

```sh
gh team-kit diff <team-slug1> <team-slug2> [repository...]
```

Compare the repositories associated with two teams and display the differences.

### Compare team permissions between two repositories

```sh
gh team-kit repo diff <repo1> <repo2> [team-slug...]
```

Compare the team permissions between two repositories and display the differences.

### Copy teams and permissions to multiple destination repos

```sh
gh team-kit repo copy <dst-repository...>
```

Copy teams and permissions from a source repository to multiple destination repositories.

### Create a new team

```sh
gh team-kit create <name> --description <description> --parent <parent-team-slug>
```

Create a new team in the specified organization with various options such as description, privacy, and notification settings.

### Delete a team

```sh
gh team-kit delete <team-slug>
```

Delete a specified team from the organization. Ensure that the team is no longer needed as this action is irreversible.

### Display a team hierarchy in a tree structure

```sh
gh team-kit tree [team-slug]
```

Display a team hierarchy in a tree structure based on the team's slug.

### Get a team by its slug

```sh
gh team-kit get [team-slug...]
```

Retrieve details of a team using the team's slug.

### List all teams in the organization

```sh
gh team-kit list [owner]
```

Retrieve and display a list of all teams in the specified organization. You can optionally filter the results by repository.

### List members of a team

```sh
gh team-kit member list <team-slug>
```

List all members of the specified team in the organization.

### List repositories for a team

```sh
gh team-kit repo list <team-slug>
```

List all repositories for the specified team in the organization.

### Move a team to a new parent

```sh
gh team-kit move <team-slug> [new-parent-slug]
```

Change the parent of an existing team in the specified organization to a new parent team. If no new parent is specified, the team will be moved to the root level.

### Remove a member from a team

```sh
gh team-kit member remove <team-slug> <username>
```

Remove a specified user from the specified team in the organization.

### Remove a repository from a team

```sh
gh team-kit repo remove <team-slug>
```

Remove a specified repository from the specified team in the organization.

### Rename an existing team

```sh
gh team-kit rename <team-slug> <new-name>
```

Rename an existing team in the specified organization to a new name.

### Sync teams and permissions to multiple destination repos

```sh
gh team-kit repo sync <dst-repository...>
```

Synchronize teams and permissions from a source repository to multiple destination repositories.

### Update a team

```sh
gh team-kit update <team-slug> --description <new-description> --parent <parent-team-slug>
```

Update the details of an existing team in the specified organization, such as its description or settings.
