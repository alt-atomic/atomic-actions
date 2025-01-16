package installer

import (
	"atomic-actions/models/installer/theme"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"strconv"
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
		fmt.Printf("Ошибка во время выбора диска: %v\n", err)
		os.Exit(1)
	}

	imageModel := model.(Disk)
	return imageModel.Result
}

func InitialDisk() Disk {
	disks := getAvailableDisks()
	if len(disks) == 0 {
		fmt.Println(theme.ErrorStyle.Render("Для установки требуется дисковое устройство размером ≥ 50 ГБ!"))
		os.Exit(1)
	}

	return Disk{
		choices:       disks,
		selected:      -1,
		confirmActive: false,
		confirmCursor: 0,
	}
}

// ----------------------------------------------------------------------------
// Изменяемая функция: учитываем только диски размером >= 50 ГБ
func getAvailableDisks() []string {
	out, err := exec.Command("lsblk", "-o", "NAME,SIZE,TYPE", "-d", "-n").Output()
	if err != nil {
		fmt.Println("Ошибка получения списка дисков:", err)
		os.Exit(1)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var disks []string

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[2] == "disk" {
			// Исключаем zram и loop
			if strings.HasPrefix(fields[0], "zram") || strings.HasPrefix(fields[0], "loop") {
				continue
			}

			// Парсим размер, если не удалось — пропускаем
			sizeGb, err := parseSize(fields[1])
			if err != nil {
				continue
			}

			// Оставляем диск, только если он ≥ 50 ГБ
			if sizeGb >= 50 {
				devicePath := "/dev/" + fields[0]
				displayName := fmt.Sprintf("%s (%s)", devicePath, fields[1])
				disks = append(disks, displayName)
			}
		}
	}

	// Если подходящих дисков нет — вернём пустой список
	return disks
}

// parseSize парсит строку вида "100G", "512M", "2T" и возвращает размер в ГБ.
// Если строка не распознана, возвращаем ошибку.
func parseSize(sizeStr string) (float64, error) {
	if len(sizeStr) < 2 {
		return 0, fmt.Errorf("неизвестный формат размера: %s", sizeStr)
	}
	unit := sizeStr[len(sizeStr)-1]    // последняя буква: G / M / T / и т.д.
	valStr := sizeStr[:len(sizeStr)-1] // всё, кроме последней буквы

	value, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, fmt.Errorf("ошибка парсинга числа из %s: %w", valStr, err)
	}

	switch unit {
	case 'G':
		// Обычные гигабайты
		return value, nil
	case 'M':
		// Мегабайты -> делим на 1024, получаем ГБ
		return value / 1024.0, nil
	case 'T':
		// Терабайты -> умножаем на 1024, получаем ГБ
		return value * 1024.0, nil
	default:
		// Если попался какой-то другой (например, KiB, GiB — но lsblk обычно пишет G/M/T)
		return 0, fmt.Errorf("неизвестная единица измерения: %c", unit)
	}
}

// ----------------------------------------------------------------------------

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
					// Извлекаем только путь устройства ("/dev/sda"), без объёма в скобках
					selectedDisk := m.choices[m.selected]
					m.Result = strings.Split(selectedDisk, " ")[0]
					return m, tea.Quit
				} else {
					// Отмена подтверждения
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

		checked := " "
		if m.selected == i {
			checked = theme.SelectedStyle.Render("x")
		}

		body += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	if m.confirmActive {
		body += "\nВы уверены, что хотите выбрать диск " +
			theme.SelectedStyle.Render(m.choices[m.selected]) + "?\n"
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
