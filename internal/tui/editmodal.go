package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	selectOptions []string
	selectCursor  int
	selectValue   string

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
		// Clamp the typed buffer to [min, max] only at confirmation time.
		v := m.numberValue
		if v < m.numberMin {
			v = m.numberMin
		}
		if v > m.numberMax {
			v = m.numberMax
		}
		return v
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
	switch msg.String() {
	case "tab":
		// Autocomplete for paths
		if m.isPath {
			m.autocomplete()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *EditModal) autocomplete() {
	value := m.textInput.Value()
	if value == "" {
		return
	}

	value = expandHome(value)

	dir := filepath.Dir(value)
	prefix := filepath.Base(value)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			completed := filepath.Join(dir, name)
			m.textInput.SetValue(completed + string(filepath.Separator))
			m.textInput.CursorEnd()
			return
		}
	}
}

func (m EditModal) updateBool(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "h", "l", "space":
		m.boolValue = !m.boolValue
	}
	return m, nil
}

func (m EditModal) updateNumber(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		if m.numberValue > m.numberMin {
			m.numberValue--
		}
	case "right", "l":
		if m.numberValue < m.numberMax {
			m.numberValue++
		}
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Allow free typing into a buffer; the value is clamped to [min, max]
		// at confirmation time (see getValue). A loose upper bound just guards
		// against int overflow on absurd input.
		if n, err := strconv.Atoi(msg.String()); err == nil {
			if newVal := m.numberValue*10 + n; newVal < 1_000_000 {
				m.numberValue = newVal
			}
		}
	case "backspace":
		m.numberValue = m.numberValue / 10
	}
	return m, nil
}

func (m EditModal) updateSelect(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectCursor > 0 {
			m.selectCursor--
		}
	case "down", "j":
		if m.selectCursor < len(m.selectOptions)-1 {
			m.selectCursor++
		}
	}
	return m, nil
}

func (m EditModal) updateMultiSelect(msg tea.KeyMsg) (EditModal, tea.Cmd) {
	if len(m.multiOrder) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.multiCursor > 0 {
			m.multiCursor--
		}
	case "down", "j":
		if m.multiCursor < len(m.multiOrder)-1 {
			m.multiCursor++
		}
	case "space":
		// Toggle selection
		opt := m.multiOrder[m.multiCursor]
		m.multiSelected[opt] = !m.multiSelected[opt]
	case "ctrl+up", "K":
		// Move item up
		if m.multiCursor > 0 {
			m.multiOrder[m.multiCursor], m.multiOrder[m.multiCursor-1] =
				m.multiOrder[m.multiCursor-1], m.multiOrder[m.multiCursor]
			m.multiCursor--
		}
	case "ctrl+down", "J":
		// Move item down
		if m.multiCursor < len(m.multiOrder)-1 {
			m.multiOrder[m.multiCursor], m.multiOrder[m.multiCursor+1] =
				m.multiOrder[m.multiCursor+1], m.multiOrder[m.multiCursor]
			m.multiCursor++
		}
	}
	return m, nil
}

// View renders the modal
func (m EditModal) View() string {
	if !m.visible {
		return ""
	}

	var content string
	switch m.mode {
	case ModalText:
		content = m.viewText()
	case ModalBool:
		content = m.viewBool()
	case ModalNumber:
		content = m.viewNumber()
	case ModalSelect:
		content = m.viewSelect()
	case ModalMultiSelect:
		content = m.viewMultiSelect()
	}

	return m.renderModal(content)
}

func (m EditModal) renderModal(content string) string {
	// Modal style with border
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(m.width)

	// Title
	title := TitleStyle.Render(m.title)

	// Combine
	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", content)

	return modalStyle.Render(inner)
}

func (m EditModal) viewText() string {
	input := InputFocusedStyle.Width(m.width - 8).Render(m.textInput.View())
	help := DimStyle.Render("Tab: autocomplete  Enter: confirm  Esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Left, input, "", help)
}

func (m EditModal) viewBool() string {
	offStyle := ButtonStyle
	onStyle := ButtonStyle

	if m.boolValue {
		onStyle = ButtonActiveStyle
	} else {
		offStyle = ButtonActiveStyle
	}

	toggle := lipgloss.JoinHorizontal(lipgloss.Center,
		offStyle.Render("  OFF  "),
		"  ",
		onStyle.Render("  ON  "),
	)

	help := DimStyle.Render("←/→: toggle  Enter: confirm  Esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Center, toggle, "", help)
}

func (m EditModal) viewNumber() string {
	// Number display with arrows
	leftArrow := DimStyle.Render("◀")
	rightArrow := DimStyle.Render("▶")
	if m.numberValue > m.numberMin {
		leftArrow = TextStyle.Render("◀")
	}
	if m.numberValue < m.numberMax {
		rightArrow = TextStyle.Render("▶")
	}

	numDisplay := BoldStyle.Render(fmt.Sprintf(" %d ", m.numberValue))
	display := lipgloss.JoinHorizontal(lipgloss.Center,
		leftArrow, "  ", numDisplay, "  ", rightArrow,
	)

	limits := DimStyle.Render(fmt.Sprintf("(%d - %d)", m.numberMin, m.numberMax))
	help := DimStyle.Render("←/→: adjust  Enter: confirm  Esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Center, display, limits, "", help)
}

func (m EditModal) viewSelect() string {
	var lines []string

	for i, opt := range m.selectOptions {
		prefix := "  "
		style := ListItemStyle
		suffix := ""

		if i == m.selectCursor {
			prefix = "> "
			style = SelectedItemStyle
		}
		if opt == m.selectValue {
			suffix = " " + SuccessStyle.Render(IconCheck)
		}

		lines = append(lines, style.Render(prefix+opt+suffix))
	}

	list := lipgloss.JoinVertical(lipgloss.Left, lines...)
	help := DimStyle.Render("↑/↓: select  Enter: confirm  Esc: cancel")

	return lipgloss.JoinVertical(lipgloss.Left, list, "", help)
}

func (m EditModal) viewMultiSelect() string {
	var lines []string

	for i, opt := range m.multiOrder {
		prefix := "  "
		style := ListItemStyle
		checkbox := DimStyle.Render("[ ]")

		if m.multiSelected[opt] {
			checkbox = SuccessStyle.Render("[x]")
		}

		if i == m.multiCursor {
			prefix = "> "
			style = SelectedItemStyle
		}

		line := fmt.Sprintf("%s%s %s", prefix, checkbox, opt)
		lines = append(lines, style.Render(line))
	}

	list := lipgloss.JoinVertical(lipgloss.Left, lines...)
	help := DimStyle.Render("Space: toggle  Ctrl+↑/↓: reorder  Enter: confirm")

	return lipgloss.JoinVertical(lipgloss.Left, list, "", help)
}

// Visible returns whether the modal is visible
func (m EditModal) Visible() bool {
	return m.visible
}
