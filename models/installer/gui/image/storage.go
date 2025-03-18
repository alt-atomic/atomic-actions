package image

import (
	_ "embed" // Включаем поддержку go:embed
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed icons/translate.png
var iconLanguage []byte

//go:embed icons/docker.png
var iconImage []byte

//go:embed icons/disk.png
var iconDisk []byte

//go:embed icons/folder.png
var iconFilesystem []byte

//go:embed icons/loader.png
var iconLoader []byte

//go:embed icons/user.png
var iconUser []byte

//go:embed icons/layers.png
var iconResult []byte

const (
	IconLanguage = iota
	IconImage
	IconDisk
	IconFilesystem
	IconBoot
	IconUser
	IconResult
)

// NewIconFromEmbed возвращает готовый к вставке в UI виджет
func NewIconFromEmbed(iconType int) gtk.Widgetter {
	var icon []byte
	switch iconType {
	case IconLanguage:
		icon = iconLanguage
	case IconImage:
		icon = iconImage
	case IconDisk:
		icon = iconDisk
	case IconFilesystem:
		icon = iconFilesystem
	case IconBoot:
		icon = iconLoader
	case IconUser:
		icon = iconUser
	case IconResult:
		icon = iconResult
	default:
		return nil
	}

	glibBytes := glib.NewBytesWithGo(icon)
	texture, err := gdk.NewTextureFromBytes(glibBytes)
	if err != nil {
		log.Println("Ошибка создания gdk.Texture из байтов:", err)
		return gtk.NewPictureForPaintable(nil)
	}

	pic := gtk.NewPictureForPaintable(texture)
	return pic
}
