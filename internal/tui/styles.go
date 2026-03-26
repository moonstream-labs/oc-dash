package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	colorGreen  = lipgloss.Color("#a6e3a1")
	colorYellow = lipgloss.Color("#f9e2af")
	colorBlue   = lipgloss.Color("#89b4fa")
	colorRed    = lipgloss.Color("#f38ba8")
	colorGray   = lipgloss.Color("#6c7086")
	colorText   = lipgloss.Color("#cdd6f4")
	colorDim    = lipgloss.Color("#585b70")
	colorBg     = lipgloss.Color("#1e1e2e")
	colorBorder = lipgloss.Color("#313244")
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Padding(0, 1)

	listItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(lipgloss.Color("#313244")).
				Bold(true).
				Padding(0, 1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorDim).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true).
				Padding(0, 1).
				MarginBottom(1)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	activityUserStyle = lipgloss.NewStyle().
				Foreground(colorBlue)

	activityAssistantStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	activityToolStyle = lipgloss.NewStyle().
				Foreground(colorYellow)
)

// StatusIndicator returns the colored status symbol for a session status.
func StatusIndicator(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(colorGreen).Render("●")
	case "waiting":
		return lipgloss.NewStyle().Foreground(colorYellow).Render("◉")
	case "idle":
		return lipgloss.NewStyle().Foreground(colorBlue).Render("○")
	case "error":
		return lipgloss.NewStyle().Foreground(colorRed).Render("✕")
	default:
		return lipgloss.NewStyle().Foreground(colorGray).Render("◌")
	}
}

// StatusTag returns a short colored tag for a status.
func StatusTag(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(colorGreen).Render("[R]")
	case "waiting":
		return lipgloss.NewStyle().Foreground(colorYellow).Render("[W]")
	case "idle":
		return lipgloss.NewStyle().Foreground(colorBlue).Render("[I]")
	case "error":
		return lipgloss.NewStyle().Foreground(colorRed).Render("[E]")
	default:
		return lipgloss.NewStyle().Foreground(colorGray).Render("[C]")
	}
}
