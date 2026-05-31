package tui

import "audiosort/pkg/models"

// scanCompleteMsg is sent when the initial scan is complete
type scanCompleteMsg struct {
	books []models.Audiobook
	err   error
}

// processProgressMsg is sent to update processing progress
type processProgressMsg struct {
	current int
	total   int
}

// processCompleteMsg is sent when all processing is complete
type processCompleteMsg struct {
	summary *models.Summary
	err     error
}

// backToMenuMsg asks the root model to leave the current sub-view and return
// to the main menu. In standalone (CLI sub-command) mode there is no menu, so
// the standalone wrapper translates it into a quit. Sub-views emit this on
// esc/q instead of calling tea.Quit directly, so the parent decides what
// "leave" means.
type backToMenuMsg struct{}

// quitMsg asks the whole application to exit. Both the root model and the
// standalone wrapper translate it into tea.Quit.
type quitMsg struct{}
