# CE0066 $ID Encoding Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the gen-native encoder so new AccessRule/MemberAccess elements write a proper BSON Binary `$ID` instead of an empty string, preventing `InvalidCastException` in `mx check`.

**Architecture:** Single-point fix in `modelsdk/codec/encoder.go` — `idToBinarySubtype0("")` currently returns `""` (empty string); change it to generate a fresh UUID so every new element gets a Binary `$ID` regardless of call site. Two regression tests gate the fix: one at the encoder layer, one at the backend integration layer.

**Tech Stack:** Go, `go.mongodb.org/mongo-driver/bson`, `modelsdk/mpr.GenerateID()`, `modelsdk/mpr.IDToBsonBinary()`

---

### Task 1: Failing encoder-level test for empty-ID new element

**Files:**
- Modify: `modelsdk/codec/encoder_test.go`

The existing `TestEncoderNewElementHasBinaryID` only tests an element that already has an ID set. We need a test that catches the bug: a new element with no ID assigned must still get a Binary `$ID` in the output.

- [ ] **Step 1: Add the failing test to `modelsdk/codec/encoder_test.go`**

Open `modelsdk/codec/encoder_test.go`. After the existing `TestEncoderNewElementHasBinaryID` function, add:

```go
func TestEncoderNewElementWithNoIDHasBinaryID(t *testing.T) {
	// New element with no ID assigned (the bug case: NewAccessRule() leaves id="").
	elem := &element.Base{}
	elem.SetTypeName("Test$NoID")
	elem.MarkDirty(63) // new element flag

	enc := &Encoder{}
	out, err := enc.Encode(elem)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	raw := bson.Raw(out)
	idVal, err := raw.LookupErr("$ID")
	if err != nil {
		t.Fatalf("$ID field missing from encoded output")
	}
	if idVal.Type.String() != "binary" {
		t.Errorf("$ID type = %q, want \"binary\" — new element with no pre-set ID wrote a non-binary $ID", idVal.Type)
	}
	// Also confirm the binary is 16 bytes (a real UUID, not zero-length).
	bin := idVal.Binary()
	if len(bin.Data) != 16 {
		t.Errorf("$ID binary length = %d, want 16", len(bin.Data))
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./modelsdk/codec/... -run TestEncoderNewElementWithNoIDHasBinaryID -v
```

Expected output (FAIL):
```
--- FAIL: TestEncoderNewElementWithNoIDHasBinaryID
    encoder_test.go:NN: $ID type = "string", want "binary"
```

---

### Task 2: Fix `idToBinarySubtype0` in the encoder

**Files:**
- Modify: `modelsdk/codec/encoder.go:153-158`

- [ ] **Step 3: Apply the fix**

In `modelsdk/codec/encoder.go`, replace the `idToBinarySubtype0` function:

**Before (lines 151–158):**
```go
// idToBinarySubtype0 converts a UUID string to BSON Binary subtype 0
// using the Mendix byte-swap convention (via sdk/mpr.IDToBsonBinary).
func idToBinarySubtype0(id element.ID) any {
	if id == "" {
		return id
	}
	return mpr.IDToBsonBinary(string(id))
}
```

**After:**
```go
// idToBinarySubtype0 converts a UUID string to BSON Binary subtype 0
// using the Mendix byte-swap convention (via sdk/mpr.IDToBsonBinary).
// When id is empty (new element, no ID assigned), a fresh UUID is generated
// so the BSON always carries a valid Binary $ID.
func idToBinarySubtype0(id element.ID) any {
	if id == "" {
		return mpr.IDToBsonBinary(mpr.GenerateID())
	}
	return mpr.IDToBsonBinary(string(id))
}
```

- [ ] **Step 4: Run the new test to confirm it passes**

```bash
go test ./modelsdk/codec/... -run TestEncoderNewElementWithNoIDHasBinaryID -v
```

Expected:
```
--- PASS: TestEncoderNewElementWithNoIDHasBinaryID (0.00s)
PASS
```

- [ ] **Step 5: Run the full encoder test suite to confirm no regressions**

```bash
go test ./modelsdk/codec/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add modelsdk/codec/encoder.go modelsdk/codec/encoder_test.go
git commit -m "fix(codec): generate UUID for new elements with no $ID instead of writing empty string"
```

---

### Task 3: Add backend integration regression test for Binary `$ID`

**Files:**
- Modify: `mdl/backend/mpr/security_entity_access_gen_test.go`

The encoder fix is validated. Now lock in the end-to-end BSON shape at the backend layer: after `addEntityAccessRuleViaModelsdk` runs, every `$ID` in the written BSON (AccessRule + all its MemberAccesses) must be a `primitive.Binary`, not a string.

- [ ] **Step 7: Add the `allIDsAreBinary` helper + regression test**

Open `mdl/backend/mpr/security_entity_access_gen_test.go`. Add after the existing `countVersionedEntries` helper (around line 176):

```go
// allIDsAreBinary recursively walks a bson.D and returns the path of any $ID
// field whose value is not primitive.Binary. Returns nil if all $ID fields are
// Binary (the passing case).
func allIDsAreBinary(doc bson.D, prefix string) []string {
	var bad []string
	for _, f := range doc {
		path := prefix + "." + f.Key
		if f.Key == "$ID" {
			if _, ok := f.Value.(primitive.Binary); !ok {
				bad = append(bad, fmt.Sprintf("%s = %T(%v)", path, f.Value, f.Value))
			}
			continue
		}
		switch v := f.Value.(type) {
		case bson.D:
			bad = append(bad, allIDsAreBinary(v, path)...)
		case bson.A:
			for i, item := range v {
				if sub, ok := item.(bson.D); ok {
					bad = append(bad, allIDsAreBinary(sub, fmt.Sprintf("%s[%d]", path, i))...)
				}
			}
		}
	}
	return bad
}
```

