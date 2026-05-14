// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// fixturePath is the canonical Stage 4 read fixture: a v2 MPR with
// mprcontents/ folder, modules, and at least one microflow.
const treeFixturePath = "../../testdata/expr-checker/minimal.mpr"

// TestBuildProjectTree_MicroflowsAppear verifies the JSON tree built
// from the canonical fixture contains microflow nodes — this is the
// surface that A1 migrates from sdk/mpr.Reader.ListMicroflows to
// mprread.ListMicroflows; both implementations must produce the same
// observable shape (microflow leaves under their owning module).
func TestBuildProjectTree_MicroflowsAppear(t *testing.T) {
	dst := copyMPRTreeForTest(t, treeFixturePath, t.TempDir())
	tree, err := buildProjectTree(dst)
	if err != nil {
		t.Fatalf("buildProjectTree(%s): %v", dst, err)
	}
	if len(tree) == 0 {
		t.Fatal("buildProjectTree returned empty tree")
	}
	if !treeContainsType(tree, "microflow") {
		t.Errorf("project tree does not contain any node with Type=\"microflow\"")
	}
}

// treeContainsType walks the tree and reports whether any node has the
// given Type.
func treeContainsType(nodes []*TreeNode, want string) bool {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if n.Type == want {
			return true
		}
		if treeContainsType(n.Children, want) {
			return true
		}
	}
	return false
}

func copyMPRTreeForTest(t *testing.T, srcMPR, dstDir string) string {
	t.Helper()
	dstMPR := filepath.Join(dstDir, filepath.Base(srcMPR))
	if err := copyOneFileForTest(srcMPR, dstMPR); err != nil {
		t.Fatalf("copy %s -> %s: %v", srcMPR, dstMPR, err)
	}
	srcContents := filepath.Join(filepath.Dir(srcMPR), "mprcontents")
	if info, err := os.Stat(srcContents); err == nil && info.IsDir() {
		dstContents := filepath.Join(dstDir, "mprcontents")
		if err := copyDirForTest(srcContents, dstContents); err != nil {
			t.Fatalf("copy contents %s -> %s: %v", srcContents, dstContents, err)
		}
	}
	return dstMPR
}

func copyOneFileForTest(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDirForTest(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyOneFileForTest(p, target)
	})
}
