# blog-aggregator (gator)

A command-line RSS feed aggregator, written in Go. It stores users, feeds,
and posts in Postgres, lets users follow feeds, and can continuously
aggregate new posts in the background.

This was built as a guided project for boot.dev.

## Requirements

- Go (1.25+)
- PostgreSQL, running and reachable

## Installation

Install the CLI with `go install`:

```
go install github.com/blageo/blog-aggregator@latest
```

This builds a binary named `blog-aggregator` (Go names the binary after the
module, not the repo). If you'd rather type `gator`, rename or alias it,
e.g.:

```
ln -s "$(go env GOPATH)/bin/blog-aggregator" "$(go env GOPATH)/bin/gator"
```

The rest of this README refers to the command as `gator` for brevity, but
substitute `blog-aggregator` if you didn't create an alias.

## Configuration

Before running the program, create a config file at `~/.gatorconfig.json`
with your Postgres connection string and a current user name:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

`current_user_name` is updated automatically by `login`/`register`; you can
leave it empty initially.

## Database migrations

Schema migrations live in `sql/schema` and use the
[Goose](https://github.com/pressly/goose) format. Install goose and run the
migrations against your database before first use:

```
go install github.com/pressly/goose/v3/cmd/goose@latest
cd sql/schema
goose postgres "postgres://username:password@localhost:5432/gator?sslmode=disable" up
```

## Commands

- `register <name>` - creates a new user and logs in as them.
- `login <name>` - sets the current user (must already be registered).
- `addfeed <name> <url>` - adds a new RSS feed and follows it as the
  current user. Example: `gator addfeed "Boot.dev Blog" https://blog.boot.dev/index.xml`
- `feeds` - lists all feeds that have been added, across all users.
- `follow <url>` - follows an existing feed (by URL) as the current user.
- `following` - lists the feeds the current user follows.
- `unfollow <url>` - unfollows a feed.
- `browse [limit]` - prints recent posts from feeds the current user
  follows. `limit` is optional and defaults to a small number of posts.
- `agg <duration>` - runs continuously, fetching the next feed due for
  refresh on each tick. Example: `gator agg 1m` polls once per minute.
  Run this in a separate terminal/process while you use the other
  commands.

## Typical first run

```
gator register alice
gator addfeed "Boot.dev Blog" https://blog.boot.dev/index.xml
gator agg 1m        # in another terminal, leave running
gator browse 5
```
