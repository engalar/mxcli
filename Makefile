# Makefile for ModelSDKGo
#
# Usage:
#   make build     - Build mxcli for current platform
#   make release   - Build mxcli for all platforms (macOS, Windows, Linux)
#   make test      - Run unit tests
#   make test-integration - Run integration tests (requires mx/mxbuild)
#   make test-mdl  - Run MDL integration tests (requires Docker)
#   make lint      - Lint all code (Go + TypeScript)
#   make lint-go   - Lint Go code (fmt + vet)
#   make lint-ts   - Lint TypeScript code (tsc --noEmit)
#   make grammar   - Regenerate ANTLR parser
#   make sbom      - Generate CycloneDX SBOM (Go + TypeScript)
#   make sbom-report - Generate Markdown dependency report
#   make clean     - Remove build artifacts

BINARY_NAME = mxcli
BUILD_DIR = bin
CMD_PATH = ./cmd/mxcli

# Version info (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: build release clean test test-mdl grammar completions sync-skills sync-commands sync-lint-rules sync-changelog sync-all docs documentation vscode-ext vscode-install source-tree sbom sbom-report lint lint-go lint-ts fmt vet

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

# Sync VS Code extension (.vsix) for embedding
sync-vsix:
	@if [ -f vscode-mdl/vscode-mdl-*.vsix ]; then \
		src=$$(ls vscode-mdl/vscode-mdl-*.vsix | head -1); \
		if [ ! -f cmd/mxcli/vscode-mdl.vsix ] || ! cmp -s "$$src" cmd/mxcli/vscode-mdl.vsix; then \
			cp "$$src" cmd/mxcli/vscode-mdl.vsix; \
			echo "Synced vscode-mdl.vsix"; \
		fi; \
	elif [ ! -f cmd/mxcli/vscode-mdl.vsix ]; then \
		echo "Warning: No .vsix found. Creating empty placeholder."; \
		touch cmd/mxcli/vscode-mdl.vsix; \
	fi

# Sync changelog to cmd/mxcli for embedding
sync-changelog:
	$(call copy-if-changed,CHANGELOG.md,cmd/mxcli/changelog.md)

# Sync skills, commands, lint rules, and changelog
sync-all: sync-skills sync-commands sync-lint-rules sync-vsix sync-changelog

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

# Run integration tests (requires mx binary / mxbuild)
test-integration:
	CGO_ENABLED=0 go test -tags integration -count=1 -timeout 30m ./...

# Run MDL integration tests (requires Docker and a Mendix project)
# Usage: make test-mdl MPR=path/to/app.mpr
MPR ?= app.mpr
test-mdl: build
	$(BUILD_DIR)/$(BINARY_NAME) test mdl-examples/doctype-tests/microflow-spec.test.mdl -p $(MPR)

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

# Lint TypeScript code (VS Code extension)
lint-ts:
	cd vscode-mdl && bun install --silent && bun run lint
	@echo "TypeScript lint passed"

# Regenerate ANTLR parser from MDLLexer.g4 and MDLParser.g4
grammar:
	$(MAKE) -C mdl/grammar generate

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Build VS Code extension (.vsix) with build-time version info
vscode-ext:
	@echo "Building VS Code extension..."
	cd vscode-mdl && bun install && \
		bunx esbuild src/extension.ts --bundle --outfile=dist/extension.js \
			--external:vscode --format=cjs --platform=node \
			--define:__BUILD_TIME__="'$(BUILD_TIME)'" \
			--define:__GIT_COMMIT__="'$(VERSION)'" && \
		bunx @vscode/vsce package --no-dependencies
	@echo "Built vscode-mdl/$$(ls vscode-mdl/*.vsix)"

# Install VS Code extension
vscode-install: vscode-ext
	code --install-extension vscode-mdl/vscode-mdl-*.vsix
	@echo "Extension installed. Reload VS Code to activate."

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
