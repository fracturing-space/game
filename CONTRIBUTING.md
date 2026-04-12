# Contributing

This repository uses `servicekit` to keep core contributor tooling consistent across service repos.

## Prerequisites

- Go `1.26`- `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` available on `PATH`- `sqlc` only if you need to regenerate checked-in SQLite query bindings

## First-Time Setup

```bash
make install-hooks
```The repo-owned hooks run `make check` before `git commit`, merge commits, and `git am`. Use Git's standard `--no-verify` escape hatch only when you need to capture work in progress.

## Development Workflow

- Run `make test` for fast feedback while implementing.- Run `make test-race` when you are changing concurrency, storage, or streaming behavior.
- Run `make modernize` when Go can apply safe source rewrites that the repo expects.
- Run `make vet` for baseline static analysis.
- Run `make cover` when you need updated coverage artifacts.
- Run `make check` before sending a change for review.

## Generated And Derived Files- `make proto` regenerates checked-in protobuf Go stubs under `api/gen/go/...`
- `make sqlc` regenerates checked-in SQLite bindings under `internal/storage/sqlite/db/...`
- `make normgen` regenerates checked-in normalization helpers under `internal/.../zz_normalize.go`
## Service-Specific Notes

For behavior changes, start with [docs/architecture/game-service.md](docs/architecture/game-service.md).

Keep transport adapters thin; domain rules belong in contract, module, and service packages.

Repo admins should use [docs/github/setup.md](docs/github/setup.md) for manual GitHub setup.
