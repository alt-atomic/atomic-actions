package installer

import (
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

	// Шаг 3: Выбор файловой системы
	typeFileSystem := RunFilesystemStep()
	if typeFileSystem == "" {
		log.Println("Файловая система не выбрана.")
		return
	}

	// Шаг 4: Выбор типа загрузки
	typeBoot := RunBootModeStep()
	if typeBoot == "" {
		log.Println("Boot режим не выбран.")
		return
	}

	// Шаг 3: Уничтожение данных и создание разметки
	if err := prepareDisk(diskResult, typeFileSystem, typeBoot); err != nil {
		log.Fatalf("Ошибка подготовки диска: %v\n", err)
	}

	// Шаг 4: Установка с использованием bootc
	if err := installToFilesystem(imageResult, diskResult, typeBoot, typeFileSystem); err != nil {
		log.Fatalf("Ошибка установки: %v\n", err)
	}

	partitions, err := getNamedPartitions(diskResult, typeBoot)
	if err != nil {
		log.Fatalf("Ошибка получения именованных разделов: %v", err)
	}

	if err := cleanupTemporaryPartition(partitions, diskResult); err != nil {
		log.Fatalf("Ошибка очистки временного раздела: %v", err)
	}

	log.Println("Установка завершена успешно!")
}

func cleanupTemporaryPartition(partitions map[string]PartitionInfo, diskResult string) error {
	log.Println("Удаление временного раздела и расширение root-раздела...")

	// Размонтируем временный раздел
	log.Printf("Размонтирование временного раздела %s...\n", partitions["temp"].Path)
	if err := unmount("/var/lib/containers"); err != nil {
		return fmt.Errorf("ошибка размонтирования временного раздела: %v", err)
	}

	// Удаляем временный раздел
	log.Printf("Удаление временного раздела %s...\n", partitions["temp"].Path)
	cmd := exec.Command("parted", "-s", diskResult, "rm", partitions["temp"].Number)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка удаления временного раздела: %v", err)
	}

	// Расширяем root-раздел
	log.Printf("Расширение root-раздела %s до 100%%...\n", partitions["root"].Path)
	cmd = exec.Command("parted", "-s", diskResult, "resizepart", partitions["root"].Number, "100%")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка изменения размера root-раздела: %v", err)
	}

	// Проверяем тип файловой системы root-раздела
	log.Printf("Проверка типа файловой системы раздела %s...\n", partitions["root"].Path)
	cmd = exec.Command("blkid", "-o", "value", "-s", "TYPE", partitions["root"].Path)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ошибка проверки типа файловой системы: %v", err)
	}

	fsType := strings.TrimSpace(string(output))
	log.Printf("Тип файловой системы: %s\n", fsType)

	if fsType == "btrfs" {
		// Для btrfs используем btrfs filesystem resize
		mountPoint := "/mnt/btrfs-root"
		log.Printf("Изменение размера файловой системы btrfs на разделе %s...\n", partitions["root"].Path)

		// Монтируем раздел
		if err := mountDisk(partitions["root"].Path, mountPoint, ""); err != nil {
			return fmt.Errorf("ошибка монтирования btrfs-раздела: %v", err)
		}
		defer unmountDisk(mountPoint) // Размонтируем после завершения

		// Выполняем resize на точке монтирования
		cmd = exec.Command("btrfs", "filesystem", "resize", "max", mountPoint)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка изменения размера файловой системы btrfs: %v", err)
		}
	} else if fsType == "ext4" {
		// Для ext4 используем resize2fs
		cmd = exec.Command("e2fsck", "-f", "-y", partitions["root"].Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Printf("Проверка и исправление файловой системы ext4 на разделе %s...\n", partitions["root"].Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка проверки файловой системы ext4: %v", err)
		}

		log.Printf("Изменение размера файловой системы ext4 на разделе %s...\n", partitions["root"].Path)
		cmd = exec.Command("resize2fs", partitions["root"].Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка изменения размера файловой системы ext4: %v", err)
		}
	} else {
		return fmt.Errorf("неподдерживаемая файловая система: %s", fsType)
	}

	log.Println("Временный раздел удалён, root-раздел расширен.")
	return nil
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
		"podman",
		"rsync",
		"wipefs",
		"parted",
		"mkfs.fat",
		"mkfs.ext4",
		"mount",
		"umount",
		"blkid",
		"lsblk",
	}
	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("команда %s не найдена в PATH", cmd)
		}
	}
	return nil
}

