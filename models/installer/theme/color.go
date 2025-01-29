package theme

import "github.com/charmbracelet/lipgloss"

// Определение цветов
var (
	cursorColor   = lipgloss.Color("205") // Розовый (Pink)
	selectedColor = lipgloss.Color("34")  // Зеленый (Green)
	headerColor   = lipgloss.Color("220") // Желтый (Yellow)
	footerColor   = lipgloss.Color("167") // Красновато-розовый (Deep Pink)
	successColor  = lipgloss.Color("34")  // Зеленый (Green)
	errorColor    = lipgloss.Color("160") // Красный (Red)
	loadingColor  = lipgloss.Color("33")  // Голубой (Cyan)
	warningColor  = lipgloss.Color("214") // Оранжевый (Orange)
	infoColor     = lipgloss.Color("240") // Серый (Gray)

	WarningsStyle    = lipgloss.NewStyle().Bold(true).Foreground(warningColor)
	CursorStyle      = lipgloss.NewStyle().Foreground(cursorColor)
	SelectedStyle    = lipgloss.NewStyle().Foreground(selectedColor)
	HeaderStyle      = lipgloss.NewStyle().Bold(true).Foreground(headerColor)
	FooterStyle      = lipgloss.NewStyle().Bold(true).Foreground(footerColor)
	SuccessStyle     = lipgloss.NewStyle().Bold(true).Foreground(successColor)
	ErrorStyle       = lipgloss.NewStyle().Bold(true).Foreground(errorColor)
	SuccessInfoStyle = lipgloss.NewStyle().Bold(true).Foreground(successColor)

	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Foreground(lipgloss.Color("69")).
			Background(lipgloss.Color("236"))
	LoadingStyle = lipgloss.NewStyle().Bold(true).Italic(true).Foreground(loadingColor)
)
