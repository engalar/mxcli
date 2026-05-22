# Makefile for ModelSDKGo
#
# Usage:
#   make build     - Build mxcli for current platform
#   make release   - Build mxcli for all platforms (macOS, Windows, Linux)
#   make test      - Run unit tests
#   make check-mdl - Check MDL syntax for all doctype example scripts
#   make test-integration - Run integration tests (requires mx/mxbuild)
#   make test-mdl  - Run MDL integration tests (requires Docker)
#   make lint      - Lint all code (Go + TypeScript)
#   make lint-go   - Lint Go code (fmt + vet)
#   make lint-ts   - Lint TypeScript code (tsc --noEmit)
#   make grammar   - Regenerate ANTLR parser
#   make docs-site - Build documentation site (mdbook)
#   make docs-serve - Serve docs site locally with live reload
#   make sbom      - Generate CycloneDX SBOM (Go + TypeScript)
#   make sbom-report - Generate Markdown dependency report
#   make mine-exprgrammar MINE_MPR=path/to/app.mpr - Re-mine generated/exprgrammar/mined.go from an MPR
#   make clean     - Remove build artifacts

BINARY_NAME = mxcli
BUILD_DIR = bin
CMD_PATH = ./cmd/mxcli

# Version info (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Clean version for VS Code extension (must be valid semver: major.minor.patch)
VSCE_VERSION = $(shell echo "$(VERSION)" | sed 's/^v//; s/-.*//' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$$' || echo "0.0.0")

.PHONY: build build-debug release clean test test-mdl report report-bench report-reset-baseline grammar completions sync-skills sync-commands sync-lint-rules sync-changelog sync-all docs documentation docs-site docs-serve source-tree sbom sbom-report lint lint-go fmt vet update-helpdesk-golden test-helpdesk-regression

# Helper: copy file only if content differs (avoids mtime updates that invalidate go build cache)
# Usage: $(call copy-if-changed,src,dst)
define copy-if-changed
	@if [ ! -f $(2) ] || ! cmp -s $(1) $(2); then cp $(1) $(2); fi
endef

# Sync skills from .claude/skills/mendix to cmd/mxcli/skills for embedding
sync-skills:
	@mkdir -p cmd/mxcli/skills
	@changed=0; for f in .claude/skills/mendix/*.md; do \
		dst="cmd/mxcli/skills/$$(basename $$f)"; \
		if [ ! -f "$$dst" ] || ! cmp -s "$$f" "$$dst"; then \
			cp "$$f" "$$dst"; changed=$$((changed + 1)); \
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

# Sync skills, commands, lint rules, and changelog
sync-all: sync-skills sync-commands sync-lint-rules sync-changelog

# Generate LSP completion items from grammar (only rewrites file if content changed)
completions:
	@CGO_ENABLED=0 go run ./cmd/gen-completions -lexer mdl/grammar/MDLLexer.g4 -output cmd/mxcli/lsp_completions_gen.go.tmp
	@if [ ! -f cmd/mxcli/lsp_completions_gen.go ] || ! cmp -s cmd/mxcli/lsp_completions_gen.go.tmp cmd/mxcli/lsp_completions_gen.go; then \
		mv cmd/mxcli/lsp_completions_gen.go.tmp cmd/mxcli/lsp_completions_gen.go; \
		echo "Updated cmd/mxcli/lsp_completions_gen.go"; \
	else \
		rm cmd/mxcli/lsp_completions_gen.go.tmp; \
	fi

# Build for current platform (auto-syncs skills and commands)
build: sync-all completions
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/source_tree ./cmd/source_tree
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/source_tree"

# Build with debug tools (includes bson discover/compare/dump)
build-debug: sync-all completions
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags debug $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-debug (debug build with bson tools)"

# Build for all platforms (CGO_ENABLED=0 for cross-compilation)
release: clean sync-all
	@mkdir -p $(BUILD_DIR)
	@echo "Building release binaries..."

	@echo "  -> Linux (amd64)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

	@echo "  -> Linux (arm64)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

	@echo "  -> macOS (amd64 - Intel)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)

	@echo "  -> macOS (arm64 - Apple Silicon)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

	@echo "  -> Windows (amd64)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

	@echo "  -> Windows (arm64)"
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_PATH)

	@echo ""
	@echo "Release binaries:"
	@ls -lh $(BUILD_DIR)/

# Run tests
test:
	CGO_ENABLED=0 go test ./...

# Run full test suite and generate layered report (terminal + HTML)
# Output: coverage/report.html, coverage/bench-baseline.json
report:
	@mkdir -p coverage
	CGO_ENABLED=0 go test -v -json -coverprofile=coverage/coverage.out ./... > coverage/test-results.json 2>&1 || true
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html 2>/dev/null || true
	@if command -v benchstat >/dev/null 2>&1; then \
		CGO_ENABLED=0 go test -bench=. -benchmem -count=3 ./... > coverage/bench-results.txt 2>/dev/null || true; \
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

# Run only benchmarks and update the baseline
report-bench:
	@mkdir -p coverage
	CGO_ENABLED=0 go test -bench=. -benchmem -count=3 ./... > coverage/bench-results.txt
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat coverage/bench-baseline.json coverage/bench-results.txt > coverage/bench-diff.txt || true; \
		cat coverage/bench-diff.txt; \
	fi

# Reset benchmark baseline (use after major refactors)
report-reset-baseline:
	echo '[]' > coverage/bench-baseline.json
	@echo "Baseline reset."

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

# Run integration tests (requires mx binary / mxbuild)
test-integration:
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m ./...

# Regenerate testdata/helpdesk-golden/ from helpdesk-app.mdl.
# Run after intentional changes to helpdesk-app.mdl; then commit the result.
update-helpdesk-golden:
	CGO_ENABLED=0 go test ./internal/goldenfs/ \
		-tags linux,integration \
		-run TestHelpdeskGolden_Update \
		-update-golden \
		-v -timeout 10m

# Run both helpdesk regression layers (BSON + describe MDL).
# Requires testdata/helpdesk-golden/ to exist (run update-helpdesk-golden first).
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

# Lint all code (Go + TypeScript)
lint: lint-go lint-ts

# Lint Go code
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


# Generate documentation from ANTLR4 grammar
docs: documentation
documentation:
	@echo "Generating MDL grammar documentation..."
	@mkdir -p docs/06-mdl-reference
	@CGO_ENABLED=0 go run ./cmd/grammardoc \
		-grammar mdl/grammar/MDLParser.g4 \
		-lexer mdl/grammar/MDLLexer.g4 \
		-output docs/06-mdl-reference/grammar-reference.md \
		-title "MDL Grammar Reference"
	@echo "Documentation generated at docs/06-mdl-reference/grammar-reference.md"

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

# expr-hints-md — regenerates docs/06-mdl-reference/expr-hints.md from the
# canonical hint registry in mdl/exprcheck/hints/registry.go.
.PHONY: expr-hints-md
expr-hints-md:
	GOPROXY=https://goproxy.cn,direct go run ./cmd/expr-hints-md
