# MDL Syntax Showcase

Grammar coverage test suite. Each file covers one or more MDL grammar production rules.

## Running

    make test-showcase

Or manually:

    find mdl-examples/syntax-showcase -name "*.mdl" | sort | \
        xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || exit 1'

## File Groups

| Prefix | Count | Content |
|--------|-------|---------|
| 00-setup | 1 | Shared module/entity/enum definitions |
| expr-01..19 | 19 | Mendix expression layer |
| xpath-01..06 | 6 | XPath constraint syntax |
| act-01..30 | 30 | Microflow activities |
| ctrl-01..13 | 13 | Control flow (legacy + modern {} forms) |
| ddl-01..10 | 10 | DDL layer |
