package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Title styles
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	// Subtitle/header styles
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	// Normal text styles
	normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Selected item style
	selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	// Unselected item style
	itemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Status styles
	statusPendingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	statusProcessingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	statusDoneStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82"))

	statusErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	statusSkippedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	// Error message style
	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	// Help text style
	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	// Dimmed text
	dimmedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	// Checkmark for selected items
	checkboxSelected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		SetString("[x]")

	checkboxUnselected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		SetString("[ ]")

	// Progress bar styles
	progressFullChar  = "█"
	progressEmptyChar = "░"
	progressBarStyle  = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	progressBarEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// statusToStyle returns the appropriate style for a given status
func statusToStyle(status string) lipgloss.Style {
	switch status {
	case "pending":
		return statusPendingStyle
	case "processing":
		return statusProcessingStyle
	case "done":
		return statusDoneStyle
	case "error":
		return statusErrorStyle
	case "skipped":
		return statusSkippedStyle
	default:
		return normalStyle
	}
}

// renderProgressBar renders a progress bar with the given percentage (0-1)
func renderProgressBar(progress float64, width int) string {
	if width <= 0 {
		width = 40
	}

	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	empty := width - filled

	bar := progressBarStyle.Render(lipgloss.NewStyle().Width(filled).Render(lipgloss.NewStyle().SetString(progressFullChar).String())) +
		progressBarEmptyStyle.Render(lipgloss.NewStyle().Width(empty).Render(lipgloss.NewStyle().SetString(progressEmptyChar).String()))

	return bar
}
