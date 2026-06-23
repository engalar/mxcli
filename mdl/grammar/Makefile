# Makefile for MDL grammar generation
#
# Usage:
#   make generate    - Generate Go parser from MDLLexer.g4 and MDLParser.g4
#   make clean       - Remove generated files
#
# Prerequisites:
#   - ANTLR4 must be installed (https://www.antlr.org/)
#   - macOS:  brew install antlr4
#   - Linux:  apt install antlr4  (or download JAR manually)
#   - Or use: pip install antlr4-tools

LEXER = MDLLexer.g4
PARSER = MDLParser.g4
OUTPUT_DIR = parser
PACKAGE = parser

# Domain grammar files (imported by MDLParser.g4 via the `import` directive)
DOMAIN_FILES = \
    domains/MDLDomainModel.g4 \
    domains/MDLMicroflow.g4 \
    domains/MDLPage.g4 \
    domains/MDLSecurity.g4 \
    domains/MDLAgent.g4 \
    domains/MDLWorkflow.g4 \
    domains/MDLService.g4 \
    domains/MDLCatalog.g4 \
    domains/MDLSettings.g4

# Find antlr4 command (try common locations)
ANTLR4 := $(shell which antlr4 2>/dev/null || which antlr 2>/dev/null)

.PHONY: generate clean check-antlr

check-antlr:
ifndef ANTLR4
	$(error ANTLR4 not found. Install with: brew install antlr4 (macOS) or pip install antlr4-tools)
endif

generate: check-antlr $(LEXER) $(PARSER) $(DOMAIN_FILES)
	@mkdir -p $(OUTPUT_DIR)
	bash -c '$(ANTLR4) -Dlanguage=Go -no-visitor -package $(PACKAGE) -lib domains -o $(OUTPUT_DIR) $(LEXER) $(PARSER)'
	@echo "Generated Go parser in $(OUTPUT_DIR)/"

clean:
	rm -rf $(OUTPUT_DIR)
