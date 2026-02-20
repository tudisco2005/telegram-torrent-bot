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

# check if transmission-remote is available
if ! command -v transmission-remote &> /dev/null; then
    echo "Error: transmission-remote is not installed. Please install it and try again."
    exit 1
fi

# Esegui il bot con tutti gli argomenti passati
# Se non vengono passati argomenti, il bot leggerà i valori dal file .env
chmod +x ../bin/telegram-torrent-bot
exec ../bin/telegram-torrent-bot "$@"