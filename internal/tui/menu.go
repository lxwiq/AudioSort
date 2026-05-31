package tui

import (
	"audiosort/internal/cache"
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

// SetSize updates the cached terminal dimensions.
func (m MenuModel) SetSize(width, height int) subView {
	m.width = width
	m.height = height
	return m
}

// Update handles menu input
func (m MenuModel) Update(msg tea.Msg) (subView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, func() tea.Msg { return quitMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "enter", " ":
			action := m.items[m.cursor].title
			return m, func() tea.Msg { return menuActionMsg{action: action} }

		// Quick keys - "f" for search to avoid conflict with the "/" search idiom
		case "s", "1":
			return m, func() tea.Msg { return menuActionMsg{action: "Scan"} }
		case "f", "2":
			return m, func() tea.Msg { return menuActionMsg{action: "Search"} }
		case "c", "3":
			return m, func() tea.Msg { return menuActionMsg{action: "Config"} }
		case "x", "4":
			return m, func() tea.Msg { return menuActionMsg{action: "Cache"} }
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

// MainAppModel is the root application model. It is the single authority over
// the application lifecycle: sub-views never call tea.Quit themselves, they
// emit backToMenuMsg/quitMsg which the root interprets here.
type MainAppModel struct {
	config *config.Config
	view   AppView
	menu   MenuModel
	active subView // the focused non-menu sub-view (nil while on the menu)

	// cache is the single bbolt handle, opened lazily and shared by every view
	// (scan, cache). bbolt takes an exclusive file lock, so a second handle
	// would dead-lock; reuse this one everywhere.
	cache *cache.Store

	width  int
	height int
}

// ensureCache opens the shared cache handle on first use. The cache is optional,
// so a failure leaves it nil.
func (m MainAppModel) ensureCache() MainAppModel {
	if m.cache == nil {
		m.cache, _ = cache.NewStore(config.CachePath())
	}
	return m
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
		if mm, ok := m.menu.SetSize(msg.Width, msg.Height).(MenuModel); ok {
			m.menu = mm
		}
		if m.active != nil {
			m.active = m.active.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyMsg:
		// ctrl+c always quits, regardless of the focused view.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case quitMsg:
		return m, tea.Quit

	case backToMenuMsg:
		m.view = ViewMenu
		m.active = nil
		return m, nil

	case menuActionMsg:
		return m.handleMenuAction(msg.action)

	case pathConfirmMsg:
		if msg.confirmed {
			m = m.ensureCache()
			m.active = NewScanModel(msg.path, m.config, m.cache).SetSize(m.width, m.height)
			m.view = ViewScan
			return m, m.active.Init()
		}
		m.view = ViewMenu
		m.active = nil
		return m, nil
	}

	// Route everything else to the focused sub-view.
	var cmd tea.Cmd
	if m.view == ViewMenu || m.active == nil {
		next, c := m.menu.Update(msg)
		if menu, ok := next.(MenuModel); ok {
			m.menu = menu
		}
		cmd = c
	} else {
		m.active, cmd = m.active.Update(msg)
	}
	return m, cmd
}

// handleMenuAction handles menu selections
func (m MainAppModel) handleMenuAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "Scan":
		m.active = NewPathInputModel("Scan Audiobooks", "Enter path to audiobooks...", "").SetSize(m.width, m.height)
		m.view = ViewPathInput
		return m, m.active.Init()

	case "Search":
		m = m.ensureCache()
		m.active = NewSearchModel(m.config, m.cache, "").SetSize(m.width, m.height)
		m.view = ViewSearch
		return m, m.active.Init()

	case "Config":
		m.active = NewConfigModel(m.config).SetSize(m.width, m.height)
		m.view = ViewConfig
		return m, m.active.Init()

	case "Cache":
		m = m.ensureCache()
		m.active = NewCacheModel(m.cache, config.CachePath()).SetSize(m.width, m.height)
		m.view = ViewCache
		return m, m.active.Init()
	}

	return m, nil
}

// View renders the current view
func (m MainAppModel) View() string {
	if m.view == ViewMenu || m.active == nil {
		return m.menu.View()
	}
	return m.active.View()
}

// Message types
type pathConfirmMsg struct {
	path      string
	confirmed bool
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

	_, err := newProgram(NewMainAppModel(cfg)).Run()
	return err
}
