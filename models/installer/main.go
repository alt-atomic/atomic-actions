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
	typeBoot := "legacy" // legacy или UEFI делать проверку dmidecode | grep -i "EFI"
	// Тип файловой системы для root
	typeFileSystem := "btrfs"

	// Шаг 3: Уничтожение данных и создание разметки
	if err := prepareDisk(diskResult, typeFileSystem, typeBoot); err != nil {
		log.Fatalf("Ошибка подготовки диска: %v\n", err)
	}

	return
	// Шаг 4: Установка с использованием bootc
	//if err := installToFilesystem(imageResult, diskResult); err != nil {
	//	log.Fatalf("Ошибка установки: %v\n", err)
	//}
	//
	//log.Println("Установка завершена успешно!")
}

// checkRoot проверяет, запущен ли установщик от имени root
func checkRoot() {
	if syscall.Geteuid() != 0 {
		log.Println("Установщик должен быть запущен с правами суперпользователя (root).")
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
func prepareDisk(disk string, rootFileSystem string, typeBoot string) error {
	log.Printf("Подготовка диска %s с файловой системой %s в режиме %s\n", disk, rootFileSystem, typeBoot)

	// Команды для разметки
	var commands [][]string

	if typeBoot == "legacy" {
		commands = [][]string{
			{"wipefs", "--all", disk},
			{"parted", "-s", disk, "mklabel", "gpt"},
			{"parted", "-s", disk, "mkpart", "primary", "1MiB", "3MiB"},                    // BIOS Boot Partition (2 МиБ)
			{"parted", "-s", disk, "set", "1", "bios_grub", "on"},                          // BIOS Boot Partition
			{"parted", "-s", disk, "mkpart", "primary", "fat32", "3MiB", "1003MiB"},        // EFI раздел (1 ГБ)
			{"parted", "-s", disk, "set", "2", "boot", "on"},                               // EFI раздел
			{"parted", "-s", disk, "mkpart", "primary", "ext4", "1003MiB", "3003MiB"},      // Boot раздел (2 ГБ)
			{"parted", "-s", disk, "mkpart", "primary", rootFileSystem, "3003MiB", "100%"}, // Root раздел
		}
	} else if typeBoot == "UEFI" {
		commands = [][]string{
			{"wipefs", "--all", disk},
			{"parted", "-s", disk, "mklabel", "gpt"},
			{"parted", "-s", disk, "mkpart", "primary", "fat32", "1MiB", "601MiB"},         // EFI раздел (600 МБ)
			{"parted", "-s", disk, "set", "1", "boot", "on"},                               // EFI раздел
			{"parted", "-s", disk, "mkpart", "primary", "ext4", "601MiB", "2601MiB"},       // Boot раздел (2 ГБ)
			{"parted", "-s", disk, "mkpart", "primary", rootFileSystem, "2601MiB", "100%"}, // Root раздел
		}
	} else {
		return fmt.Errorf("неизвестный тип загрузки: %s", typeBoot)
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка выполнения команды %s: %v", args[0], err)
		}
	}

	partitions, err := getNamedPartitions(disk, typeBoot)
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	if len(partitions) < 3 {
		return fmt.Errorf("недостаточно разделов на диске")
	}

	var partitionList []string
	for key, value := range partitions {
		partitionList = append(partitionList, fmt.Sprintf("%s: %s", key, value))
	}
	log.Printf("Partitions: %s\n", strings.Join(partitionList, ", "))

	formats := []struct {
		cmd  string
		args []string
	}{
		{"mkfs.vfat", []string{"-F32", partitions["efi"]}}, // Форматирование EFI раздела
		{"mkfs.ext4", []string{partitions["boot"]}},        // Форматирование boot раздела
	}

	if rootFileSystem == "ext4" {
		formats = append(formats, struct {
			cmd  string
			args []string
		}{"mkfs.ext4", []string{partitions["root"]}})
	} else if rootFileSystem == "btrfs" {
		formats = append(formats, struct {
			cmd  string
			args []string
		}{"mkfs.btrfs", []string{partitions["root"]}})
	} else {
		return fmt.Errorf("неизвестная файловая система: %s", rootFileSystem)
	}

	for _, format := range formats {
		cmd := exec.Command(format.cmd, format.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка форматирования %s: %v", format.args[0], err)
		}
	}

	if rootFileSystem == "btrfs" {
		if err := createBtrfsSubvolumes(partitions["root"]); err != nil {
			return fmt.Errorf("ошибка создания подтомов Btrfs: %v", err)
		}
	}

	log.Printf("Диск %s успешно подготовлен.\n", disk)
	return nil
}

// createBtrfsSubvolumes создает подтомы Btrfs
func createBtrfsSubvolumes(rootPartition string) error {
	mountPoint := "/mnt/btrfs-setup"
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}
	defer os.RemoveAll(mountPoint)

	if err := mountDisk(rootPartition, mountPoint); err != nil {
		return fmt.Errorf("ошибка монтирования Btrfs раздела: %v", err)
	}
	defer unmountDisk(mountPoint)

	subvolumes := []string{"@", "@home", "@var"}
	for _, subvol := range subvolumes {
		subvolPath := fmt.Sprintf("%s/%s", mountPoint, subvol)
		if _, err := os.Stat(subvolPath); os.IsNotExist(err) {
			cmd := exec.Command("btrfs", "subvolume", "create", subvolPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ошибка создания подтома %s: %v", subvol, err)
			}
		} else {
			log.Printf("Подтом %s уже существует, пропуск.", subvol)
		}
	}

	return nil
}

