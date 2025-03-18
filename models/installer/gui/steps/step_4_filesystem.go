package steps

import (
	"atomic-actions/models/installer/gui/image"
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateFilesystemStep возвращает GUI-шаг выбора файловой системы.
func CreateFilesystemStep(onFsSelected func(string), onCancel func()) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconFilesystem)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	outerBox.Append(wrapper)

	centerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	outerBox.Append(centerBox)

	// Список вариантов (для ComboBoxText)
	fsChoices := []string{
		"btrfs (Будут добавлены subvolume:@, @home, @var)",
		"ext4 (Установка в корень /)",
	}

	combo := gtk.NewComboBoxText()
	for _, choice := range fsChoices {
		combo.AppendText(choice)
	}
	combo.SetActive(0)
	centerBox.Append(combo)

	// Метка, которая будет показывать дополнительные описания
	noteLabel := gtk.NewLabel("")
	noteLabel.SetHAlign(gtk.AlignStart)
	noteLabel.SetMarginTop(10)
	noteLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(noteLabel)

	// Изначально для btrfs
	noteLabel.SetLabel("btrfs – рекомендуемый выбор, хорошо сочетается с атомарным образом")

	// Меняем описание при смене выбора
	combo.ConnectChanged(func() {
		activeIndex := combo.Active()
		if activeIndex < 0 {
			noteLabel.SetLabel("")
			return
		}
		if activeIndex == 0 {
			noteLabel.SetLabel("btrfs – рекомендуемый выбор, хорошо сочетается с атомарным образом")
		} else {
			noteLabel.SetLabel("ext4 – классическая, проверенная ФС")
		}
	})

	// Горизонтальный контейнер для кнопок внизу
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

	// Обработчик "Назад"
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	// Обработчик "Выбрать"
	chooseBtn.ConnectClicked(func() {
		activeIndex := combo.Active()
		if activeIndex < 0 {
			fmt.Println("Файловая система не выбрана")
			return
		}

		chosenStr := fsChoices[activeIndex]
		fsName := chosenStr
		if idx := strings.Index(fsName, " "); idx != -1 {
			fsName = fsName[:idx]
		}

		onFsSelected(fsName)
	})

	return outerBox
}
