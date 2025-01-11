package installer

import (
	theme "atomic-actions/models/theme"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type ImagePodman struct {
	Names []string `json:"Names"`
}

type Image struct {
	Result        string   // Результат выбора
	choices       []string // Список изображений
	cursor        int      // Текущая позиция курсора
	selected      int      // Выбранный элемент
	confirmActive bool     // Меню подтверждения
	confirmCursor int      // Курсор в меню подтверждения
	inputActive   bool     // Поле ввода активно
	menuCursor    int      // Курсор в меню "ОК" и "Отмена"
	inputText     string   // Текущий текст ввода
	textCursor    int      // Курсор в тексте инпута
	footerMessage string   // Сообщение для footer
	loading       bool     // Прелоадер
	inputFocused  bool     // Фокус находится на инпуте
}

func RunImageStep() string {
	p := tea.NewProgram(InitialImage())

	model, err := p.Run()
	if err != nil {
		fmt.Printf("Ошибка во время выбора образа: %v\n", err)
		os.Exit(1)
	}

	imageModel := model.(Image)
	return imageModel.Result
}

func InitialImage() Image {
	images, err := getAvailableImages()
	footerMessage := ""
	if err != nil {
		log.Printf(err.Error())
		footerMessage = theme.ErrorStyle.Render(err.Error())
		images = []string{}
	}

	log.Printf(strings.Join(images, "\n"))
	images = append(images, "Выбрать свой образ")
	return Image{
		choices:       images,
		selected:      -1,
		confirmActive: false,
		inputActive:   false,
		menuCursor:    0,
		textCursor:    0,
		inputText:     "",
		footerMessage: footerMessage,
	}
}

// Получение доступных изображений через podman
func getAvailableImages() ([]string, error) {
	// Выполнить команду podman images --format json
	out, err := exec.Command("sudo", "podman", "images", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка изображений: %v", err)
	}

	// Парсинг JSON-ответа
	var imagesData []ImagePodman
	if err := json.Unmarshal(out, &imagesData); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Фильтровать образы с непустыми Names
	var images []string
	for _, image := range imagesData {
		if len(image.Names) > 0 {
			images = append(images, image.Names...)
		}
	}

	return images, nil
}

// Проверка изображения с помощью skopeo
func validateImage(image string) (string, error) {
	cmd := exec.Command("skopeo", "inspect", "docker://"+image)
	output, err := cmd.Output()
	if err != nil {
		return string(err.(*exec.ExitError).Stderr), err
	}
	return string(output), nil
}

func (m Image) Init() tea.Cmd {
	return nil
}

func (m Image) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.footerMessage = ""

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if strings.HasPrefix(msg.String(), "error:") {
			m.loading = false
			m.footerMessage = theme.ErrorStyle.Render(strings.TrimPrefix(msg.String(), "error:"))
		} else if strings.HasPrefix(msg.String(), "success:") {
			m.loading = false
			image := strings.TrimPrefix(msg.String(), "success:")
			m.footerMessage = theme.SuccessStyle.Render("Валидное изображение: " + image)
			m.choices = append(m.choices[:len(m.choices)-1], image, "Загрузить свое изображение")
			m.inputActive = false
			m.inputText = ""
			m.textCursor = 0
		} else if m.inputActive {
			return m.updateInputOrMenu(msg)
		} else if m.confirmActive {
			return m.updateConfirmation(msg)
		} else {
			return m.updateChoices(msg)
		}
	}
	return m, nil
}

