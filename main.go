package main

import (
"bytes"
"fmt"
"log"
"net"
"net/http"
"os"
"os/exec"
"os/signal"
"sync"
"syscall"
)

const (
port    = 8080
monitor = "HEADLESS-2"
quality = 70
)

var (
currentFrame []byte
frameMu      sync.RWMutex
)

func main() {
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go captureLoop()

http.HandleFunc("/", serveHTML)
http.HandleFunc("/frame.jpg", handleFrame)

ip := getLocalIP()
fmt.Println("=============================================")
fmt.Println("  SANAL MONİTÖR PAYLAŞIMI")
fmt.Println("=============================================")
fmt.Printf("  Monitör: %s\n", monitor)
fmt.Printf("  Adres: http://%s:%d\n", ip, port)
fmt.Printf("  Kalite: %d%%\n", quality)
fmt.Println("=============================================")

go func() {
<-sigChan
fmt.Println("\nKapatılıyor...")
os.Exit(0)
}()

log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func captureLoop() {
var buf bytes.Buffer

for {
buf.Reset()

cmd := exec.Command("grim", "-o", monitor, "-t", "jpeg", "-q", fmt.Sprintf("%d", quality), "-")
cmd.Stdout = &buf

if err := cmd.Run(); err == nil && buf.Len() > 0 {
data := make([]byte, buf.Len())
copy(data, buf.Bytes())

frameMu.Lock()
currentFrame = data
frameMu.Unlock()
}
}
}

func handleFrame(w http.ResponseWriter, r *http.Request) {
frameMu.RLock()
frame := currentFrame
frameMu.RUnlock()

if len(frame) == 0 {
http.Error(w, "Frame yok", 503)
return
}

w.Header().Set("Content-Type", "image/jpeg")
w.Header().Set("Cache-Control", "no-store")
w.Write(frame)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
html := `<!DOCTYPE html>
<html><head><title>2. Ekran</title>
<style>*{margin:0;padding:0}body{background:#000}img{width:100vw;height:100vh;object-fit:contain}</style>
</head><body><img id="s"><script>
const s=document.getElementById('s');
let l=false;
setInterval(()=>{
  if(l)return;l=true;
  const i=new Image();
  i.onload=()=>{s.src=i.src;l=false;};
  i.onerror=()=>l=false;
  i.src='/frame.jpg?'+Date.now();
},33);
</script></body></html>`
w.Header().Set("Content-Type", "text/html")
w.Write([]byte(html))
}

func getLocalIP() string {
addrs, _ := net.InterfaceAddrs()
for _, addr := range addrs {
if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
return ipnet.IP.String()
}
}
return "localhost"
}
