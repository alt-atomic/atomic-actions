package installer

import (
	"atomic-actions/models/installer/theme"
	"errors"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type UserCreation struct {
	Username       string // Имя пользователя
	Password       string // Пароль
	PasswordRepeat string // Подтверждение пароля

	cursor        int    // Текущая позиция курсора
	focusedField  int    // Фокус текущего поля: 0 - Username, 1 - Password, 2 - PasswordRepeat, 3 - Начать установку, 4 - Отмена
	footerMessage string // Сообщение для footer
	success       bool   // Флаг успешного создания
	errorMessage  string // Сообщение об ошибке
}

func RunUserCreationStep() (*UserCreation, error) {
	p := tea.NewProgram(InitialUserCreation())

	model, err := p.Run()
	if err != nil {
		fmt.Printf("Ошибка во время создания пользователя: %v\n", err)
		os.Exit(1)
	}

	userModel := model.(UserCreation)
	if userModel.success {
		return &userModel, nil
	}

	return nil, errors.New(userModel.errorMessage)
}

func InitialUserCreation() UserCreation {
	return UserCreation{
		Username:       "",
		Password:       "",
		PasswordRepeat: "",
		focusedField:   0,
		footerMessage:  "Введите данные нового пользователя.",
		success:        false,
	}
}

func (m UserCreation) Init() tea.Cmd {
	return nil
}

func (m UserCreation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "down":
			m.focusedField = (m.focusedField + 1) % 5 // Переключение между всеми полями
			if m.focusedField < 3 {
				m.cursor = 0 // Сбрасываем курсор только для инпутов
			}
		case "shift+tab", "up":
			m.focusedField = (m.focusedField + 4) % 5 // Переключение между всеми полями
			if m.focusedField < 3 {
				m.cursor = 0 // Сбрасываем курсор только для инпутов
			}
		case "left":
			if m.focusedField < 3 && m.cursor > 0 {
				m.cursor-- // Перемещение курсора влево в инпутах
			}
		case "right":
			if m.focusedField < 3 {
				fieldLength := len(m.getFieldValue())
				if m.cursor < fieldLength {
					m.cursor++ // Перемещение курсора вправо в инпутах
				}
			}
		case "enter":
			if m.focusedField == 3 {
				// Логика для кнопки "Начать установку"
				if m.Password != m.PasswordRepeat {
					m.errorMessage = "Пароли не совпадают. Попробуйте снова."
				} else if m.Username == "" || m.Password == "" {
					m.errorMessage = "Имя пользователя и пароль не могут быть пустыми."
				} else {
					m.success = true
					m.footerMessage = "Пользователь успешно создан."
					return m, tea.Quit
				}
			} else if m.focusedField == 4 {
				// Логика для кнопки "Отмена"
				return m, tea.Quit
			}
		default:
			if m.focusedField < 3 {
				// Обработка ввода текста только для инпутов
				current := m.getFieldValue()
				newValue, newCursor := handleTextInputWithCursor(current, msg, m.cursor)
				m.setFieldValue(newValue)
				m.cursor = newCursor
			}
		}
	}

	return m, nil
}

// Получить значение текущего поля
func (m *UserCreation) getFieldValue() string {
	switch m.focusedField {
	case 0:
		return m.Username
	case 1:
		return m.Password
	case 2:
		return m.PasswordRepeat
	default:
		return ""
	}
}

// Установить значение текущего поля
func (m *UserCreation) setFieldValue(value string) {
	switch m.focusedField {
	case 0:
		m.Username = value
	case 1:
		m.Password = value
	case 2:
		m.PasswordRepeat = value
	}
}

