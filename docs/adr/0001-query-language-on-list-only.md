# Query language on list only

Projects are selected with a GitHub-style query (`pj list -f 'api tag:go NOT is:missing'`) that only `list` accepts. Every other command takes projects by name or path, and batch work pipes `pj list -f ... -o paths` into them. One place to select keeps the other commands small, and a destructive command like `rm` can never act on more than the projects it was explicitly given.

## Considered Options

- **`field op value` terms** (`-f 'name~api'`, `=`, `!=`, `~`, `!~`): needed shell quoting for `~` and `!`, had no text search or OR, and read unlike any search people already know.
- **`--filter` on every multi-project command**: rejected because a query on `rm` or `tag detect --apply` can silently match more than intended.

## Consequences

- Commands that accept several projects must resolve every argument before acting, and fail without changes if any argument is missing or ambiguous.
- A command where "no arguments" means "everything" must require an explicit `--all`, because `xargs` runs it with no arguments when the query matches nothing.
