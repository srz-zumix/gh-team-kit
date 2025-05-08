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

### Add a repository to a team

```sh
gh team-kit repo add <team-slug> <permission>
```

### Check if a user is a member of a team

```sh
gh team-kit member check <team-slug> <username>
```

### Check team permissions for a repository

```sh
gh team-kit repo check <team-slug>
```

### Compare repositories between two teams

```sh
gh team-kit diff <team-slug1> <team-slug2> [repository...]
```

### Compare team permissions between two repositories

```sh
gh team-kit repo diff <repo1> <repo2> [team-slug...]
```

### Copy teams and permissions to multiple destination repos

```sh
gh team-kit repo copy <dst-repository...>
```

### Create a new team

```sh
gh team-kit create <name> --description <description> --parent <parent-team-slug> --disable-notification --secret --owner <owner>
```

### Delete a team

```sh
gh team-kit delete <team>
```

### Display a team hierarchy in a tree structure

```sh
gh team-kit tree [team-slug]
```

### Get a team by its slug

```sh
gh team-kit get [team-slug...]
```

### List all teams in the organization

```sh
gh team-kit list [owner]
```

### List members of a team

```sh
gh team-kit member list <team-slug>
```

### List repositories for a team

```sh
gh team-kit repo list <team-slug>
```

### Remove a member from a team

```sh
gh team-kit member remove <team-slug> <username>
```

### Remove a repository from a team

```sh
gh team-kit repo remove <team-slug>
```

### Sync teams and permissions to multiple destination repos

```sh
gh team-kit repo sync <dst-repository...>
```
