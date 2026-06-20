// SPDX-License-Identifier: Apache-2.0

package ast

// ImageItem represents an image to add to a collection: IMAGE "name" FROM FILE 'path'.
type ImageItem struct {
	Name     string // Image name (e.g. "logo")
	FilePath string // Path to image file on disk
}

// CreateImageCollectionStmt represents:
//
//	CREATE IMAGE COLLECTION Module.Name [EXPORT LEVEL 'Public'] [COMMENT '...'] [(IMAGE "name" FROM FILE 'path', ...)]
type CreateImageCollectionStmt struct {
	Name           QualifiedName
	CreateOrModify bool
	ExportLevel    string // "Hidden" (default) or "Public"
	Comment        string
	Images         []ImageItem
}

func (s *CreateImageCollectionStmt) isStatement() {}
func (s *CreateImageCollectionStmt) TypeName() string { return "CreateImageCollection" }

// DropImageCollectionStmt represents: DROP IMAGE COLLECTION Module.Name
type DropImageCollectionStmt struct {
	Name QualifiedName
}

func (s *DropImageCollectionStmt) isStatement() {}
func (s *DropImageCollectionStmt) TypeName() string { return "DropImageCollection" }

// AlterImageCollectionStmt represents ALTER IMAGE COLLECTION Module.Name action [, action...]
type AlterImageCollectionStmt struct {
	Name    QualifiedName
	Actions []ImageCollectionAction
}

func (s *AlterImageCollectionStmt) isStatement() {}
func (s *AlterImageCollectionStmt) TypeName() string { return "AlterImageCollection" }

// ImageCollectionAction is the interface for all ALTER IMAGE COLLECTION sub-actions.
type ImageCollectionAction interface{ isImageCollectionAction() }

// AddImageAction: ADD IMAGE name FROM FILE 'path'
type AddImageAction struct {
	ImageName string
	FilePath  string
}

// DropImageAction: DROP IMAGE name
type DropImageAction struct {
	ImageName string
}

// RenameImageAction: RENAME IMAGE oldName TO newName
type RenameImageAction struct {
	From string
	To   string
}

// SetImageAction: SET IMAGE name FROM FILE 'path'
type SetImageAction struct {
	ImageName string
	FilePath  string
}

// MoveImageCollectionAction: MOVE TO Module.Name
type MoveImageCollectionAction struct {
	Target QualifiedName
}

// ExportImageAction: EXPORT IMAGE name TO FILE 'path'
type ExportImageAction struct {
	ImageName string
	FilePath  string
}

func (a *AddImageAction) isImageCollectionAction()            {}
func (a *DropImageAction) isImageCollectionAction()           {}
func (a *RenameImageAction) isImageCollectionAction()         {}
func (a *SetImageAction) isImageCollectionAction()            {}
func (a *MoveImageCollectionAction) isImageCollectionAction() {}
func (a *ExportImageAction) isImageCollectionAction()         {}
