package steps

import (
	"atomic-actions/models/installer/gui/image"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateBootLoaderStep – GUI-шаг выбора загрузчика (UEFI или LEGACY).
func CreateBootLoaderStep(onBootModeSelected func(string), onCancel func()) gtk.Widgetter {
	// Внешний вертикальный контейнер
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconBoot)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	outerBox.Append(wrapper)

	// Центральный контейнер, чтобы ComboBox занял место по центру
	centerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	outerBox.Append(centerBox)

	// Проверяем поддержку UEFI
	uefiSupported := checkUEFISupport()

	// Формируем список
	var choices []string
	if uefiSupported {
		choices = []string{
			"UEFI (рекомендуется для современных систем)",
			"LEGACY (совместимый вариант)",
		}
	} else {
		choices = []string{
			"LEGACY (UEFI не поддерживается)",
		}
	}

	combo := gtk.NewComboBoxText()
	for _, c := range choices {
		combo.AppendText(c)
	}
	combo.SetActive(0)
	centerBox.Append(combo)

	// Доп. лейбл ниже
	infoLabel := gtk.NewLabel("")
	infoLabel.SetHAlign(gtk.AlignStart)
	infoLabel.SetMarginTop(10)
	infoLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(infoLabel)

	// Начальный текст
	if uefiSupported {
		infoLabel.SetLabel("Ваш компьютер поддерживает UEFI загрузку - это рекомендуемый выбор.")
	} else {
		infoLabel.SetLabel("UEFI не поддерживается на данной системе, используем LEGACY.")
	}

	// При смене выбора (если нужно динамически менять подсказку)
	combo.ConnectChanged(func() {
		idx := combo.Active()
		if !uefiSupported || idx < 0 {
			return
		}
		if idx == 0 {
			infoLabel.SetLabel("Выбран режим UEFI - это рекомендуемый выбор для современных систем.")
		} else {
			infoLabel.SetLabel("Выбран LEGACY - более совместимый вариант для BIOS/UEFI.")
		}
	})

	// Горизонтальный контейнер для кнопок внизу
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	backBtn := gtk.NewButtonWithLabel("Назад")
	chooseBtn := gtk.NewButtonWithLabel("Выбрать")
	backBtn.SetSizeRequest(120, 40)
	chooseBtn.SetSizeRequest(120, 40)
	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(backBtn)
	buttonBox.Append(chooseBtn)
	outerBox.Append(buttonBox)

	// Обработка "Назад"
	backBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	// «Выбрать»
	chooseBtn.ConnectClicked(func() {
		active := combo.Active()
		if active < 0 {
			log.Println("Тип загрузки не выбран.")
			return
		}
		chosenStr := choices[active]
		// Обрезаем "UEFI " или "LEGACY "
		if idx := strings.Index(chosenStr, " "); idx != -1 {
			chosenStr = chosenStr[:idx]
		}

		fmt.Printf("Пользователь выбрал тип загрузки: %s\n", chosenStr)
		onBootModeSelected(chosenStr)
	})

	return outerBox
}

// checkUEFISupport – упрощённая проверка наличия каталога /sys/firmware/efi/efivars
func checkUEFISupport() bool {
	_, err := os.Stat("/sys/firmware/efi/efivars")
	return err == nil
}
