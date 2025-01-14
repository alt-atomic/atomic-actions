package installer

import (
	"atomic-actions/models/installer/theme"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"strings"
)

type Disk struct {
	Result        string   // Результат выбора
	choices       []string // Элементы списка
	cursor        int      // Текущая позиция курсора
	selected      int      // Выбранный элемент (только один)
	confirmActive bool     // Включено ли меню подтверждения
	confirmCursor int      // Позиция курсора в меню подтверждения
}

func RunDiskStep() string {
	p := tea.NewProgram(InitialDisk())

	model, err := p.Run()
	if err != nil {
		fmt.Printf("Ошибка во время выбора образа: %v\n", err)
		os.Exit(1)
	}

	imageModel := model.(Disk)
	return imageModel.Result
}

func InitialDisk() Disk {
	disks := getAvailableDisks()
	return Disk{
		choices:       disks,
		selected:      -1,
		confirmActive: false,
		confirmCursor: 0,
	}
}

func getAvailableDisks() []string {
	out, err := exec.Command("lsblk", "-o", "NAME,SIZE,TYPE", "-d", "-n").Output()
	if err != nil {
		return []string{"Ошибка получения списка дисков"}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var disks []string

	// Исключаем устройства zram и loop
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[2] == "disk" { // Оставляем только устройства типа "disk"
			if strings.HasPrefix(fields[0], "zram") || strings.HasPrefix(fields[0], "loop") {
				continue
			}

			// Формируем отображаемое название
			devicePath := "/dev/" + fields[0]
			displayName := devicePath + " (" + fields[1] + ")"
			disks = append(disks, displayName)
		}
	}
	return disks
}

func (m Disk) Init() tea.Cmd {
	return nil
}

func (m Disk) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
					// Извлекаем только путь устройства, убирая объем
					selectedDisk := m.choices[m.selected]
					m.Result = strings.Split(selectedDisk, " ")[0] // Берем только "/dev/vdX"
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

func (m Disk) View() string {
	header := theme.HeaderStyle.Render("Выберите диск:")

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
		body += "\nВы уверены, что хотите выбрать диск " + theme.SelectedStyle.Render(m.choices[m.selected]) + "?\n"
		confirmOptions := []string{"Да", "Отмена"}
		for i, option := range confirmOptions {
			cursor := " "
			if m.confirmCursor == i {
				cursor = theme.CursorStyle.Render(">")
			}
			body += fmt.Sprintf("%s %s\n", cursor, option)
		}
	}

	footer := "\nВнимание! Все данные на диске будут уничтожены.\n"
	return header + "\n\n" + body + theme.FooterStyle.Render(footer)
}
