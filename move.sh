#!/bin/bash

# --- CONFIGURAZIONE ---
# IMPORTANTE: NON mettere spazi prima o dopo l'uguale!
DESTINAZIONE="/home/dipi/MyTorrentBot/bot/data/downloads"
LOGFILE="/home/dipi/MyTorrentBot/bot/data/transmission-move.log"

# --- LOGICA ---

# Crea la cartella log/destinazione se non esiste
if [ ! -d "$DESTINAZIONE" ]; then
    mkdir -p "$DESTINAZIONE"
    chmod 777 "$DESTINAZIONE"
fi

echo "---------------------------------------------------" >> "$LOGFILE"
echo "[$(date)] Torrent completato: $TR_TORRENT_NAME" >> "$LOGFILE"

# Verifica variabili
if [ -z "$TR_TORRENT_DIR" ] || [ -z "$TR_TORRENT_NAME" ]; then
    echo "[ERRORE] Variabili Transmission mancanti." >> "$LOGFILE"
    exit 1
fi

# COSTRUZIONE DEL PERCORSO COMPLETO
# $TR_TORRENT_DIR = /var/lib/transmission-daemon/Downloads
# $TR_TORRENT_NAME = Nome del file o della cartella del torrent
# Mettendoli insieme con / otteniamo il percorso assoluto esatto
FULL_PATH="$TR_TORRENT_DIR/$TR_TORRENT_NAME"

echo "[INFO] Sorgente rilevata: '$FULL_PATH'" >> "$LOGFILE"
echo "[INFO] Destinazione: '$DESTINAZIONE'" >> "$LOGFILE"

# Verifica esistenza sorgente
if [ ! -e "$FULL_PATH" ]; then
    echo "[ERRORE] Il file o la cartella non esiste: '$FULL_PATH'" >> "$LOGFILE"
    exit 1
fi

# ESECUZIONE SPOSTAMENTO
# Le virgolette attorno a "$FULL_PATH" gestiscono spazi e caratteri speciali automaticamente
mv "$FULL_PATH" "$DESTINAZIONE/" 2>> "$LOGFILE"

STATUS=$?

if [ $STATUS -eq 0 ]; then
    echo "[SUCCESS] Spostamento riuscito." >> "$LOGFILE"
else
    echo "[ERRORE] Codice errore mv: $STATUS" >> "$LOGFILE"
fi