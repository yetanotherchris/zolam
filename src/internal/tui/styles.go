package tui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	MenuItemStyle = lipgloss.NewStyle()

	SelectedMenuItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED"))

	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333")).
			Foreground(lipgloss.Color("#FFF")).
			Padding(0, 1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EAB308"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666"))

	DocStyle = lipgloss.NewStyle().
			Margin(1)
)
