package tui

import "github.com/charmbracelet/lipgloss"

// Theme colors - consistent across the app
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorAccent    = lipgloss.Color("#F59E0B") // Amber

	// Status colors
	ColorSuccess = lipgloss.Color("#10B981") // Green
	ColorWarning = lipgloss.Color("#F59E0B") // Amber
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorInfo    = lipgloss.Color("#3B82F6") // Blue

	// Neutral colors
	ColorText      = lipgloss.Color("#E5E7EB") // Light gray
	ColorTextDim   = lipgloss.Color("#6B7280") // Gray
	ColorBorder    = lipgloss.Color("#374151") // Dark gray
	ColorBg        = lipgloss.Color("#111827") // Very dark
	ColorBgSubtle  = lipgloss.Color("#1F2937") // Dark
	ColorHighlight = lipgloss.Color("#374151") // Selection bg
)

// Common styles
var (
	// App frame
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Header/Logo
	LogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			MarginBottom(1)

	// Section header
	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder).
			MarginTop(1).
			MarginBottom(1)

	// Text styles
	TextStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	// Status badges
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ErrorMsgStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// List items
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				Background(ColorHighlight).
				Foreground(ColorText).
				Bold(true).
				PaddingLeft(2)

	// Box/Card style
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// Help bar
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorBorder)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	TableSelectedStyle = lipgloss.NewStyle().
				Background(ColorHighlight).
				Foreground(ColorText).
				Padding(0, 1)

	// Progress bar
	ProgressFullStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	ProgressEmptyStyle = lipgloss.NewStyle().
				Foreground(ColorBorder)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	InputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgSubtle).
			Padding(0, 2).
			MarginRight(1)

	ButtonActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorPrimary).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)
)

// Logo ASCII art
const Logo = `
 █████╗ ██╗   ██╗██████╗ ██╗ ██████╗ ███████╗ ██████╗ ██████╗ ████████╗
██╔══██╗██║   ██║██╔══██╗██║██╔═══██╗██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝
███████║██║   ██║██║  ██║██║██║   ██║███████╗██║   ██║██████╔╝   ██║
██╔══██║██║   ██║██║  ██║██║██║   ██║╚════██║██║   ██║██╔══██╗   ██║
██║  ██║╚██████╔╝██████╔╝██║╚██████╔╝███████║╚██████╔╝██║  ██║   ██║
╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═╝ ╚═════╝ ╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝
`

const LogoSmall = `█▀█ █░█ █▀▄ █ █▀█ █▀ █▀█ █▀█ ▀█▀
█▀█ █▄█ █▄▀ █ █▄█ ▄█ █▄█ █▀▄ ░█░`

// Icons
const (
	IconCheck    = "✓"
	IconCross    = "✗"
	IconArrow    = "→"
	IconBullet   = "•"
	IconPlay     = "▶"
	IconPause    = "⏸"
	IconFolder   = "📁"
	IconBook     = "📚"
	IconMusic    = "🎵"
	IconSearch   = "🔍"
	IconSettings = "⚙"
	IconCache    = "💾"
	IconSpinner  = "◐◓◑◒"
)

// Helper functions
func RenderProgressBar(progress float64, width int) string {
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

	fullBar := ""
	for i := 0; i < filled; i++ {
		fullBar += "█"
	}

	emptyBar := ""
	for i := 0; i < empty; i++ {
		emptyBar += "░"
	}

	return ProgressFullStyle.Render(fullBar) + ProgressEmptyStyle.Render(emptyBar)
}

func RenderHelpKey(key, desc string) string {
	return HelpKeyStyle.Render(key) + " " + HelpDescStyle.Render(desc)
}

func RenderStatusBadge(status string) string {
	switch status {
	case "done", "success":
		return SuccessStyle.Render("[" + IconCheck + "]")
	case "error":
		return ErrorMsgStyle.Render("[" + IconCross + "]")
	case "warning", "skipped":
		return WarningStyle.Render("[!]")
	case "processing":
		return InfoStyle.Render("[...]")
	default:
		return DimStyle.Render("[-]")
	}
}
