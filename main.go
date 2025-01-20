package main

import (
	"atomic-actions/models/installer"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	systemActionsPath = "/usr/local/share/atomic-actions/actions"
	userActionsPath   = ".local/share/atomic-actions/actions"
)

type ActionSettings struct {
	Commands    []string `json:"commands"`
	Description string   `json:"description"`
	Sudo        bool     `json:"sudo"`
}

func main() {
	logFile := setLogger()
	defer logFile.Close()

	// Определяем флаги
	helpFlag := flag.Bool("h", false, "Показать список команд")
	flag.Parse()

	// Аргументы командной строки
	args := flag.Args()

	// Пути к директориям с actions
	systemPath := systemActionsPath
	userPath := filepath.Join(os.Getenv("HOME"), userActionsPath)

	// Загружаем команды из обоих путей
	commands := mergeCommands(
		loadActionsWithDescriptions(systemPath),
		loadActionsWithDescriptions(userPath),
	)

	// Добавляем команду installer вручную
	commands["install-system"] = Command{
		Description: "Установка Alt Atomic на диск \nВнимание! Блочное устройство не должно быть смонтировано в системе.",
		Handler: func(args []string) {
			installer.RunInstaller()
		},
	}

	if *helpFlag || len(args) == 0 {
		printHelp(commands)
		return
	}

	// Обрабатываем аргументы: объединяем первую часть команды и подкоманду
	if len(args) > 1 {
		args[0] = fmt.Sprintf("%s %s", args[0], args[1])
		args = append(args[:1], args[2:]...) // Убираем подкоманду из списка аргументов
	}

	// Ищем команду в карте
	if command, exists := commands[args[0]]; exists {
		command.Handler(args[1:])
	} else {
		fmt.Printf("Неизвестная команда: %s\n", args[0])
		printHelp(commands)
	}
}

// mergeCommands объединяет команды из двух карт
func mergeCommands(cmd1, cmd2 map[string]Command) map[string]Command {
	merged := make(map[string]Command)

	// Добавляем команды из первой карты
	for k, v := range cmd1 {
		merged[k] = v
	}

	// Добавляем команды из второй карты
	for k, v := range cmd2 {
		if _, exists := merged[k]; !exists {
			merged[k] = v
		} else {
			log.Printf("Предупреждение: Команда %s дублируется. Используется версия из первой карты.\n", k)
		}
	}

	return merged
}

type Command struct {
	Description string
	Handler     func(args []string)
}

func printHelp(commands map[string]Command) {
	// Группируем команды
	groupedCommands := make(map[string][]string)
	descriptions := make(map[string]string)

	for cmd, details := range commands {
		parts := strings.SplitN(cmd, " ", 2)
		if len(parts) > 1 {
			groupedCommands[parts[0]] = append(groupedCommands[parts[0]], parts[1])
			descriptions[parts[0]] = details.Description
		} else {
			groupedCommands[cmd] = []string{}
			descriptions[cmd] = details.Description
		}
	}

	// Сортируем группы команд по ключу (алфавитно)
	keys := make([]string, 0, len(groupedCommands))
	for key := range groupedCommands {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Создаем стиль с отступами для ячеек
	cellStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2)

	// Создаем таблицу
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers(cellStyle.Render("Команда"), cellStyle.Render("Описание"))

	// Добавляем строки в таблицу
	for _, group := range keys {
		cmds := groupedCommands[group]
		sort.Strings(cmds) // Сортируем подкоманды внутри группы
		commandList := fmt.Sprintf("atomic-actions %s", group)
		if len(cmds) > 0 {
			commandList = fmt.Sprintf("atomic-actions %s %s", group, strings.Join(cmds, ", "))
		}
		t.Row(
			cellStyle.Render(commandList),
			cellStyle.Render(descriptions[group]),
		)
	}

	fmt.Println(t)
}

func setLogger() *os.File {
	tempDir := os.TempDir()
	logFilePath := filepath.Join(tempDir, "debug.log")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Println("fatal:", err)
		//os.Exit(1)
	}

	// Настраиваем MultiWriter для записи в файл и консоль
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	return logFile
}

func loadActionsWithDescriptions(basePath string) map[string]Command {
	commands := make(map[string]Command)

	_ = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {

			settingsPath := filepath.Join(path, "settings.json")
			if _, err := os.Stat(settingsPath); err == nil {
				settings, err := parseSettings(settingsPath)
				if err != nil {
					log.Printf("Ошибка чтения settings.json в %s: %v\n", path, err)
					return nil
				}

				dirName := filepath.Base(path)
				if len(settings.Commands) > 0 {
					for _, cmd := range settings.Commands {
						commandName := fmt.Sprintf("%s %s", dirName, cmd)
						commands[commandName] = Command{
							Description: settings.Description,
							Handler:     generateActionHandler(dirName, cmd, settings.Sudo),
						}
					}
				} else {
					commands[dirName] = Command{
						Description: settings.Description,
						Handler:     generateActionHandler(dirName, "", settings.Sudo),
					}
				}
			}
		}
		return nil
	})

	return commands
}

func parseSettings(settingsPath string) (*ActionSettings, error) {
	data, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings ActionSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func generateActionHandler(action, command string, sudo bool) func(args []string) {
	return func(args []string) {
		scriptPaths := []string{
			filepath.Join(systemActionsPath, action, "main.sh"),
			filepath.Join(os.Getenv("HOME"), userActionsPath, action, "main.sh"),
		}

		var scriptPath string
		for _, path := range scriptPaths {
			if _, err := os.Stat(path); err == nil {
				scriptPath = path
				break
			}
		}

		if scriptPath == "" {
			log.Printf("Скрипт для действия %s не найден в доступных директориях\n", action)
			return
		}

		// Формируем команду для запуска
		cmdArgs := append([]string{scriptPath, command}, args...)
		if sudo {
			cmdArgs = append([]string{"sudo"}, cmdArgs...)
		}

		fmt.Printf("Выполняется: %s\n", strings.Join(cmdArgs, " "))

		// Запускаем процесс
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Ошибка выполнения скрипта: %v\n", err)
		}
	}
}
