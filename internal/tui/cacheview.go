package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"audiosort/internal/cache"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CacheModel is the model for the cache TUI
type CacheModel struct {
	cachePath string
	cacheInfo cacheStats

	// State
	confirmClear bool
	confirmFocus bool // true = yes, false = no
	cleared      bool
	message      string
	err          error

	// Dimensions
	width  int
	height int
}

type cacheStats struct {
	path    string
	exists  bool
	size    int64
	sizeStr string
	modTime string
}

// cacheCleared message
type cacheClearedMsg struct {
	err error
}

// NewCacheModel creates a new cache model
func NewCacheModel(cachePath string) CacheModel {
	if cachePath == "" {
		home, _ := os.UserHomeDir()
		cachePath = filepath.Join(home, ".cache", "audiosort", "metadata.db")
	}

	stats := getCacheStats(cachePath)

	return CacheModel{
		cachePath: cachePath,
		cacheInfo: stats,
		width:     80,
		height:    24,
	}
}

func getCacheStats(path string) cacheStats {
	stats := cacheStats{path: path}

	info, err := os.Stat(path)
	if err != nil {
		stats.exists = false
		return stats
	}

	stats.exists = true
	stats.size = info.Size()
	stats.sizeStr = formatBytes(info.Size())
	stats.modTime = info.ModTime().Format("2006-01-02 15:04:05")

	// Try to get entry count
	store, err := cache.NewStore(path)
	if err == nil {
		// Note: We'd need to add a method to get count
		// For now, just show file stats
		store.Close()
	}

	return stats
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Init initializes the cache model
func (m CacheModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the cache model
func (m CacheModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.confirmClear {
			// Handle confirmation dialog
			switch msg.String() {
			case "left", "h":
				m.confirmFocus = true
			case "right", "l":
				m.confirmFocus = false
			case "enter":
				if m.confirmFocus {
					// Clear cache
					return m, m.doClearCache()
				} else {
					m.confirmClear = false
				}
			case "esc", "n":
				m.confirmClear = false
			case "y":
				return m, m.doClearCache()
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "c":
			// Show confirmation
			if m.cacheInfo.exists && !m.cleared {
				m.confirmClear = true
				m.confirmFocus = false // Default to "No"
			}

		case "r":
			// Refresh stats
			m.cacheInfo = getCacheStats(m.cachePath)
			m.message = "Stats refreshed"
		}

	case cacheClearedMsg:
		m.confirmClear = false
		if msg.err != nil {
			m.err = msg.err
			m.message = "Error: " + msg.err.Error()
		} else {
			m.cleared = true
			m.message = "Cache cleared successfully!"
			m.cacheInfo = getCacheStats(m.cachePath)
		}
	}

	return m, nil
}

// View renders the cache view
func (m CacheModel) View() string {
	var sections []string

	// Header
	sections = append(sections, HeaderCompact("Cache Management"))
	sections = append(sections, "")

	// Cache info
	sections = append(sections, m.renderCacheInfo())

	// Confirmation dialog
	if m.confirmClear {
		sections = append(sections, "")
		sections = append(sections, m.renderConfirmDialog())
	}

	// Message
	if m.message != "" && !m.confirmClear {
		sections = append(sections, "")
		if m.err != nil {
			sections = append(sections, Alert("error", m.message))
		} else if m.cleared {
			sections = append(sections, Alert("success", m.message))
		} else {
			sections = append(sections, Alert("info", m.message))
		}
	}

	// Footer
	sections = append(sections, "")
	if m.confirmClear {
		sections = append(sections, Footer("←/→", "select", "enter", "confirm", "esc", "cancel"))
	} else {
		sections = append(sections, Footer("c", "clear cache", "r", "refresh", "q", "quit"))
	}

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m CacheModel) renderCacheInfo() string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Cache Information"))
	lines = append(lines, "")

	if !m.cacheInfo.exists {
		lines = append(lines, DimStyle.Render("No cache file found"))
		lines = append(lines, "")
		lines = append(lines, TextStyle.Render("Path: ")+DimStyle.Render(m.cachePath))
	} else {
		lines = append(lines, BoldStyle.Render("Path:     ")+TextStyle.Render(m.cacheInfo.path))
		lines = append(lines, BoldStyle.Render("Size:     ")+TextStyle.Render(m.cacheInfo.sizeStr))
		lines = append(lines, BoldStyle.Render("Modified: ")+TextStyle.Render(m.cacheInfo.modTime))

		if m.cleared {
			lines = append(lines, "")
			lines = append(lines, SuccessStyle.Render(IconCheck+" Cache has been cleared"))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return BoxStyle.Width(m.width - 6).Render(content)
}

func (m CacheModel) renderConfirmDialog() string {
	yesStyle := ButtonStyle
	noStyle := ButtonStyle

	if m.confirmFocus {
		yesStyle = ButtonActiveStyle
	} else {
		noStyle = ButtonActiveStyle
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Left,
		yesStyle.Render(" Yes "),
		"  ",
		noStyle.Render(" No "),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		WarningStyle.Render("Clear all cached metadata?"),
		"",
		DimStyle.Render("This will remove all cached book information."),
		DimStyle.Render("You may need to re-fetch metadata for processed books."),
		"",
		buttons,
	)

	return BoxStyle.
		BorderForeground(ColorWarning).
		Width(m.width - 6).
		Align(lipgloss.Center).
		Render(content)
}

func (m CacheModel) doClearCache() tea.Cmd {
	return func() tea.Msg {
		err := os.Remove(m.cachePath)
		if err != nil && !os.IsNotExist(err) {
			return cacheClearedMsg{err: err}
		}
		return cacheClearedMsg{}
	}
}

// RunCache runs the cache TUI
func RunCache(cachePath string) error {
	if cachePath == "" {
		home, _ := os.UserHomeDir()
		cachePath = filepath.Join(home, ".cache", "audiosort", "metadata.db")
	}

	p := tea.NewProgram(
		NewCacheModel(cachePath),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
