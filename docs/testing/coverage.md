# Testing And Coverage

This repository uses the standard `servicekit` verification flow for Go service repos.

## Commands- `make proto`: regenerate checked-in protobuf Go stubs

- `make test`: fast local verification during implementation- `make test-race`: run the test suite with the race detector

- `make vet`: run baseline static analysis
- `make cover`: generate coverage artifacts under `.tmp/coverage/`
- `make install-hooks`: configure repo-owned Git hooks for this checkout
- `make check`: formatting, static analysis, tests, and coverage verification- `make check-coverage`: enforce package floors from `docs/reference/coverage-floors.json`


Recommended contributor setup:

```bash
make install-hooks
```

## Coverage Policy

Coverage is a regression guardrail, not a vanity target.

- thin server wiring in `cmd/` is excluded from coverage
- checked-in generated transport code in `api/gen/` is excluded from coverage- generated sqlc bindings in `internal/storage/sqlite/db/` are excluded from coverage
- package floors live in [docs/reference/coverage-floors.json](../reference/coverage-floors.json)

## Coverage Artifacts

`make cover` writes:

- `.tmp/coverage/coverage.raw`
- `.tmp/coverage/coverage.out`
- `.tmp/coverage/coverage.func`
- `.tmp/coverage/coverage.html`## Service-Specific Notes

Scenario planning notes: [docs/testing/scenario-requirements.md](scenario-requirements.md).

