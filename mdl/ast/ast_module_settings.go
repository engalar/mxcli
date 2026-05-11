// SPDX-License-Identifier: Apache-2.0

package ast

// AlterModuleJarDepStmt represents: ALTER MODULE name <jarDepAction>+
type AlterModuleJarDepStmt struct {
	ModuleName string
	Actions    []JarDepAction
}

func (s *AlterModuleJarDepStmt) isStatement() {}

// JarDepAction is the common interface for JAR dependency sub-actions.
type JarDepAction interface {
	isJarDepAction()
}

// AddJarDepAction represents: ADD JAR DEPENDENCY (group = '...', artifact = '...', version = '...' [, included = true])
type AddJarDepAction struct {
	Group    string
	Artifact string
	Version  string
	Included bool // defaults to true
}

func (a *AddJarDepAction) isJarDepAction() {}

// SetJarDepVersionAction represents: SET JAR DEPENDENCY 'group:artifact' VERSION '...'
type SetJarDepVersionAction struct {
	Coordinate string // "group:artifact"
	Version    string
}

func (a *SetJarDepVersionAction) isJarDepAction() {}

// SetJarDepIncludedAction represents: SET JAR DEPENDENCY 'group:artifact' INCLUDED true|false
type SetJarDepIncludedAction struct {
	Coordinate string
	Included   bool
}

func (a *SetJarDepIncludedAction) isJarDepAction() {}

// DropJarDepAction represents: DROP JAR DEPENDENCY 'group:artifact'
type DropJarDepAction struct {
	Coordinate string
}

func (a *DropJarDepAction) isJarDepAction() {}

// AddJarDepExclusionAction represents: SET JAR DEPENDENCY 'group:artifact' ADD EXCLUSION 'exc:artifact'
type AddJarDepExclusionAction struct {
	Coordinate string
	Exclusion  string
}

func (a *AddJarDepExclusionAction) isJarDepAction() {}

// DropJarDepExclusionAction represents: SET JAR DEPENDENCY 'group:artifact' DROP EXCLUSION 'exc:artifact'
type DropJarDepExclusionAction struct {
	Coordinate string
	Exclusion  string
}

func (a *DropJarDepExclusionAction) isJarDepAction() {}
