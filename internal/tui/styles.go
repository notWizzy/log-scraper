package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/notWizzy/log-scraper/internal/model"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ADADAD"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3D3D5C")).
			Foreground(lipgloss.Color("#FAFAFA"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	boldStyle = lipgloss.NewStyle().Bold(true)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

var categoryColors = map[model.Category]lipgloss.Color{
	model.CategoryOOMKill: lipgloss.Color("#FF4444"),
	model.CategoryTimeout: lipgloss.Color("#FFAA00"),
	model.CategoryDBLock:  lipgloss.Color("#FF6600"),
	model.CategoryPanic:   lipgloss.Color("#FF0066"),
	model.CategorySignal:  lipgloss.Color("#CC00FF"),
	model.CategoryDiskIO:  lipgloss.Color("#FF3333"),
	model.CategoryNetwork: lipgloss.Color("#FFCC00"),
}

func categoryBadge(c model.Category) string {
	color, ok := categoryColors[c]
	if !ok {
		color = lipgloss.Color("#AAAAAA")
	}
	return badgeStyle.
		Foreground(lipgloss.Color("#000000")).
		Background(color).
		Render(c.String())
}

func severityStyle(s model.Severity) lipgloss.Style {
	switch s {
	case model.SeverityFatal:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))
	case model.SeverityCritical:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4444"))
	case model.SeverityError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6666"))
	case model.SeverityWarn:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	}
}
