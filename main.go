package main

import (
	"atomic-actions/models/installer"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	logFile := setLogger()
	defer logFile.Close()

	installer.Run()
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

	teaLogWriter := io.MultiWriter(os.Stdout, logFile)
	teaLog := log.New(teaLogWriter, "debug ", log.LstdFlags)
	f, err := tea.LogToFileWith(logFilePath, "debug", teaLog)
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	//log.Printf("Логи записываются в файл: %s\n", logFilePath)

	return logFile
}
