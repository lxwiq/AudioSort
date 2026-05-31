package tui

import (
	"testing"

	"audiosort/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// key builds a tea.KeyMsg from a key string the way bubbletea reports it.
func key(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case " ", "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// msgFromCmd executes a tea.Cmd and returns the message it produced.
func msgFromCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	return cmd()
}

func TestMenuQuitEmitsQuitMsg(t *testing.T) {
	m := NewMenuModel(config.DefaultConfig())
	_, cmd := m.Update(key("q"))
	if _, ok := msgFromCmd(t, cmd).(quitMsg); !ok {
		t.Fatalf("menu 'q' should emit quitMsg")
	}
}

func TestMenuQuickKeyEmitsAction(t *testing.T) {
	m := NewMenuModel(config.DefaultConfig())
	_, cmd := m.Update(key("f"))
	msg, ok := msgFromCmd(t, cmd).(menuActionMsg)
	if !ok || msg.action != "Search" {
		t.Fatalf("menu 'f' should emit menuActionMsg{Search}, got %#v", msg)
	}
}

func TestSubViewsEscReturnToMenu(t *testing.T) {
	cfg := config.DefaultConfig()
	cases := map[string]subView{
		"search": NewSearchModel(cfg, nil, ""),
		"config": NewConfigModel(cfg),
		"cache":  NewCacheModel(nil, ""),
		"scan":   NewScanModel("/tmp", cfg, nil),
	}
	for name, sv := range cases {
		_, cmd := sv.Update(key("esc"))
		if _, ok := msgFromCmd(t, cmd).(backToMenuMsg); !ok {
			t.Fatalf("%s 'esc' should emit backToMenuMsg", name)
		}
	}
}

func TestStandaloneTranslatesNavigationToQuit(t *testing.T) {
	s := standaloneModel{sub: NewSearchModel(config.DefaultConfig(), nil, "")}

	for _, nav := range []tea.Msg{backToMenuMsg{}, quitMsg{}} {
		_, cmd := s.Update(nav)
		if _, ok := msgFromCmd(t, cmd).(tea.QuitMsg); !ok {
			t.Fatalf("standalone wrapper should translate %T into tea.Quit", nav)
		}
	}
}

func TestMainAppNavigationAndCursorPreservation(t *testing.T) {
	m := NewMainAppModel(config.DefaultConfig())

	// Move the menu cursor down so we can later assert it is preserved.
	next, _ := m.Update(key("j"))
	m = next.(MainAppModel)
	if m.menu.cursor != 1 {
		t.Fatalf("menu cursor should be 1 after 'j', got %d", m.menu.cursor)
	}

	// Enter the Config view.
	next, _ = m.Update(menuActionMsg{action: "Config"})
	m = next.(MainAppModel)
	if m.view != ViewConfig || m.active == nil {
		t.Fatalf("expected ViewConfig with an active sub-view, got view=%d active=%v", m.view, m.active)
	}

	// Returning to the menu must clear the active view but keep the cursor.
	next, _ = m.Update(backToMenuMsg{})
	m = next.(MainAppModel)
	if m.view != ViewMenu || m.active != nil {
		t.Fatalf("expected ViewMenu with no active sub-view, got view=%d active=%v", m.view, m.active)
	}
	if m.menu.cursor != 1 {
		t.Fatalf("menu cursor should be preserved at 1 after returning, got %d", m.menu.cursor)
	}
}

func TestWindowSizePropagatesToActiveView(t *testing.T) {
	m := NewMainAppModel(config.DefaultConfig())
	next, _ := m.Update(menuActionMsg{action: "Search"})
	m = next.(MainAppModel)

	next, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(MainAppModel)

	sm, ok := m.active.(SearchModel)
	if !ok {
		t.Fatalf("active view should be a SearchModel, got %T", m.active)
	}
	if sm.width != 120 || sm.height != 40 {
		t.Fatalf("window size should propagate to active view, got %dx%d", sm.width, sm.height)
	}
}