// isMounted проверяет, примонтирован ли путь
func isMounted(path string) bool {
	cmd := exec.Command("mountpoint", "-q", path)
	err := cmd.Run()
	return err == nil
}

// validateDisk проверяет существование диска
func validateDisk(disk string) bool {
	if _, err := os.Stat(disk); os.IsNotExist(err) {
		return false
	}
	return true
}

// unmount размонтирует путь, если он примонтирован
func unmount(path string) error {
	if isMounted(path) {
		log.Printf("Размонтирование %s...\n", path)
		cmd := exec.Command("umount", path)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка размонтирования %s: %v", path, err)
		}
		log.Printf("%s успешно размонтирован.\n", path)
	}
	return nil
}

// prepareDisk выполняет подготовку диска
func prepareDisk(disk string, rootFileSystem string, typeBoot string) error {
	paths := []string{"/mnt/target/boot/efi", "/mnt/target/boot", "/var/lib/containers", "/mnt/target"}

	for _, path := range paths {
		_ = unmount(path)
	}

	log.Printf("Подготовка диска %s с файловой системой %s в режиме %s\n", disk, rootFileSystem, typeBoot)

	// Команды для разметки
	var commands [][]string

	if typeBoot == "LEGACY" {
		commands = [][]string{
			{"wipefs", "--all", disk},
			{"parted", "-s", disk, "mklabel", "gpt"},
			{"parted", "-s", disk, "mkpart", "primary", "1MiB", "3MiB"},                        // BIOS Boot Partition (2 МиБ)
			{"parted", "-s", disk, "set", "1", "bios_grub", "on"},                              // BIOS Boot Partition
			{"parted", "-s", disk, "mkpart", "primary", "fat32", "3MiB", "1003MiB"},            // EFI раздел (1 ГБ)
			{"parted", "-s", disk, "set", "2", "boot", "on"},                                   // EFI раздел
			{"parted", "-s", disk, "mkpart", "primary", "ext4", "1003MiB", "3003MiB"},          // Boot раздел (2 ГБ)
			{"parted", "-s", disk, "mkpart", "primary", rootFileSystem, "3003MiB", "20000MiB"}, // Root раздел
			{"parted", "-s", disk, "mkpart", "primary", "ext4", "20000MiB", "50000MiB"},        // Временный раздел
		}
	} else if typeBoot == "UEFI" {
		commands = [][]string{
			{"wipefs", "--all", disk},
			{"parted", "-s", disk, "mklabel", "gpt"},
			{"parted", "-s", disk, "mkpart", "primary", "fat32", "1MiB", "601MiB"},             // EFI раздел (600 МБ)
			{"parted", "-s", disk, "set", "1", "boot", "on"},                                   // EFI раздел
			{"parted", "-s", disk, "mkpart", "primary", "ext4", "601MiB", "2601MiB"},           // Boot раздел (2 ГБ)
			{"parted", "-s", disk, "mkpart", "primary", rootFileSystem, "2601MiB", "20000MiB"}, // Root раздел
			{"parted", "-s", disk, "mkpart", "primary", "ext4", "20000MiB", "50000MiB"},        // Временный раздел
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
		{"mkfs.vfat", []string{"-F32", partitions["efi"].Path}}, // Форматирование EFI раздела
		{"mkfs.ext4", []string{partitions["boot"].Path}},        // Форматирование boot раздела
	}

	if rootFileSystem == "ext4" {
		formats = append(formats, struct {
			cmd  string
			args []string
		}{"mkfs.ext4", []string{partitions["root"].Path}})

	} else if rootFileSystem == "btrfs" {
		formats = append(formats, struct {
			cmd  string
			args []string
		}{"mkfs.btrfs", []string{"-f", partitions["root"].Path}})
	} else {
		return fmt.Errorf("неизвестная файловая система: %s", rootFileSystem)
	}

	formats = append(formats, struct {
		cmd  string
		args []string
	}{"mkfs.ext4", []string{partitions["temp"].Path}})

	for _, format := range formats {
		cmd := exec.Command(format.cmd, format.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка форматирования %s: %v", format.args[0], err)
		}
	}

	if rootFileSystem == "btrfs" {
		if err := createBtrfsSubVolumes(partitions["root"].Path); err != nil {
			return fmt.Errorf("ошибка создания подтомов Btrfs: %v", err)
		}
	}

	// Создание временного раздела
	tempCommands := [][]string{
		{"mkdir", "-p", "/var/lib/containers"},
		{"mount", partitions["temp"].Path, "/var/lib/containers"},
	}

	for _, args := range tempCommands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка выполнения команды %s: %v", args[0], err)
		}
	}
	log.Printf("Диск %s успешно подготовлен.\n", disk)

	return nil
}

