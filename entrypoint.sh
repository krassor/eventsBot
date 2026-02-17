#!/bin/sh

echo "--- Starting Entrypoint Script ---"

mkdir -p ${CONFIG_FILEPATH}

echo "--- Dirs created. Listing contents of VOLUME_PATH: ---"
ls -la ${VOLUME_PATH}

exec /bin/eventsBot "$@"
