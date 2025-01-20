#!/usr/bin/env bash

# Проверяем тип команды
command="${1:-}"
shift || true

case "$command" in
  update)
    echo "Running system update"
    ;;

  install)
    echo "Running install with arguments: $@"
    ;;

  remove)
    echo "Running remove with arguments: $@"
    ;;

  *)
    if [ -n "$command" ]; then
      err "Unsupported command: $command"
    else
      echo "Usage: $0 {update|install|remove} [arguments]"
    fi
    ;;
esac