package tui

import (
	"fmt"
	"strings"

	"audiosort/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fieldSpec is the single declarative description of an editable config field.
// It replaces the former parallel enum + openModal + applyModalResult +
// rebuildItems switches, which had to be kept in sync by hand.
//
// label is both the row label AND the join key with the modal: it must equal
// the field name passed to NewXxxModal, because EditModalResult.Field carries
// that human label back (not an index), so a row can be matched without relying
// on the cursor position.
type fieldSpec struct {
	label string
	desc  string
	value func(c *config.Config) string
	modal func(c *config.Config) EditModal
	apply func(c *config.Config, v any)
}

func configFields() []fieldSpec {
	return []fieldSpec{
		{
			label: "Sources",
			desc:  "Metadata sources to query",
			value: func(c *config.Config) string { return strings.Join(c.Sources, ", ") },
			modal: func(c *config.Config) EditModal {
				return NewMultiSelectModal("Sources", "Edit Sources", config.AvailableSources, c.Sources)
			},
			apply: func(c *config.Config, v any) {
				if s, ok := v.([]string); ok {
					c.Sources = s
				}
			},
		},
		{
			label: "Output Format",
			desc:  "Default output format preset",
			value: func(c *config.Config) string { return c.OutputFormat },
			modal: func(c *config.Config) EditModal {
				return NewSelectModal("Output Format", "Edit Output Format", config.AvailableFormats, c.OutputFormat)
			},
			apply: func(c *config.Config, v any) {
				if s, ok := v.(string); ok {
					c.OutputFormat = s
				}
			},
		},
		{
			label: "Outputs",
			desc:  "Sidecar files written next to each book (opf/cover/json)",
			value: func(c *config.Config) string { return strings.Join(c.Outputs, ", ") },
			modal: func(c *config.Config) EditModal {
				return NewMultiSelectModal("Outputs", "Edit Outputs", config.AvailableOutputs, c.Outputs)
			},
			apply: func(c *config.Config, v any) {
				if s, ok := v.([]string); ok {
					c.Outputs = s
				}
			},
		},
		{
			label: "Default Output",
			desc:  "Default destination directory",
			value: func(c *config.Config) string { return c.DefaultOutput },
			modal: func(c *config.Config) EditModal {
				return NewTextModal("Default Output", "Edit Default Output", c.DefaultOutput, true)
			},
			apply: func(c *config.Config, v any) {
				if s, ok := v.(string); ok {
					c.DefaultOutput = expandHome(s)
				}
			},
		},
		{
			label: "Copy Mode",
			desc:  "Copy files instead of moving",
			value: func(c *config.Config) string { return fmt.Sprintf("%v", c.CopyMode) },
			modal: func(c *config.Config) EditModal {
				return NewBoolModal("Copy Mode", "Edit Copy Mode", c.CopyMode)
			},
			apply: func(c *config.Config, v any) {
				if b, ok := v.(bool); ok {
					c.CopyMode = b
				}
			},
		},
		{
			label: "Workers",
			desc:  "Number of parallel workers",
			value: func(c *config.Config) string { return fmt.Sprintf("%d", c.ParallelWorkers) },
			modal: func(c *config.Config) EditModal {
				return NewNumberModal("Workers", "Edit Workers", c.ParallelWorkers, 1, 32)
			},
			apply: func(c *config.Config, v any) {
				if n, ok := v.(int); ok {
					c.ParallelWorkers = n
				}
			},
		},
		{
			label: "Skip Existing",
			desc:  "Skip already processed books",
			value: func(c *config.Config) string { return fmt.Sprintf("%v", c.SkipExisting) },
			modal: func(c *config.Config) EditModal {
				return NewBoolModal("Skip Existing", "Edit Skip Existing", c.SkipExisting)
			},
			apply: func(c *config.Config, v any) {
				if b, ok := v.(bool); ok {
					c.SkipExisting = b
				}
			},
		},
		{
			label: "Language",
			desc:  "Preferred metadata language",
			value: func(c *config.Config) string { return c.PreferredLanguage },
			modal: func(c *config.Config) EditModal {
				return NewTextModal("Language", "Edit Language", c.PreferredLanguage, false)
			},
			apply: func(c *config.Config, v any) {
				if s, ok := v.(string); ok {
					c.PreferredLanguage = s
				}
			},
		},
	}
}

// cloneConfig returns an independent copy so edits stay on the working copy
// until the user saves.
func cloneConfig(c *config.Config) *config.Config {
	cp := *c
	cp.Sources = append([]string(nil), c.Sources...)
	cp.Outputs = append([]string(nil), c.Outputs...)
	return &cp
}

// ConfigModel is the model for the config TUI
type ConfigModel struct {
	config  *config.Config // shared, persisted config (written only on save)
	working *config.Config // working copy edited in the UI
	fields  []fieldSpec

	cursor int

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

// NewConfigModel creates a new config model
func NewConfigModel(cfg *config.Config) ConfigModel {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	return ConfigModel{
		config:  cfg,
		working: cloneConfig(cfg),
		fields:  configFields(),
		width:   80,
		height:  24,
	}
}

// Init initializes the config model
func (m ConfigModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the cached terminal dimensions.
func (m ConfigModel) SetSize(width, height int) subView {
	m.width = width
	m.height = height
	return m
}

// Update handles messages for the config model
func (m ConfigModel) Update(msg tea.Msg) (subView, tea.Cmd) {
	// Handle modal result
	if result, ok := msg.(EditModalResult); ok {
		m.editing = false
		if result.Confirmed {
			m.applyModalResult(result)
		}
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
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}

		case "enter":
			return m, m.openModal()

		case "s":
			m.save()

		case "r":
			m.working = config.DefaultConfig()
			m.message = "Reset to defaults (press 's' to save)"
			m.saved = false
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
	footerText := Footer("↑/↓", "navigate", "Enter", "edit", "s", "save", "r", "reset", "q", "back")
	sections = append(sections, footerText)

	baseView := AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)

	// If modal is open, overlay it
	if m.editing {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.modal.View(),
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
	for _, f := range m.fields {
		if len(f.label) > maxKeyLen {
			maxKeyLen = len(f.label)
		}
	}

	for i, f := range m.fields {
		style := ListItemStyle
		prefix := "  "

		if i == m.cursor {
			style = SelectedItemStyle
			prefix = "> "
		}

		// Format: key (padded) : value
		key := BoldStyle.Render(fmt.Sprintf("%-*s", maxKeyLen, f.label))
		val := f.value(m.working)
		value := TextStyle.Render(val)
		if val == "" {
			value = DimStyle.Render("(not set)")
		}

		line := fmt.Sprintf("%s%s : %s", prefix, key, value)

		// Add description for selected item
		if i == m.cursor {
			line += "\n" + strings.Repeat(" ", len(prefix)+maxKeyLen+3) + DimStyle.Render(f.desc)
		}

		lines = append(lines, style.Render(line))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return BoxStyle.Width(m.width - 6).Render(content)
}

func (m *ConfigModel) openModal() tea.Cmd {
	f := m.fields[m.cursor]
	m.modal = f.modal(m.working)
	m.editing = true
	return m.modal.Init()
}

// applyModalResult matches the edited field by its human label (carried in
// result.Field) rather than the cursor position, then applies the value to the
// working copy.
func (m *ConfigModel) applyModalResult(result EditModalResult) {
	for _, f := range m.fields {
		if f.label == result.Field {
			f.apply(m.working, result.Value)
			break
		}
	}

	m.message = "Value updated (press 's' to save)"
	m.saved = false
}

// commit copies the validated working copy into the shared config so the rest
// of the app sees the change through the same pointer. It does not persist to
// disk (the Sources slice is copied so the two configs never alias).
func (m *ConfigModel) commit() {
	*m.config = *m.working
	m.config.Sources = append([]string(nil), m.working.Sources...)
	m.config.Outputs = append([]string(nil), m.working.Outputs...)
}

// save validates the working copy, commits it to the shared config and persists
// it to disk.
func (m *ConfigModel) save() {
	if err := m.working.Validate(); err != nil {
		m.message = "Error: " + err.Error()
		m.saved = false
		return
	}

	m.commit()

	if err := m.config.Save(); err != nil {
		m.message = "Error: " + err.Error()
		m.saved = false
		return
	}

	m.saved = true
	m.message = "Configuration saved!"
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

	_, err := newStandaloneProgram(NewConfigModel(cfg)).Run()
	return err
}
