package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
)

// viewScanning renders the scanning state
func (m Model) viewScanning() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AudioSort - Scanning"))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString(" Scanning for audiobooks in: ")
	b.WriteString(dimmedStyle.Render(m.sourcePath))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Please wait..."))

	return b.String()
}

// viewBookList renders the main book selection list
func (m Model) viewBookList() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("AudioSort - Select Audiobooks to Process"))
	b.WriteString("\n\n")

	if len(m.books) == 0 {
		b.WriteString(errorStyle.Render("No audiobooks found in the specified directory."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press q to quit"))
		return b.String()
	}

	// Show statistics
	totalSelected := 0
	for _, selected := range m.selected {
		if selected {
			totalSelected++
		}
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("Found %d audiobook(s) | %d selected", len(m.books), totalSelected)))
	b.WriteString("\n\n")

	// Calculate visible window
	height := m.height - 10 // Reserve space for header/footer
	if height <= 0 {
		height = 10
	}

	start := m.cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(m.books) {
		end = len(m.books)
		start = end - height
		if start < 0 {
			start = 0
		}
	}

	// Render book list
	for i := start; i < end && i < len(m.books); i++ {
		book := m.books[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := checkboxUnselected.String()
		if m.selected[i] {
			checkbox = checkboxSelected.String()
		}

		// Format book info
		title := filepath.Base(book.Path)
		if book.Metadata != nil && book.Metadata.Title != "" {
			title = book.Metadata.Title
			if len(book.Metadata.Authors) > 0 {
				title += " - " + book.Metadata.Authors[0]
			}
		}

		// Truncate if too long
		maxWidth := m.width - 20
		if maxWidth <= 0 {
			maxWidth = 60
		}
		if len(title) > maxWidth {
			title = title[:maxWidth-3] + "..."
		}

		status := string(book.Status)
		statusStyled := statusToStyle(status).Render(status)

		line := fmt.Sprintf("%s%s %s %s", cursor, checkbox, title, statusStyled)

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Show scroll indicator if needed
	if len(m.books) > height {
		b.WriteString("\n")
		b.WriteString(dimmedStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.books))))
	}

	// Help text
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/k: up • ↓/j: down • space: select • a: select all • enter: process • s: skip • q: quit • ?: help"))

	return b.String()
}

// viewProcessing renders the processing state
func (m Model) viewProcessing() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AudioSort - Processing"))
	b.WriteString("\n\n")

	// Show progress
	totalSelected := 0
	for _, selected := range m.selected {
		if selected {
			totalSelected++
		}
	}

	if totalSelected > 0 {
		progress := m.progress
		b.WriteString(fmt.Sprintf("Processing %d audiobook(s)...\n\n", totalSelected))
		b.WriteString(renderProgressBar(progress, m.width-10))
		b.WriteString(fmt.Sprintf(" %.0f%%\n", progress*100))
	} else {
		b.WriteString(m.spinner.View())
		b.WriteString(" Processing audiobooks...\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Please wait..."))

	return b.String()
}

// viewDone renders the completion summary
func (m Model) viewDone() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AudioSort - Complete"))
	b.WriteString("\n\n")

	if m.summary != nil {
		b.WriteString(headerStyle.Render("Summary"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Total:     %d\n", m.summary.Total))
		b.WriteString(statusDoneStyle.Render(fmt.Sprintf("Processed: %d\n", m.summary.Processed)))
		b.WriteString(statusSkippedStyle.Render(fmt.Sprintf("Skipped:   %d\n", m.summary.Skipped)))
		b.WriteString(statusErrorStyle.Render(fmt.Sprintf("Errors:    %d\n", m.summary.Errors)))
		b.WriteString(fmt.Sprintf("Duration:  %s\n", m.summary.Duration))

		if m.summary.Errors > 0 && len(m.summary.Failures) > 0 {
			b.WriteString("\n")
			b.WriteString(errorStyle.Render("Errors:"))
			b.WriteString("\n")
			for _, failure := range m.summary.Failures {
				if failure.Audiobook != nil {
					b.WriteString(fmt.Sprintf("  - %s: %v\n", filepath.Base(failure.Audiobook.Path), failure.Error))
				}
			}
		}
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("Error: "))
		b.WriteString(m.err.Error())
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press q to quit"))

	return b.String()
}

// viewHelp renders the help screen
func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AudioSort - Help"))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString("  ↑/k       Move cursor up\n")
	b.WriteString("  ↓/j       Move cursor down\n")
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Selection"))
	b.WriteString("\n")
	b.WriteString("  space     Toggle selection of current item\n")
	b.WriteString("  a         Select all audiobooks\n")
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Actions"))
	b.WriteString("\n")
	b.WriteString("  enter     Process selected audiobooks\n")
	b.WriteString("  s         Skip selected audiobooks\n")
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Other"))
	b.WriteString("\n")
	b.WriteString("  ?         Toggle this help screen\n")
	b.WriteString("  q         Quit application\n")

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press ? to return"))

	return b.String()
}

// newSpinner creates a new spinner for loading states
func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = statusProcessingStyle
	return s
}
