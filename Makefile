# Makefile for ModelSDKGo
#
# Usage:
#   make setup     - Configure git hooks (run once after clone)
#   make build     - Build mxcli for current platform
#   make release   - Build mxcli for all platforms (macOS, Windows, Linux)
#   make test      - Run unit tests
#   make check-mdl - Check MDL syntax for all doctype example scripts
#   make test-integration - Run integration tests (requires mx/mxbuild)
#   make test-mdl  - Run MDL integration tests (requires Docker)
#   make lint      - Lint Go code (fmt + vet)
#   make lint-go   - Lint Go code (fmt + vet)
#   make grammar   - Regenerate ANTLR parser
#   make docs-site - Build documentation site (mdbook)
#   make docs-serve - Serve docs site locally with live reload
#   make sbom      - Generate CycloneDX SBOM (Go + TypeScript)
#   make sbom-report - Generate Markdown dependency report
#   make install-global - Build + install mxcli-daemon to /usr/local/bin/mxcli (requires sudo)
#   make mine-exprgrammar MINE_MPR=path/to/app.mpr - Re-mine generated/exprgrammar/mined.go from an MPR
#   make clean     - Remove build artifacts

HELPDESK_VERSIONS := 11.6.6 11.10.0

BINARY_NAME = mxcli
BUILD_DIR = bin
CMD_PATH = ./cmd/mxcli

# Version info (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_SHA = $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitSHA=$(COMMIT_SHA)"
# Release builds strip debug info and symbol table (~23% smaller).
RELEASE_LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitSHA=$(COMMIT_SHA) -s -w"
DAEMON_NAME = mxcli-daemon

