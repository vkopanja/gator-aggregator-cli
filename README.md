# Gator, RSS feed aggregator - boot.dev

A RSS aggregator written in Go during the guided course on [boot.dev](https://boot.dev).

You need to have Go installed and PostgreSQL running either standalone or preferably in a Docker container.
I have chosen to use a DB named `gator` instead of the default `postgres`.

## Install Go

### macOS (with Homebrew [brew.sh](https://brew.sh/))

```
brew install go
```

### Linux

#### Ubuntu/Debian

```
sudo apt update && sudo apt install golang-go
```

#### Fedora

```
sudo dnf install golang
```

#### Arch

```
sudo pacman -S go
```

### Verify

```
go version
```

Uses latest [goose](https://github.com/pressly/goose) for migrations and [sqlc](https://github.com/sqlc-dev/sqlc) for
generating SQL code.

Easiest way to install `goose` is to use Go itself: `go install github.com/pressly/goose/v3/cmd/goose@latest`

Basic config is stored in `~/.gatorconfig.json` and should look like this:

```json
{
  "db_url": "postgres://<user>:<password>@<hostname>:<port>/<db-name>?sslmode=disable",
  "current_user_name": "WILL BE SET WITH CLI"
}
```

## Quick start

Just run `make build` and run it `./bin/gator <command> [params]` - when it starts type `help` for list of available
commands.

Some of the commands to run are:

- `gator register <username>` &larr; will create a new user in `users` table and set it as the `current_user_name`
- `gator login <username>` &larr; update the `current_user_name` to set the current user
- `gator users` &larr; list all registered users
- `gator feeds` &larr; list all the feeds and the username who created them
- `gator addfeed <name> <url>` &larr; e.g. `gator addfeed "Boot Dev" https://blog.boot.dev/index.xml`

These are just few of the available commands, type `gator help` for more info.
