# Design Decisions

Key architectural decisions made during the development of ModelSDK Go, with rationale and trade-offs.

## Pure Go SQLite (No CGO)

**Decision:** Use `modernc.org/sqlite` instead of `mattn/go-sqlite3`.

**Rationale:** `mattn/go-sqlite3` requires CGO and a C compiler, complicating cross-compilation and deployment. The pure Go implementation works on all platforms without external dependencies.

**Trade-off:** Slightly lower SQLite performance compared to the C implementation. In practice, MPR files are small enough that this is not measurable.

## BSON with MongoDB Driver

**Decision:** Use `go.mongodb.org/mongo-driver` for BSON parsing.

**Rationale:** Mendix stores model elements as BSON documents. The MongoDB Go driver provides a mature, well-tested BSON implementation. No actual MongoDB connection is needed -- only the BSON codec is used.

## SQL-Like DSL (MDL)

**Decision:** Create a custom SQL-like language instead of using JSON, YAML, or a general-purpose language.

**Rationale:**
- SQL is familiar to most developers
- LLMs generate SQL-like syntax with high accuracy
- 5-10x more token-efficient than JSON representations
- Human-readable diffs are easier to review than binary or JSON changes
- High information density for scanning microflow logic

**Trade-off:** Requires building and maintaining a custom parser (ANTLR4).

## ANTLR4 for Parsing

**Decision:** Use ANTLR4 instead of hand-written parser or Go-native parser generators.

**Rationale:**
- Grammar is shareable across languages (Go, TypeScript, Java, Python)
- Well-defined grammar file serves as the language specification
- Rich tooling for grammar debugging and visualization
- Generated parser handles error recovery automatically

**Trade-off:** ANTLR4 Go target generates large files. Regeneration requires the ANTLR4 tool (`make grammar`).

## Storage Names in BSON

**Decision:** Always use `storageName` (not `qualifiedName`) for the `$Type` field in BSON.

**Rationale:** Studio Pro uses storage names internally. Using qualified names causes `TypeCacheUnknownTypeException`. The two names are often identical, but critical exceptions exist (e.g., `DomainModels$Entity` vs `DomainModels$EntityImpl`).

## Inverted Association Pointer Names

**Decision:** Preserve Mendix's counter-intuitive naming where `ParentPointer` points to the FROM entity and `ChildPointer` points to the TO entity.

**Rationale:** Changing the field names would create a mapping layer that hides the actual BSON structure, making debugging harder. By keeping the original names, developers can directly compare Go structs with raw BSON output.

**Trade-off:** Every developer working with associations must learn the inverted semantics. Documentation and comments are essential.

## Embedded Widget Templates

**Decision:** Embed widget templates as JSON files compiled into the binary via `go:embed`.

**Rationale:** Pluggable widgets require exact property schemas that change between Mendix versions. Embedding templates ensures the binary is self-contained and version-matched. Extracting templates from Studio Pro guarantees correctness.

**Trade-off:** New Mendix versions or widgets require extracting and adding new template files.

## Catalog as SQLite

**Decision:** Use an in-memory SQLite database for the project metadata catalog.

**Rationale:** SQLite provides indexing, FTS5 full-text search, and standard SQL querying without any external dependency. The catalog is rebuilt from the MPR on each session, so persistence is not needed.

## Fluent API Layer

**Decision:** Provide both a low-level SDK (direct struct construction) and a high-level fluent API (`api/` package).

**Rationale:** The low-level SDK gives full control over BSON structure, which is needed for complex operations. The fluent API reduces boilerplate for common tasks and makes test code more readable.

**Trade-off:** Two API surfaces to maintain and document.
