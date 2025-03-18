package console

import (
	"atomic-actions/models/installer/theme"
	"bytes"
	"encoding/json"
	"errors"
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

type Choice struct {
	Name        string
	Description string
}

type Image struct {
	Result        string
	choices       []Choice
	cursor        int
	selected      int
	confirmActive bool
	confirmCursor int
	inputActive   bool
	menuCursor    int
	inputText     string
	textCursor    int
	footerMessage string
	loading       bool
	inputFocused  bool
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
		images = []Choice{}
	}

	images = append(images, Choice{Name: "Выбрать свой образ"})
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

func getAvailableImages() ([]Choice, error) {
	out, err := exec.Command("sudo", "podman", "images", "--format", "json").Output()
	if err != nil {
		return addDefaultImage(nil), nil
	}

	var imagesData []ImagePodman
	if err := json.Unmarshal(out, &imagesData); err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
		return addDefaultImage(nil), nil
	}

	var images []Choice
	for _, image := range imagesData {
		for _, name := range image.Names {
			images = append(images, Choice{Name: name, Description: ""})
		}
	}

	return addDefaultImage(images), nil
}

func addDefaultImage(images []Choice) []Choice {
	images = append(images,
		Choice{
			Name:        "ghcr.io/alt-gnome/alt-atomic:latest",
			Description: "Образ GNOME. Рекомендуемый",
		},
		Choice{
			Name:        "ghcr.io/alt-gnome/alt-atomic:latest-nv",
			Description: "Образ GNOME для NVIDIA (OPEN драйвер)",
		},
		Choice{
			Name:        "ghcr.io/alt-atomic/alt-kde:latest",
			Description: "Образ KDE (только для тестирования)",
		},
		Choice{
			Name:        "ghcr.io/alt-atomic/alt-kde:latest-nv",
			Description: "Образ KDE для NVIDIA (OPEN драйвер, только для тестирования)",
		},
	)
	return images
}

func validateImage(image string) (string, error) {
	cmd := exec.Command("skopeo", "inspect", "docker://"+image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stderr.String(), err
		}
		return "Ошибка выполнения команды: проверьте что skopeo установлен", err
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
			m.choices = append(m.choices[:len(m.choices)-1], Choice{Name: image}, Choice{Name: "Выбрать свой образ"})
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
			m.menuCursor = 1
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
				m.loading = true
				return m, tea.Batch(func() tea.Msg {
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
			m.Result = m.choices[m.selected].Name
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
		if m.choices[m.cursor].Name == "Выбрать свой образ" {
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
	header := theme.HeaderStyle.Render("Добро пожаловать в установку Alt Atomic \nВыберите образ:")

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

		desc := ""
		if choice.Description != "" {
			desc = theme.LoadingStyle.Render(" - " + choice.Description)
		}
		body += fmt.Sprintf("%s [%s] %s%s\n", cursor, checked, choice.Name, desc)
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
		selectedName := m.choices[m.selected].Name
		body += "\nВы уверены, что хотите выбрать изображение " + theme.SelectedStyle.Render(selectedName) + "?\n"
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
