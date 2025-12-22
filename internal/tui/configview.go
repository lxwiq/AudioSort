package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"audiosort/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigModel is the model for the config TUI
type ConfigModel struct {
	config *config.Config

	// Navigation
	cursor int
	items  []configItem

	// State
	saved   bool
	message string

	// Modal editing
	editing bool
	modal   EditModal

	// Dimensions
	width  int
	height int
}

type configItem struct {
	key   string
	value string
	desc  string
}

type configField int

const (
	fieldSources configField = iota
	fieldOutputFormat
	fieldDefaultOutput
	fieldCopyMode
	fieldWorkers
	fieldSkipExisting
	fieldLanguage
)

// NewConfigModel creates a new config model
func NewConfigModel(cfg *config.Config) ConfigModel {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	items := []configItem{
		{key: "Sources", value: strings.Join(cfg.Sources, ", "), desc: "Metadata sources to query"},
		{key: "Output Format", value: cfg.OutputFormat, desc: "Default output format preset"},
		{key: "Default Output", value: cfg.DefaultOutput, desc: "Default destination directory"},
		{key: "Copy Mode", value: fmt.Sprintf("%v", cfg.CopyMode), desc: "Copy files instead of moving"},
		{key: "Workers", value: fmt.Sprintf("%d", cfg.ParallelWorkers), desc: "Number of parallel workers"},
		{key: "Skip Existing", value: fmt.Sprintf("%v", cfg.SkipExisting), desc: "Skip already processed books"},
		{key: "Language", value: cfg.PreferredLanguage, desc: "Preferred metadata language"},
	}

	return ConfigModel{
		config: cfg,
		items:  items,
		width:  80,
		height: 24,
	}
}

// Init initializes the config model
func (m ConfigModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the config model
func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle modal result
	if result, ok := msg.(EditModalResult); ok {
		m.editing = false
		m.applyModalResult(result)
		return m, nil
	}

	// If modal is open, delegate to modal
	if m.editing {
		var cmd tea.Cmd
		m.modal, cmd = m.modal.Update(msg)
		if !m.modal.Visible() {
			m.editing = false
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			if m.editing {
				m.editing = false
				return m, nil
			}
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "enter":
			return m, m.openModal()

		case "s":
			// Save config
			if err := m.config.Save(); err != nil {
				m.message = "Error: " + err.Error()
			} else {
				m.saved = true
				m.message = "Configuration saved!"
			}

		case "r":
			// Reset to defaults
			m.config = config.DefaultConfig()
			m.items = m.rebuildItems()
			m.message = "Reset to defaults"
		}
	}

	return m, nil
}

// View renders the config view
func (m ConfigModel) View() string {
	var sections []string

	// Header
	sections = append(sections, HeaderCompact("Configuration"))
	sections = append(sections, "")

	// Config items
	sections = append(sections, m.renderConfigList())

	// Message
	if m.message != "" {
		sections = append(sections, "")
		if m.saved {
			sections = append(sections, Alert("success", m.message))
		} else {
			sections = append(sections, Alert("info", m.message))
		}
	}

	// Footer
	sections = append(sections, "")
	footerText := Footer("↑/↓", "navigate", "Enter", "edit", "s", "save", "r", "reset", "q", "quit")
	sections = append(sections, footerText)

	baseView := AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)

	// If modal is open, overlay it
	if m.editing {
		modalView := m.modal.View()

		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			modalView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#1a1a1a")),
		)
	}

	return baseView
}

