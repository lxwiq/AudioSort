package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Header renders the app header with logo
func Header(width int, title string) string {
	logo := LogoStyle.Render(LogoSmall)
	subtitle := SubtitleStyle.Render(title)

	content := lipgloss.JoinVertical(lipgloss.Left, logo, subtitle)

	return BoxStyle.Width(width - 4).Render(content)
}

// HeaderCompact renders a compact header
func HeaderCompact(title string) string {
	return TitleStyle.Render("AudioSort") + " " + DimStyle.Render("|") + " " + SubtitleStyle.Render(title)
}

// Footer renders the help footer
func Footer(keys ...string) string {
	var items []string
	for i := 0; i < len(keys)-1; i += 2 {
		items = append(items, RenderHelpKey(keys[i], keys[i+1]))
	}
	return HelpBarStyle.Render(strings.Join(items, DimStyle.Render("  •  ")))
}

// Section renders a section with title
func Section(title string, content string) string {
	header := SectionStyle.Render(title)
	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

// Alert renders an alert message
func Alert(alertType, message string) string {
	var style lipgloss.Style
	var icon string

	switch alertType {
	case "success":
		style = SuccessStyle
		icon = IconCheck
	case "error":
		style = ErrorMsgStyle
		icon = IconCross
	case "warning":
		style = WarningStyle
		icon = "!"
	default:
		style = InfoStyle
		icon = "i"
	}

	return BoxStyle.BorderForeground(style.GetForeground()).Render(
		style.Render(icon + " " + message),
	)
}

// Spinner creates a new spinner model
func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return s
}

// SpinnerWithText renders a spinner with text
func SpinnerWithText(s spinner.Model, text string) string {
	return s.View() + " " + TextStyle.Render(text)
}

// Progress renders a progress section
func Progress(current, total int, label string, width int) string {
	if total == 0 {
		total = 1
	}
	progress := float64(current) / float64(total)
	percent := fmt.Sprintf("%d%%", int(progress*100))

	bar := RenderProgressBar(progress, width-20)
	counter := DimStyle.Render(fmt.Sprintf("%d/%d", current, total))

	return lipgloss.JoinVertical(lipgloss.Left,
		TextStyle.Render(label),
		bar+" "+percent,
		counter,
	)
}

// truncate truncates a string to the given number of runes, appending an
// ellipsis when it is cut. It operates on runes (not bytes) so multi-byte
// UTF-8 titles are never corrupted mid-character.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(r[:maxLen])
	}
	return string(r[:maxLen-3]) + "..."
}
