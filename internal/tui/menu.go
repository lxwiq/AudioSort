package tui

import (
	"audiosort/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppView represents the current view in the application
type AppView int

const (
	ViewMenu AppView = iota
	ViewPathInput
	ViewScan
	ViewSearch
	ViewConfig
	ViewCache
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
		{title: "Search", desc: "Search for book metadata", key: "f"},
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

		// Quick keys - changed "/" to "f" to avoid conflicts
		case "s", "1":
			return m, func() tea.Msg {
				return menuActionMsg{action: "Scan"}
			}
		case "f", "2":
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

// MainAppModel is the root application model
type MainAppModel struct {
	config  *config.Config
	view    AppView
	menu    MenuModel
	path    PathInputModel
	scan    Model
	search  SearchModel
	cfgView ConfigModel
	cache   CacheModel

	// Pending data between views
	pendingPath string

	width  int
	height int
}

// NewMainAppModel creates a new main app model
func NewMainAppModel(cfg *config.Config) MainAppModel {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	return MainAppModel{
		config: cfg,
		view:   ViewMenu,
		menu:   NewMenuModel(cfg),
		width:  80,
		height: 24,
	}
}

// Init initializes the main app
func (m MainAppModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages
func (m MainAppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to active view
		return m.forwardMessage(msg)

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// ESC returns to menu from sub-views
		if msg.String() == "esc" && m.view != ViewMenu && m.view != ViewScan {
			m.view = ViewMenu
			m.menu = NewMenuModel(m.config)
			return m, nil
		}

	case menuActionMsg:
		return m.handleMenuAction(msg.action)

	case pathConfirmMsg:
		if msg.confirmed {
			m.pendingPath = msg.path
			m.view = ViewScan
			m.scan = New(msg.path, m.config)
			return m, m.scan.Init()
		}
		m.view = ViewMenu
		return m, nil

	case scanDoneMsg:
		m.view = ViewMenu
		m.menu = NewMenuModel(m.config)
		return m, nil
	}

	return m.forwardMessage(msg)
}

// forwardMessage forwards messages to the active view
func (m MainAppModel) forwardMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.view {
	case ViewMenu:
		var newMenu tea.Model
		newMenu, cmd = m.menu.Update(msg)
		m.menu = newMenu.(MenuModel)

	case ViewPathInput:
		var newPath tea.Model
		newPath, cmd = m.path.Update(msg)
		m.path = newPath.(PathInputModel)

	case ViewScan:
		var newScan tea.Model
		newScan, cmd = m.scan.Update(msg)
		m.scan = newScan.(Model)

	case ViewSearch:
		var newSearch tea.Model
		newSearch, cmd = m.search.Update(msg)
		m.search = newSearch.(SearchModel)

	case ViewConfig:
		var newCfg tea.Model
		newCfg, cmd = m.cfgView.Update(msg)
		m.cfgView = newCfg.(ConfigModel)

	case ViewCache:
		var newCache tea.Model
		newCache, cmd = m.cache.Update(msg)
		m.cache = newCache.(CacheModel)
	}

	return m, cmd
}

// handleMenuAction handles menu selections
func (m MainAppModel) handleMenuAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "Scan":
		m.view = ViewPathInput
		m.path = NewPathInputModel("Scan Audiobooks", "Enter path to audiobooks...", "")
		return m, m.path.Init()

	case "Search":
		m.view = ViewSearch
		m.search = NewSearchModel(m.config, "")
		return m, m.search.Init()

	case "Config":
		m.view = ViewConfig
		m.cfgView = NewConfigModel(m.config)
		return m, m.cfgView.Init()

	case "Cache":
		m.view = ViewCache
		m.cache = NewCacheModel("")
		return m, m.cache.Init()
	}

	return m, nil
}

// View renders the current view
func (m MainAppModel) View() string {
	switch m.view {
	case ViewMenu:
		return m.menu.View()
	case ViewPathInput:
		return m.path.View()
	case ViewScan:
		return m.scan.View()
	case ViewSearch:
		return m.search.View()
	case ViewConfig:
		return m.cfgView.View()
	case ViewCache:
		return m.cache.View()
	default:
		return "Unknown view"
	}
}

// Message types
type pathConfirmMsg struct {
	path      string
	confirmed bool
}

type scanDoneMsg struct{}

// RunMenu runs the main menu
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

// RunMenuWithAction runs the complete application
func RunMenuWithAction(cfg *config.Config) error {
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			cfg = config.DefaultConfig()
		}
	}

	p := tea.NewProgram(
		NewMainAppModel(cfg),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
