#!/bin/sh

echo "--- Starting Entrypoint Script ---"

# Создаем все необходимые директории, используя переменные окружения
mkdir -p ${CONFIG_FILEPATH}

echo "--- Dirs created. Listing contents of VOLUME_PATH: ---"
ls -la ${VOLUME_PATH}

# Запускаем ваше Go-приложение
exec /bin/eventsBot "$@"
