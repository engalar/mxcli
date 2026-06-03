// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type scriptBufEntry struct {
	containerID     string
	containmentName string
	unitType        string
	contents        []byte
}

// ScriptBuffer accumulates write operations during an EXECUTE SCRIPT block.
// No SQL connection is held; a single atomic BatchWrite is issued at Commit time.
type ScriptBuffer struct {
	inserts map[string]scriptBufEntry
	updates map[string][]byte
	reader  *modelsdkmpr.Reader
}

func newScriptBuffer(r *modelsdkmpr.Reader) *ScriptBuffer {
	return &ScriptBuffer{
		inserts: make(map[string]scriptBufEntry),
		updates: make(map[string][]byte),
		reader:  r,
	}
}

func (buf *ScriptBuffer) AddInsert(unitID, containerID, containmentName, unitType string, contents []byte) error {
	buf.inserts[unitID] = scriptBufEntry{
		containerID: containerID, containmentName: containmentName,
		unitType: unitType, contents: contents,
	}
	buf.reader.SetScriptOverlay(unitID, contents)
	buf.reader.AppendScriptInsert(modelsdkmpr.ScriptInsertEntry{
		ID: unitID, ContainerID: containerID,
		ContainmentName: containmentName, UnitType: unitType, Contents: contents,
	})
	return nil
}

func (buf *ScriptBuffer) AddUpdate(unitID string, contents []byte) error {
	buf.updates[unitID] = contents
	buf.reader.SetScriptOverlay(unitID, contents)
	return nil
}

func (buf *ScriptBuffer) Rollback() {
	buf.inserts = nil
	buf.updates = nil
	buf.reader.ClearScriptMode()
}

func (buf *ScriptBuffer) toBatchOps() []modelsdkmpr.BatchWriteOp {
	ops := make([]modelsdkmpr.BatchWriteOp, 0, len(buf.inserts)+len(buf.updates))
	for id, e := range buf.inserts {
		ops = append(ops, modelsdkmpr.BatchWriteOp{
			Insert: true, UnitID: id, ContainerID: e.containerID,
			ContainmentName: e.containmentName, UnitType: e.unitType, Contents: e.contents,
		})
	}
	for id, contents := range buf.updates {
		ops = append(ops, modelsdkmpr.BatchWriteOp{Insert: false, UnitID: id, Contents: contents})
	}
	return ops
}
