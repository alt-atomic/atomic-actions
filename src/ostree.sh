#!/bin/bash
set -e

echo "Running ostree.sh"

# Проверка `ostree refs`
if ostree --repo=/sysroot/ostree/repo refs | grep -q .; then
  echo "OSTree refs exist. Skipping commit creation."
else
  echo "No OSTree refs found. Creating initial commit."

  ## Инициализация OSTree репозитория
  mkdir -p /sysroot/ostree/repo
  ostree --repo=/sysroot/ostree/repo init --mode=archive

  ## Подготовка временной директории
  mkdir -p /tmp/rootfscopy

  rsync -aA \
    --exclude=/home \
    --exclude=/var/root \
    --exclude=/var/home \
    --exclude=/dev \
    --exclude=/proc \
    --exclude=/sys \
    --exclude=/run \
    --exclude=/boot \
    --exclude=/var/log \
    --exclude=/tmp \
    --exclude=/var/tmp \
    --exclude=/var/lib/containers \
    --exclude=/var/lib/openvpn/dev \
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
fi

echo "End ostree.sh"