// Обработка ввода текста с учетом позиции курсора
func handleTextInputWithCursor(current string, msg tea.KeyMsg, cursor int) (string, int) {
	switch msg.String() {
	case "backspace":
		if cursor > 0 {
			current = current[:cursor-1] + current[cursor:]
			cursor--
		}
	case "delete":
		if cursor < len(current) {
			current = current[:cursor] + current[cursor+1:]
		}
	default:
		if len(msg.String()) == 1 {
			current = current[:cursor] + msg.String() + current[cursor:]
			cursor++
		}
	}
	return current, cursor
}

func (m UserCreation) View() string {
	header := theme.HeaderStyle.Render("Создание нового пользователя")

	// Минимальная ширина поля
	const fieldWidth = 20

	// Функция для добавления ширины к тексту
	padRight := func(text string, width int) string {
		if len(text) < width {
			return text + strings.Repeat(" ", width-len(text))
		}
		return text
	}

	// Поле для имени пользователя
	usernameField := "Имя пользователя:\n"
	if m.focusedField == 0 {
		usernameField += theme.InputStyle.Render(padRight(m.Username, fieldWidth)[:m.cursor]+"|"+padRight(m.Username, fieldWidth)[m.cursor:]) + "\n"
	} else {
		usernameField += theme.InputStyle.Render(padRight(m.Username, fieldWidth)) + "\n"
	}

	// Поле для пароля
	passwordField := "Пароль:\n"
	if m.focusedField == 1 {
		maskedPassword := strings.Repeat("*", len(m.Password))
		passwordField += theme.InputStyle.Render(padRight(maskedPassword, fieldWidth)[:m.cursor]+"|"+padRight(maskedPassword, fieldWidth)[m.cursor:]) + "\n"
	} else {
		passwordField += theme.InputStyle.Render(strings.Repeat("*", len(m.Password))+strings.Repeat(" ", fieldWidth-len(m.Password))) + "\n"
	}

	// Поле для подтверждения пароля
	passwordRepeatField := "Повторите пароль:\n"
	if m.focusedField == 2 {
		maskedRepeatPassword := strings.Repeat("*", len(m.PasswordRepeat))
		passwordRepeatField += theme.InputStyle.Render(padRight(maskedRepeatPassword, fieldWidth)[:m.cursor]+"|"+padRight(maskedRepeatPassword, fieldWidth)[m.cursor:]) + "\n"
	} else {
		passwordRepeatField += theme.InputStyle.Render(strings.Repeat("*", len(m.PasswordRepeat))+strings.Repeat(" ", fieldWidth-len(m.PasswordRepeat))) + "\n"
	}

	// Создание стилей для кнопок
	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")). // Белый текст
		Background(lipgloss.Color("33")). // Синий фон
		Padding(0, 2).
		Margin(1, 0).
		Align(lipgloss.Center)

	buttonCancelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).  // Белый текст
		Background(lipgloss.Color("238")). // Серый фон
		Padding(0, 2).
		Margin(1, 0).
		Align(lipgloss.Center)

	selectedButtonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).   // Черный текст
		Background(lipgloss.Color("220")). // Желтый фон
		Padding(0, 2).
		Margin(1, 0).
		Align(lipgloss.Center)

	// Устанавливаем стиль для каждой кнопки ровно один раз
	startButton := buttonStyle.Render("Начать установку")
	cancelButton := buttonCancelStyle.Render("Отмена")

	// Изменяем стиль только для выбранной кнопки
	if m.focusedField == 3 {
		startButton = selectedButtonStyle.Render("Начать установку")
	}
	if m.focusedField == 4 {
		cancelButton = selectedButtonStyle.Render("Отмена")
	}

	// Сообщения в футере
	footer := "\n" + m.footerMessage
	if m.errorMessage != "" {
		footer += "\n" + theme.ErrorStyle.Render(m.errorMessage)
	}

	// Собираем итоговый вид
	// Сборка итогового представления
	return strings.Join([]string{
		header,
		"",
		usernameField,
		passwordField,
		passwordRepeatField,
		"",
		startButton,
		cancelButton,
		footer,
	}, "\n")
}
