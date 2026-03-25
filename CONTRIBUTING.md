# Contributing

## Development

Run the full test suite:

```bash
go test ./...
```

Format the repository before sending changes:

```bash
gofmt -w *.go
```

## Scope Discipline

This repository is only for the low-level Clinic SDK.

Keep these concerns out of this repo:

- workflow orchestration
- diagnosis logic
- item selection policy
- application-specific integration behavior

## Pull Request Expectations

- keep the public surface small
- prefer additive changes over breaking renames
- add or update tests for behavior changes
- update README or examples when public usage changes
