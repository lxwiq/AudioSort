package tui

import (
	"context"
	"time"

	"audiosort/internal/core"
	"audiosort/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
)

// scanCmd performs the initial scan for audiobooks
func scanCmd(path string, scanner *core.Scanner) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		audiobookChan, err := scanner.Scan(ctx, path)
		if err != nil {
			return scanCompleteMsg{err: err}
		}

		// Collect all audiobooks from the channel
		var books []models.Audiobook
		for book := range audiobookChan {
			books = append(books, book)
		}

		return scanCompleteMsg{books: books}
	}
}

// processCmd processes selected audiobooks through the pipeline
func processCmd(books []models.Audiobook, pipeline *core.Pipeline, sourcePath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		summary, err := pipeline.Run(ctx, sourcePath)
		if err != nil {
			return processCompleteMsg{err: err}
		}

		return processCompleteMsg{summary: summary}
	}
}

// tickCmd sends periodic tick messages for spinner animation
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
