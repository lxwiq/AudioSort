package tui

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PathInputModel is a TUI for entering a directory path
type PathInputModel struct {
	input     textinput.Model
	title     string
	err       error
	confirmed bool
	cancelled bool

	width  int
	height int
}

// Note: pathConfirmMsg is defined in menu.go and used for communication

// NewPathInputModel creates a new path input model
func NewPathInputModel(title, placeholder, defaultValue string) PathInputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	if defaultValue != "" {
		ti.SetValue(defaultValue)
	} else {
		// Use current directory as default
		if cwd, err := os.Getwd(); err == nil {
			ti.SetValue(cwd)
		}
	}

	return PathInputModel{
		input:  ti,
		title:  title,
		width:  80,
		height: 24,
	}
}

// Init initializes the path input
func (m PathInputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles input
func (m PathInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = m.width - 20
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, func() tea.Msg {
				return pathConfirmMsg{path: "", confirmed: false}
			}

		case "enter":
			path := m.input.Value()

			// Expand ~ to home directory
			if len(path) > 0 && path[0] == '~' {
				if home, err := os.UserHomeDir(); err == nil {
					path = filepath.Join(home, path[1:])
				}
			}

			// Validate path exists
			info, err := os.Stat(path)
			if err != nil {
				m.err = err
				return m, nil
			}
			if !info.IsDir() {
				m.err = os.ErrInvalid
				return m, nil
			}

			m.confirmed = true
			return m, func() tea.Msg {
				return pathConfirmMsg{path: path, confirmed: true}
			}

		case "tab":
			// Simple tab completion - find matching directories
			m.autocomplete()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.err = nil // Clear error on input change
	return m, cmd
}

// View renders the path input
func (m PathInputModel) View() string {
	var sections []string

	// Header
	sections = append(sections, HeaderCompact(m.title))
	sections = append(sections, "")

	// Instructions
	sections = append(sections, TextStyle.Render("Enter the path to your audiobooks folder:"))
	sections = append(sections, "")

	// Input box
	inputStyle := InputFocusedStyle.Width(m.width - 10)
	sections = append(sections, inputStyle.Render(m.input.View()))

	// Error message
	if m.err != nil {
		sections = append(sections, "")
		sections = append(sections, ErrorMsgStyle.Render("Path not found or not a directory"))
	}

	// Hints
	sections = append(sections, "")
	sections = append(sections, DimStyle.Render("Tip: Use ~ for home directory, Tab for autocomplete"))

	// Footer
	sections = append(sections, "")
	sections = append(sections, Footer("enter", "confirm", "tab", "autocomplete", "esc", "cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return AppStyle.Width(m.width).Height(m.height).Render(content)
}

// autocomplete tries to complete the current path
func (m *PathInputModel) autocomplete() {
	path := m.input.Value()
	if path == "" {
		return
	}

	// Expand ~
	if len(path) > 0 && path[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Get directory and prefix
	dir := filepath.Dir(path)
	prefix := filepath.Base(path)

	// If path ends with separator, list that directory
	if path[len(path)-1] == filepath.Separator {
		dir = path
		prefix = ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Find first matching directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if prefix == "" || len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			completed := filepath.Join(dir, name)
			m.input.SetValue(completed + string(filepath.Separator))
			m.input.CursorEnd()
			return
		}
	}
}

// Path returns the entered path
func (m PathInputModel) Path() string {
	return m.input.Value()
}

// Confirmed returns true if the path was confirmed
func (m PathInputModel) Confirmed() bool {
	return m.confirmed
}

// Cancelled returns true if the input was cancelled
func (m PathInputModel) Cancelled() bool {
	return m.cancelled
}

// RunPathInput runs the path input dialog and returns the selected path
func RunPathInput(title, placeholder, defaultValue string) (string, bool, error) {
	p := tea.NewProgram(
		NewPathInputModel(title, placeholder, defaultValue),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	m := finalModel.(PathInputModel)
	if m.Cancelled() {
		return "", false, nil
	}

	return m.Path(), m.Confirmed(), nil
}
