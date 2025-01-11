package installer

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Run запускает процесс установки
func Run() {
	// Проверка прав суперпользователя
	checkRoot()

	// Проверка наличия необходимых команд
	if err := checkCommands(); err != nil {
		log.Fatalf("Необходимая команда отсутствует: %v\n", err)
	}

	// Шаг 1: Выбор образа
	imageResult := RunImageStep()
	if imageResult == "" {
		log.Println("Образ не был выбран.")
		return
	}
	log.Printf("Выбранный образ: %s\n\n", imageResult)

	// Шаг 2: Выбор диска
	diskResult := RunDiskStep()
	if diskResult == "" {
		log.Println("Диск не был выбран.")
		return
	}

	if !validateDisk(diskResult) {
		log.Fatalf("Выбранный диск %s недействителен или не существует.\n", diskResult)
	}

	log.Printf("Выбранный диск: %s\n", diskResult)

	// Подтверждение удаления данных
	if !confirmAction(fmt.Sprintf("Вы уверены, что хотите уничтожить все данные на диске %s?", diskResult)) {
		log.Println("Операция отменена пользователем.")
		return
	}

	// Тип файловой системы для root
	typeFileSystem := "ext4"

	// Шаг 3: Уничтожение данных и создание разметки
	if err := prepareDisk(diskResult, typeFileSystem); err != nil {
		log.Fatalf("Ошибка подготовки диска: %v\n", err)
	}

	// Шаг 4: Установка с использованием bootc
	if err := installToFilesystem(imageResult, diskResult); err != nil {
		log.Fatalf("Ошибка установки: %v\n", err)
	}

	log.Println("Установка завершена успешно!")
	fmt.Println("Установка завершена успешно!")
}

// checkRoot проверяет, запущен ли установщик от имени root
func checkRoot() {
	if syscall.Geteuid() != 0 {
		fmt.Println("Установщик должен быть запущен с правами суперпользователя (root).")
		os.Exit(1)
	}
}

// checkCommands проверяет наличие необходимых системных команд
func checkCommands() error {
	commands := []string{
		"wipefs",
		"parted",
		"mkfs.fat",
		"mkfs.ext4",
		"mount",
		"umount",
		"blkid",
		"bootc",
		"lsblk",
	}
	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("команда %s не найдена в PATH", cmd)
		}
	}
	return nil
}

// confirmAction запрашивает у пользователя подтверждение действия
func confirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (y/N): ", prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Ошибка чтения ввода: %v\n", err)
			return false
		}
		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Пожалуйста, ответьте 'y' или 'n'.")
		}
	}
}

// validateDisk проверяет существование диска
func validateDisk(disk string) bool {
	if _, err := os.Stat(disk); os.IsNotExist(err) {
		return false
	}
	return true
}

// prepareDisk выполняет подготовку диска
func prepareDisk(disk string, rootFileSystem string) error {
	log.Printf("Подготовка диска %s с root файловой системой %s...\n", disk, rootFileSystem)

	commands := [][]string{
		{"wipefs", "--all", disk},
		{"parted", "-s", disk, "mklabel", "gpt"},
		{"parted", "-s", disk, "mkpart", "primary", "fat32", "1MiB", "601MiB"},
		{"parted", "-s", disk, "set", "1", "boot", "on"}, // Установка флага boot для первого раздела
		{"parted", "-s", disk, "mkpart", "primary", "ext4", "601MiB", "1601MiB"},
		{"parted", "-s", disk, "set", "2", "legacy_boot", "on"}, // Установка legacy_boot для второго раздела
		{"parted", "-s", disk, "mkpart", "primary", "ext4", "1601MiB", "100%"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка выполнения команды %s: %v", args[0], err)
		}
	}

	partitions, err := getPartitionNames(disk)
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	if len(partitions) < 3 {
		return fmt.Errorf("недостаточно разделов на диске")
	}

	// Форматирование разделов
	formats := []struct {
		cmd  string
		args []string
	}{
		{"mkfs.fat", []string{"-F32", partitions[0]}}, // Форматирование EFI раздела
		{"mkfs.ext4", []string{partitions[1]}},        // Форматирование boot раздела
		{"mkfs.ext4", []string{partitions[2]}},        // Форматирование root раздела
	}

	for _, format := range formats {
		cmd := exec.Command(format.cmd, format.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка форматирования %s: %v", format.args[0], err)
		}
	}

	log.Printf("Диск %s успешно подготовлен.\n", disk)
	return nil
}

// installToFilesystem выполняет установку с использованием bootc
func installToFilesystem(image string, disk string) error {
	mountPoint := "/mnt/target"
	partitions, err := getPartitionNames(disk)
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	if len(partitions) < 3 {
		return fmt.Errorf("недостаточно разделов на диске")
	}

	rootPartition := partitions[2]
	if err := mountDisk(rootPartition, mountPoint); err != nil {
		return fmt.Errorf("ошибка монтирования root раздела: %v", err)
	}
	defer unmountDisk(mountPoint)

	cmd := exec.Command("bootc", "install-to-filesystem",
		"--skip-fetch-check", "--generic-image", "--disable-selinux",
		fmt.Sprintf("--root-mount-spec=UUID=%s", getUUID(partitions[2])),
		fmt.Sprintf("--boot-mount-spec=UUID=%s", getUUID(partitions[1])),
		fmt.Sprintf("--efi-mount-spec=UUID=%s", getUUID(partitions[0])),
		fmt.Sprintf("--source-imgref=%s", image),
		mountPoint,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Выполняется установка...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка выполнения bootc: %v", err)
	}

	log.Println("Установка прошла успешно.")
	return nil
}

// getPartitionNames возвращает имена разделов на диске
func getPartitionNames(disk string) ([]string, error) {
	cmd := exec.Command("lsblk", "-ln", "-o", "NAME,TYPE", disk)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения lsblk: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var partitions []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "part" { // Проверяем, что тип устройства "part"
			partitions = append(partitions, "/dev/"+fields[0])
		}
	}

	return partitions, nil
}

// mountDisk монтирует указанный раздел в точку монтирования
func mountDisk(disk string, mountPoint string) error {
	log.Printf("Монтирование диска %s в %s...\n", disk, mountPoint)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}
	cmd := exec.Command("mount", disk, mountPoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка монтирования диска: %v", err)
	}
	return nil
}

// unmountDisk размонтирует указанную точку монтирования
func unmountDisk(mountPoint string) {
	log.Printf("Размонтирование %s...\n", mountPoint)
	cmd := exec.Command("umount", mountPoint)
	if err := cmd.Run(); err != nil {
		log.Printf("Ошибка размонтирования %s: %v\n", mountPoint, err)
	}
}

// getUUID возвращает UUID указанного раздела
func getUUID(disk string) string {
	cmd := exec.Command("blkid", "-s", "UUID", "-o", "value", disk)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Ошибка получения UUID для %s: %v\n", disk, err)
		return ""
	}
	return strings.TrimSpace(string(output))
}
