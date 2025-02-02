#!/usr/bin/env bash

# Версия Conty
CONTY_VERSION="1.27.1"

# Ссылка на релиз Conty (lite-вариант)
CONTY_URL="https://github.com/Kron4ek/Conty/releases/download/${CONTY_VERSION}/conty_lite.sh"

# Куда устанавливаем
INSTALL_DIR="${HOME}/.local/share/conty"

# Полный путь
CONTY_SCRIPT="${INSTALL_DIR}/conty_lite.sh"

# Папка, куда Conty будет складывать .desktop-файлы
DESKTOP_DIR="${HOME}/.local/share/applications/Conty"

# Завершаем работу при ошибках
set -e

case "$1" in
  enable)
    echo "Включение игрового режима"
    echo "Установка Conty версии ${CONTY_VERSION}..."

    # Создаём директорию, если её нет
    mkdir -p "${INSTALL_DIR}"

    # Проверяем, не скачан ли уже conty_lite.sh
    if [ -f "${CONTY_SCRIPT}" ]; then
      echo "Файл '${CONTY_SCRIPT}' уже существует. Пропускаю скачивание."
    else
      echo "Скачиваю ${CONTY_URL} в ${INSTALL_DIR}..."
      curl -L "${CONTY_URL}" -o "${CONTY_SCRIPT}"
      chmod +x "${CONTY_SCRIPT}"
    fi

    echo "Запускаю Conty (генерируем .desktop-файлы)..."
    "${CONTY_SCRIPT}" -d

    echo "Очистка лишних .desktop-файлов из '${DESKTOP_DIR}'..."
    # Список файлов, которые хотим оставить
    FILES_TO_KEEP=(
      "com.github.tkashkin.gamehub-conty.desktop"
      "com.usebottles.bottles-conty.desktop"
      "duckstation-conty.desktop"
      "net.lutris.Lutris-conty.desktop"
      "org.libretro.RetroArch-conty.desktop"
      "pcsx2-conty.desktop"
      "playonlinux4-conty.desktop"
      "steam-conty.desktop"
      "steam-native-conty.desktop"
      "wine-conty.desktop"
    )

    if [ -d "${DESKTOP_DIR}" ]; then
      for desktop_file in "${DESKTOP_DIR}"/*.desktop; do
        base_name="$(basename "$desktop_file")"
        keep_file="no"
        for keep in "${FILES_TO_KEEP[@]}"; do
          if [[ "$base_name" == "$keep" ]]; then
            keep_file="yes"
            break
          fi
        done
        if [[ "$keep_file" == "no" ]]; then
          rm -f "$desktop_file"
        fi
      done
    fi

    echo "Игровой режим включен. Все связанные файлы установлены."
    ;;

  disable)
    echo "Отключение игрового режима"

    if [ -f "${CONTY_SCRIPT}" ]; then
      rm -f "${CONTY_SCRIPT}"
      echo "Удалён файл '${CONTY_SCRIPT}'."
    fi

    if [ -d "${DESKTOP_DIR}" ]; then
      rm -rf "${DESKTOP_DIR}"
      echo "Удалена папка '${DESKTOP_DIR}'."
    fi

    echo "Игровой режим отключен. Все связанные файлы удалены."
    ;;

  *)
    echo "Использование: $0 {enable|disable}"
    exit 1
    ;;
esac