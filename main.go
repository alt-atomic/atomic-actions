package main

import (
	"atomic-actions/models/installer"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	logFile := setLogger()
	defer logFile.Close()

	// Определяем флаги
	helpFlag := flag.Bool("h", false, "Показать список команд")
	flag.Parse()

	// Аргументы командной строки
	args := flag.Args()

	// Логика выполнения
	if *helpFlag {
		printHelp()
		return
	}

	if len(args) == 0 {
		printHelp()
		return
	}

	switch args[0] {
	case "install":
		fmt.Println("Выполняется установка Alt Atomic на диск...")
		installer.Run()
	default:
		fmt.Printf("Неизвестная команда: %s\n", args[0])
		printHelp()
	}
}

func printHelp() {
	fmt.Println("Доступные команды:")
	fmt.Println("  install  - Установка Alt Atomic на диск")
}

func setLogger() *os.File {
	tempDir := os.TempDir()
	logFilePath := filepath.Join(tempDir, "debug.log")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}

	// Настраиваем MultiWriter для записи в файл и консоль
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	return logFile
}
