package steps

import (
	"atomic-actions/models/installer/gui/image"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// ImagePodman – структура для парсинга "podman images --format json"
type ImagePodman struct {
	Names []string `json:"Names"`
}

// Choice – элемент списка доступных образов
type Choice struct {
	Name        string
	ShortText   string
	Description string
}

// getAvailableImages – заглушка вместо реального podman.
func getAvailableImages() ([]Choice, error) {
	var images []Choice
	return addDefaultImage(images), nil
}

// addDefaultImage – добавляет «стандартные» образы
func addDefaultImage(images []Choice) []Choice {
	if images == nil {
		images = []Choice{}
	}
	images = append(
		images,
		Choice{
			Name:        "ghcr.io/alt-gnome/alt-atomic:latest",
			ShortText:   "GNOME",
			Description: "Образ GNOME. Рекомендуемый",
		},
		Choice{
			Name:        "ghcr.io/alt-gnome/alt-atomic:latest-nv",
			ShortText:   "GNOME NVIDIA",
			Description: "Образ GNOME для NVIDIA. OPEN драйвер",
		},
		Choice{
			Name:        "ghcr.io/alt-atomic/alt-kde:latest",
			ShortText:   "KDE",
			Description: "Образ KDE. Находится в стадии тестирования, не рекомендуется",
		},
		Choice{
			Name:        "ghcr.io/alt-atomic/alt-kde:latest-nv",
			ShortText:   "KDE NVIDIA",
			Description: "Образ KDE для NVIDIA. Находится в стадии тестирования, не рекомендуется",
		},
	)
	return images
}

// validateImage – проверяем образ через `skopeo inspect`.
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
		return "Ошибка выполнения команды (проверьте, что skopeo установлен)", err
	}
	return string(output), nil
}

