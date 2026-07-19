// Package golden_test contains the MDL→BSON integration test harness.
// It verifies that running MDL through the real executor/backend/writer
// pipeline produces BSON identical to golden snapshots.
package golden_test

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/modelsdk/codec/golden"
	"github.com/mendixlabs/mxcli/modelsdk/codec/golden/memory"
	"github.com/mendixlabs/mxcli/modelsdk/codec/golden/snapshot"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type mdlGoldenCase struct {
	Name           string
	SetupMDL       string
	MDL            string
	GoldenFile     string // path relative to testdata/
	TargetUnitType string
}

func goldenCases() []mdlGoldenCase {
	return []mdlGoldenCase{
		{
			Name: "nanoflow/create",
			SetupMDL: `create module Administration;
create entity Administration.Account (FullName: String(200));`,
			MDL: `create nanoflow Administration.AdminNanoflow () returns list of Administration.Account as $AccountList
{
  synchronize;
  retrieve $AccountList from Administration.Account
    where [FullName = empty and System.owner = '[%CurrentUser%]']
    sort by FullName desc;
  return $AccountList;
}`,
			GoldenFile:     "nanoflow/create.golden.json",
			TargetUnitType: "Microflows$Nanoflow",
		},
	}
}

func readUnitBSONByType(db *sql.DB, unitType string) ([]byte, error) {
	rows, err := db.Query(`SELECT Contents FROM Unit`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var contents []byte
		if err := rows.Scan(&contents); err != nil {
			continue
		}
		if len(contents) < 5 {
			continue
		}
		var doc bson.D
		if err := bson.Unmarshal(contents, &doc); err != nil {
			continue
		}
		for _, elem := range doc {
			if elem.Key == "$Type" {
				if typeStr, ok := elem.Value.(string); ok && typeStr == unitType {
					return contents, nil
				}
			}
		}
	}
	return nil, sql.ErrNoRows
}

func TestMDLToGolden(t *testing.T) {
	for _, tc := range goldenCases() {
		t.Run(tc.Name, func(t *testing.T) {
			// Create a temp MPR file for the backend to connect to.
			tmpDir := t.TempDir()
			mprPath := filepath.Join(tmpDir, "test.mpr")

			// Use memory.NewFile to create the MPR with correct schema.
			memMPR, err := memory.NewFile(mprPath, "11.12.1")
			if err != nil {
				t.Fatalf("memory.NewFile: %v", err)
			}
			memMPR.Close()

			// Open backend on the temp MPR.
			backend := mprbackend.New()
			if err := backend.Connect(mprPath); err != nil {
				t.Fatalf("backend.Connect: %v", err)
			}

			// Create executor and set backend.
			output := &bytes.Buffer{}
			exec := executor.New(output)
			exec.SetBackend(backend)

			// Execute setup MDL.
			if tc.SetupMDL != "" {
				if err := executeMDL(exec, tc.SetupMDL); err != nil {
					t.Fatalf("setup MDL: %v", err)
				}
			}

			// Execute main MDL.
			if err := executeMDL(exec, tc.MDL); err != nil {
				t.Fatalf("main MDL: %v", err)
			}

			// Disconnect backend so SQLite file is released for re-opening.
			backend.Disconnect()

			// Read BSON back from the MPR file.
			readDB, err := sql.Open("sqlite", mprPath)
			if err != nil {
				t.Fatalf("open sqlite for readback: %v", err)
			}
			defer readDB.Close()

			bsonData, err := readUnitBSONByType(readDB, tc.TargetUnitType)
			if err != nil {
				t.Fatalf("readUnitBSONByType: %v", err)
			}

			// Handle GOLDEN_WRITE=1 mode.
			goldenPath := filepath.Join("testdata", tc.GoldenFile)
			if os.Getenv("GOLDEN_WRITE") == "1" {
				snap, err := snapshot.NewUnitSnapshot(tc.TargetUnitType, bsonData)
				if err != nil {
					t.Fatalf("NewUnitSnapshot: %v", err)
				}
				if err := snapshot.WriteSnapshotToFile(snap, goldenPath); err != nil {
					t.Fatalf("WriteSnapshotToFile: %v", err)
				}
				t.Logf("Wrote golden file: %s", goldenPath)
				return
			}

			// Load golden.
			goldenSnap, err := snapshot.ReadSnapshotFromFile(goldenPath)
			if err != nil {
				t.Fatalf("LoadGolden: %v", err)
			}

			// Compare binary — strict byte-level BSON comparison.
			result, err := snapshot.CompareCanonical(bsonData, goldenSnap, snapshot.CompareBinary)
			if err != nil {
				t.Fatalf("CompareCanonical: %v", err)
			}
			for _, d := range result.Diffs {
				t.Errorf("%s", golden.FormatDiff(d))
			}
		})
	}
}

func executeMDL(exec *executor.Executor, mdl string) error {
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		return errs[0]
	}
	return exec.ExecuteProgram(prog)
}
