package tui

import (
	"context"
	"fmt"
	"path/filepath"

	"audiosort/internal/cache"
	"audiosort/internal/config"
	"audiosort/internal/core"
	"audiosort/pkg/models"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// subView is the common contract implemented by every screen. It mirrors the
// Elm-style tea.Model but Update returns a subView (so the root can hold the
// active screen without per-type assertions) and SetSize lets the root push
// the authoritative terminal dimensions down at creation time.
type subView interface {
	Init() tea.Cmd
	Update(tea.Msg) (subView, tea.Cmd)
	View() string
	SetSize(width, height int) subView
}

// standaloneModel wraps a single sub-view so it can run as the root program of
// a CLI sub-command. There is no menu in that mode, so backToMenuMsg and
// quitMsg both become tea.Quit.
type standaloneModel struct {
	sub subView
}

// newStandaloneProgram builds a program that runs a single sub-view directly.
func newStandaloneProgram(sub subView) *tea.Program {
	return newProgram(standaloneModel{sub: sub})
}

func (s standaloneModel) Init() tea.Cmd { return s.sub.Init() }

func (s standaloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.sub = s.sub.SetSize(msg.Width, msg.Height)
		return s, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return s, tea.Quit
		}
	case backToMenuMsg, quitMsg:
		return s, tea.Quit
	}

	var cmd tea.Cmd
	s.sub, cmd = s.sub.Update(msg)
	return s, cmd
}

func (s standaloneModel) View() string { return s.sub.View() }

// state represents the current state of the scan view
type state int

const (
	stateScanning state = iota
	stateReady
	stateProcessing
	stateDone
)

// ScanModel drives the scan / selection / processing screen.
type ScanModel struct {
	state  state
	config *config.Config

	// Paths
	sourcePath string
	destPath   string

	// Book data
	books    []models.Audiobook
	cursor   int
	selected map[int]bool

	// Pipeline components
	scanner  *core.Scanner
	pipeline *core.Pipeline

	// Shared cache handle (owned by the root model / standalone entry point)
	cache *cache.Store

	// Cancellation for the in-flight scan / processing run. Stored on the model
	// (copied by closure across Update copies, which all share one cancel func)
	// so leaving the view can stop the work.
	ctx    context.Context
	cancel context.CancelFunc

	// State flags
	showHelp bool
	dryRun   bool // when true, simulate processing without moving any files

	// Processing state
	progress       float64
	processedCount int
	progressCh     chan core.ProgressUpdate
	summary        *models.Summary

	// UI components
	spinner spinner.Model
	width   int
	height  int
	err     error
}

// NewScanModel creates a new scan model with the given source path, configuration
// and shared cache handle (which may be nil; the cache is optional).
func NewScanModel(sourcePath string, cfg *config.Config, store *cache.Store) ScanModel {
	scanner := core.NewScanner(cfg.ParallelWorkers)
	ctx, cancel := context.WithCancel(context.Background())

	return ScanModel{
		state:      stateScanning,
		config:     cfg,
		sourcePath: sourcePath,
		destPath:   cfg.DefaultOutput,
		scanner:    scanner,
		cache:      store,
		ctx:        ctx,
		cancel:     cancel,
		selected:   make(map[int]bool),
		spinner:    NewSpinner(),
		width:      80,
		height:     24,
	}
}

// Init initializes the scan view
func (m ScanModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		scanCmd(m.ctx, m.sourcePath, m.scanner),
	)
}

// SetSize updates the cached terminal dimensions.
func (m ScanModel) SetSize(width, height int) subView {
	m.width = width
	m.height = height
	return m
}

// Update handles messages and updates the model
func (m ScanModel) Update(msg tea.Msg) (subView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case scanCompleteMsg:
		return m.handleScanComplete(msg)

	case processCompleteMsg:
		return m.handleProcessComplete(msg)

	case processProgressMsg:
		m.processedCount = msg.current
		if msg.total > 0 {
			m.progress = float64(msg.current) / float64(msg.total)
		}
		// Keep listening for the next update until the channel closes.
		return m, listenProgress(m.progressCh)
	}

	return m, nil
}

// View renders the current view based on state
func (m ScanModel) View() string {
	if m.showHelp {
		return m.viewHelp()
	}

	switch m.state {
	case stateScanning:
		return m.viewScanning()
	case stateReady:
		return m.viewBookList()
	case stateProcessing:
		return m.viewProcessing()
	case stateDone:
		return m.viewDone()
	default:
		return "Unknown state"
	}
}

// handleKeyPress processes keyboard input based on current state
func (m ScanModel) handleKeyPress(msg tea.KeyMsg) (subView, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "q", "esc":
		// Cancel any in-flight scan/processing before leaving.
		if m.cancel != nil {
			m.cancel()
		}
		return m, func() tea.Msg { return backToMenuMsg{} }
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	// State-specific keys
	switch m.state {
	case stateReady:
		return m.handleReadyKeys(msg)
	}

	return m, nil
}

