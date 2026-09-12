package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorPink    = lipgloss.Color("#F38BA8")
	colorMauve   = lipgloss.Color("#CBA6F7")
	colorGreen   = lipgloss.Color("#A6E3A1")
	colorYellow  = lipgloss.Color("#F9E2AF")
	colorPeach   = lipgloss.Color("#FAB387")
	colorCyan    = lipgloss.Color("#89DCEB")
	colorSubtext = lipgloss.Color("#A6ADC8")
	colorBase    = lipgloss.Color("#1E1E2E")
	colorSurface = lipgloss.Color("#313244")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorMauve).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Background(colorSurface).
			Foreground(colorYellow).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorYellow).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Background(colorBase).
				Foreground(colorSubtext).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSurface).
				Padding(0, 2)

	nowPlayingBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Italic(true)
)

// titledPanel renders a rounded-border box with the title embedded into the
// top-left of the border, mimicking a labeled frame.
func titledPanel(title, content string, width, height int, borderColor lipgloss.Color) string {
	st := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width)
	if height > 0 {
		st = st.Height(height)
	}
	box := st.Render(content)
	if title == "" {
		return box
	}
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	total := lipgloss.Width(lines[0])
	titleStr := " " + title + " "
	tw := lipgloss.Width(titleStr)
	fill := total - 3 - tw
	if fill < 0 {
		fill = 0
	}
	bStyle := lipgloss.NewStyle().Foreground(borderColor)
	tStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)
	lines[0] = bStyle.Render("╭─") + tStyle.Render(titleStr) + bStyle.Render(strings.Repeat("─", fill)+"╮")
	return strings.Join(lines, "\n")
}
