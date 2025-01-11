package main

import (
	"atomic-actions/models/installer"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	installer.Run()
}
