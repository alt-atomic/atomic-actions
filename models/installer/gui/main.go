package main

import (
	"fmt"
	"os"
	"unsafe"

	"atomic-actions/models/installer/gui/steps"

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

func NewAdwApplicationWindow(app *adw.Application) *adw.ApplicationWindow {
	gtkApp := (*gtk.Application)(unsafe.Pointer(app))
	return adw.NewApplicationWindow(gtkApp)
}

func onActivate(app *adw.Application) {
	window := NewAdwApplicationWindow(app)
	window.SetDefaultSize(900, 700)
	window.SetTitle("Установщик")

	toolbarView := adw.NewToolbarView()

	mainHeader := adw.NewHeaderBar()
	boldLabel := gtk.NewLabel("")
	boldLabel.SetUseMarkup(true)
	boldLabel.SetLabel("<b>Atomic Installer</b>")
	mainHeader.SetTitleWidget(boldLabel)
	toolbarView.AddTopBar(mainHeader)

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

	var chosenImage string
	var chosenDisk string
	var chosenFilesystem string
	var chosenBootMode string
	var chosenUsername string
	var chosenPassword string
	var chosenLang string

	// Имена шагов
	stepTitles := []string{
		"1. Выбор языка",
		"2. Выбор образа",
		"3. Выбор диска",
		"4. Выбор файловой системы",
		"5. Выбор загрузчика",
		"6. Выбор пользователя",
		"6. Итог",
	}

	stepsCount := len(stepTitles)

	stepDone := make([]bool, stepsCount)

	var currentStep int // индекс текущего шага

	var stepsArr []func() gtk.Widgetter

	// Функция, которая будет обновлять отображение контента
	var updateStep func()

	updateStep = func() {
		stepLabel.SetLabel(stepTitles[currentStep])

		// Создаём новый box
		newContent := gtk.NewBox(gtk.OrientationVertical, 10)
		newContent.SetMarginTop(20)
		newContent.SetMarginBottom(20)
		newContent.SetMarginStart(20)
		newContent.SetMarginEnd(20)

		// Генерируем виджет для текущего шага
		stepWidget := stepsArr[currentStep]()
		newContent.Append(stepWidget)

		toolbarView.SetContent(newContent)

		// «Назад» доступен, если это не первый шаг
		backBtn.SetSensitive(currentStep > 0)

		// «Вперёд» доступен, только если этот шаг уже завершён
		nextBtn.SetSensitive(stepDone[currentStep])

		if currentStep == stepsCount-1 {
			nextBtn.SetTooltipText("Готово")
		} else {
			nextBtn.SetTooltipText("Вперёд")
		}
	}

	// Заполняем stepsArr
	stepsArr = []func() gtk.Widgetter{
		// Шаг 1: Выбор языка
		func() gtk.Widgetter {
			return steps.CreateLanguageStep(
				func(lang string) {
					chosenLang = lang
					fmt.Println("Шаг: Выбор языка – выбран:", chosenLang)
					// Помечаем шаг выполненным, разрешаем "Вперёд" или сразу переходим
					stepDone[0] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
				func() {
					os.Exit(0)
				},
			)
		},
		// Шаг 2: Выбор образа
		func() gtk.Widgetter {
			return steps.CreateImageStep(
				func(selected string) {
					chosenImage = selected
					fmt.Println("Пользователь выбрал образ:", chosenImage)
					stepDone[1] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
				func() {
					if currentStep > 0 {
						currentStep--
						updateStep()
					}
				},
			)
		},
		// Шаг 3: Выбор диска
		func() gtk.Widgetter {
			return steps.CreateDiskStep(
				func(disk string) {
					chosenDisk = disk
					fmt.Println("Пользователь выбрал диск:", chosenDisk)
					stepDone[2] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
				func() {
					if currentStep > 0 {
						currentStep--
						updateStep()
					}
				},
			)
		},
		// Шаг 4: Выбор файловой системы
		func() gtk.Widgetter {
			return steps.CreateFilesystemStep(
				func(fs string) {
					chosenFilesystem = fs
					fmt.Println("Выбрана ФС:", chosenFilesystem)
					stepDone[3] = true
					nextBtn.SetSensitive(true)

					// Можно сразу переходить к следующему шагу:
					currentStep++
					updateStep()
				},
				func() {
					// onCancel
					if currentStep > 0 {
						currentStep--
						updateStep()
					}
				},
			)
		},
		// Шаг 5: Выбор загрузчика
		func() gtk.Widgetter {
			return steps.CreateBootLoaderStep(
				func(bootMode string) {
					chosenBootMode = bootMode
					fmt.Println("Выбран загрузчик:", chosenBootMode)
					stepDone[4] = true
					nextBtn.SetSensitive(true)
					// Можно сразу перейти дальше:
					currentStep++
					updateStep()
				},
				func() {
					// onCancel
					// Вернуться на предыдущий шаг
					if currentStep > 0 {
						currentStep--
						updateStep()
					}
				},
			)
		},
		// Шаг 6: Создание пользователя
		func() gtk.Widgetter {
			return steps.CreateUserStep(
				func(username, password string) {
					chosenUsername = username
					chosenPassword = password
					fmt.Println("Создан пользователь:", chosenUsername, chosenPassword)
					stepDone[5] = true
					nextBtn.SetSensitive(true)

					currentStep++
					updateStep()
				},
				func() {
					if currentStep > 0 {
						currentStep--
						updateStep()
					}
				},
			)
		},
		// Шаг 7: Итог
		func() gtk.Widgetter {
			return steps.CreateSummaryStep(
				chosenLang,
				chosenImage,
				chosenDisk,
				chosenFilesystem,
				chosenBootMode,
				chosenUsername,
				chosenPassword,
				func() {
					if currentStep > 0 {
						currentStep--
						updateStep()
					}
				},
				func() {
					fmt.Println("Запуск установки с параметрами:")
					fmt.Println("Язык:", chosenLang)
					fmt.Println("Образ:", chosenImage)
					fmt.Println("Диск:", chosenDisk)
					// Можно завершить программу, начать реальную установку и т.д.
				},
			)
		},
	}

	// Изначально 0-й шаг, он не завершён
	currentStep = 0
	stepDone[0] = false

	for i := 1; i < stepsCount; i++ {
		stepDone[i] = false
	}

	updateStep()

	// Кнопка «Назад»
	backBtn.ConnectClicked(func() {
		if currentStep > 0 {
			currentStep--
			updateStep()
		}
	})

	// Кнопка «Вперёд»
	nextBtn.ConnectClicked(func() {
		// Если мы не на последнем шаге – переходим на следующий (если он есть)
		if currentStep < stepsCount-1 {
			currentStep++
			updateStep()
		} else {
			// На последнем шаге – завершаем
			fmt.Println("Все шаги завершены!")
			fmt.Println("Итоговый образ:", chosenImage)
			fmt.Println("Итоговый диск:", chosenDisk)
			window.Close()
		}
	})

	window.SetContent(toolbarView)
	window.SetVisible(true)
}
