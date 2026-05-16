// SPDX-License-Identifier: Apache-2.0

// Package modelsdk provides a Go library for reading and modifying Mendix projects.
//
// This library is a Go alternative to the Mendix Model SDK and Mendix Platform SDK,
// allowing direct manipulation of Mendix project files (.mpr) on disk.
//
// # Overview
//
// Mendix projects are stored in .mpr files which are SQLite databases containing
// BSON-encoded model elements. This library provides:
//
//   - Reading and parsing MPR files (both v1 and v2 formats)
//   - Type-safe access to all Mendix model elements
//   - Creating, updating, and deleting model elements
//   - Exporting models to JSON format
//
// # Quick Start
//
//	package main
//
//	import (
//	    "fmt"
//	    "github.com/mendixlabs/mxcli"
//	)
//
//	func main() {
//	    // Open a Mendix project via the public modelsdk API.
//	    reader, err := modelsdk.Open("/path/to/MyApp.mpr")
//	    if err != nil {
//	        panic(err)
//	    }
//	    defer reader.Close()
//
//	    // List all modules
//	    modules, err := reader.ListModules()
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    for _, m := range modules {
//	        fmt.Printf("Module: %s\n", m.Name)
//	    }
//	}
//
// # MPR File Formats
//
// The library supports both MPR v1 (single file) and MPR v2 (with mprcontents folder)
// formats. MPR v2 was introduced in Mendix Studio Pro 10.18.
//
// # Model Structure
//
// The Mendix model is organized hierarchically:
//
//   - Project
//   - Modules
//   - Domain Models (Entities, Attributes, Associations)
//   - Microflows and Nanoflows
//   - Pages, Layouts, and Snippets
//   - Enumerations and Constants
//   - Scheduled Events
//
// Each element has a unique ID and belongs to a container element.
//
// # Thread Safety
//
// The Reader is safe for concurrent read access. The Writer should only be used
// from a single goroutine. For concurrent modifications, use transactions.
//
// # Error Handling
//
// All functions that can fail return an error. Errors include:
//
//   - File not found
//   - Invalid MPR format
//   - Element not found
//   - BSON parsing errors
//   - SQLite errors
package modelsdk

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// Version is the library version.
const Version = "0.1.0"

// Re-export commonly used types for convenience.
type (
	// ID is a unique identifier for model elements.
	ID = model.ID

	// Module represents a Mendix module.
	Module = model.Module

	// Project represents a Mendix project.
	Project = model.Project

	// Enumeration represents an enumeration type.
	Enumeration = model.Enumeration

	// Constant represents a constant value.
	Constant = model.Constant

	// ConstantDataType represents the data type of a constant.
	ConstantDataType = model.ConstantDataType

	// ScheduledEvent represents a scheduled event.
	ScheduledEvent = model.ScheduledEvent

	// Microflow represents a microflow (modelsdk gen-typed).
	Microflow = genMf.Microflow

	// Nanoflow represents a nanoflow (modelsdk gen-typed).
	Nanoflow = genMf.Nanoflow

	// Page represents a page.
	Page = genPg.Page

	// Layout represents a layout.
	Layout = genPg.Layout

	// Snippet represents a page snippet.
	Snippet = genPg.Snippet

	// Reader provides read-only access to Mendix project files.
	Reader = mmpr.Reader

	// Writer provides read-write access to Mendix project files.
	Writer = mmpr.Writer
)

// Open opens an MPR file for reading.
func Open(path string) (*Reader, error) {
	return mmpr.Open(path)
}

// OpenForWriting opens an MPR file for reading and writing.
func OpenForWriting(path string) (*Writer, error) {
	return mmpr.NewWriter(path)
}

// NewPage creates a new page.
func NewPage(name string) *Page {
	p := genPg.NewPage()
	p.SetName(name)
	return p
}

// GenerateID generates a new unique ID for model elements.
func GenerateID() ID {
	return ID(mmpr.GenerateID())
}
