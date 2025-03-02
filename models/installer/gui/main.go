package main

import (
	"fmt"
	"os"
	"unsafe"

	"atomic-actions/models/installer/gui/steps" // Импортируем пакет с шагами
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	app := adw.NewApplication("com.example.AdwExampleApp", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() {
		onActivate(app)
	})
	os.Exit(app.Run(os.Args))
}

// Обёртка для создания окна AdwApplicationWindow из adw.Application
func NewAdwApplicationWindow(app *adw.Application) *adw.ApplicationWindow {
	gtkApp := (*gtk.Application)(unsafe.Pointer(app))
	return adw.NewApplicationWindow(gtkApp)
}

func onActivate(app *adw.Application) {
	window := NewAdwApplicationWindow(app)
	window.SetDefaultSize(600, 400)
	window.SetTitle("Установщик")

	toolbarView := adw.NewToolbarView()

	// Первый top bar: Заголовок
	mainHeader := adw.NewHeaderBar()
	boldLabel := gtk.NewLabel("")
	boldLabel.SetUseMarkup(true)
	boldLabel.SetLabel("<b>Atomic Installer</b>")
	mainHeader.SetTitleWidget(boldLabel)
	toolbarView.AddTopBar(mainHeader)

	// Второй top bar: Навигация (кнопки с иконками)
	navCenterBox := gtk.NewCenterBox()
	backBtn := gtk.NewButton()
	backIcon := gtk.NewImageFromIconName("go-previous-symbolic")
	backBtn.SetChild(backIcon)
	backBtn.AddCSSClass("circular")
	backBtn.AddCSSClass("flat")
	nextBtn := gtk.NewButton()
	nextIcon := gtk.NewImageFromIconName("go-next-symbolic")
	nextBtn.SetChild(nextIcon)
	nextBtn.AddCSSClass("circular")
	nextBtn.AddCSSClass("flat")
	stepLabel := gtk.NewLabel("")

	leftBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	leftBox.SetMarginStart(20)
	leftBox.Append(backBtn)
	rightBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	rightBox.SetMarginEnd(20)
	rightBox.Append(nextBtn)

	navCenterBox.SetStartWidget(leftBox)
	navCenterBox.SetCenterWidget(stepLabel)
	navCenterBox.SetEndWidget(rightBox)
	toolbarView.AddTopBar(navCenterBox)

	// Массив шагов и логика переключения
	stepsArr := []func() gtk.Widgetter{
		steps.CreateImageStep,
		steps.CreateDiskStep,
		// Дополните другими шагами: steps.CreateFilesystemStep, steps.CreateBootStep, steps.CreateUserStep
	}
	stepTitles := []string{
		"1. Выбор образа",
		"2. Выбор диска",
		"3. Выбор ФС",
		"4. Настройки загрузчика",
		"5. Создание пользователя",
	}
	currentStep := 0

	// Функция обновления шага, создающая новый контейнер для контента
	var updateStep func()
	updateStep = func() {
		stepLabel.SetLabel(stepTitles[currentStep])
		// Создаем новый контейнер для центрального контента
		newContentBox := gtk.NewBox(gtk.OrientationVertical, 10)
		newContentBox.SetMarginTop(20)
		newContentBox.SetMarginBottom(20)
		newContentBox.SetMarginStart(20)
		newContentBox.SetMarginEnd(20)
		newContentBox.Append(stepsArr[currentStep]())

		toolbarView.SetContent(newContentBox)

		backBtn.SetSensitive(currentStep > 0)
		if currentStep == len(stepsArr)-1 {
			nextBtn.SetTooltipText("Готово")
		} else {
			nextBtn.SetTooltipText("Вперёд")
		}
	}
	updateStep()

	backBtn.ConnectClicked(func() {
		if currentStep > 0 {
			currentStep--
			updateStep()
		}
	})
	nextBtn.ConnectClicked(func() {
		if currentStep < len(stepsArr)-1 {
			currentStep++
			updateStep()
		} else {
			fmt.Println("Все шаги завершены!")
			window.Close()
		}
	})

	window.SetContent(toolbarView)
	window.SetVisible(true)
}
