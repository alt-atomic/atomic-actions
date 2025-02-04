#!/usr/bin/env bash

# Версия Conty
CONTY_VERSION="1.27.1"

# Ссылка на релиз Conty (lite-вариант)
CONTY_URL="https://github.com/Kron4ek/Conty/releases/download/$CONTY_VERSION/conty_lite.sh"

LOCAL_DATADIR="$HOME/.local/share"

CONTY_DATADIR="$LOCAL_DATADIR/Conty"
CONTY_FILES="$CONTY_DATADIR/files"
CONTY_SCRIPT="${CONTY_DATADIR}/conty_lite"

# Папка, куда Conty будет складывать иконки для .desktop-файлов
ICONS_USER_DIR="${LOCAL_DATADIR}/icons"
PIXMAPS_USER_DIR="${LOCAL_DATADIR}/pixmaps/Conty"

# Папка .desktop файлов Conty
DESKTOP_USER_DIR="${LOCAL_DATADIR}/applications"

CUR_TEMP_DIR=$(mktemp -d)
CONTY_MNT_POINT="$CUR_TEMP_DIR/conty_mnt"

APP_FOLDER_ID="$Atomic-Game-Mode-Folder"
APP_FOLDER_NAME="Atomic Games"

# Завершаем работу при ошибках
set -e

case "$1" in
  enable)
    mkdir -p "${CONTY_DATADIR}"
    mkdir -p "$CONTY_MNT_POINT"

    conty_files=()
    desktop_files=()

    # Проверяем, не скачан ли уже conty_lite.sh
    if [ -f "${CONTY_SCRIPT}" ]; then
      echo "Conty file exists. Skip downloading"
    else
      echo "Download ${CONTY_URL} to ${CONTY_DATADIR}..."
      curl -L "${CONTY_URL}" -o "${CONTY_SCRIPT}"
      chmod +x "${CONTY_SCRIPT}"
    fi

    export BASE_DIR="$CUR_TEMP_DIR"
    export CUSTOM_MNT="$CONTY_MNT_POINT"

    mount_output=$("${CONTY_SCRIPT}" -m 2>&1)
    if [[ "$mount_output" == "The image has been unmounted" ]]; then
      mount_output=$("${CONTY_SCRIPT}" -m 2>&1)
    fi

    DESKTOP_CONTY_DIR="${CONTY_MNT_POINT}/usr/share/applications"
    ICONS_CONTY_DIR="${CONTY_MNT_POINT}/usr/share/icons"
    PIXMAPS_CONTY_DIR="${CONTY_MNT_POINT}/usr/share/pixmaps"

    mkdir -p "$DESKTOP_USER_DIR"
    mkdir -p "$ICONS_USER_DIR"
    mkdir -p "$PIXMAPS_USER_DIR"

    # Укажем список файлов, которые хотим оставить
    FILES_TO_KEEP=(
      "com.github.tkashkin.gamehub.desktop"
      "com.usebottles.bottles.desktop"
      "duckstation.desktop"
      "faugus-launcher.desktop"
      "org.gnome.Zenity.desktop"
      "net.lutris.Lutris.desktop"
      "org.libretro.RetroArch.desktop"
      "pcsx2.desktop"
      "steam-native.desktop"
      "steamtinkerlaunch.desktop"
      "wine.desktop"
    )

     # Перебираем все сгенерированные .desktop-файлы
    if [ -d "$DESKTOP_CONTY_DIR" ]; then
      for desktop_file in "${DESKTOP_CONTY_DIR}"/*.desktop; do
        desktop_base_name="$(basename "$desktop_file")"

        # Проверяем, входит ли файл в список нужных
        keep_file="no"
        for keep in "${FILES_TO_KEEP[@]}"; do
          if [[ "$desktop_base_name" == "$keep" ]]; then
            keep_file="yes"
            break
          fi
        done

        # Если не совпало ни с одним нужным — удаляем
        if [[ "$keep_file" == "yes" ]]; then
          tmp_desktop_path="$CUR_TEMP_DIR/$desktop_base_name"
          conty_desktop_file="$DESKTOP_USER_DIR/$(echo "$desktop_base_name" | sed 's/\.desktop$/-conty.desktop/')"
          cp "$desktop_file" "$tmp_desktop_path"

          icon_name=$(grep -E '^Icon=' "$desktop_file" | cut -d'=' -f2)
          if [[ -n "$icon_name" ]]; then
            icon_paths=($(find "$ICONS_CONTY_DIR" -type f \( -name "$icon_name.png" -o -name "$icon_name.svg" \)))
            pixmaps_path=$(find "$PIXMAPS_CONTY_DIR" -type f \( -name "$icon_name.png" -o -name "$icon_name.svg" \) | head -n 1)  # Can be only one pixmap icon

            local_pixmap_path=$(echo "$pixmaps_path" | sed "s|^$PIXMAPS_CONTY_DIR|$PIXMAPS_USER_DIR|")

            if [[ -n "$icon_paths" ]]; then
              for icon_path in "${icon_paths[@]}"; do
                local_icon_path=$(echo "$icon_path" | sed "s|^$ICONS_CONTY_DIR|$ICONS_USER_DIR|")
                mkdir -p $(dirname "$local_icon_path")
                cp "$icon_path" "$local_icon_path"
                conty_files+=("$local_icon_path")
                #echo "Icon copyed: $icon_path > $local_icon_path"
              done
            else
              if [[ -n "$pixmaps_path" ]]; then
                mkdir -p $(dirname "$local_pixmap_path")
                cp "$pixmaps_path" "$local_pixmap_path"
                #echo "Pixmap icon copyed: $pixmaps_path > $local_pixmap_path"
                sed -i "s|^Icon[[:space:]]*=[[:space:]]*.*|Icon=$local_pixmap_path|" "$tmp_desktop_path"
                conty_files+=("$local_pixmap_path")
              fi
            fi
            sed -i "s|^Exec=\(.*\)$|Exec=\"$CONTY_SCRIPT\" \1|" "$tmp_desktop_path"
          fi
          mv "$tmp_desktop_path" "$conty_desktop_file"
          conty_files+=("$conty_desktop_file")
          desktop_files+=("$conty_desktop_file")
          echo "Desktop exported: $conty_desktop_file"
        fi
      done
    fi

    printf "%s\n" "${conty_files[@]}" > "$CONTY_FILES"

    gtk4-update-icon-cache -f -t "$ICONS_USER_DIR/hicolor" &> /dev/null

    "${CONTY_SCRIPT}" -m &> /dev/null

    echo "Gamemode enabled"
    ;;

  disable)

    if [ -f "$CONTY_FILES" ]; then
      while IFS= read -r file; do
        rm -f -- "$file"
      done < "$CONTY_FILES"
    fi

    rm -f "$CONTY_FILES"

    echo "Gamemode disabled."
    ;;

  *)
    echo "Usage: $0 {enable|disable}"
    exit 1
    ;;
esac