func (m Image) updateInputOrMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.menuCursor == 0 {
		m.inputFocused = true

		switch msg.String() {
		case "left":
			if m.textCursor > 0 {
				m.textCursor--
			}
		case "right":
			if m.textCursor < len(m.inputText) {
				m.textCursor++
			}
		case "backspace":
			if m.textCursor > 0 {
				m.inputText = m.inputText[:m.textCursor-1] + m.inputText[m.textCursor:]
				m.textCursor--
			}
		case "enter", "down":
			m.menuCursor = 1 // Переключаемся на меню "ОК/Отмена"
			m.inputFocused = false
		case "esc":
			m.inputActive = false
			m.inputText = ""
			m.textCursor = 0
			m.inputFocused = false
		default:
			if len(msg.String()) == 1 {
				m.inputText = m.inputText[:m.textCursor] + msg.String() + m.inputText[m.textCursor:]
				m.textCursor++
			}
		}
	} else {
		m.inputFocused = false

		switch msg.String() {
		case "up":
			if m.menuCursor > 1 {
				m.menuCursor--
			} else {
				m.menuCursor = 0
				m.inputFocused = true
			}
		case "down":
			if m.menuCursor < 2 {
				m.menuCursor++
			}
		case "enter", " ":
			if m.menuCursor == 1 && len(m.inputText) > 0 {
				// Переходим в режим загрузки
				m.loading = true
				return m, tea.Batch(func() tea.Msg {
					// Проверка изображения выполняется асинхронно
					output, err := validateImage(m.inputText)
					if err != nil {
						return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("error:" + output)}
					}
					return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("success:" + m.inputText)}
				})
			} else if m.menuCursor == 2 {
				m.inputActive = false
				m.inputText = ""
				m.textCursor = 0
			}
		}
	}
	return m, nil
}

func (m Image) updateConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.confirmCursor > 0 {
			m.confirmCursor--
		}
	case "down":
		if m.confirmCursor < 1 {
			m.confirmCursor++
		}
	case "enter", " ":
		if m.confirmCursor == 0 {
			m.Result = m.choices[m.selected]
			return m, tea.Quit
		} else if m.confirmCursor == 1 {
			m.selected = -1
			m.confirmActive = false
		}
	}
	return m, nil
}

func (m Image) updateChoices(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "enter", " ":
		if m.cursor == len(m.choices)-1 {
			m.inputActive = true
			m.menuCursor = 0
		} else {
			m.selected = m.cursor
			m.confirmActive = true
			m.confirmCursor = 0
		}
	}
	return m, nil
}

func (m Image) View() string {
	header := theme.HeaderStyle.Render("Добро пожаловать в установку Alt Atomic ✨\nВыберите образ:")

	var body string
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = theme.CursorStyle.Render(">")
		}
		checked := " "
		if m.selected == i {
			checked = theme.SelectedStyle.Render("x")
		}
		body += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	if m.inputActive {
		if m.loading {
			body += "\n" + theme.LoadingStyle.Render("Проверка... Пожалуйста, подождите.") + "\n"
		} else {
			body += "\nУкажите путь к изображению:\n"

			renderedInput := m.inputText
			if m.inputFocused {
				renderedInput = m.inputText[:m.textCursor] + theme.CursorStyle.Render("|") + m.inputText[m.textCursor:]
			}
			body += theme.InputStyle.Render(renderedInput) + "\n\n"

			confirmOptions := []string{"ОК", "Отмена"}
			for i, option := range confirmOptions {
				cursor := " "
				if m.menuCursor == i+1 {
					cursor = theme.CursorStyle.Render(">")
				}
				body += fmt.Sprintf("%s %s\n", cursor, option)
			}
		}
	}

	if m.Result == "" && m.selected != -1 {
		body += "\nВы уверены, что хотите выбрать изображение " + theme.SelectedStyle.Render(m.choices[m.selected]) + "?\n"
		confirmOptions := []string{"Да", "Отмена"}
		for i, option := range confirmOptions {
			cursor := " "
			if m.confirmCursor == i {
				cursor = theme.CursorStyle.Render(">")
			}
			body += fmt.Sprintf("%s %s\n", cursor, option)
		}
	}

	footer := m.footerMessage
	return header + "\n\n" + body + theme.FooterStyle.Render(footer)
}
