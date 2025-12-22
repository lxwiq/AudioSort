package tui

import (
	"audiosort/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents a menu option
type MenuItem struct {
	title string
	desc  string
	key   string
}

// MenuModel is the main menu TUI
type MenuModel struct {
	config *config.Config
	items  []MenuItem
	cursor int

	width  int
	height int
}

// NewMenuModel creates a new menu model
func NewMenuModel(cfg *config.Config) MenuModel {
	items := []MenuItem{
		{title: "Scan", desc: "Scan and organize audiobooks", key: "s"},
		{title: "Search", desc: "Search for book metadata", key: "/"},
		{title: "Config", desc: "View and manage settings", key: "c"},
		{title: "Cache", desc: "Manage metadata cache", key: "x"},
	}

	return MenuModel{
		config: cfg,
		items:  items,
		width:  80,
		height: 24,
	}
}

// menuActionMsg is sent when a menu item is selected
type menuActionMsg struct {
	action string
}

// Init initializes the menu
func (m MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles menu input
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "enter", " ":
			return m, func() tea.Msg {
				return menuActionMsg{action: m.items[m.cursor].title}
			}

		// Quick keys
		case "s", "1":
			return m, func() tea.Msg {
				return menuActionMsg{action: "Scan"}
			}
		case "/", "2":
			return m, func() tea.Msg {
				return menuActionMsg{action: "Search"}
			}
		case "c", "3":
			return m, func() tea.Msg {
				return menuActionMsg{action: "Config"}
			}
		case "x", "4":
			return m, func() tea.Msg {
				return menuActionMsg{action: "Cache"}
			}
		}
	}

	return m, nil
}

// View renders the menu
func (m MenuModel) View() string {
	var sections []string

	// Logo
	logo := LogoStyle.Render(Logo)
	sections = append(sections, logo)

	// Tagline
	tagline := DimStyle.Render("Organize your audiobook collection with style")
	sections = append(sections, tagline)
	sections = append(sections, "")

	// Menu items
	for i, item := range m.items {
		style := ListItemStyle
		cursor := "  "

		if i == m.cursor {
			style = SelectedItemStyle
			cursor = "> "
		}

		keyHint := HelpKeyStyle.Render("[" + item.key + "]")
		title := BoldStyle.Render(item.title)
		desc := DimStyle.Render(" - " + item.desc)

		line := cursor + keyHint + " " + title + desc
		sections = append(sections, style.Render(line))
	}

	sections = append(sections, "")

	// Version info
	version := DimStyle.Render("v1.0.0")
	sections = append(sections, version)

	// Footer
	sections = append(sections, "")
	sections = append(sections, Footer("↑/↓", "navigate", "enter", "select", "q", "quit"))

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center the content
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// RunMenu runs the main menu and returns the selected action
func RunMenu(cfg *config.Config) (string, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	p := tea.NewProgram(
		NewMenuModel(cfg),
		tea.WithAltScreen(),
	)

	model, err := p.Run()
	if err != nil {
		return "", err
	}

	// Check if an action was selected
	if m, ok := model.(MenuModel); ok {
		_ = m // Menu closed normally
	}

	return "", nil
}

// RunMenuWithAction runs the menu and executes the selected action
func RunMenuWithAction(cfg *config.Config) error {
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
	}

	for {
		// Create and run the menu
		p := tea.NewProgram(
			&menuRunner{
				menu:   NewMenuModel(cfg),
				config: cfg,
			},
			tea.WithAltScreen(),
		)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		runner, ok := finalModel.(*menuRunner)
		if !ok || runner.quit {
			return nil
		}

		// Action was handled, loop back to menu
	}
}

// menuRunner wraps the menu and handles action execution
type menuRunner struct {
	menu      MenuModel
	config    *config.Config
	action    string
	quit      bool
	inSubmenu bool
}

func (m *menuRunner) Init() tea.Cmd {
	return m.menu.Init()
}

func (m *menuRunner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case menuActionMsg:
		m.action = msg.action
		// Execute the action
		return m, m.executeAction(msg.action)

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quit = true
			return m, tea.Quit
		}
	}

	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu.(MenuModel)
	return m, cmd
}

func (m *menuRunner) View() string {
	return m.menu.View()
}

func (m *menuRunner) executeAction(action string) tea.Cmd {
	return func() tea.Msg {
		var err error

		switch action {
		case "Scan":
			// For scan, we need a path - show a simple prompt or use current dir
			err = Run(".", m.config)
		case "Search":
			err = RunSearch("", m.config)
		case "Config":
			err = RunConfig(m.config)
		case "Cache":
			err = RunCache("")
		}

		if err != nil {
			return errorMsg{err: err}
		}

		// Return to menu
		return nil
	}
}
