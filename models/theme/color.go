package installer

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	cursorColor   = lipgloss.Color("205")
	selectedColor = lipgloss.Color("34")
	headerColor   = lipgloss.Color("220")
	footerColor   = lipgloss.Color("167")
	successColor  = lipgloss.Color("34")
	errorColor    = lipgloss.Color("160")
	loadingColor  = lipgloss.Color("33")
	warningColor  = lipgloss.Color("214")

	WarningsStyle = lipgloss.NewStyle().Bold(true).Foreground(warningColor)
	CursorStyle   = lipgloss.NewStyle().Foreground(cursorColor)
	SelectedStyle = lipgloss.NewStyle().Foreground(selectedColor)
	HeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(headerColor)
	FooterStyle   = lipgloss.NewStyle().Bold(true).Foreground(footerColor)
	SuccessStyle  = lipgloss.NewStyle().Bold(true).Foreground(successColor)
	ErrorStyle    = lipgloss.NewStyle().Bold(true).Foreground(errorColor)
	InfoStyle     = lipgloss.NewStyle().Foreground(successColor)

	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Foreground(lipgloss.Color("69")).
			Background(lipgloss.Color("236"))
	LoadingStyle = lipgloss.NewStyle().Bold(true).Italic(true).Foreground(loadingColor)
)
