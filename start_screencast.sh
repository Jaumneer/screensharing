#!/bin/bash

# Portal üzerinden ekran seçimi yap ve node_id'yi döndür
# Bu script xdg-desktop-portal ile etkileşim kurar

PORTAL_DEST="org.freedesktop.portal.Desktop"
PORTAL_PATH="/org/freedesktop/portal/desktop"
PORTAL_SCREENCAST="org.freedesktop.portal.ScreenCast"

# Session token oluştur
SESSION_TOKEN="screenshare_$$"

# Session oluştur
SESSION_PATH=$(gdbus call --session \
    --dest $PORTAL_DEST \
    --object-path $PORTAL_PATH \
    --method $PORTAL_SCREENCAST.CreateSession \
    "{'session_handle_token': <'$SESSION_TOKEN'>}" 2>/dev/null | grep -oP "'/[^']+'" | tr -d "'")

if [ -z "$SESSION_PATH" ]; then
    echo "Session oluşturulamadı" >&2
    exit 1
fi

sleep 0.5

# Kaynak seç (1=Monitor, 2=Window, 4=Virtual)
# cursor_mode: 1=hidden, 2=embedded
RESULT=$(gdbus call --session \
    --dest $PORTAL_DEST \
    --object-path $PORTAL_PATH \
    --method $PORTAL_SCREENCAST.SelectSources \
    "$SESSION_PATH" \
    "{'types': <uint32 1>, 'cursor_mode': <uint32 2>, 'multiple': <false>}" 2>/dev/null)

sleep 1

# Başlat
START_RESULT=$(gdbus call --session \
    --dest $PORTAL_DEST \
    --object-path $PORTAL_PATH \
    --method $PORTAL_SCREENCAST.Start \
    "$SESSION_PATH" "" "{}" 2>/dev/null)

# Node ID'yi çıkar
NODE_ID=$(echo "$START_RESULT" | grep -oP "node_id[^0-9]+\K[0-9]+")

if [ -z "$NODE_ID" ]; then
    echo "Node ID alınamadı" >&2
    exit 1
fi

echo "$NODE_ID"
