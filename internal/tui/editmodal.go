package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// ModalMode defines the type of editor in the modal
type ModalMode int

const (
	ModalText ModalMode = iota
	ModalBool
	ModalNumber
	ModalSelect
	ModalMultiSelect
)

// Available options for select fields
var (
	AvailableSources = []string{"googlebooks", "bookinfo", "openlibrary"}
	AvailableFormats = []string{"audiobookshelf", "plex", "json", "all"}
)

// EditModalResult is sent when the modal is closed
type EditModalResult struct {
	Confirmed bool
	Field     string
	Value     any
}

// EditModal is a modal dialog for editing config values
type EditModal struct {
	mode    ModalMode
	field   string
	title   string
	visible bool

	// Text input
	textInput textinput.Model

	// Bool toggle
	boolValue bool

	// Number input
	numberValue int
	numberMin   int
	numberMax   int

	// Select (single choice)
	selectOptions  []string
	selectCursor   int
	selectValue    string

	// MultiSelect (multiple choices with ordering)
	multiOptions  []string
	multiSelected map[string]bool
	multiOrder    []string
	multiCursor   int

	// Dimensions
	width  int
	height int
}
