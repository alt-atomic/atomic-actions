#!/bin/bash
set -e

echo "Checking ostree commit"

# Проверка существования репозитория
if [ -d "/sysroot/ostree/repo" ]; then
  if ostree --repo=/sysroot/ostree/repo refs 2>/dev/null | grep -q .; then
    echo "OSTree refs exist. Skipping commit creation."
    exit 0
  fi
fi

echo "Создание ostree репозитория, пожалуйста ожидайте"

## Инициализация OSTree репозитория
mkdir -p /sysroot/ostree/repo
ostree --repo=/sysroot/ostree/repo init --mode=archive

## Подготовка временной директории
mkdir -p /tmp/rootfscopy

rsync -aA \
  --exclude=/home \
  --exclude=/dev \
  --exclude=/proc \
  --exclude=/sys \
  --exclude=/run \
  --exclude=/boot \
  --exclude=/tmp \
  --exclude=/etc \
  --exclude=/var \
  --exclude=/output \
  / /tmp/rootfscopy/

mkdir -p /tmp/rootfscopy/var/tmp

ostree --repo=/sysroot/ostree/repo commit \
  --branch=alt/atomic \
  --subject "Initial ALT Atomic Commit" \
  --tree=dir=/tmp/rootfscopy

## Очистка временной директории
rm -rf /tmp/rootfscopy

echo "Initial OSTree commit created."