// handleReadyKeys processes keys in the ready state
func (m ScanModel) handleReadyKeys(msg tea.KeyMsg) (subView, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.books)-1 {
			m.cursor++
		}
	case " ":
		m.selected[m.cursor] = !m.selected[m.cursor]
	case "a":
		allSelected := len(m.selected) == len(m.books)
		m.selected = make(map[int]bool)
		if !allSelected {
			for i := range m.books {
				m.selected[i] = true
			}
		}
	case "s":
		for i := range m.selected {
			if m.selected[i] {
				m.books[i].Status = models.StatusSkipped
			}
		}
		m.selected = make(map[int]bool)
	case "d":
		m.dryRun = !m.dryRun
	case "enter":
		return m.startProcessing()
	}

	return m, nil
}

// handleScanComplete processes the scan completion message
func (m ScanModel) handleScanComplete(msg scanCompleteMsg) (subView, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = stateDone
		return m, nil
	}

	m.books = msg.books
	m.state = stateReady

	m.selected = make(map[int]bool)
	for i := range m.books {
		m.selected[i] = true
	}

	return m, nil
}

// handleProcessComplete processes the completion message
func (m ScanModel) handleProcessComplete(msg processCompleteMsg) (subView, tea.Cmd) {
	m.state = stateDone

	if msg.err != nil {
		m.err = msg.err
	} else {
		m.summary = msg.summary
	}

	return m, nil
}

// selectedBooks returns the audiobooks the user has selected for processing.
func (m ScanModel) selectedBooks() []models.Audiobook {
	var books []models.Audiobook
	for i, selected := range m.selected {
		if selected && i >= 0 && i < len(m.books) {
			books = append(books, m.books[i])
		}
	}
	return books
}

// selectedCount returns how many books are currently selected.
func (m ScanModel) selectedCount() int {
	count := 0
	for _, selected := range m.selected {
		if selected {
			count++
		}
	}
	return count
}

// startProcessing initiates the processing pipeline
func (m ScanModel) startProcessing() (subView, tea.Cmd) {
	selectedBooks := m.selectedBooks()
	if len(selectedBooks) == 0 {
		return m, nil
	}

	m.state = stateProcessing
	m.progress = 0.0

	// Presentation no longer assembles sources/cache itself: the factory in
	// internal/core does it, reusing the shared cache handle.
	fetcher := core.BuildFetcher(m.config, m.cache)
	m.pipeline = core.BuildPipeline(m.config, m.sourcePath, m.destPath, fetcher, m.dryRun)

	// Buffer the channel for the whole run so the pipeline never blocks on a
	// slow UI consumer; listenProgress drains it at the UI's pace.
	m.progressCh = make(chan core.ProgressUpdate, len(selectedBooks)+1)

	return m, tea.Batch(
		m.spinner.Tick,
		processCmd(m.ctx, selectedBooks, m.pipeline, m.progressCh),
		listenProgress(m.progressCh),
	)
}

// View methods

