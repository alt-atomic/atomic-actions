package steps

import (
	"atomic-actions/models/installer/gui/image"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateLanguageStep – шаг выбора языка.
func CreateLanguageStep(onLanguageSelected func(string), onCancel func()) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconLanguage)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	box.Append(wrapper)

	// Список языков
	languages := []string{
		"Русский",
		"English",
	}

	combo := gtk.NewComboBoxText()
	combo.SetSizeRequest(300, -1)
	for _, lang := range languages {
		combo.AppendText(lang)
	}
	combo.SetActive(0)

	centerBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	centerBox.SetHExpand(true)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	centerBox.Append(combo)

	box.Append(centerBox)

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
	box.Append(buttonBox)

	backBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	chooseBtn.ConnectClicked(func() {
		activeIdx := combo.Active()
		if activeIdx < 0 {
			fmt.Println("Язык не выбран.")
			return
		}
		selectedLang := languages[activeIdx]
		fmt.Printf("Выбран язык: %s\n", selectedLang)
		onLanguageSelected(selectedLang)
	})

	return box
}
