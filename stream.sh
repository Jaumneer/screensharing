#!/bin/bash

# Ayarlar
PORT=8080
MONITOR="eDP-1"  # Monitör adı (hyprctl monitors ile kontrol edin)
QUALITY=90       # JPEG kalitesi (1-100)
FPS=60           # Frame rate
SCALE="2944x1840" # Çözünürlük (orijinal 2944x1840)

# IP adresini bul
IP=$(ip route get 1 | awk '{print $7; exit}')

echo "============================================="
echo "  Ekran Paylaşımı Başlatılıyor"
echo "============================================="
echo "Monitör: $MONITOR"
echo "Tablet'ten şu adrese gidin:"
echo "  http://$IP:$PORT"
echo "============================================="
echo "Durdurmak için Ctrl+C"
echo ""

# wf-recorder ile ekranı yakala, ffmpeg ile MJPEG stream yap
wf-recorder -o "$MONITOR" -c rawvideo -x yuv420p -f pipe:1 2>/dev/null | \
ffmpeg -f rawvideo -pix_fmt yuv420p -s 2944x1840 -r $FPS -i pipe:0 \
    -vf "scale=$SCALE" \
    -c:v mjpeg -q:v 2 -f mjpeg \
    -listen 1 "http://0.0.0.0:$PORT" 2>/dev/null