func createBtrfsSubVolumes(rootPartition string) error {
	mountPoint := "/mnt/btrfs-setup"
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}
	defer os.RemoveAll(mountPoint)

	if err := mountDisk(rootPartition, mountPoint, "rw,subvol=/"); err != nil {
		return fmt.Errorf("ошибка монтирования Btrfs раздела: %v", err)
	}
	defer unmountDisk(mountPoint)

	subVolumes := []string{"@", "@home", "@var"}
	for _, subVol := range subVolumes {
		subVolPath := fmt.Sprintf("%s/%s", mountPoint, subVol)
		if _, err := os.Stat(subVolPath); os.IsNotExist(err) {
			cmd := exec.Command("btrfs", "subvolume", "create", subVolPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ошибка создания подтома %s: %v", subVol, err)
			}
		} else {
			log.Printf("Подтом %s уже существует, пропуск.", subVol)
		}
	}

	return nil
}

// installToFilesystem выполняет установку с использованием bootc
func installToFilesystem(image string, disk string, typeBoot string, rootFileSystem string) error {
	mountPoint := "/mnt/target"
	mountBtrfsVar := "/mnt/btrfs/var"
	mountBtrfsHome := "/mnt/btrfs/home"
	mountPointBoot := "/mnt/target/boot"
	efiMountPoint := "/mnt/target/boot/efi"
	var installCmd string

	// Получаем именованные разделы
	partitions, err := getNamedPartitions(disk, typeBoot)
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	// Монтируем разделы
	if rootFileSystem == "btrfs" {
		if err := mountDisk(partitions["root"].Path, mountPoint, "subvol=@"); err != nil {
			return fmt.Errorf("ошибка монтирования корневого подтома: %v", err)
		}
	} else {
		if err := mountDisk(partitions["root"].Path, mountPoint, ""); err != nil {
			return fmt.Errorf("ошибка монтирования root раздела: %v", err)
		}
	}

	if err := mountDisk(partitions["boot"].Path, mountPointBoot, ""); err != nil {
		return fmt.Errorf("ошибка монтирования boot раздела: %v", err)
	}

	if err := mountDisk(partitions["efi"].Path, efiMountPoint, ""); err != nil {
		return fmt.Errorf("ошибка монтирования EFI раздела: %v", err)
	}

	// Выполняем установку с использованием bootc
	if typeBoot == "UEFI" {
		installCmd = fmt.Sprintf(
			"[ -f /usr/libexec/init-ostree.sh ] && /usr/libexec/init-ostree.sh; bootc install to-filesystem --skip-fetch-check --disable-selinux %s",
			"/mnt/target",
		)
	} else {
		installCmd = fmt.Sprintf(
			"[ -f /usr/libexec/init-ostree.sh ] && /usr/libexec/init-ostree.sh; bootc install to-filesystem --skip-fetch-check --generic-image --disable-selinux %s",
			"/mnt/target",
		)
	}

	cmd := exec.Command("sudo", "podman", "run", "--rm", "--privileged", "--pid=host",
		"--security-opt", "label=type:unconfined_t",
		"-v", "/var/lib/containers:/var/lib/containers",
		"-v", "/dev:/dev",
		"-v", "/mnt/target:/mnt/target",
		"--security-opt", "label=disable",
		image,
		"sh", "-c", installCmd,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Выполняется установка...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка выполнения bootc: %v", err)
	}

	unmountDisk(efiMountPoint)
	unmountDisk(mountPointBoot)
	unmountDisk(mountPoint)

	if rootFileSystem == "btrfs" {
		if err := mountDisk(partitions["root"].Path, mountPoint, "rw,subvol=@"); err != nil {
			return fmt.Errorf("ошибка повторного монтирования корневого подтома: %v", err)
		}

		if err := mountDisk(partitions["root"].Path, mountBtrfsVar, "subvol=@var"); err != nil {
			return fmt.Errorf("ошибка монтирования подтома @var: %v", err)
		}

		if err := mountDisk(partitions["root"].Path, mountBtrfsHome, "subvol=@home"); err != nil {
			return fmt.Errorf("ошибка монтирования подтома @home: %v", err)
		}

		ostreeDeployPath, err := findOstreeDeployPath(mountPoint)
		if err != nil {
			return fmt.Errorf("ошибка поиска ostree deploy пути: %v", err)
		}

		// Копируем содержимое /var в подтом @var
		if err := copyWithRsync(fmt.Sprintf("%s/var/", ostreeDeployPath), mountBtrfsVar); err != nil {
			return fmt.Errorf("ошибка копирования /var в @var: %v", err)
		}

		// Копируем содержимое /home в подтом @home
		if err := copyWithRsync(fmt.Sprintf("%s/home/", ostreeDeployPath), mountBtrfsHome); err != nil {
			return fmt.Errorf("ошибка копирования /home в @home: %v", err)
		}

		// Очищаем содержимое /var, но оставляем папку
		if err := clearDirectory(fmt.Sprintf("%s/var", ostreeDeployPath)); err != nil {
			return fmt.Errorf("ошибка очистки содержимого /var: %v", err)
		}
	} else {
		if err := mountDisk(partitions["root"].Path, mountPoint, "rw"); err != nil {
			return fmt.Errorf("ошибка повторного монтирования root раздела: %v", err)
		}
	}

	if err := mountDisk(partitions["boot"].Path, mountPointBoot, "rw"); err != nil {
		return fmt.Errorf("ошибка повторного монтирования boot раздела: %v", err)
	}

	if err := mountDisk(partitions["efi"].Path, efiMountPoint, "rw"); err != nil {
		return fmt.Errorf("ошибка повторного монтирования EFI раздела: %v", err)
	}

	// Генерация fstab
	log.Println("Генерация fstab...")
	if err := generateFstab(mountPoint, partitions, rootFileSystem); err != nil {
		return fmt.Errorf("ошибка генерации fstab: %v", err)
	}

	unmountDisk(efiMountPoint)
	unmountDisk(mountPointBoot)
	unmountDisk(mountBtrfsVar)
	unmountDisk(mountBtrfsHome)
	unmountDisk(mountPoint)
	return nil
}

