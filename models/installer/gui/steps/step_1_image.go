package steps

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateImageStep возвращает виджет для шага выбора образа.
func CreateImageStep() gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationVertical, 12)

	// Основная метка
	label := gtk.NewLabel("бла бла что-то написано")
	box.Append(label)

	// Горизонтальный контейнер для кнопок
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20) // Отступ сверху, в пикселях

	// Создаем кнопки "Отмена" и "Выбрать"
	cancelBtn := gtk.NewButtonWithLabel("Отмена")
	chooseBtn := gtk.NewButtonWithLabel("Выбрать")

	buttonBox.Append(cancelBtn)
	buttonBox.Append(chooseBtn)

	// Добавляем контейнер с кнопками в основной box
	box.Append(buttonBox)

	return box
}
