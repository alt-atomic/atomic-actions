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

	// Шаг 2: Выбор Диска
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
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" || response == "" {
			return false
		} else {
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

// Уничтожение данных и создание разметки
func prepareDisk(disk string, rootFileSystem string) error {
	log.Printf("Подготовка диска %s с root файловой системой %s...\n", disk, rootFileSystem)

	// Уничтожение данных на диске
	cmd := exec.Command("wipefs", "--all", disk)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка очистки диска: %v", err)
	}

	// Создание новой GPT-разметки
	cmd = exec.Command("parted", "-s", disk, "mklabel", "gpt")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка создания GPT-разметки: %v", err)
	}

	// Создание EFI раздела (512 MB)
	cmd = exec.Command("parted", "-s", disk, "mkpart", "primary", "1MiB", "513MiB")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка создания EFI раздела: %v", err)
	}

	// Создание boot раздела (1 GB)
	cmd = exec.Command("parted", "-s", disk, "mkpart", "primary", "513MiB", "1.5GiB")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка создания boot раздела: %v", err)
	}

	// Создание root раздела (остаток диска)
	cmd = exec.Command("parted", "-s", disk, "mkpart", "primary", "1.5GiB", "100%")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка создания root раздела: %v", err)
	}

	// Получение имен разделов
	efiPartition, err := getPartitionName(disk, 1)
	if err != nil {
		return fmt.Errorf("ошибка получения имени EFI раздела: %v", err)
	}
	bootPartition, err := getPartitionName(disk, 2)
	if err != nil {
		return fmt.Errorf("ошибка получения имени boot раздела: %v", err)
	}
	rootPartition, err := getPartitionName(disk, 3)
	if err != nil {
		return fmt.Errorf("ошибка получения имени root раздела: %v", err)
	}

	// Форматирование EFI раздела
	cmd = exec.Command("mkfs.fat", "-F32", efiPartition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка форматирования EFI раздела: %v", err)
	}

	// Форматирование boot раздела
	cmd = exec.Command("mkfs."+rootFileSystem, bootPartition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка форматирования boot раздела: %v", err)
	}

	// Форматирование root раздела
	cmd = exec.Command("mkfs."+rootFileSystem, rootPartition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка форматирования root раздела: %v", err)
	}

	log.Printf("Диск %s успешно подготовлен.\n", disk)
	return nil
}

func formatPartitions(disk string, rootFileSystem string) error {
	// Получение имен разделов
	efiPartition, err := getPartitionName(disk, 1)
	if err != nil {
		return fmt.Errorf("ошибка получения имени EFI раздела: %v", err)
	}
	bootPartition, err := getPartitionName(disk, 2)
	if err != nil {
		return fmt.Errorf("ошибка получения имени boot раздела: %v", err)
	}
	rootPartition, err := getPartitionName(disk, 3)
	if err != nil {
		return fmt.Errorf("ошибка получения имени root раздела: %v", err)
	}

	// Форматирование EFI раздела
	cmd := exec.Command("mkfs.fat", "-F32", efiPartition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка форматирования EFI раздела: %v", err)
	}

	// Форматирование boot раздела
	cmd = exec.Command("mkfs."+rootFileSystem, bootPartition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка форматирования boot раздела: %v", err)
	}

	// Форматирование root раздела
	cmd = exec.Command("mkfs."+rootFileSystem, rootPartition)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка форматирования root раздела: %v", err)
	}

	return nil
}

// installToFilesystem выполняет установку с использованием bootc
func installToFilesystem(image string, disk string) error {
	mountPoint := "/mnt/target"

	// Получение имен разделов
	efiPartition, err := getPartitionName(disk, 1)
	if err != nil {
		return fmt.Errorf("ошибка получения имени EFI раздела: %v", err)
	}
	bootPartition, err := getPartitionName(disk, 2)
	if err != nil {
		return fmt.Errorf("ошибка получения имени boot раздела: %v", err)
	}
	rootPartition, err := getPartitionName(disk, 3)
	if err != nil {
		return fmt.Errorf("ошибка получения имени root раздела: %v", err)
	}

	// Получение UUID разделов
	rootUUID := getUUID(rootPartition)
	if rootUUID == "" {
		return fmt.Errorf("не удалось получить UUID для раздела %s", rootPartition)
	}

	bootUUID := getUUID(bootPartition)
	if bootUUID == "" {
		return fmt.Errorf("не удалось получить UUID для раздела %s", bootPartition)
	}

	efiUUID := getUUID(efiPartition)
	if efiUUID == "" {
		return fmt.Errorf("не удалось получить UUID для раздела %s", efiPartition)
	}

	// Монтируем root раздел
	if err := mountDisk(rootPartition, mountPoint); err != nil {
		return fmt.Errorf("ошибка монтирования root раздела: %v", err)
	}
	defer unmountDisk(mountPoint)

	// Выполнение установки с использованием bootc
	cmd := exec.Command(
		"bootc", "install-to-filesystem",
		"--skip-fetch-check",
		"--generic-image",
		"--disable-selinux",
		fmt.Sprintf("--root-mount-spec=UUID=%s", rootUUID),
		fmt.Sprintf("--boot-mount-spec=UUID=%s", bootUUID),
		fmt.Sprintf("--efi-mount-spec=UUID=%s", efiUUID),
		fmt.Sprintf("--source-imgref=%s", image),
		mountPoint,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Выполняется установка...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка выполнения bootc: %v", err)
	}

	// Монтируем EFI раздел для загрузчика
	efiMountPoint := fmt.Sprintf("%s/boot/efi", mountPoint)
	if err := mountDisk(efiPartition, efiMountPoint); err != nil {
		return fmt.Errorf("ошибка монтирования EFI раздела: %v", err)
	}
	defer unmountDisk(efiMountPoint)

	log.Println("Установка прошла успешно.")
	return nil
}

// mountDisk монтирует указанный раздел в точку монтирования
func mountDisk(disk string, mountPoint string) error {
	log.Printf("Монтирование диска %s в %s...\n", disk, mountPoint)

	// Создание точки монтирования, если не существует
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}

	// Монтирование раздела
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

// getPartitionName возвращает имя раздела по его номеру
func getPartitionName(disk string, number int) (string, error) {
	cmd := exec.Command("lsblk", "-ln", "-o", "NAME", disk)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения lsblk: %v", err)
	}
	partitions := strings.Fields(string(output))
	if number-1 < 0 || number-1 >= len(partitions) {
		return "", fmt.Errorf("неверный номер раздела: %d", number)
	}
	// Определение полного пути к разделу
	return "/dev/" + partitions[number-1], nil
}
