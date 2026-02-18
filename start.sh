#!/bin/bash

# Script di avvio per transmission-telegram-bot
# Se i parametri non vengono passati dalla riga di comando, vengono letti dal file .env

# Vai alla directory src
cd "$(dirname "$0")/src" || exit 1

# Verifica se il binario esiste, altrimenti compila
if [ ! -f "../bin/telegram-torrent-bot" ]; then
    echo "Building the bot..."
    go build -o ../bin/telegram-torrent-bot || exit 1
fi

chmod 777 ./move.sh

# check if transmission-remote is available
if ! command -v transmission-remote &> /dev/null; then
    echo "Error: transmission-remote is not installed. Please install it and try again."
    exit 1
fi

# check if in trasmission config move script is set, if not set warning 
if ! sudo grep -q '"script-torrent-done-enabled": true' /var/lib/transmission-daemon/info/settings.json; then
    echo "Warning: Failed to configure transmission-remote, Downloads will remain in trasmission default folder."

    echo "To fix this, stop transmission and add the following lines to your transmission settings.json:"
    echo '    "script-torrent-done-enabled": true,'
    echo '    "script-torrent-done-filename": "<ABSOLUTE_PATH_TO_MOVE_SCRIPT>",'
    echo "Then restart transmission and run this script again."
fi

# Esegui il bot con tutti gli argomenti passati
# Se non vengono passati argomenti, il bot leggerà i valori dal file .env
chmod +x ../bin/telegram-torrent-bot
exec ../bin/telegram-torrent-bot "$@"