// installToFilesystem выполняет установку с использованием bootc
func installToFilesystem(image string, disk string, typeBoot string) error {
	mountPoint := "/mnt/target"
	mountPointBoot := "/mnt/target/boot"
	efiMountPoint := "/mnt/target/boot/efi"

	// Получаем именованные разделы
	partitions, err := getNamedPartitions(disk, typeBoot)
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	// Монтирование root раздела
	if err := mountDisk(partitions["root"], mountPoint); err != nil {
		return fmt.Errorf("ошибка монтирования root раздела: %v", err)
	}
	defer unmountDisk(mountPoint)

	// Монтирование boot раздела
	if err := mountDisk(partitions["boot"], mountPointBoot); err != nil {
		return fmt.Errorf("ошибка монтирования boot раздела: %v", err)
	}
	defer unmountDisk(mountPointBoot)

	// Монтирование EFI раздела, если используется UEFI
	if typeBoot == "UEFI" {
		if err := mountDisk(partitions["efi"], efiMountPoint); err != nil {
			return fmt.Errorf("ошибка монтирования EFI раздела: %v", err)
		}
		defer unmountDisk(efiMountPoint)
	}

	// Получение UUID для разделов
	efiUUID := ""
	if typeBoot == "UEFI" {
		efiUUID = getUUID(partitions["efi"])
		if efiUUID == "" {
			return fmt.Errorf("не удалось получить UUID для EFI раздела %s", partitions["efi"])
		}
	}

	bootUUID := getUUID(partitions["boot"])
	if bootUUID == "" {
		return fmt.Errorf("не удалось получить UUID для boot раздела %s", partitions["boot"])
	}

	rootUUID := getUUID(partitions["root"])
	if rootUUID == "" {
		return fmt.Errorf("не удалось получить UUID для root раздела %s", partitions["root"])
	}

	// Получение текущего рабочего каталога
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Ошибка получения текущего рабочего каталога: %v", err)
	}

	// Команда для установки
	cmd := exec.Command("sudo", "podman", "run", "--rm", "--privileged", "--pid=host",
		"--security-opt", "label=type:unconfined_t",
		"-v", "/var/lib/containers:/var/lib/containers",
		"-v", "/dev:/dev",
		"-v", "/mnt/target:/mnt/target",
		"-v", fmt.Sprintf("%s:/output", currentDir),
		"--security-opt", "label=disable",
		image,
		"sh", "-c", fmt.Sprintf(
			"/output/src/ostree.sh && bootc install to-filesystem --skip-fetch-check --generic-image --disable-selinux "+
				"--root-mount-spec=UUID=%s --boot-mount-spec=UUID=%s %s",
			rootUUID, bootUUID, "/mnt/target",
		),
	)

	// Выполнение команды установки
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Выполняется установка...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка выполнения bootc: %v", err)
	}

	log.Println("Установка прошла успешно.")
	return nil
}

// getNamedPartitions возвращает мапу с именованными разделами в зависимости от типа загрузки
func getNamedPartitions(disk string, typeBoot string) (map[string]string, error) {
	partitions, err := getPartitions(disk)
	if err != nil {
		return nil, err
	}

	if typeBoot == "legacy" && len(partitions) < 4 {
		return nil, fmt.Errorf("недостаточно разделов на диске для режима legacy")
	} else if typeBoot == "UEFI" && len(partitions) < 3 {
		return nil, fmt.Errorf("недостаточно разделов на диске для режима UEFI")
	}

	namedPartitions := make(map[string]string)
	if typeBoot == "legacy" {
		namedPartitions["bios"] = partitions[0] // BIOS Boot Partition
		namedPartitions["efi"] = partitions[1]  // EFI Partition
		namedPartitions["boot"] = partitions[2] // Boot Partition
		namedPartitions["root"] = partitions[3] // Root Partition
	} else if typeBoot == "UEFI" {
		namedPartitions["efi"] = partitions[0]  // EFI Partition
		namedPartitions["boot"] = partitions[1] // Boot Partition
		namedPartitions["root"] = partitions[2] // Root Partition
	}

	return namedPartitions, nil
}

// getPartitionNames возвращает список всех разделов на указанном диске
func getPartitions(disk string) ([]string, error) {
	cmd := exec.Command("lsblk", "-ln", "-o", "NAME,TYPE", disk)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения lsblk: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var partitions []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "part" { // Проверяем, что это раздел
			partitions = append(partitions, "/dev/"+fields[0])
		}
	}

	return partitions, nil
}

// mountDisk монтирует указанный раздел в точку монтирования
func mountDisk(disk string, mountPoint string) error {
	fmt.Printf("Монтирование диска %s в %s...\n", disk, mountPoint)
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
