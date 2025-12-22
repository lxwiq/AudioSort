package tui

import (
	"audiosort/internal/config"
	"audiosort/internal/core"
	"audiosort/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
)

// state represents the current state of the TUI application
type state int

const (
	stateScanning state = iota
	stateReady
	stateProcessing
	stateDone
	stateHelp
)

// Model is the main application model for the TUI
type Model struct {
	state   state
	config  *config.Config

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
	scanning   bool
	processing bool
	showHelp   bool

	// Processing state
	progress float64
	summary  *models.Summary

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
		spinner:    newSpinner(),
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
		// Only allow quit in done state
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
		// Toggle selection
		m.selected[m.cursor] = !m.selected[m.cursor]
	case "a":
		// Select all
		allSelected := len(m.selected) == len(m.books)
		m.selected = make(map[int]bool)
		if !allSelected {
			for i := range m.books {
				m.selected[i] = true
			}
		}
	case "s":
		// Skip selected
		for i := range m.selected {
			if m.selected[i] {
				m.books[i].Status = models.StatusSkipped
			}
		}
		m.selected = make(map[int]bool)
	case "enter":
		// Process selected books
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
	m.scanning = false
	m.state = stateReady

	// Auto-select all books
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
	m.processing = false
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
	// Collect selected books
	var selectedBooks []models.Audiobook
	for i, selected := range m.selected {
		if selected && i < len(m.books) {
			selectedBooks = append(selectedBooks, m.books[i])
		}
	}

	if len(selectedBooks) == 0 {
		// Nothing to process
		return m, nil
	}

	m.state = stateProcessing
	m.processing = true
	m.progress = 0.0

	// Create pipeline
	// Note: This is a simplified version. In a real implementation,
	// you would need to properly initialize the metadata fetcher and writers
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

// Run starts the TUI application
func Run(sourcePath string, cfg *config.Config) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Set default output if not configured
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