Then add the following test after `TestAddEntityAccessRuleViaModelsdk_GenNative`:

```go
// TestAddEntityAccessRuleViaModelsdk_BinaryID is a regression guard for the
// empty-string $ID bug: when NewAccessRule/NewMemberAccess creates elements
// with no pre-set ID, the encoder must auto-generate a UUID and write it as
// BSON Binary — not as an empty string that causes InvalidCastException in
// Studio Pro / mx check.
func TestAddEntityAccessRuleViaModelsdk_BinaryID(t *testing.T) {
	b, dmID := seedEntityForAccessTest(t)

	if err := b.addEntityAccessRuleViaModelsdk(
		dmID, "Customer",
		[]string{"TestModule.UserRole"},
		true, false,
		"ReadWrite", "",
		[]mpr.EntityMemberAccess{
			{AttributeRef: "TestModule.Customer.Name", AccessRights: "ReadWrite"},
		},
	); err != nil {
		t.Fatalf("addEntityAccessRuleViaModelsdk: %v", err)
	}

	// Read back the raw DomainModel BSON and scan every $ID field.
	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(dmID))
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if bad := allIDsAreBinary(doc, "DomainModel"); len(bad) > 0 {
		t.Errorf("found non-Binary $ID fields (Studio Pro will crash with InvalidCastException):\n  %s",
			strings.Join(bad, "\n  "))
	}
}
```

- [ ] **Step 8: Add missing imports to the test file**

Check the top of `mdl/backend/mpr/security_entity_access_gen_test.go`. It currently imports:
```go
import (
    "testing"
    "go.mongodb.org/mongo-driver/bson"
    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/sdk/domainmodel"
    "github.com/mendixlabs/mxcli/sdk/mpr"
)
```

Add the missing imports:
```go
import (
    "fmt"
    "strings"
    "testing"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"

    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/sdk/domainmodel"
    "github.com/mendixlabs/mxcli/sdk/mpr"
)
```

- [ ] **Step 9: Run the new backend test to confirm it passes**

```bash
go test ./mdl/backend/mpr/... -run TestAddEntityAccessRuleViaModelsdk_BinaryID -v
```

Expected:
```
--- PASS: TestAddEntityAccessRuleViaModelsdk_BinaryID (0.00s)
PASS
```

- [ ] **Step 10: Run the full backend mpr test suite**

```bash
go test ./mdl/backend/mpr/... -v 2>&1 | tail -20
```

Expected: all tests PASS.

- [ ] **Step 11: Commit**

```bash
git add mdl/backend/mpr/security_entity_access_gen_test.go
git commit -m "test(backend): regression guard — AccessRule/MemberAccess $ID must be Binary after addEntityAccessRuleViaModelsdk"
```

---

### Task 4: Verify with mx check on macnica

**Files:** none (verification only)

- [ ] **Step 12: Run a GRANT against a fresh copy of macnica**

```bash
# Copy the original backup so the mxunit files are in known-good state
cp /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr.v1.bak \
   /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr 2>/dev/null || true
```

Then run a GRANT that exercises the fixed path:

```bash
go run ./cmd/mxcli -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr \
  -c "grant ContractorRegistration.User on ContractorRegistration.ContractorApplication (create, read *, write *);"
```

Expected:
```
Granted access on ContractorRegistration.ContractorApplication to ContractorRegistration.User
```

- [ ] **Step 13: Run mx check**

```bash
timeout 120 /home/claude_dev/.mxcli/mxbuild/11.6.6/modeler/mx check \
  /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr 2>&1 | head -40
```

**Pass criterion:** No `InvalidCastException` and no `GetGuidFromBson` error.

If CE0066 still appears, capture the entity name and open a Layer 2 investigation
(compare that entity's BSON MemberAccesses against a Studio Pro-created access rule
for the same entity — see spec section "Layer 2").

- [ ] **Step 14: Run full build + test suite**

```bash
make build && make test
```

Expected: no errors.

- [ ] **Step 15: Final commit if any cleanup is needed**

If Step 14 surfaced any issues, fix them and commit. Otherwise nothing to do.

---

## Self-Review

**Spec coverage:**
- ✅ Fix `idToBinarySubtype0` → Task 2, Step 3
- ✅ Encoder-level test for empty ID → Task 1
- ✅ Backend integration regression test → Task 3
- ✅ mx check verification → Task 4
- ✅ Deferred: Layer 2 CE0066 investigation if mx check still fails after fix

**Placeholder scan:** No TBD/TODO. All code blocks are complete.

**Type consistency:**
- `primitive.Binary` — correctly imported from `go.mongodb.org/mongo-driver/bson/primitive` ✓
- `mpr.GenerateID()` — in `modelsdk/mpr/utils.go:13` ✓
- `mpr.IDToBsonBinary(string)` — in `modelsdk/mpr/utils.go:31` ✓
- `allIDsAreBinary(doc bson.D, prefix string) []string` — defined in Task 3, used in same task ✓
- `bson.D`, `bson.A` — already imported in test file ✓
