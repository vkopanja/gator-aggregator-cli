# Gator, blog aggregator - boot.dev

A blog aggregator written in Go during the guided course on [boot.dev](https://boot.dev).

Uses latest [goose](https://github.com/pressly/goose) for migrations and [sqlc](https://github.com/sqlc-dev/sqlc) for generating SQL code. Basic config is stored in `~/.gatorconfig.json`

```json
{
  "db_url": "postgres://<user>:<password>@<hostname>:<port>/<db-name>?sslmode=disable",
  "current_user_name": "WILL BE SET WITH CLI"
}
```

## Quick start

Just run `make build` and run it `./bin/gator <command> [params]` - when it starts type `help` for list of available commands.