func clearDirectory(path string) error {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("ошибка чтения содержимого директории %s: %v", path, err)
	}

	for _, entry := range dirEntries {
		entryPath := fmt.Sprintf("%s/%s", path, entry.Name())

		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("ошибка удаления %s: %v", entryPath, err)
		}
	}

	return nil
}

// copyWithRsync копирование с использованием команды rsync
func copyWithRsync(src string, dst string) error {
	cmd := exec.Command("rsync", "-aHAXv", src, dst)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Копирование с использованием rsync: %s -> %s\n", src, dst)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка выполнения rsync: %v", err)
	}
	return nil
}

// находит путь к папке, заканчивающейся на .0
func findOstreeDeployPath(mountPoint string) (string, error) {
	deployPath := fmt.Sprintf("%s/ostree/deploy/default/deploy", mountPoint)
	entries, err := os.ReadDir(deployPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения директории %s: %v", deployPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".0") {
			return fmt.Sprintf("%s/%s", deployPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("не найдена папка, в %s", deployPath)
}

func generateFstab(mountPoint string, partitions map[string]PartitionInfo, rootFileSystem string) error {
	ostreeDeployPath, err := findOstreeDeployPath(mountPoint)
	if err != nil {
		return fmt.Errorf("ошибка поиска ostree deploy пути: %v", err)
	}
	fstabPath := fmt.Sprintf("%s/etc/fstab", ostreeDeployPath)

	log.Printf("Генерация %s...\n", fstabPath)

	fstabContent := "# Auto generate fstab from atomic-actions installer \n"

	if rootFileSystem == "btrfs" {
		fstabContent += fmt.Sprintf(
			"UUID=%s / btrfs subvol=@,compress=zstd:1,x-systemd.device-timeout=0 0 0\n",
			getUUID(partitions["root"].Path),
		)
		fstabContent += fmt.Sprintf(
			"UUID=%s /home btrfs subvol=@home,compress=zstd:1,x-systemd.device-timeout=0 0 0\n",
			getUUID(partitions["root"].Path),
		)
		fstabContent += fmt.Sprintf(
			"UUID=%s /var btrfs subvol=@var,compress=zstd:1,x-systemd.device-timeout=0 0 0\n",
			getUUID(partitions["root"].Path),
		)
	} else if rootFileSystem == "ext4" {
		fstabContent += fmt.Sprintf(
			"UUID=%s / ext4 defaults 1 1\n",
			getUUID(partitions["root"].Path),
		)
	} else {
		return fmt.Errorf("неизвестная файловая система: %s", rootFileSystem)
	}

	fstabContent += fmt.Sprintf(
		"UUID=%s /boot ext4 defaults 1 2\n",
		getUUID(partitions["boot"].Path),
	)
	fstabContent += fmt.Sprintf(
		"UUID=%s /boot/efi vfat umask=0077,shortname=winnt 0 2\n",
		getUUID(partitions["efi"].Path),
	)

	file, err := os.Create(fstabPath)
	if err != nil {
		return fmt.Errorf("ошибка создания %s: %v", fstabPath, err)
	}
	defer file.Close()

	_, err = file.WriteString(fstabContent)
	if err != nil {
		return fmt.Errorf("ошибка записи в %s: %v", fstabPath, err)
	}

	log.Printf("Файл %s успешно создан.\n", fstabPath)
	return nil
}

type PartitionInfo struct {
	Path   string
	Number string
}

func getNamedPartitions(disk string, typeBoot string) (map[string]PartitionInfo, error) {
	partitions, err := getPartitions(disk)
	if err != nil {
		return nil, err
	}

	fmt.Println("Список разделов:")
	for i, partition := range partitions {
		fmt.Printf("Раздел %d: %s\n", i+1, partition)
	}
	if typeBoot == "LEGACY" && len(partitions) < 4 {
		return nil, fmt.Errorf("недостаточно разделов на диске для режима LEGACY")
	} else if typeBoot == "UEFI" && len(partitions) < 3 {
		return nil, fmt.Errorf("недостаточно разделов на диске для режима UEFI")
	}

	// Карта с информацией о разделах
	namedPartitions := make(map[string]PartitionInfo)

	if typeBoot == "LEGACY" {
		namedPartitions["bios"] = PartitionInfo{Path: partitions[0], Number: "1"} // BIOS Boot Partition
		namedPartitions["efi"] = PartitionInfo{Path: partitions[1], Number: "2"}  // EFI Partition
		namedPartitions["boot"] = PartitionInfo{Path: partitions[2], Number: "3"} // Boot Partition
		namedPartitions["root"] = PartitionInfo{Path: partitions[3], Number: "4"} // Root Partition
		namedPartitions["temp"] = PartitionInfo{Path: partitions[4], Number: "5"} // Temporary Partition
	} else if typeBoot == "UEFI" {
		namedPartitions["efi"] = PartitionInfo{Path: partitions[0], Number: "1"}  // EFI Partition
		namedPartitions["boot"] = PartitionInfo{Path: partitions[1], Number: "2"} // Boot Partition
		namedPartitions["root"] = PartitionInfo{Path: partitions[2], Number: "3"} // Root Partition
		namedPartitions["temp"] = PartitionInfo{Path: partitions[3], Number: "4"} // Temporary Partition
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
func mountDisk(disk string, mountPoint string, options string) error {
	fmt.Printf("Монтирование диска %s в %s с опциями '%s'\n", disk, mountPoint, options)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}
	args := []string{}
	if options != "" {
		args = append(args, "-o", options)
	}
	args = append(args, disk, mountPoint)
	cmd := exec.Command("mount", args...)
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
