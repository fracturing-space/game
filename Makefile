PROTO_DIR ?= api/proto
GEN_GO_DIR ?= api/gen/go
PROTO_FILES := $(wildcard $(PROTO_DIR)/game/v1/*.proto)
GO_TEST_CACHE_DIR ?= $(CURDIR)/.tmp/go-cache
GO_TEST_TMP_DIR ?= $(CURDIR)/.tmp/go-build
COVER_DIR ?= $(CURDIR)/.tmp/coverage
COVER_RAW ?= $(COVER_DIR)/coverage.raw
COVER_OUT ?= $(COVER_DIR)/coverage.out
COVER_FUNC ?= $(COVER_DIR)/coverage.func
COVER_HTML ?= $(COVER_DIR)/coverage.html
SERVICEKIT_VERSION ?= main
COVER_EXCLUDE_REGEX ?= (^github\.com/fracturing-space/game/cmd/|^github\.com/fracturing-space/game/api/gen/|^github\.com/fracturing-space/game/internal/storage/sqlite/db/)
COVERAGE_FLOORS_FILE ?= docs/reference/coverage-floors.json
STATICCHECK_VERSION ?= 2026.1

.PHONY: proto sqlc normgen normgen-check test test-race vet staticcheck cover check-coverage fmt-check install-hooks modernize modernize-check check

proto:
	@mkdir -p "$(GEN_GO_DIR)"
	protoc \
		-I "$(PROTO_DIR)" \
		--go_out="$(GEN_GO_DIR)" \
		--go_opt=paths=source_relative \
		--go-grpc_out="$(GEN_GO_DIR)" \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)


normgen:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go run ./tools/normgen

normgen-check:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go run ./tools/normgen check


sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
	gofmt -w internal/storage/sqlite/db/*.go

test:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go test ./...

test-race:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go test -race ./...

vet:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go vet ./...

staticcheck:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...

modernize:
	# Some fixes only become applicable after earlier rewrites, so a second
	# pass helps the tree reach a fixed point without extra shell logic.
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go fix ./...
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go fix ./...

modernize-check:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	@GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" bash -euo pipefail -c '\
		diff_file="$$(mktemp)"; \
		trap '\''rm -f "$$diff_file"'\'' EXIT; \
		go fix -diff ./... > "$$diff_file"; \
		if [ -s "$$diff_file" ]; then \
			echo "Go modernization needed. Run '\''make modernize'\''."; \
			cat "$$diff_file"; \
			exit 1; \
		fi; \
		echo "Go modernization check passed." \
	'

cover:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)" "$(COVER_DIR)"
	@rm -f "$(COVER_RAW)" "$(COVER_OUT)" "$(COVER_FUNC)" "$(COVER_HTML)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go test ./... -covermode=set -coverprofile="$(COVER_RAW)"
	@bash -euo pipefail -c '\
		if [ -n "$(COVER_EXCLUDE_REGEX)" ]; then \
			{ \
				head -n 1 "$(COVER_RAW)"; \
				tail -n +2 "$(COVER_RAW)" | grep -Ev "$(COVER_EXCLUDE_REGEX)" || true; \
			} > "$(COVER_OUT)"; \
		else \
			cp "$(COVER_RAW)" "$(COVER_OUT)"; \
		fi'
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go tool cover -func="$(COVER_OUT)" > "$(COVER_FUNC)"
	GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go tool cover -html="$(COVER_OUT)" -o "$(COVER_HTML)"
	@awk '/^total:/{print}' "$(COVER_FUNC)"

check-coverage:
	@mkdir -p "$(GO_TEST_CACHE_DIR)" "$(GO_TEST_TMP_DIR)"
	@bash -euo pipefail -c '\
		if [ -f "./tools/coveragefloors/main.go" ]; then \
			GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go run ./tools/coveragefloors check -profile="$(COVER_OUT)" -floors="$(COVERAGE_FLOORS_FILE)"; \
		else \
			GOCACHE="$(GO_TEST_CACHE_DIR)" GOTMPDIR="$(GO_TEST_TMP_DIR)" go run github.com/fracturing-space/servicekit/cmd/coveragefloors@$(SERVICEKIT_VERSION) check -profile="$(COVER_OUT)" -floors="$(COVERAGE_FLOORS_FILE)"; \
		fi'

fmt-check:
	@bash -euo pipefail -c '\
		files="$$(find . -type f -name "*.go" -not -path "./.tmp/*" | sort)"; \
		if [ -z "$$files" ]; then \
			echo "No Go files found."; \
			exit 0; \
		fi; \
		unformatted="$$(gofmt -l $$files)"; \
		if [ -n "$$unformatted" ]; then \
			echo "Go files need formatting:"; \
			printf "%s\n" "$$unformatted"; \
			exit 1; \
		fi; \
		echo "Go formatting check passed." \
	'

install-hooks:

	git config extensions.worktreeConfig true
	git config --worktree core.hooksPath .githooks
	git config --worktree core.bare false
	@printf 'Installed Git hooks at %s\n' "$$(git config --worktree --get core.hooksPath)"


check:
	$(MAKE) normgen-check
	$(MAKE) modernize-check
	$(MAKE) fmt-check
	$(MAKE) vet
	$(MAKE) staticcheck
	$(MAKE) test
	$(MAKE) cover
	$(MAKE) check-coverage
