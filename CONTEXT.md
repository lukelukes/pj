# pj

pj keeps a personal catalog of local projects so they can be found, opened and scripted by name.

## Language

### Catalog

**Catalog**:
The persisted collection of every project pj knows about.

**Project**:
A named local directory recorded in the catalog, with optional description, editor and tags.

**Tag**:
A lowercase label attached to a project. A project holds a set of tags.

**Missing project**:
A project whose recorded path no longer exists on disk.
_Avoid_: stale project, broken project

### Querying

**Query**:
The text given to one `--filter`, selecting the projects that satisfy every term in it.
_Avoid_: filter expression, selector, search string

**Term**:
One whitespace-separated piece of a query, optionally negated.

**Qualifier**:
The `key:` prefix of a term that names what the term tests, such as `tag:` or `is:`.
_Avoid_: field, operator

**Text term**:
A term without a qualifier, matched against a project's name and description.
_Avoid_: bare word, free text
