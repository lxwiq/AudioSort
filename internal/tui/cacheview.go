package tui

import (
	"fmt"
	"os"

	"audiosort/internal/cache"
	"audiosort/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CacheModel is the model for the cache TUI
type CacheModel struct {
	store     *cache.Store // shared handle (may be nil)
	cachePath string
	cacheInfo cacheStats

	// Entry counts (computed on load / refresh / clear, not per render)
	metaCount      int
	processedCount int

	// State
	confirmClear bool
	confirmFocus bool // true = yes, false = no
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

// NewCacheModel creates a new cache model bound to the shared cache handle
// (which may be nil). The path is used only for file-level statistics.
func NewCacheModel(store *cache.Store, cachePath string) CacheModel {
	if cachePath == "" {
		cachePath = config.CachePath()
	}

	m := CacheModel{
		store:     store,
		cachePath: cachePath,
		width:     80,
		height:    24,
	}
	return m.refreshStats()
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

	return stats
}

// refreshStats recomputes the file stats and entry counts once, so View never
// has to touch disk or open a bbolt transaction per frame.
func (m CacheModel) refreshStats() CacheModel {
	m.cacheInfo = getCacheStats(m.cachePath)
	if m.store != nil {
		m.metaCount, m.processedCount = m.store.Stats()
	}
	return m
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

// SetSize updates the cached terminal dimensions.
func (m CacheModel) SetSize(width, height int) subView {
	m.width = width
	m.height = height
	return m
}

// Update handles messages for the cache model
func (m CacheModel) Update(msg tea.Msg) (subView, tea.Cmd) {
	switch msg := msg.(type) {
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
					return m, m.doClearCache()
				}
				m.confirmClear = false
			case "esc", "n":
				m.confirmClear = false
			case "y":
				return m, m.doClearCache()
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }

		case "c":
			// Reset any stale status, then ask for confirmation.
			m.message = ""
			m.err = nil
			if m.cacheInfo.exists {
				m.confirmClear = true
				m.confirmFocus = false // Default to "No"
			} else {
				m.message = "No cache file to clear"
			}

		case "r":
			m = m.refreshStats()
			m.message = "Stats refreshed"
			m.err = nil
		}

	case cacheClearedMsg:
		m.confirmClear = false
		if msg.err != nil {
			m.err = msg.err
			m.message = "Error: " + msg.err.Error()
		} else {
			m.err = nil
			m = m.refreshStats()
			m.message = "Cache cleared successfully!"
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
		} else {
			sections = append(sections, Alert("success", m.message))
		}
	}

	// Footer
	sections = append(sections, "")
	if m.confirmClear {
		sections = append(sections, Footer("←/→", "select", "enter", "confirm", "esc", "cancel"))
	} else {
		sections = append(sections, Footer("c", "clear cache", "r", "refresh", "q", "back"))
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
		if m.store != nil {
			lines = append(lines, BoldStyle.Render("Entries:  ")+TextStyle.Render(
				fmt.Sprintf("%d metadata, %d processed", m.metaCount, m.processedCount)))
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

// doClearCache clears the cache through the shared store handle (no second
// bbolt open, no file removal of an open database).
func (m CacheModel) doClearCache() tea.Cmd {
	store := m.store
	return func() tea.Msg {
		if store == nil {
			return cacheClearedMsg{err: fmt.Errorf("cache is not available")}
		}
		return cacheClearedMsg{err: store.Clear()}
	}
}

// RunCache runs the cache TUI
func RunCache(cachePath string) error {
	if cachePath == "" {
		cachePath = config.CachePath()
	}

	store, _ := cache.NewStore(cachePath) // optional; nil on failure
	_, err := newStandaloneProgram(NewCacheModel(store, cachePath)).Run()
	return err
}
