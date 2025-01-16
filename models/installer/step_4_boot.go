package installer

import (
	"atomic-actions/models/installer/theme"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"strings"
)

type BootMode struct {
	Result        string
	choices       []string // Список выбора (UEFI, LEGACY)
	cursor        int      // Текущая позиция курсора
	selected      int      // Выбранный элемент (только один)
	confirmActive bool     // Включено ли меню подтверждения
	confirmCursor int      // Позиция курсора в меню подтверждения
	infoMessage   string   // Информация о поддержке UEFI
	uefiSupported bool     // Состояние поддержки компьютером
}

func RunBootModeStep() string {
	// Сначала проверяем поддержку UEFI
	if !checkUEFISupport() {
		fmt.Println(theme.WarningsStyle.Render("Система не поддерживает UEFI. Автоматически выбран LEGACY."))
		return "LEGACY"
	}

	// Если поддержка есть, запускаем TUI
	p := tea.NewProgram(InitialBootMode())
	model, err := p.Run()
	if err != nil {
		fmt.Printf("Ошибка во время выбора типа загрузки: %v\n", err)
		os.Exit(1)
	}

	bootModel := model.(BootMode)
	// Убираем объяснение из возвращаемого результата
	return strings.Split(bootModel.Result, " ")[0]
}

func InitialBootMode() BootMode {
	uefiSupported := checkUEFISupport()
	infoMessage := ""

	if uefiSupported {
		infoMessage = "Ваш компьютер поддерживает UEFI загрузку, это - рекомендуемый выбор."
	} else {
		infoMessage = "Ваш компьютер не поддерживает UEFI, рекомендуем выбрать LEGACY."
	}

	return BootMode{
		choices:       []string{"UEFI (рекомендуется для современных систем)", "LEGACY (совместимый вариант UEFI|Legacy bios)"},
		selected:      -1,
		confirmActive: false,
		confirmCursor: 0,
		infoMessage:   infoMessage,
		uefiSupported: uefiSupported,
	}
}

func checkUEFISupport() bool {
	_, err := os.Stat("/sys/firmware/efi/efivars")
	return err == nil
}

func (m BootMode) Init() tea.Cmd {
	return nil
}

func (m BootMode) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmActive {
			switch msg.String() {
			case "up", "k":
				if m.confirmCursor > 0 {
					m.confirmCursor--
				}
			case "down", "j":
				if m.confirmCursor < 1 {
					m.confirmCursor++
				}
			case "enter", " ":
				if m.confirmCursor == 0 {
					m.Result = m.choices[m.selected]
					return m, tea.Quit
				} else {
					m.selected = -1
					m.confirmActive = false
				}
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		} else {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter", " ":
				m.selected = m.cursor
				m.confirmActive = true
			}
		}
	}
	return m, nil
}

func (m BootMode) View() string {
	header := theme.HeaderStyle.Render("Выберите тип загрузки:")

	var body string
	for i, choice := range m.choices {
		cursor := ""
		if m.cursor == i {
			cursor = theme.CursorStyle.Render(">")
		}

		checked := " " // Не выбрано
		if m.selected == i {
			checked = theme.SelectedStyle.Render("x")
		}

		body += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	if m.confirmActive {
		body += "\nВы уверены, что хотите выбрать " + theme.SelectedStyle.Render(strings.Split(m.choices[m.selected], " ")[0]) + "?\n"
		confirmOptions := []string{"Да", "Отмена"}
		for i, option := range confirmOptions {
			cursor := " "
			if m.confirmCursor == i {
				cursor = theme.CursorStyle.Render(">")
			}
			body += fmt.Sprintf("%s %s\n", cursor, option)
		}
	}

	footer := "\n"
	if m.uefiSupported {
		footer += theme.SuccessStyle.Render(m.infoMessage)
	} else {
		footer += theme.WarningsStyle.Render(m.infoMessage)
	}

	return header + "\n\n" + body + footer
}
