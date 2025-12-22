package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	isPath    bool // Enable path autocomplete for text mode

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

// NewTextModal creates a modal for text input
func NewTextModal(field, title, value string, isPath bool) EditModal {
	ti := textinput.New()
	ti.SetValue(value)
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return EditModal{
		mode:      ModalText,
		field:     field,
		title:     title,
		visible:   true,
		textInput: ti,
		isPath:    isPath,
		width:     60,
		height:    10,
	}
}

// NewBoolModal creates a modal for boolean toggle
func NewBoolModal(field, title string, value bool) EditModal {
	return EditModal{
		mode:      ModalBool,
		field:     field,
		title:     title,
		visible:   true,
		boolValue: value,
		width:     40,
		height:    8,
	}
}

// NewNumberModal creates a modal for number input
func NewNumberModal(field, title string, value, min, max int) EditModal {
	return EditModal{
		mode:        ModalNumber,
		field:       field,
		title:       title,
		visible:     true,
		numberValue: value,
		numberMin:   min,
		numberMax:   max,
		width:       40,
		height:      8,
	}
}

// NewSelectModal creates a modal for single selection
func NewSelectModal(field, title string, options []string, current string) EditModal {
	cursor := 0
	for i, opt := range options {
		if opt == current {
			cursor = i
			break
		}
	}

	return EditModal{
		mode:          ModalSelect,
		field:         field,
		title:         title,
		visible:       true,
		selectOptions: options,
		selectCursor:  cursor,
		selectValue:   current,
		width:         40,
		height:        len(options) + 6,
	}
}

// NewMultiSelectModal creates a modal for multiple selection with ordering
func NewMultiSelectModal(field, title string, allOptions []string, selected []string) EditModal {
	selectedMap := make(map[string]bool)
	for _, s := range selected {
		selectedMap[s] = true
	}

	// Build ordered list: selected first (in order), then unselected
	order := make([]string, 0, len(allOptions))
	order = append(order, selected...)
	for _, opt := range allOptions {
		if !selectedMap[opt] {
			order = append(order, opt)
		}
	}

	return EditModal{
		mode:          ModalMultiSelect,
		field:         field,
		title:         title,
		visible:       true,
		multiOptions:  allOptions,
		multiSelected: selectedMap,
		multiOrder:    order,
		multiCursor:   0,
		width:         45,
		height:        len(allOptions) + 8,
	}
}

// Init initializes the modal
func (m EditModal) Init() tea.Cmd {
	if m.mode == ModalText {
		return textinput.Blink
	}
	return nil
}

// Update handles messages for the modal
func (m EditModal) Update(msg tea.Msg) (EditModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.visible = false
			return m, func() tea.Msg {
				return EditModalResult{Confirmed: false, Field: m.field}
			}

		case "enter":
			m.visible = false
			return m, func() tea.Msg {
				return EditModalResult{
					Confirmed: true,
					Field:     m.field,
					Value:     m.getValue(),
				}
			}
		}

		// Mode-specific handling
		switch m.mode {
		case ModalText:
			return m.updateText(msg)
		case ModalBool:
			return m.updateBool(msg)
		case ModalNumber:
			return m.updateNumber(msg)
		case ModalSelect:
			return m.updateSelect(msg)
		case ModalMultiSelect:
			return m.updateMultiSelect(msg)
		}
	}

	// Pass through to text input if in text mode
	if m.mode == ModalText {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m EditModal) getValue() any {
	switch m.mode {
	case ModalText:
		return m.textInput.Value()
	case ModalBool:
		return m.boolValue
	case ModalNumber:
		return m.numberValue
	case ModalSelect:
		if len(m.selectOptions) == 0 || m.selectCursor < 0 || m.selectCursor >= len(m.selectOptions) {
			return ""
		}
		return m.selectOptions[m.selectCursor]
	case ModalMultiSelect:
		// Return only selected items in order
		var result []string
		for _, opt := range m.multiOrder {
			if m.multiSelected[opt] {
				result = append(result, opt)
			}
		}
		return result
	}
	return nil
}

func (m EditModal) updateText(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	return m, nil
}

func (m EditModal) updateBool(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	return m, nil
}

func (m EditModal) updateNumber(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	return m, nil
}

func (m EditModal) updateSelect(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	return m, nil
}

func (m EditModal) updateMultiSelect(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	return m, nil
}