// CreateImageStep – виджет для шага выбора образа.
func CreateImageStep(onImageSelected func(string), onCancel func()) gtk.Widgetter {
	// ВЕРТИКАЛЬНЫЙ box – «корневой»
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)

	iconWidget := image.NewIconFromEmbed(image.IconImage)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	outerBox.Append(wrapper)

	// Создадим центральный контейнер (centerBox), который займёт всё свободное пространство
	centerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	centerBox.SetVExpand(true) // чтобы растягиваться по вертикали
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	outerBox.Append(centerBox)

	// Получаем список «стандартных» образов
	images, err := getAvailableImages()
	if err != nil {
		log.Println("Ошибка при получении списка образов:", err)
	}
	// Добавляем пункт «кастомный» (последним)
	images = append(images, Choice{
		Name:        "добавить свой образ",
		Description: "",
	})

	customChoiceIndex := len(images) - 1

	combo := gtk.NewComboBoxText()
	var comboCount = len(images)

	for _, img := range images {
		label := img.Name
		if img.ShortText != "" {
			label += "  " + img.ShortText
		}
		combo.AppendText(label)
	}
	combo.SetActive(0)
	centerBox.Append(combo)

	// Лейбл для описания
	descLabel := gtk.NewLabel("")
	descLabel.SetHAlign(gtk.AlignStart)
	descLabel.SetMarginTop(10)
	descLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(descLabel)

	// Поле для ввода кастомного образа (по умолчанию скрыто)
	customEntry := gtk.NewEntry()
	customEntry.SetPlaceholderText("Введите образ (например, repo/image:tag)")
	customEntry.SetVisible(false)
	centerBox.Append(customEntry)

	// Кнопка "Проверить и добавить" + Спиннер
	checkButton := gtk.NewButtonWithLabel("Проверить и добавить")
	checkButton.SetVisible(false)

	spinner := gtk.NewSpinner()
	spinner.SetHAlign(gtk.AlignCenter)
	spinner.SetVAlign(gtk.AlignCenter)

	stack := gtk.NewStack()
	stack.AddNamed(checkButton, "button")
	stack.AddNamed(spinner, "spinner")
	stack.SetVisibleChildName("button")
	centerBox.Append(stack)

	// Лейбл для результата проверки
	checkResultLabel := gtk.NewLabel("")
	checkResultLabel.SetHAlign(gtk.AlignStart)
	checkResultLabel.SetMarginTop(4)
	checkResultLabel.SetVisible(false)
	centerBox.Append(checkResultLabel)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	cancelBtn := gtk.NewButtonWithLabel("Назад")
	chooseBtn := gtk.NewButtonWithLabel("Выбрать")

	cancelBtn.SetSizeRequest(120, 40)
	chooseBtn.SetSizeRequest(120, 40)
	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(cancelBtn)
	buttonBox.Append(chooseBtn)

	outerBox.Append(buttonBox)

	var customImageValid string

	// При смене пункта в combo
	combo.ConnectChanged(func() {
		checkResultLabel.SetVisible(false)
		checkResultLabel.SetLabel("")
		customEntry.SetText("")
		customImageValid = ""

		activeIndex := combo.Active()
		if activeIndex < 0 {
			return
		}

		if activeIndex == customChoiceIndex {
			// Кастомный пункт
			customEntry.SetVisible(true)
			checkButton.SetVisible(true)
			stack.SetVisibleChildName("button")
			descLabel.SetLabel("Ввести свой образ вручную и проверить.")
		} else {
			// Стандартный пункт
			customEntry.SetVisible(false)
			checkButton.SetVisible(false)
			stack.SetVisibleChildName("button")

			desc := images[activeIndex].Description
			if desc == "" {
				desc = "Без описания."
			}
			descLabel.SetLabel(desc)
		}
	})

	// Нажали кнопку "Проверить и добавить"
	checkButton.ConnectClicked(func() {
		imageName := strings.TrimSpace(customEntry.Text())
		if imageName == "" {
			checkResultLabel.SetLabel("Введите корректное имя образа.")
			checkResultLabel.SetVisible(true)
			return
		}

		// Переходим на spinner
		stack.SetVisibleChildName("spinner")
		spinner.Start()

		checkButton.SetSensitive(false)
		cancelBtn.SetSensitive(false)
		chooseBtn.SetSensitive(false)

		// Запускаем проверку в горутине
		go func(img string) {
			var mu sync.Mutex
			mu.Lock()
			out, err := validateImage(img)
			mu.Unlock()

			// Возврат в UI-поток
			glib.IdleAdd(func() bool {
				spinner.Stop()
				stack.SetVisibleChildName("button")
				checkButton.SetSensitive(true)
				cancelBtn.SetSensitive(true)
				chooseBtn.SetSensitive(true)

				if err != nil {
					checkResultLabel.SetLabel("Ошибка проверки образа:\n" + out)
					checkResultLabel.SetVisible(true)
					customImageValid = ""
				} else {
					checkResultLabel.SetLabel("Образ проверен и добавлен в список.")
					checkResultLabel.SetVisible(true)
					customImageValid = imageName

					images = append(images, Choice{
						Name:        imageName,
						Description: "",
					})

					combo.AppendText(imageName)
					comboCount++
					combo.SetActive(comboCount - 1)
				}
				return false
			})
		}(imageName)
	})

	// Нажали «Выйти»
	cancelBtn.ConnectClicked(func() {
		onCancel()
	})

	// Нажали «Выбрать»
	chooseBtn.ConnectClicked(func() {
		activeIndex := combo.Active()
		if activeIndex < 0 {
			fmt.Println("Ничего не выбрано.")
			return
		}

		var resultImage string
		if activeIndex == customChoiceIndex {
			if customImageValid == "" {
				checkResultLabel.SetLabel("Сперва проверьте введённый образ.")
				checkResultLabel.SetVisible(true)
				return
			}
			resultImage = customImageValid
		} else if activeIndex >= len(images) {
			resultImage = images[activeIndex].Name
		} else {
			resultImage = images[activeIndex].Name
		}

		onImageSelected(resultImage)
	})

	// Изначальное описание (для пункта 0)
	if combo.Active() == 0 {
		desc := images[0].Description
		if desc == "" {
			descLabel.SetLabel("Без описания.")
		} else {
			descLabel.SetLabel(desc)
		}
	}

	return outerBox
}
