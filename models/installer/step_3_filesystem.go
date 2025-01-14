package installer

import (
	theme "atomic-actions/models/theme"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"strings"
)

type Filesystem struct {
	Result        string   // Результат выбора
	choices       []string // Элементы списка
	cursor        int      // Текущая позиция курсора
	selected      int      // Выбранный элемент (только один)
	confirmActive bool     // Включено ли меню подтверждения
	confirmCursor int      // Позиция курсора в меню подтверждения
}

func RunFilesystemStep() string {
	p := tea.NewProgram(InitialFilesystem())

	model, err := p.Run()
	if err != nil {
		fmt.Printf("Ошибка во время выбора файловой системы: %v\n", err)
		os.Exit(1)
	}

	fsModel := model.(Filesystem)
	return fsModel.Result
}

func InitialFilesystem() Filesystem {
	return Filesystem{
		choices:       []string{"btrfs (Будут добавлены subvolume:@, @home, @var)", "ext4 (Установка в корень /)"},
		selected:      -1,
		confirmActive: false,
		confirmCursor: 0,
	}
}

func (m Filesystem) Init() tea.Cmd {
	return nil
}

func (m Filesystem) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
					// Извлекаем только название файловой системы без пояснений
					m.Result = m.choices[m.selected]
					if idx := strings.Index(m.Result, " "); idx != -1 {
						m.Result = m.Result[:idx]
					}
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

func (m Filesystem) View() string {
	header := theme.HeaderStyle.Render("Выберите файловую систему:")

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
		body += "\nВы уверены, что хотите выбрать файловую систему " + theme.SelectedStyle.Render(strings.Split(m.choices[m.selected], " ")[0]) + "?\n"
		confirmOptions := []string{"Да", "Отмена"}
		for i, option := range confirmOptions {
			cursor := " "
			if m.confirmCursor == i {
				cursor = theme.CursorStyle.Render(">")
			}
			body += fmt.Sprintf("%s %s\n", cursor, option)
		}
	}

	footer := "\nBtrfs - это современная файловая система, она хорошо подходит для концепции ostree.\n"
	return header + "\n\n" + body + theme.InfoStyle.Render(footer)
}
