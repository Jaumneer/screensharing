package main

import (
"bytes"
"encoding/json"
"fmt"
"log"
"net"
"net/http"
"os"
"os/exec"
"os/signal"
"sync"
"sync/atomic"
"syscall"
"time"
)

const (
port       = 8080
monitor    = "HEADLESS-2"
minQuality = 30
maxQuality = 85
)

var (
frame      []byte
frameMu    sync.RWMutex
frameSeq   atomic.Uint64
frameTime  atomic.Int64

// Adaptif ayarlar
quality    atomic.Int32
stats      ClientStats
statsMu    sync.RWMutex
)

type ClientStats struct {
RTT        float64 `json:"rtt"`        // ms
Bandwidth  float64 `json:"bandwidth"`  // KB/s
FrameDrops int     `json:"frameDrops"`
LastSeq    uint64  `json:"lastSeq"`
}

type StatsResponse struct {
Quality   int     `json:"quality"`
FrameSeq  uint64  `json:"frameSeq"`
Timestamp int64   `json:"timestamp"`
}

func init() {
quality.Store(60) // Başlangıç kalitesi
}

func main() {
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go captureLoop()
go adaptiveLoop()

http.HandleFunc("/frame.jpg", handleFrame)
http.HandleFunc("/stats", handleStats)
http.HandleFunc("/", serveHTML)

ip := getLocalIP()
fmt.Println("=============================================")
fmt.Println("  ADAPTIVE SCREEN SHARE")
fmt.Println("=============================================")
fmt.Printf("  Monitör: %s\n", monitor)
fmt.Printf("  URL: http://%s:%d\n", ip, port)
fmt.Printf("  Kalite: %d-%d%% (adaptif)\n", minQuality, maxQuality)
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
q := quality.Load()

cmd := exec.Command("grim", "-o", monitor, "-t", "jpeg", "-q", fmt.Sprintf("%d", q), "-")
cmd.Stdout = &buf

if err := cmd.Run(); err == nil && buf.Len() > 0 {
data := make([]byte, buf.Len())
copy(data, buf.Bytes())

frameMu.Lock()
frame = data
frameMu.Unlock()

frameSeq.Add(1)
frameTime.Store(time.Now().UnixMilli())
}
}
}

// Adaptif kalite ayarlama
func adaptiveLoop() {
ticker := time.NewTicker(500 * time.Millisecond)
defer ticker.Stop()

for range ticker.C {
statsMu.RLock()
s := stats
statsMu.RUnlock()

currentQ := quality.Load()
newQ := currentQ

// RTT bazlı ayarlama
if s.RTT > 150 {
// Yüksek gecikme - kaliteyi düşür
newQ = currentQ - 10
} else if s.RTT > 80 {
newQ = currentQ - 5
} else if s.RTT < 30 && s.Bandwidth > 500 {
// Düşük gecikme, iyi bant - kaliteyi artır
newQ = currentQ + 5
} else if s.RTT < 50 && s.Bandwidth > 300 {
newQ = currentQ + 2
}

// Frame drop varsa kaliteyi düşür
if s.FrameDrops > 0 {
newQ = currentQ - (int32(s.FrameDrops) * 5)
}

// Sınırları uygula
if newQ < minQuality {
newQ = minQuality
}
if newQ > maxQuality {
newQ = maxQuality
}

if newQ != currentQ {
quality.Store(newQ)
fmt.Printf("📊 Kalite: %d%% (RTT: %.0fms, BW: %.0fKB/s, Drops: %d)\n", 
newQ, s.RTT, s.Bandwidth, s.FrameDrops)
}
}
}

func handleFrame(w http.ResponseWriter, r *http.Request) {
frameMu.RLock()
data := frame
seq := frameSeq.Load()
ts := frameTime.Load()
frameMu.RUnlock()

if len(data) == 0 {
http.Error(w, "No frame", 503)
return
}

w.Header().Set("Content-Type", "image/jpeg")
w.Header().Set("Cache-Control", "no-store")
w.Header().Set("X-Frame-Seq", fmt.Sprintf("%d", seq))
w.Header().Set("X-Frame-Time", fmt.Sprintf("%d", ts))
w.Header().Set("X-Quality", fmt.Sprintf("%d", quality.Load()))
w.Write(data)
}

// Client stats'ları alır ve sunucu stats'larını döner
func handleStats(w http.ResponseWriter, r *http.Request) {
if r.Method == "POST" {
var clientStats ClientStats
if err := json.NewDecoder(r.Body).Decode(&clientStats); err == nil {
statsMu.Lock()
stats = clientStats
statsMu.Unlock()
}
}

resp := StatsResponse{
Quality:   int(quality.Load()),
FrameSeq:  frameSeq.Load(),
Timestamp: time.Now().UnixMilli(),
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(resp)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
html := `<!DOCTYPE html><html><head><title>Screen</title>
<style>*{margin:0;padding:0}body{background:#000;color:#fff;font-family:monospace}
img{width:100vw;height:100vh;object-fit:contain}
#stats{position:fixed;top:5px;left:5px;background:rgba(0,0,0,0.7);padding:5px 10px;font-size:12px;border-radius:4px}</style>
</head><body>
<div id="stats">FPS: 0 | Q: 0 | RTT: 0ms</div>
<img id="s">
<script>
const s=document.getElementById('s'),st=document.getElementById('stats');
let fps=0,lastSeq=0,drops=0,rtt=0,bw=0,lastTime=0,lastSize=0;

async function sendStats(){
  const start=Date.now();
  try{
    const r=await fetch('/stats',{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({rtt,bandwidth:bw,frameDrops:drops,lastSeq})});
    rtt=(rtt*0.7)+(Date.now()-start)*0.3;
    drops=0;
  }catch(e){}
}

function load(){
  const i=new Image(),start=Date.now();
  i.onload=()=>{
    const now=Date.now(),dt=now-start;
    if(lastSize>0)bw=(lastSize/1024)/(dt/1000);
    lastSize=i.src.length;
    s.src=i.src;
    fps++;
    requestAnimationFrame(load);
  };
  i.onerror=()=>{drops++;setTimeout(load,30);};
  i.src='/frame.jpg?'+Date.now();
}

load();
setInterval(sendStats,500);
setInterval(()=>{
  st.textContent='FPS: '+fps+' | RTT: '+Math.round(rtt)+'ms | BW: '+Math.round(bw)+'KB/s';
  fps=0;
},1000);
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