func (m ScanModel) viewScanning() string {
	var sections []string

	sections = append(sections, Header(m.width, "Scanning Audiobooks"))
	sections = append(sections, "")
	sections = append(sections, SpinnerWithText(m.spinner, "Scanning "+m.sourcePath+"..."))
	sections = append(sections, "")
	sections = append(sections, DimStyle.Render("Looking for audio files..."))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m ScanModel) viewBookList() string {
	var sections []string

	sections = append(sections, HeaderCompact("Scan Results"))
	sections = append(sections, "")

	// Summary
	summary := fmt.Sprintf("Found %d audiobooks, %d selected", len(m.books), m.selectedCount())
	sections = append(sections, SubtitleStyle.Render(summary))

	// Dry-run banner
	if m.dryRun {
		sections = append(sections, WarningStyle.Render("DRY RUN: files will NOT be moved (press d to toggle)"))
	}
	sections = append(sections, "")

	// Book list
	sections = append(sections, m.renderBookList())

	// Footer
	sections = append(sections, "")
	dryLabel := "dry-run: off"
	if m.dryRun {
		dryLabel = "dry-run: on"
	}
	sections = append(sections, Footer(
		"↑/↓", "navigate",
		"space", "toggle",
		"a", "select all",
		"d", dryLabel,
		"enter", "process",
		"esc", "back",
	))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m ScanModel) renderBookList() string {
	if len(m.books) == 0 {
		return Alert("warning", "No audiobooks found in "+m.sourcePath)
	}

	var lines []string
	maxVisible := m.height - 15
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.books) {
		end = len(m.books)
	}

	for i := start; i < end; i++ {
		book := m.books[i]
		style := ListItemStyle

		// Checkbox
		checkbox := DimStyle.Render("[ ] ")
		if m.selected[i] {
			checkbox = SuccessStyle.Render("[x] ")
		}

		// Cursor
		cursor := "  "
		if i == m.cursor {
			style = SelectedItemStyle
			cursor = InfoStyle.Render("> ")
		}

		// Book info: prefer the title detected from embedded tags.
		name := filepath.Base(book.Path)
		if book.Probe != nil && book.Probe.Title != "" {
			name = book.Probe.Title
		}
		status := RenderStatusBadge(string(book.Status))

		line := cursor + checkbox + truncate(name, m.width-20) + " " + status
		lines = append(lines, style.Render(line))
	}

	// Scroll indicator
	if len(m.books) > maxVisible {
		scrollInfo := DimStyle.Render(fmt.Sprintf("  [%d/%d]", m.cursor+1, len(m.books)))
		lines = append(lines, scrollInfo)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m ScanModel) viewProcessing() string {
	var sections []string

	sections = append(sections, HeaderCompact("Processing"))
	sections = append(sections, "")
	label := "Processing audiobooks..."
	if m.dryRun {
		label = "Simulating (dry run)..."
	}
	sections = append(sections, SpinnerWithText(m.spinner, label))
	sections = append(sections, "")

	total := m.selectedCount()
	if total == 0 {
		total = 1
	}

	sections = append(sections, Progress(m.processedCount, total, "Progress", m.width-10))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m ScanModel) viewDone() string {
	var sections []string

	sections = append(sections, HeaderCompact("Complete"))
	sections = append(sections, "")

	if m.dryRun && m.err == nil {
		sections = append(sections, Alert("warning", "DRY RUN — no files were moved"))
		sections = append(sections, "")
	}

	if m.err != nil {
		sections = append(sections, Alert("error", m.err.Error()))
	} else if m.summary != nil {
		// Summary box
		var summaryLines []string
		summaryLines = append(summaryLines, TitleStyle.Render("Processing Summary"))
		summaryLines = append(summaryLines, "")
		summaryLines = append(summaryLines, SuccessStyle.Render(fmt.Sprintf("%s Processed: %d", IconCheck, m.summary.Processed)))
		summaryLines = append(summaryLines, WarningStyle.Render(fmt.Sprintf("! Skipped:   %d", m.summary.Skipped)))
		summaryLines = append(summaryLines, ErrorMsgStyle.Render(fmt.Sprintf("%s Errors:    %d", IconCross, m.summary.Errors)))
		summaryLines = append(summaryLines, "")
		summaryLines = append(summaryLines, DimStyle.Render(fmt.Sprintf("Total: %d in %s", m.summary.Total, m.summary.Duration.Round(1e9).String())))

		content := lipgloss.JoinVertical(lipgloss.Left, summaryLines...)
		sections = append(sections, BoxStyle.Width(m.width-6).Render(content))

		// Show error details if any
		if len(m.summary.Failures) > 0 {
			sections = append(sections, "")
			sections = append(sections, TitleStyle.Render("Error Details"))
			sections = append(sections, "")

			maxErrors := 10 // Limit to avoid overflow
			for i, failure := range m.summary.Failures {
				if i >= maxErrors {
					remaining := len(m.summary.Failures) - maxErrors
					sections = append(sections, DimStyle.Render(fmt.Sprintf("  ... and %d more errors", remaining)))
					break
				}

				bookName := "Unknown"
				if failure.Audiobook != nil && failure.Audiobook.Path != "" {
					bookName = filepath.Base(failure.Audiobook.Path)
				}

				errMsg := "Unknown error"
				if failure.Error != nil {
					errMsg = failure.Error.Error()
				}

				sections = append(sections, ErrorMsgStyle.Render(fmt.Sprintf("  %s %s", IconCross, bookName)))
				sections = append(sections, DimStyle.Render(fmt.Sprintf("    %s", truncate(errMsg, m.width-10))))
			}
		}
	}

	sections = append(sections, "")
	sections = append(sections, Footer("esc", "back to menu", "q", "back to menu"))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m ScanModel) viewHelp() string {
	var sections []string

	sections = append(sections, HeaderCompact("Help"))
	sections = append(sections, "")

	helpContent := []struct{ key, desc string }{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"space", "Toggle selection"},
		{"a", "Select/deselect all"},
		{"s", "Skip selected"},
		{"d", "Toggle dry-run (no file moves)"},
		{"enter", "Process selected"},
		{"?", "Toggle help"},
		{"esc/q", "Back to menu"},
	}

	var lines []string
	for _, h := range helpContent {
		lines = append(lines, RenderHelpKey(h.key, h.desc))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	sections = append(sections, BoxStyle.Width(m.width-6).Render(content))

	sections = append(sections, "")
	sections = append(sections, Footer("?", "close help"))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

// Run starts the scan TUI as a standalone CLI sub-command
func Run(sourcePath string, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	store, _ := cache.NewStore(config.CachePath()) // optional; nil on failure
	_, err := newStandaloneProgram(NewScanModel(sourcePath, cfg, store)).Run()
	return err
}
