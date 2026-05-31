package tui

import (
	"context"

	"audiosort/internal/core"
	"audiosort/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
)

// scanCmd performs the initial scan for audiobooks under the given context, so
// the caller can cancel a long scan when leaving the view.
func scanCmd(ctx context.Context, path string, scanner *core.Scanner) tea.Cmd {
	return func() tea.Msg {
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

// processCmd processes the already-selected audiobooks through the pipeline,
// honouring the selection (no re-scan) and streaming progress over the channel.
// The context lets the caller cancel an in-flight run.
func processCmd(ctx context.Context, books []models.Audiobook, pipeline *core.Pipeline, progress chan<- core.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		summary := pipeline.ProcessBooks(ctx, books, progress)
		return processCompleteMsg{summary: summary}
	}
}

// listenProgress waits for the next progress update and turns it into a
// processProgressMsg. When the channel is closed it returns nil so the listen
// loop stops. The model re-issues this command after each update it receives.
func listenProgress(ch <-chan core.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return nil
		}
		return processProgressMsg{current: update.Done, total: update.Total}
	}
}
