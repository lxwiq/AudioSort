package tui

import (
	"fmt"
	"path/filepath"

	"audiosort/internal/config"
	"audiosort/internal/core"
	"audiosort/pkg/models"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// state represents the current state of the TUI application
type state int

const (
	stateScanning state = iota
	stateReady
	stateProcessing
	stateDone
)

// Model is the main application model for the TUI
type Model struct {
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

	// State flags
	showHelp bool

	// Processing state
	progress       float64
	processedCount int
	summary        *models.Summary

	// UI components
	spinner spinner.Model
	width   int
	height  int
	err     error
}

// New creates a new TUI model with the given source path and configuration
func New(sourcePath string, cfg *config.Config) Model {
	scanner := core.NewScanner(cfg.ParallelWorkers)

	return Model{
		state:      stateScanning,
		config:     cfg,
		sourcePath: sourcePath,
		destPath:   cfg.DefaultOutput,
		scanner:    scanner,
		selected:   make(map[int]bool),
		spinner:    NewSpinner(),
		width:      80,
		height:     24,
	}
}

// Init initializes the TUI application
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		scanCmd(m.sourcePath, m.scanner),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case scanCompleteMsg:
		return m.handleScanComplete(msg)

	case metadataFetchedMsg:
		return m.handleMetadataFetched(msg)

	case processCompleteMsg:
		return m.handleProcessComplete(msg)

	case processProgressMsg:
		m.processedCount = msg.current
		m.progress = float64(msg.current) / float64(msg.total)
		return m, nil

	case tickMsg:
		return m, tickCmd()

	case errorMsg:
		m.err = msg.err
		m.state = stateDone
		return m, nil
	}

	return m, nil
}

// View renders the current view based on state
func (m Model) View() string {
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
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	// State-specific keys
	switch m.state {
	case stateReady:
		return m.handleReadyKeys(msg)
	case stateDone:
		return m, nil
	}

	return m, nil
}

// handleReadyKeys processes keys in the ready state
func (m Model) handleReadyKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "enter":
		return m.startProcessing()
	}

	return m, nil
}

// handleScanComplete processes the scan completion message
func (m Model) handleScanComplete(msg scanCompleteMsg) (tea.Model, tea.Cmd) {
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

// handleMetadataFetched processes metadata fetch results
func (m Model) handleMetadataFetched(msg metadataFetchedMsg) (tea.Model, tea.Cmd) {
	if msg.index >= 0 && msg.index < len(m.books) {
		if msg.err == nil && msg.metadata != nil {
			m.books[msg.index].Metadata = msg.metadata
		}
	}
	return m, nil
}

// handleProcessComplete processes the completion message
func (m Model) handleProcessComplete(msg processCompleteMsg) (tea.Model, tea.Cmd) {
	m.state = stateDone

	if msg.err != nil {
		m.err = msg.err
	} else {
		m.summary = msg.summary
	}

	return m, nil
}

// startProcessing initiates the processing pipeline
func (m Model) startProcessing() (tea.Model, tea.Cmd) {
	var selectedBooks []models.Audiobook
	for i, selected := range m.selected {
		if selected && i < len(m.books) {
			selectedBooks = append(selectedBooks, m.books[i])
		}
	}

	if len(selectedBooks) == 0 {
		return m, nil
	}

	m.state = stateProcessing
	m.progress = 0.0

	opts := core.PipelineOptions{
		SourcePath: m.sourcePath,
		DestPath:   m.destPath,
		Workers:    m.config.ParallelWorkers,
		CopyMode:   m.config.CopyMode,
		SkipExist:  m.config.SkipExisting,
		DryRun:     false,
	}

	m.pipeline = core.NewPipeline(opts)

	return m, tea.Batch(
		m.spinner.Tick,
		processCmd(selectedBooks, m.pipeline, m.sourcePath),
	)
}

// View methods

func (m Model) viewScanning() string {
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

func (m Model) viewBookList() string {
	var sections []string

	sections = append(sections, HeaderCompact("Scan Results"))
	sections = append(sections, "")

	// Summary
	selectedCount := 0
	for _, selected := range m.selected {
		if selected {
			selectedCount++
		}
	}
	summary := fmt.Sprintf("Found %d audiobooks, %d selected", len(m.books), selectedCount)
	sections = append(sections, SubtitleStyle.Render(summary))
	sections = append(sections, "")

	// Book list
	sections = append(sections, m.renderBookList())

	// Footer
	sections = append(sections, "")
	sections = append(sections, Footer(
		"↑/↓", "navigate",
		"space", "toggle",
		"a", "select all",
		"enter", "process",
		"?", "help",
	))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m Model) renderBookList() string {
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

		// Book info
		name := filepath.Base(book.Path)
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

func (m Model) viewProcessing() string {
	var sections []string

	sections = append(sections, HeaderCompact("Processing"))
	sections = append(sections, "")
	sections = append(sections, SpinnerWithText(m.spinner, "Processing audiobooks..."))
	sections = append(sections, "")

	// Progress bar
	total := 0
	for _, s := range m.selected {
		if s {
			total++
		}
	}
	if total == 0 {
		total = 1
	}

	sections = append(sections, Progress(m.processedCount, total, "Progress", m.width-10))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m Model) viewDone() string {
	var sections []string

	sections = append(sections, HeaderCompact("Complete"))
	sections = append(sections, "")

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
	}

	sections = append(sections, "")
	sections = append(sections, Footer("q", "quit"))

	return AppStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

func (m Model) viewHelp() string {
	var sections []string

	sections = append(sections, HeaderCompact("Help"))
	sections = append(sections, "")

	helpContent := []struct{ key, desc string }{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"space", "Toggle selection"},
		{"a", "Select/deselect all"},
		{"s", "Skip selected"},
		{"enter", "Process selected"},
		{"?", "Toggle help"},
		{"q", "Quit"},
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

// Run starts the TUI application
func Run(sourcePath string, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if cfg.DefaultOutput == "" {
		cfg.DefaultOutput = "./organized"
	}

	p := tea.NewProgram(
		New(sourcePath, cfg),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