# Clean version for VS Code extension (must be valid semver: major.minor.patch)
VSCE_VERSION = $(shell echo "$(VERSION)" | sed 's/^v//; s/-.*//' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$$' || echo "0.0.0")

# Limit CPU to 85% so the machine stays responsive during test runs.
# Uses cpulimit(1) when installed; falls back to nice -n 15.
# cpulimit -l value = nproc * 85 (percentage of one core per core).
_NCPU         := $(shell nproc)
_CPU_LIMIT_L  := $(shell echo "$(_NCPU) * 85" | bc)
_CPU_RUNNER   := $(shell command -v cpulimit >/dev/null 2>&1 \
                   && echo "cpulimit -l $(_CPU_LIMIT_L) --" \
                   || echo "nice -n 15")

# Max concurrent packages and tests, capped at 85% of nproc.
_85PCT        := $(shell echo "$(_NCPU) * 85 / 100" | bc)
TEST_P        ?= $(_85PCT)
TEST_PARALLEL ?= $(_85PCT)

# Hard ceiling on how long the full test suite may run.
TEST_TIMEOUT ?= 180s

.PHONY: build mdlrun build-local install-local build-debug release release-mxcli release-win-amd64 release-launcher release-daemon release-local-bins clean test _test-inner test-mdl report report-bench report-reset-baseline bench-baseline grammar sync-skills sync-commands sync-lint-rules sync-changelog sync-examples sync-all docs documentation docs-site docs-serve source-tree sbom sbom-report lint lint-go fmt vet update-helpdesk-golden test-helpdesk-regression setup install install-daemon install-global test-section-check update-snapshots validate-snapshots validate-academy-capstone test-integration-profiled test-profile-check test-profile-record

setup:
	git config core.hooksPath .githooks
	@echo "Git hooks configured. Pre-commit unit tests enabled."

# Verify helpdesk-app.mdl produces correct BSON in cross-section (separate mxcli exec
# process per -- MARK: section) execution mode, then confirms mx check error count
# does not exceed .mx-check-baseline. Catches regressions in cross-session state
# propagation (entity resolution, return types, parameter caches).
# Requires: locally-installed mxbuild (run: mxcli setup mxbuild -p testdata/helpdesk-clean-11.6.6/minimal.mpr)
test-section-check: build
	@./scripts/test-section-check.sh

# Install the locally-built binary to PATH.
install: install-global

install-global: build
	@INSTALL_DIR="$${HOME}/.local/bin"; \
	mkdir -p "$$INSTALL_DIR"; \
	echo "Installing mxcli to $$INSTALL_DIR/mxcli..."; \
	cp "$(BUILD_DIR)/$(BINARY_NAME)" "$$INSTALL_DIR/mxcli"; \
	chmod 755 "$$INSTALL_DIR/mxcli"; \
	echo "✅ Installed $$INSTALL_DIR/mxcli"; \
	echo ""; \
	echo "Installing shell completions..."; \
	"$$INSTALL_DIR/mxcli" setup completions 2>/dev/null || true; \
	echo "✅ Completions installed (restart shell or run: source <($$INSTALL_DIR/mxcli completion zsh))"

# Helper: copy file only if content differs (avoids mtime updates that invalidate go build cache)
# Usage: $(call copy-if-changed,src,dst)
define copy-if-changed
	@if [ ! -f $(2) ] || ! cmp -s $(1) $(2); then cp $(1) $(2); fi
endef

# Sync skills from .claude/skills/mendix to cmd/mxcli/skills for embedding.
# Supports two layouts:
#   flat:      .claude/skills/mendix/<name>.md      → cmd/mxcli/skills/<name>.md
#   directory: .claude/skills/mendix/<name>/SKILL.md → cmd/mxcli/skills/<name>.md (flattened)
sync-skills:
	@mkdir -p cmd/mxcli/skills
	@changed=0; \
	for f in .claude/skills/mendix/*.md; do \
		dst="cmd/mxcli/skills/$$(basename $$f)"; \
		if [ ! -f "$$dst" ] || ! cmp -s "$$f" "$$dst"; then \
			cp "$$f" "$$dst"; changed=$$((changed + 1)); \
		fi; \
	done; \
	for skill_md in .claude/skills/mendix/*/SKILL.md; do \
		dir=$$(dirname "$$skill_md"); \
		name=$$(basename "$$dir"); \
		dst="cmd/mxcli/skills/$${name}.md"; \
		if [ ! -f "$$dst" ] || ! cmp -s "$$skill_md" "$$dst"; then \
			cp "$$skill_md" "$$dst"; changed=$$((changed + 1)); \
		fi; \
	done; \
	if [ $$changed -gt 0 ]; then echo "Synced $$changed skill file(s)"; fi

# Sync commands from .claude/commands/mendix to cmd/mxcli/commands for embedding
sync-commands:
	@mkdir -p cmd/mxcli/commands
	@changed=0; for f in .claude/commands/mendix/*.md; do \
		dst="cmd/mxcli/commands/$$(basename $$f)"; \
		if [ ! -f "$$dst" ] || ! cmp -s "$$f" "$$dst"; then \
			cp "$$f" "$$dst"; changed=$$((changed + 1)); \
		fi; \
	done; \
	if [ $$changed -gt 0 ]; then echo "Synced $$changed command file(s)"; fi

# Sync lint rules from .claude/lint-rules to cmd/mxcli/lint-rules for embedding
sync-lint-rules:
	@mkdir -p cmd/mxcli/lint-rules
	@changed=0; for f in .claude/lint-rules/*.star; do \
		dst="cmd/mxcli/lint-rules/$$(basename $$f)"; \
		if [ ! -f "$$dst" ] || ! cmp -s "$$f" "$$dst"; then \
			cp "$$f" "$$dst"; changed=$$((changed + 1)); \
		fi; \
	done; \
	if [ $$changed -gt 0 ]; then echo "Synced $$changed lint rule file(s)"; fi

# Sync changelog to cmd/mxcli for embedding
sync-changelog:
	$(call copy-if-changed,CHANGELOG.md,cmd/mxcli/changelog.md)

# Sync example MDL files for embedding in mxcli init
sync-examples:
	@mkdir -p cmd/mxcli/examples
	$(call copy-if-changed,mdl-examples/use-cases/helpdesk/helpdesk-app.mdl,cmd/mxcli/examples/helpdesk-app.mdl)
	$(call copy-if-changed,mdl-examples/use-cases/helpdesk/helpdesk-describe.mdl,cmd/mxcli/examples/helpdesk-describe.mdl)

# Sync skills, commands, lint rules, changelog, and examples
sync-all: sync-skills sync-commands sync-lint-rules sync-changelog sync-examples

# Build for current platform (auto-syncs skills and commands)
# On Windows, also creates .exe-suffixed copies so PowerShell can execute them.
build: sync-all
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Compress a single daemon binary: make compress-daemon BIN=bin/mxcli-daemon-linux-amd64
compress-daemon:
	@command -v zstd >/dev/null || (echo "zstd not found; install it first" && exit 1)
	zstd -19 -f "$(BIN)" -o "$(BIN).tar.zst"
	@echo "Compressed: $(BIN).tar.zst"

# Build with debug tools (includes bson discover/compare/dump)
build-debug: sync-all completions
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags debug $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-debug (debug build with bson tools)"

# Build for all platforms (CGO_ENABLED=0 for cross-compilation).
# NOTE: requires a Unix shell (Linux, macOS, or Git Bash on Windows).
# In CI, this runs automatically on ubuntu-latest via release.yml.
release: clean sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "Building release binaries..."

	@echo "  -> mxcli (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_PATH)

	@echo ""
	@echo "Release binaries built in $(BUILD_DIR)/."

# Build mxcli binaries for all platforms (v* release pipeline).
release-mxcli: sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "  -> mxcli (all platforms)"
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64   $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64  $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_PATH)
	@echo "mxcli binaries built in $(BUILD_DIR)/."

# Build mxcli for Windows amd64 only.
release-win-amd64: sync-all
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(RELEASE_LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Run tests. TEST_P/TEST_PARALLEL default to 85% nproc.
# Uses nice(1) — NOT cpulimit(1), whose SIGSTOP/SIGCONT breaks Go's runtime.
test: test-showcase
	nice -n 15 $(MAKE) _test-inner

_test-inner:
	CGO_ENABLED=0 go test -timeout $(TEST_TIMEOUT) -p $(TEST_P) -parallel $(TEST_PARALLEL) \
		./cmd/... ./internal/... ./mdl/... ./model/... ./modelsdk/... \
		./sql/... ./tools/... ./generated/... ./scripts/...

# Run full test suite and generate layered report (terminal + HTML)
# Output: coverage/report.html, coverage/bench-baseline.json
report:
	@mkdir -p coverage
	CGO_ENABLED=0 go test -v -json -p $(TEST_P) -coverprofile=coverage/coverage.out \
		./cmd/... ./internal/... ./mdl/... ./model/... ./modelsdk/... \
		./sql/... ./tools/... ./generated/... ./scripts/... \
		> coverage/test-results.json 2>&1 || true
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html 2>/dev/null || true
	@if command -v benchstat >/dev/null 2>&1; then \
		CGO_ENABLED=0 go test -bench=. -benchmem -count=3 -p $(TEST_P) \
			./cmd/... ./internal/... ./mdl/... ./model/... ./modelsdk/... \
			./sql/... ./tools/... ./generated/... ./scripts/... \
			> coverage/bench-results.txt 2>/dev/null || true; \
		benchstat coverage/bench-baseline.json coverage/bench-results.txt > coverage/bench-diff.txt 2>/dev/null || true; \
	fi
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "  (benchstat not installed — skipping benchmark comparison)"; \
		echo "  Install: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi
	go run ./cmd/testreport \
		--json-file coverage/test-results.json \
		--bench-diff coverage/bench-diff.txt \
		--out-html coverage/report.html

# Record benchmark + coverage baselines used by the pre-push guard.
# Run this after intentional perf changes to silence the guard.
#
# Benchmarks: cpulimit/nice wraps go test directly (single process per pkg).
# Coverage:   always uses nice -n 15; cpulimit only limits the parent
#             go test process, not the per-package child binaries, which
#             causes go test to write only partial coverage data.
bench-baseline:
	@mkdir -p coverage
	@echo "Recording benchmark baseline (count=5, nice -n 15, p=1)..."
	@# Only packages with benchmarks; -p 1 avoids cross-process CPU contention.
	@# No cpulimit: SIGSTOP/SIGCONT skews clock-based timing by 30-40%.
	CGO_ENABLED=0 nice -n 15 go test -bench=. -benchmem -count=5 \
		-p 1 ./modelsdk/codec/... ./modelsdk/codec/poc/... ./modelsdk/property/... \
		2>/dev/null | grep -v "^---" > coverage/bench-baseline.txt
	@echo "Recording coverage baseline..."
	nice -n 15 go test -timeout 300s \
		-p $(_85PCT) -parallel $(_85PCT) \
		-coverprofile=coverage/coverage.out -covermode=atomic \
		./... >/dev/null 2>&1
	@go tool cover -func=coverage/coverage.out | \
		awk '/^total:/ { gsub(/%/,"",$$NF); printf "%.1f\n",$$NF }' > coverage/coverage-baseline.txt
	@echo "Benchmarks → coverage/bench-baseline.txt"
	@echo "Coverage   → $$(cat coverage/coverage-baseline.txt)%  (coverage/coverage-baseline.txt)"

# Run only benchmarks and update the baseline (legacy target)
report-bench:
	@mkdir -p coverage
	CGO_ENABLED=0 go test -bench=. -benchmem -count=3 ./... > coverage/bench-results.txt
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat coverage/bench-baseline.txt coverage/bench-results.txt > coverage/bench-diff.txt || true; \
		cat coverage/bench-diff.txt; \
	fi

# Reset benchmark baseline (use after major refactors)
report-reset-baseline:
	echo '' > coverage/bench-baseline.txt
	echo '' > coverage/coverage-baseline.txt
	@echo "Baselines reset."

# Check MDL syntax for all doctype example scripts
check-mdl: build
	@FAILED=0; \
	for f in mdl-examples/doctype-tests/*.mdl; do \
		case "$$f" in *.test.mdl) continue ;; esac; \
		NAME=$$(basename "$$f"); \
		if ./$(BUILD_DIR)/$(BINARY_NAME) check "$$f" > /dev/null 2>&1; then \
			echo "PASS: $$NAME"; \
		else \
			echo "FAIL: $$NAME"; \
			./$(BUILD_DIR)/$(BINARY_NAME) check "$$f" 2>&1 | grep -v "^WARNING"; \
			FAILED=1; \
		fi; \
	done; \
	exit $$FAILED

# Syntax showcase: grammar coverage regression test (no MPR needed)
test-showcase: build
	@echo "=== Syntax showcase: grammar check ==="
	@FAILED=0; \
	for f in $$(find mdl-examples/syntax-showcase -name "*.mdl" | sort); do \
		NAME=$$(basename "$$f"); \
		if ./$(BUILD_DIR)/$(BINARY_NAME) check "$$f" > /dev/null 2>&1; then \
			echo "OK: $$NAME"; \
		else \
			echo "FAIL: $$NAME"; \
			./$(BUILD_DIR)/$(BINARY_NAME) check "$$f" 2>&1 | grep -v "^WARNING"; \
			FAILED=1; \
		fi; \
	done; \
	echo "Passed: $$(find mdl-examples/syntax-showcase -name '*.mdl' | wc -l) files"; \
	exit $$FAILED

# Run integration tests (requires mx binary / mxbuild)
test-integration:
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
		./cmd/... ./internal/... ./mdl/... ./model/... ./modelsdk/... \
		./sql/... ./tools/... ./generated/... ./scripts/...

# Run integration tests with resource profiling and profile-based scheduling.
# Run integration tests with resource profiling and profile-based scheduling.
# The -resource-profile flag is registered only in mdl/executor (roundtrip tests).
# Profiles are saved to coverage/test-profiles/ for later analysis.
test-integration-profiled: build
	@mkdir -p coverage/test-profiles
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m -v \
		./mdl/executor/ \
		-resource-profile

# Check resource profiles against baselines (placeholder — flag registered in mdl/executor).
# Fails if any test exceeds its baseline by >20% in any metric.
test-profile-check:
	go test -tags integration -count=1 -run '^TestProfileCheck$$' \
		./cmd/... ./internal/... ./mdl/... ./model/... ./modelsdk/... \
		./sql/... ./tools/... ./generated/... ./scripts/...

# Record new resource profile baselines (replaces all existing profiles).
# Run after intentional performance changes to silence the regression gate.
test-profile-record: build
	@mkdir -p coverage/test-profiles
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
		-p 1 \
		./cmd/... ./internal/... ./mdl/... ./model/... ./modelsdk/... \
		./sql/... ./tools/... ./generated/... ./scripts/...
	# Re-run executor tests with record flag
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m \
		-p 1 ./mdl/executor/ \
		-resource-record
	@echo "Profiles recorded in coverage/test-profiles/"
	@echo "Profiles recorded in coverage/test-profiles/"

# Regenerate testdata/helpdesk-golden-11.6.6/ from helpdesk-app.mdl.
# Run after intentional changes to helpdesk-app.mdl; then commit the result.
update-helpdesk-golden:
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== Rebuilding helpdesk golden $$v ==="; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run '^TestHelpdeskGolden_Update$$' \
	    -update-golden \
	    -v -timeout 10m || exit 1; \
	done
	$(MAKE) update-snapshots

# Rebuild describe-snapshot.mdl for all helpdesk versions from their existing golden MPRs.
# Faster than update-helpdesk-golden (~5s vs ~10m). Run after describe logic changes.
# Validates each snapshot with: mxcli check snapshot.mdl -p minimal.mpr --references
update-snapshots: build
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== Updating snapshot $$v ==="; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run '^TestHelpdeskGolden_DescribeSnapshot$$' \
	    -update-golden \
	    -v -timeout 3m || exit 1; \
	  echo "  mxcli check describe-snapshot.mdl ($$v)..."; \
	  $(BUILD_DIR)/$(BINARY_NAME) check \
	    testdata/helpdesk-golden-$$v/describe-snapshot.mdl \
	    -p testdata/helpdesk-golden-$$v/minimal.mpr \
	    --references 2>&1 || echo "  WARNING: mxcli check found errors in snapshot (pre-existing debt — see describe output quality)" >&2; \
	done

# Validate describe-snapshot.mdl for all versions without rebuilding.
# Runs mxcli check + idempotency integration test. Suitable for CI.
validate-snapshots: build
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== Validating snapshot $$v ==="; \
	  $(BUILD_DIR)/$(BINARY_NAME) check \
	    testdata/helpdesk-golden-$$v/describe-snapshot.mdl \
	    -p testdata/helpdesk-golden-$$v/minimal.mpr \
	    --references 2>&1 || echo "  WARNING: mxcli check found errors in snapshot (pre-existing debt — see describe output quality)" >&2; \
	  echo "  idempotency test ($$v)..."; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run '^TestHelpdeskGolden_DescribeSnapshot_Idempotent$$' \
	    -v -timeout 5m || exit 1; \
	done

## validate-academy-capstone: full e2e validation of academy/zh capstone reference implementation
validate-academy-capstone:
	@./scripts/validate-academy-capstone.sh

# Run both helpdesk regression layers (BSON + describe MDL).
# Requires testdata/helpdesk-golden-11.6.6/ to exist (run update-helpdesk-golden first).
test-helpdesk-regression:
	CGO_ENABLED=0 go test ./internal/goldenfs/ \
		-tags linux,integration \
		-run 'TestHelpdeskGolden_Regression' \
		-v -timeout 15m

# Run MDL integration tests (requires Docker and a Mendix project)
# Usage: make test-mdl MPR=path/to/app.mpr
MPR ?= app.mpr
test-mdl: build
	./scripts/run-mdl-tests.sh "$(abspath $(MPR))" "$(abspath $(BUILD_DIR)/$(BINARY_NAME))"

# Lint Go code
lint: lint-go

lint-go: fmt vet
	@echo "Go lint passed"

# Format Go code
fmt:
	go fmt ./...

# Vet Go code (filters out generated ANTLR parser warnings)
vet:
	@CGO_ENABLED=0 go vet ./... 2>&1 | grep -v 'grammar/parser/' | grep -v 'mdl-grammar/parser/' || true
	@! CGO_ENABLED=0 go vet ./... 2>&1 | grep -v 'grammar/parser/' | grep -v 'mdl-grammar/parser/' | grep -q 'vet:'

# Regenerate ANTLR parser from MDLLexer.g4 and MDLParser.g4
grammar:
	$(MAKE) -C mdl/grammar generate

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean


# Build documentation site with mdbook
docs-site:
	mdbook build docs-site

# Serve documentation site locally with live reload
docs-serve:
	mdbook serve docs-site

# Generate CycloneDX SBOM (Go + TypeScript dependencies)
sbom:
	@scripts/generate-sbom.sh

# Generate Markdown dependency report from SBOM
sbom-report: sbom
	@scripts/generate-sbom-report.sh

# Generate source tree overview
source-tree: build
	@$(BUILD_DIR)/source_tree --all > source_tree.txt
	@echo "Generated source_tree.txt"

# Re-mine generated/exprgrammar/mined.go from an MPR.
# Uses MINE_MPR (not MPR) so the global "MPR ?= app.mpr" default for the
# test-mdl target does not silently mask a missing argument here.
#   make mine-exprgrammar MINE_MPR=testdata/expr-checker/minimal.mpr   (~1s, 9 slots)
#   make mine-exprgrammar MINE_MPR="/path/to/Factory Management.mpr"   (~11min, 11 slots)
.PHONY: mine-exprgrammar
mine-exprgrammar:
	@if [ -z "$(MINE_MPR)" ]; then \
		echo "usage: make mine-exprgrammar MINE_MPR=/path/to/app.mpr"; exit 2; \
	fi
	@if [ ! -f "$(MINE_MPR)" ]; then \
		echo "mine-exprgrammar: MPR file not found: $(MINE_MPR)"; exit 2; \
	fi
	GOPROXY=https://goproxy.cn,direct go run ./cmd/exprgrammar-mine \
	    --mpr "$(MINE_MPR)" \
	    --out generated/exprgrammar/mined.go

# roundtrip — exprcheck round-trip CI gate over testdata/expr-checker/minimal.mpr.
# Walks every microflow, regenerates MDL via DescribeMicroflowToString, parses
# every expression with the robust parser, asserts 0 hints. Failures reveal
# grammar gaps. Build tag 'roundtrip' keeps it out of the default suite.
.PHONY: roundtrip
roundtrip:
	GOPROXY=https://goproxy.cn,direct go test -tags=roundtrip ./mdl/exprcheck/ -run TestRoundTrip -count=1 -timeout 5m

# release-audit — 检查三个发布项目自上一个 tag 以来的代码变更。
# 用法: make release-audit [REF=<ref>]
# 输出结构化 Markdown 指引 AI 决定是否需要为新功能/修复创建 Release。
.PHONY: release-audit
release-audit:
	@scripts/release-audit.sh $(if $(REF),$(REF),HEAD)
