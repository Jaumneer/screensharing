# Screensharing - Wayland Virtual Monitor Streaming

Wayland (Hyprland) üzerinde sanal monitör oluşturup, bu monitörü yerel ağ üzerinden tablet veya başka bir cihaza stream etmek için kullanılan araç.

## Özellikler

- 🖥️ Hyprland headless backend ile sanal monitör oluşturma
- 📸 `grim` ile hızlı JPEG ekran görüntüsü yakalama
- 🌐 HTTP üzerinden JPEG frame streaming
- 📱 Flutter tabanlı Android client (ayrı repo)
- 🔒 Sadece yerel ağ (şifreleme yok)

## Gereksinimler

- Hyprland compositor
- `grim` (Wayland screenshot tool)
- Go 1.21+

## Kurulum

```bash
git clone https://github.com/Jaumneer/screensharing.git
cd screensharing
go build -o screensharing .
```

## Kullanım

### 1. Sanal Monitör Oluştur

```bash
hyprctl output create headless
```

Bu komut `HEADLESS-2` (veya benzeri) bir monitör oluşturur.

### 2. Monitör Adını Kontrol Et

```bash
hyprctl monitors | grep HEADLESS
```

### 3. Sunucuyu Başlat

```bash
./screensharing
```

Varsayılan ayarlar:
- Port: `8080`
- Monitör: `HEADLESS-2`
- JPEG Kalite: `65%`

### 4. Bağlan

Tarayıcıdan veya Flutter client ile:
```
http://<IP_ADRESI>:8080
```

### 5. Sanal Monitörü Sil (Bitince)

```bash
hyprctl output remove HEADLESS-2
```

## Yapılandırma

`main.go` içindeki sabitler:

```go
const (
    port    = 8080        // HTTP port
    monitor = "HEADLESS-2" // Monitör adı
    quality = 65          // JPEG kalitesi (0-100)
)
```

## Flutter Client

Android tablet için Flutter uygulaması: `/home/jau/Desktop/tablet_screen/`

APK build:
```bash
cd tablet_screen
flutter build apk --release
```

## Lisans

MIT
