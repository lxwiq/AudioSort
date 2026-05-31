package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"audiosort/internal/cache"
	"audiosort/internal/config"
	"audiosort/internal/core"
	"audiosort/pkg/models"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchModel is the model for the search TUI
type SearchModel struct {
	config *config.Config
	cache  *cache.Store

	// Input
	input    textinput.Model
	query    string
	searched bool

	// Results
	results  []models.BookMetadata
	cursor   int
	selected int

	// State
	searching bool
	spinner   spinner.Model
	err       error

	// Dimensions
	width  int
	height int
}

// Search result message
type searchResultsMsg struct {
	results []models.BookMetadata
	err     error
}

// NewSearchModel creates a new search model with the given configuration,
// shared cache handle (may be nil) and optional initial query.
func NewSearchModel(cfg *config.Config, store *cache.Store, initialQuery string) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Enter book title, author, or ISBN..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50
	ti.SetValue(initialQuery)

	return SearchModel{
		config:    cfg,
		cache:     store,
		input:     ti,
		query:     initialQuery,
		searching: initialQuery != "",
		spinner:   NewSpinner(),
		width:     80,
		height:    24,
		selected:  -1,
	}
}

// Init initializes the search model
func (m SearchModel) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink}

	// If we have an initial query, search immediately. The searching flag is
	// set by the constructor so it survives into the running model.
	if m.query != "" {
		cmds = append(cmds, m.spinner.Tick, m.doSearch(m.query))
	}

	return tea.Batch(cmds...)
}

// SetSize updates the cached terminal dimensions and resizes the input.
func (m SearchModel) SetSize(width, height int) subView {
	m.width = width
	m.height = height
	m.input.Width = width - 10
	return m
}

