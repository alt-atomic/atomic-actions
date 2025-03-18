package steps

import (
	"atomic-actions/models/installer/gui/image"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateSummaryStep – финальный шаг, отображающий все выбранные параметры.
func CreateSummaryStep(
	chosenLang, chosenImage, chosenDisk, chosenFilesystem, chosenBootMode, chosenUsername, chosenPassword string,
	onCancel func(),
	onInstall func(),
) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconResult)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)
	wrapper.Append(pic)
	outerBox.Append(wrapper)

	// Сетка для полей (по центру)
	centerBox := gtk.NewBox(gtk.OrientationVertical, 8)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	outerBox.Append(centerBox)

	grid := gtk.NewGrid()
	// Настраиваем сетку, чтобы значения располагались в столбцах
	grid.SetColumnSpacing(12)
	grid.SetRowSpacing(4)
	centerBox.Append(grid)

	// Вспомогательная функция для одной строки
	var row int
	addRow := func(field, value string) {
		// Создаем лейбл для поля
		lblField := gtk.NewLabel(field + ":")
		lblField.SetHAlign(gtk.AlignEnd)

		lblValue := gtk.NewLabel("")
		lblValue.SetUseMarkup(true)
		lblValue.SetLabel("<b>" + value + "</b>")
		lblValue.SetHAlign(gtk.AlignStart)

		// Размещаем в сетке: (столбец=0,row=текущаяСтрока), (столбец=1,row=текущаяСтрока)
		grid.Attach(lblField, 0, row, 1, 1)
		grid.Attach(lblValue, 1, row, 1, 1)

		row++
	}

	// Собираем данные
	stars := strings.Repeat("*", len(chosenPassword))

	addRow("Пользователь", chosenUsername)
	addRow("Пароль", stars)
	addRow("Загрузчик", chosenBootMode)
	addRow("Выбранный образ", chosenImage)
	addRow("Язык системы", chosenLang)
	addRow("Выбранный диск", chosenDisk)
	addRow("Файловая система", chosenFilesystem)

	// Горизонтальный контейнер для кнопок
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	cancelBtn := gtk.NewButtonWithLabel("Назад")
	installBtn := gtk.NewButtonWithLabel("Начать установку")

	cancelBtn.SetSizeRequest(120, 40)
	installBtn.SetSizeRequest(160, 40)
	installBtn.AddCSSClass("suggested-action")

	buttonBox.Append(cancelBtn)
	buttonBox.Append(installBtn)
	outerBox.Append(buttonBox)

	// Обработчики
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	installBtn.ConnectClicked(func() {
		if onInstall != nil {
			onInstall()
		}
	})

	return outerBox
}