func (m ConfigModel) renderConfigList() string {
	var lines []string

	// Calculate key width for alignment
	maxKeyLen := 0
	for _, item := range m.items {
		if len(item.key) > maxKeyLen {
			maxKeyLen = len(item.key)
		}
	}

	for i, item := range m.items {
		style := ListItemStyle
		prefix := "  "

		if i == m.cursor {
			style = SelectedItemStyle
			prefix = "> "
		}

		// Format: key (padded) : value
		key := BoldStyle.Render(fmt.Sprintf("%-*s", maxKeyLen, item.key))
		value := TextStyle.Render(item.value)
		if item.value == "" {
			value = DimStyle.Render("(not set)")
		}

		line := fmt.Sprintf("%s%s : %s", prefix, key, value)

		// Add description for selected item
		if i == m.cursor {
			line += "\n" + strings.Repeat(" ", len(prefix)+maxKeyLen+3) + DimStyle.Render(item.desc)
		}

		lines = append(lines, style.Render(line))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return BoxStyle.Width(m.width - 6).Render(content)
}

func (m ConfigModel) rebuildItems() []configItem {
	return []configItem{
		{key: "Sources", value: strings.Join(m.config.Sources, ", "), desc: "Metadata sources to query"},
		{key: "Output Format", value: m.config.OutputFormat, desc: "Default output format preset"},
		{key: "Default Output", value: m.config.DefaultOutput, desc: "Default destination directory"},
		{key: "Copy Mode", value: fmt.Sprintf("%v", m.config.CopyMode), desc: "Copy files instead of moving"},
		{key: "Workers", value: fmt.Sprintf("%d", m.config.ParallelWorkers), desc: "Number of parallel workers"},
		{key: "Skip Existing", value: fmt.Sprintf("%v", m.config.SkipExisting), desc: "Skip already processed books"},
		{key: "Language", value: m.config.PreferredLanguage, desc: "Preferred metadata language"},
	}
}

func (m *ConfigModel) openModal() tea.Cmd {
	field := configField(m.cursor)

	switch field {
	case fieldSources:
		m.modal = NewMultiSelectModal("Sources", "Edit Sources", AvailableSources, m.config.Sources)
	case fieldOutputFormat:
		m.modal = NewSelectModal("Output Format", "Edit Output Format", AvailableFormats, m.config.OutputFormat)
	case fieldDefaultOutput:
		m.modal = NewTextModal("Default Output", "Edit Default Output", m.config.DefaultOutput, true)
	case fieldCopyMode:
		m.modal = NewBoolModal("Copy Mode", "Edit Copy Mode", m.config.CopyMode)
	case fieldWorkers:
		m.modal = NewNumberModal("Workers", "Edit Workers", m.config.ParallelWorkers, 1, 32)
	case fieldSkipExisting:
		m.modal = NewBoolModal("Skip Existing", "Edit Skip Existing", m.config.SkipExisting)
	case fieldLanguage:
		m.modal = NewTextModal("Language", "Edit Language", m.config.PreferredLanguage, false)
	}

	m.editing = true
	return m.modal.Init()
}

func (m *ConfigModel) applyModalResult(result EditModalResult) {
	if !result.Confirmed {
		return
	}

	field := configField(m.cursor)

	switch field {
	case fieldSources:
		if sources, ok := result.Value.([]string); ok {
			m.config.Sources = sources
		}
	case fieldOutputFormat:
		if format, ok := result.Value.(string); ok {
			m.config.OutputFormat = format
		}
	case fieldDefaultOutput:
		if path, ok := result.Value.(string); ok {
			// Expand ~
			if len(path) > 0 && path[0] == '~' {
				if home, err := os.UserHomeDir(); err == nil {
					path = filepath.Join(home, path[1:])
				}
			}
			m.config.DefaultOutput = path
		}
	case fieldCopyMode:
		if val, ok := result.Value.(bool); ok {
			m.config.CopyMode = val
		}
	case fieldWorkers:
		if val, ok := result.Value.(int); ok {
			m.config.ParallelWorkers = val
		}
	case fieldSkipExisting:
		if val, ok := result.Value.(bool); ok {
			m.config.SkipExisting = val
		}
	case fieldLanguage:
		if val, ok := result.Value.(string); ok {
			m.config.PreferredLanguage = val
		}
	}

	m.items = m.rebuildItems()
	m.message = "Value updated (press 's' to save)"
	m.saved = false
}

// RunConfig runs the config TUI
func RunConfig(cfg *config.Config) error {
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
	}

	p := tea.NewProgram(
		NewConfigModel(cfg),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