// Update handles messages for the search model
func (m SearchModel) Update(msg tea.Msg) (subView, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }

		case "enter":
			if !m.searching && !m.searched {
				// Start search
				m.query = m.input.Value()
				if m.query != "" {
					m.searching = true
					m.searched = false
					return m, tea.Batch(m.spinner.Tick, m.doSearch(m.query))
				}
			} else if m.searched && len(m.results) > 0 {
				// Select result and show details
				m.selected = m.cursor
			}

		case "up", "k":
			if m.searched && m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.searched && m.cursor < len(m.results)-1 {
				m.cursor++
			}

		case "tab":
			// Back to search
			if m.searched {
				m.searched = false
				m.results = nil
				m.cursor = 0
				m.selected = -1
				m.input.Focus()
				return m, textinput.Blink
			}

		case "q":
			if m.searched {
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
		}

		// Handle text input when not showing results
		if !m.searched && !m.searching {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}

	case spinner.TickMsg:
		if m.searching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case searchResultsMsg:
		m.searching = false
		m.searched = true
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.results = msg.results
			m.cursor = 0
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the search view
func (m SearchModel) View() string {
	var sections []string

	// Header
	sections = append(sections, HeaderCompact("Search Metadata"))
	sections = append(sections, "")

	if m.searching {
		// Searching state
		sections = append(sections, SpinnerWithText(m.spinner, "Searching..."))
	} else if m.searched {
		// Results state
		if m.err != nil {
			sections = append(sections, Alert("error", m.err.Error()))
		} else if len(m.results) == 0 {
			sections = append(sections, Alert("warning", "No results found for: "+m.query))
		} else {
			// Show results
			sections = append(sections, m.renderResults())
		}

		// Show selected book details
		if m.selected >= 0 && m.selected < len(m.results) {
			sections = append(sections, "")
			sections = append(sections, m.renderBookDetails(m.results[m.selected]))
		}
	} else {
		// Input state
		inputBox := InputFocusedStyle.Width(m.width - 6).Render(m.input.View())
		sections = append(sections, Section("Search Query", inputBox))
		sections = append(sections, "")
		sections = append(sections, DimStyle.Render("Press Enter to search"))
	}

	// Footer
	sections = append(sections, "")
	if m.searched {
		sections = append(sections, Footer("↑/↓", "navigate", "enter", "details", "tab", "new search", "q", "back"))
	} else {
		sections = append(sections, Footer("enter", "search", "esc", "back"))
	}

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

// renderResults renders the search results list
func (m SearchModel) renderResults() string {
	var items []string

	header := fmt.Sprintf("Found %d results for \"%s\"", len(m.results), m.query)
	items = append(items, SubtitleStyle.Render(header))
	items = append(items, "")

	for i, book := range m.results {
		style := ListItemStyle
		prefix := "  "

		if i == m.cursor {
			style = SelectedItemStyle
			prefix = "> "
		}

		author := "Unknown"
		if len(book.Authors) > 0 {
			author = book.Authors[0]
		}

		line := fmt.Sprintf("%s%s - %s", prefix, truncate(book.Title, 40), truncate(author, 30))
		if book.PublishYear != "" {
			line += DimStyle.Render(" (" + book.PublishYear + ")")
		}

		items = append(items, style.Render(line))
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderBookDetails renders details for a selected book
func (m SearchModel) renderBookDetails(book models.BookMetadata) string {
	var lines []string

	lines = append(lines, TitleStyle.Render(book.Title))
	lines = append(lines, "")

	// Authors
	if len(book.Authors) > 0 {
		lines = append(lines, BoldStyle.Render("Authors: ")+TextStyle.Render(strings.Join(book.Authors, ", ")))
	}

	// Narrators
	if len(book.Narrators) > 0 {
		lines = append(lines, BoldStyle.Render("Narrators: ")+TextStyle.Render(strings.Join(book.Narrators, ", ")))
	}

	// Series
	if book.Series != "" {
		seriesInfo := book.Series
		if book.SeriesPosition != "" {
			seriesInfo += " #" + book.SeriesPosition
		}
		lines = append(lines, BoldStyle.Render("Series: ")+TextStyle.Render(seriesInfo))
	}

	// Publisher & Year
	if book.Publisher != "" || book.PublishYear != "" {
		pub := book.Publisher
		if book.PublishYear != "" {
			if pub != "" {
				pub += ", "
			}
			pub += book.PublishYear
		}
		lines = append(lines, BoldStyle.Render("Publisher: ")+TextStyle.Render(pub))
	}

	// Language
	if book.Language != "" {
		lines = append(lines, BoldStyle.Render("Language: ")+TextStyle.Render(book.Language))
	}

	// ISBN
	if book.ISBN != "" {
		lines = append(lines, BoldStyle.Render("ISBN: ")+TextStyle.Render(book.ISBN))
	}

	// Source
	lines = append(lines, BoldStyle.Render("Source: ")+DimStyle.Render(book.Source))

	// Summary (truncated)
	if book.Summary != "" {
		lines = append(lines, "")
		summary := truncate(book.Summary, 200)
		lines = append(lines, DimStyle.Render(summary))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return BoxStyle.Width(m.width - 6).Render(content)
}

// doSearch creates a command to search for metadata. It uses the same source
// set as the scan pipeline (via core.MetadataSources): the configured sources,
// in order, including BookInfo — instead of the previous hard-coded
// GoogleBooks+OpenLibrary subset.
func (m SearchModel) doSearch(query string) tea.Cmd {
	cfg := m.config
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		sources := core.MetadataSources(cfg)

		var allResults []models.BookMetadata
		var lastErr error
		for _, source := range sources {
			results, err := source.Search(ctx, query)
			if err != nil {
				lastErr = err
				continue
			}
			allResults = append(allResults, results...)
		}

		// Surface an error only when no source returned anything.
		if len(allResults) == 0 && lastErr != nil {
			return searchResultsMsg{err: lastErr}
		}

		// Limit results
		if len(allResults) > 10 {
			allResults = allResults[:10]
		}

		return searchResultsMsg{results: allResults}
	}
}

// RunSearch runs the search TUI
func RunSearch(query string, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	store, _ := cache.NewStore(config.CachePath()) // optional; nil on failure
	_, err := newStandaloneProgram(NewSearchModel(cfg, store, query)).Run()
	return err
}
