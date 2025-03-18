package steps

import (
	"atomic-actions/models/installer/gui/image"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateUserStep – GUI-шаг для создания пользователя.
func CreateUserStep(onUserCreated func(string, string), onCancel func()) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconUser)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	outerBox.Append(wrapper)

	// Вставим сюда "контент" – поля ввода
	contentBox := gtk.NewBox(gtk.OrientationVertical, 12)
	contentBox.SetVExpand(true) // Чтобы занять всё пространство
	outerBox.Append(contentBox)

	// Поле "Имя пользователя"
	usernameLabel := gtk.NewLabel("Имя пользователя:")
	usernameLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(usernameLabel)

	usernameEntry := gtk.NewEntry()
	usernameEntry.SetPlaceholderText("username")
	// Ставим ширину, если нужно
	usernameEntry.SetSizeRequest(250, -1)
	contentBox.Append(usernameEntry)

	// Поле "Пароль"
	passwordLabel := gtk.NewLabel("Пароль:")
	passwordLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(passwordLabel)

	passwordEntry := gtk.NewEntry()
	passwordEntry.SetPlaceholderText("******")
	passwordEntry.SetVisibility(false)
	passwordEntry.SetInputPurpose(gtk.InputPurposePassword)
	passwordEntry.SetSizeRequest(250, -1)
	contentBox.Append(passwordEntry)

	// Поле "Повтор пароля"
	repeatLabel := gtk.NewLabel("Повторите пароль:")
	repeatLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(repeatLabel)

	repeatEntry := gtk.NewEntry()
	repeatEntry.SetPlaceholderText("******")
	repeatEntry.SetVisibility(false)
	repeatEntry.SetInputPurpose(gtk.InputPurposePassword)
	repeatEntry.SetSizeRequest(250, -1)
	contentBox.Append(repeatEntry)

	// Метка для вывода ошибок
	errorLabel := gtk.NewLabel("")
	errorLabel.SetHAlign(gtk.AlignStart)
	errorLabel.SetMarginTop(8)
	contentBox.Append(errorLabel)

	// Горизонтальный контейнер для кнопок, который пойдёт в самый низ
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	cancelBtn := gtk.NewButtonWithLabel("Назад")
	startBtn := gtk.NewButtonWithLabel("Выбрать")

	cancelBtn.SetSizeRequest(120, 40)
	startBtn.SetSizeRequest(120, 40)
	startBtn.AddCSSClass("suggested-action")

	buttonBox.Append(cancelBtn)
	buttonBox.Append(startBtn)

	// Добавляем buttonBox в outerBox – после contentBox
	outerBox.Append(buttonBox)

	// Обработчик "Отмена"
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	// Обработчик "Начать установку"
	startBtn.ConnectClicked(func() {
		userName := usernameEntry.Text()
		pass := passwordEntry.Text()
		passRepeat := repeatEntry.Text()

		// Проверка полей
		if userName == "" || pass == "" {
			errorLabel.SetLabel("Имя пользователя и пароль не могут быть пустыми.")
			return
		}
		if pass != passRepeat {
			errorLabel.SetLabel("Пароли не совпадают. Попробуйте снова.")
			return
		}

		// Если всё ок – вызываем колбэк
		fmt.Printf("Создание пользователя: %s (пароль: %d символов)\n", userName, len(pass))
		onUserCreated(userName, pass)
	})

	return outerBox
}
