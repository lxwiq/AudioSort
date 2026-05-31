package tui

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// expandHome expands a leading "~" in a path to the user's home directory.
// If expansion fails or the path does not start with "~", it is returned unchanged.
func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// newProgram builds a bubbletea program with the options shared across every
// TUI entry point. Centralising this avoids the configuration drift that came
// from seven separate tea.NewProgram call sites.
func newProgram(model tea.Model) *tea.Program {
	return tea.NewProgram(model, tea.WithAltScreen())
}
