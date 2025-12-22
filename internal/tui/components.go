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

// Table renders a simple table
func Table(headers []string, rows [][]string, selectedRow int, width int) string {
	if len(headers) == 0 {
		return ""
	}

	colWidth := (width - 4) / len(headers)
	if colWidth < 10 {
		colWidth = 10
	}

	// Render header
	var headerCells []string
	for _, h := range headers {
		cell := TableHeaderStyle.Width(colWidth).Render(truncate(h, colWidth-2))
		headerCells = append(headerCells, cell)
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Left, headerCells...)

	// Render rows
	var rowStrings []string
	rowStrings = append(rowStrings, headerRow)

	for i, row := range rows {
		var cells []string
		for _, cell := range row {
			style := TableCellStyle
			if i == selectedRow {
				style = TableSelectedStyle
			}
			cells = append(cells, style.Width(colWidth).Render(truncate(cell, colWidth-2)))
		}
		rowStrings = append(rowStrings, lipgloss.JoinHorizontal(lipgloss.Left, cells...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rowStrings...)
}

// List renders a selectable list
func List(items []string, selectedIdx int, checkable bool, checked map[int]bool) string {
	var lines []string

	for i, item := range items {
		prefix := "  "
		style := ListItemStyle

		if checkable {
			if checked[i] {
				prefix = SuccessStyle.Render("[x] ")
			} else {
				prefix = DimStyle.Render("[ ] ")
			}
		}

		if i == selectedIdx {
			style = SelectedItemStyle
			prefix = InfoStyle.Render("> ") + prefix[2:]
		}

		lines = append(lines, style.Render(prefix+item))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// InfoBox renders an info/status box
func InfoBox(title string, items map[string]string, width int) string {
	var lines []string

	for key, value := range items {
		line := BoldStyle.Render(key+":") + " " + TextStyle.Render(value)
		lines = append(lines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	box := BoxStyle.Width(width - 4)

	if title != "" {
		header := TitleStyle.Render(title)
		return box.Render(lipgloss.JoinVertical(lipgloss.Left, header, "", content))
	}

	return box.Render(content)
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
		style.Render(icon+" "+message),
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

// Divider renders a horizontal divider
func Divider(width int) string {
	line := strings.Repeat("─", width-4)
	return DimStyle.Render(line)
}

// truncate truncates a string to the given length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Confirm renders a confirmation prompt
func Confirm(message string, focused bool) string {
	yesStyle := ButtonStyle
	noStyle := ButtonStyle

	if focused {
		yesStyle = ButtonActiveStyle
	} else {
		noStyle = ButtonActiveStyle
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Left,
		yesStyle.Render("Yes"),
		noStyle.Render("No"),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		TextStyle.Render(message),
		"",
		buttons,
	)
}
