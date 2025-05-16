# gh-team-kit

gh extension of github team api

## Installation

To install the tool, you can use the following command:

```sh
gh extension install srz-zumix/gh-team-kit
```

## Commands Overview

The following commands are available in `gh-team-kit`. Each command is designed to help manage GitHub teams, repositories, and users efficiently.

- **Team Management**: Create, update, delete, move, and display team hierarchies.
- **Member Management**: Add, remove, check, and modify roles of team members.
- **Repository Management**: Add, remove, list, and compare repositories associated with teams.
- **User Management**: Add, remove, list, and check users in the organization or repositories.
- **Organization-Role Management**: Manage roles within the organization, including listing available roles.
- **Permission Management**: Check and synchronize permissions for teams and users across repositories.
- **Comparison Tools**: Compare teams, repositories, and permissions to identify differences.

Refer to the specific command sections below for detailed usage and examples.

## Usage

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
gh team-kit member add <team-slug> <username> [role]
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

#### Perform set operations on two teams members

```sh
gh team-kit member sets <[owner]/team-slug1> <|,&,-,^> <[owner]/team-slug2>
```

Perform set operations on the members of two teams. The operation can be union (`|`), intersection (`&`), difference (`-`), or symmetric difference (`^`).

#### Sync members from one team to another

```sh
gh team-kit member sync <[owner/]src-team-slug> <[owner/]dst-team-slug>
```

Sync members from the source team to the destination team. Members in the source team will be added to the destination team, and members not in the source team will be removed from the destination team.

#### Copy members from one team to another

```sh
gh team-kit member copy <[owner/]src-team-slug> <[owner/]dst-team-slug>
```

Copy members from the source team to the destination team. Members in the source team will be added to the destination team, but no members will be removed from the destination team.

#### Remove a member from a team

```sh
gh team-kit member remove <team-slug> <username>
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

#### List all users in the organization

```sh
gh team-kit user list
```

Retrieve and display a list of all users in the organization.

#### List user repositories

```sh
gh team-kit user repo <username>
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

#### Check user permissions for a repository

```sh
gh team-kit repo user check <username>
```

Check the permissions of a specified user for a repository.

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

### Organization-Role Management

#### Add a team to an organization role

```sh
gh team-kit org add <team-slug> <org-role>
```

Add a specified team to the specified role in the organization.

#### Remove a team from an organization role

```sh
gh team-kit org remove <team-slug> <org-role>
```

Remove a specified team from the specified role in the organization.

#### List organization roles

```sh
gh team-kit org role list [owner]
```

List all roles available in the organization. Optionally, specify the owner to filter roles.

#### List users assigned to an organization role

```sh
gh team-kit org user list [org-role-name]
```

Retrieve and display a list of all users assigned to a specific role in the organization. Supports options for detailed information, suspended users, and filtering by owner.

#### Add a user to an organization role

```sh
gh team-kit org user add <username> <org-role>
```

Assign a specified user to the specified role in the organization.

#### Remove a user from an organization role

```sh
gh team-kit org user remove <username> <org-role>
```

Remove a specified user from the specified role in the organization.
