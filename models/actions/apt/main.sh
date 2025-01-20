#!/bin/bash

# Проверка и активация наложенной файловой системы
run-bootc-usr-overlay() {
  local run_overlay="true"
  while read -r device mountpoint fstype options dump pass; do
    if [[ "$device" == "overlay" && "$mountpoint" == "/usr" ]]; then
      run_overlay="false"
      break
    fi
  done < /proc/mounts

  if $run_overlay; then
    echo "Activating usr-overlay..."
    bootc usr-overlay || err "Failed to activate usr-overlay"
  else
    echo "Overlay already active."
  fi
}

# Проверка состояния текущего образа
validate_and_create_containerfile() {
  local containerfile="/var/Containerfile"

  if [ -f "$containerfile" ]; then
    echo "Containerfile already exists. Skipping creation."
    return
  fi

  echo "Checking current staged image..."
  local staged_image
  staged_image=$(sudo bootc status | yq '.status.booted.image.image.image')

  if [[ -z "$staged_image" ]]; then
    err "Unable to determine the current staged image."
  fi

  if [[ "$staged_image" == containers-storage:* ]]; then
    echo "Staged image is using containers-storage. Skipping Containerfile creation."
    return
  fi

  echo "Creating default Containerfile..."
  cat <<EOF > "$containerfile"
FROM $staged_image
RUN apt-get update
EOF

  echo "Containerfile created at $containerfile with staged image: $staged_image"
}

# Переключение на новый образ
bootc-switch() {
  local podman_image_id
  podman_image_id="$(podman images -q os)"

  if [ -z "$podman_image_id" ]; then
    err "No valid image found with tag 'os'. Build the image first."
  fi

  bootc switch --transport containers-storage "$podman_image_id" || err "Failed to switch to the new image."
}

# Логирование ошибок
err() {
  echo "Error: $1" >&2
  exit 1
}

# Проверка обновлений базового образа
check_and_update_base_image() {
  local base_image="ghcr.io/skywar-design/alt-atomic:source"
  local local_image_id
  local remote_image_id

  echo "Checking for updates to the base image: $base_image..."

  # Получаем ID локального образа
  local_image_id=$(podman images --noheading --format "{{.ID}}" "$base_image" 2>/dev/null || echo "")

  if [ -z "$local_image_id" ]; then
    err "Local base image not found: $base_image. Please pull it first."
  fi

  # Получаем ID удалённого образа
  echo "Pulling the latest version of the base image..."
  remote_image_id=$(podman pull "$base_image")

  if [ "$local_image_id" != "$remote_image_id" ]; then
    echo "Base image has been updated. New image ID: $remote_image_id"
    return 0 # Указывает, что обновление есть
  else
    echo "Base image is up-to-date."
    return 1 # Указывает, что обновлений нет
  fi
}

# Перестройка и переключение системы на новый образ
rebuild_and_switch() {
  echo "Rebuilding the system image..."
  podman build --squash -t os /var || err "Failed to rebuild the image."

  echo "Switching to the updated image..."
  bootc-switch

  echo "Cleaning up old Podman images..."
  prune_old_images
}

# Удаление старых образов Podman
prune_old_images() {
  podman image prune -f || echo "Failed to prune images."
  podman images --noheading | awk '$1 == "<none>" { print $3 }' | xargs -r podman rmi -f
}

# Основная программа
set -euo pipefail

if [ "$EUID" -ne 0 ]; then
  err "This command requires root privileges."
fi

run-bootc-usr-overlay

validate_and_create_containerfile

# Проверяем тип команды
command="${1:-}"
shift || true

case "$command" in
  update)
    echo "Running system update for the container image..."
    if check_and_update_base_image; then
      rebuild_and_switch
    else
      echo "No updates found for the base image. Nothing to do."
    fi
    ;;

  install)
    echo "Running apt-get install with arguments: $@"
    for pkg in "$@"; do
      if is_package_installed "$pkg"; then
        err "Package '$pkg' is already installed. Aborting."
      fi
    done

    apt-get install -y "$@" || err "apt-get install command failed."

    # Обновление Containerfile
    echo "RUN apt-get install -y $@" >> /var/Containerfile
    echo "Updated /var/Containerfile with: RUN apt-get install -y $@"

    # Перестройка и переключение
    rebuild_and_switch
    ;;

  remove)
    echo "Running apt-get remove with arguments: $@"
    apt-get remove -y "$@" || err "apt-get remove command failed."

    # Удаление соответствующих строк из Containerfile
    for pkg in "$@"; do
      sed -i "/RUN apt-get install.*\b$pkg\b/d" /var/Containerfile
      echo "Removed RUN apt-get install line for package: $pkg"
    done

    # Перестройка и переключение
    rebuild_and_switch
    ;;

  *)
    if [ -n "$command" ]; then
      err "Unsupported command: $command"
    else
      echo "Usage: $0 {update|install|remove} [arguments]"
    fi
    ;;
esac