package steps

import (
	"atomic-actions/models/installer/gui/image"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type DiskInfo struct {
	Path   string
	Size   string
	Model  string
	SizeGB float64
}

// CreateDiskStep – виджет для выбора диска
func CreateDiskStep(onDiskSelected func(string), onCancel func()) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconDisk)
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

	// Получаем список дисков
	disks := getAvailableDisks()
	if len(disks) == 0 {
		log.Println("Не найдено подходящих дисков (≥ 60 ГБ).")
	}

	combo := gtk.NewComboBoxText()
	for _, d := range disks {
		display := fmt.Sprintf("%s (%s)", d.Path, d.Size)
		if d.Model != "" {
			display += " - " + d.Model
		}
		combo.AppendText(display)
	}
	if len(disks) > 0 {
		combo.SetActive(0)
	}
	centerBox.Append(combo)

	// Лейбл для описания
	descLabel := gtk.NewLabel("Диск размером ≥ 60 ГБ")
	descLabel.SetHAlign(gtk.AlignStart)
	descLabel.SetMarginTop(10)
	descLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(descLabel)

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

	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	chooseBtn.ConnectClicked(func() {
		active := combo.Active()
		if active < 0 {
			fmt.Println("Ничего не выбрано.")
			return
		}
		chosenDisk := disks[active].Path
		onDiskSelected(chosenDisk)
	})

	return outerBox
}

func getAvailableDisks() []DiskInfo {
	out, err := exec.Command("lsblk", "-o", "NAME,SIZE,TYPE,MODEL", "-d", "-n").Output()
	if err != nil {
		log.Println("Ошибка получения списка дисков:", err)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []DiskInfo

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name, sizeStr, devType, model := fields[0], fields[1], fields[2], fields[3]

		if devType != "disk" {
			continue
		}
		if strings.HasPrefix(name, "zram") || strings.HasPrefix(name, "loop") {
			continue
		}
		sizeGB, err := parseSize(sizeStr)
		if err != nil {
			continue
		}
		if sizeGB < 60 {
			continue
		}
		path := "/dev/" + name
		info := DiskInfo{
			Path:   path,
			Size:   sizeStr,
			Model:  model,
			SizeGB: sizeGB,
		}
		result = append(result, info)
	}
	return result
}

func parseSize(sizeStr string) (float64, error) {
	if len(sizeStr) < 2 {
		return 0, fmt.Errorf("неизвестный формат размера: %s", sizeStr)
	}
	sizeStr = strings.ReplaceAll(sizeStr, ",", ".")
	unit := sizeStr[len(sizeStr)-1]
	valStr := sizeStr[:len(sizeStr)-1]
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, fmt.Errorf("ошибка парсинга числа из %s: %w", valStr, err)
	}
	switch unit {
	case 'G':
		return val, nil
	case 'M':
		return val / 1024.0, nil
	case 'T':
		return val * 1024.0, nil
	default:
		return 0, fmt.Errorf("неизвестная единица: %c", unit)
	}
}
