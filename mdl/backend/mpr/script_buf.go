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
	inserts          map[string]scriptBufEntry
	updates          map[string][]byte
	containerUpdates map[string]string // unitID → newContainerID (for existing SQLite units)
	reader           *modelsdkmpr.Reader
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

func (buf *ScriptBuffer) AddContainerUpdate(unitID, containerID string) error {
	// For inserts (buffered in this script transaction), update the insert
	// entry's ContainerID so the BatchWrite uses the new container.
	if e, ok := buf.inserts[unitID]; ok {
		e.containerID = containerID
		buf.inserts[unitID] = e
		return nil
	}
	// For existing SQLite units, add an update that sets both Contents and
	// ContainerID on commit. We store the container ID in a separate map
	// and merge it into the BatchWrite at commit time.
	buf.updates[unitID] = nil // ensure the unit is in the updates map
	if buf.containerUpdates == nil {
		buf.containerUpdates = make(map[string]string)
	}
	buf.containerUpdates[unitID] = containerID
	return nil
}

func (buf *ScriptBuffer) Rollback() {
	buf.inserts = nil
	buf.updates = nil
	buf.containerUpdates = nil
	buf.reader.ClearScriptMode()
}

func (buf *ScriptBuffer) toBatchOps() []modelsdkmpr.BatchWriteOp {
	ops := make([]modelsdkmpr.BatchWriteOp, 0, len(buf.inserts)+len(buf.updates)+len(buf.containerUpdates))
	for id, e := range buf.inserts {
		ops = append(ops, modelsdkmpr.BatchWriteOp{
			Insert: true, UnitID: id, ContainerID: e.containerID,
			ContainmentName: e.containmentName, UnitType: e.unitType, Contents: e.contents,
		})
	}
	for id, contents := range buf.updates {
		newContainerID := ""
		if buf.containerUpdates != nil {
			newContainerID = buf.containerUpdates[id]
		}
		ops = append(ops, modelsdkmpr.BatchWriteOp{Insert: false, UnitID: id, Contents: contents, NewContainerID: newContainerID})
	}
	return ops
}
