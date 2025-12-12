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
	quality = 60
)

var (
	frame   []byte
	frameMu sync.RWMutex
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go captureLoop()

	http.HandleFunc("/frame.jpg", handleFrame)
	http.HandleFunc("/", serveHTML)

	ip := getLocalIP()
	fmt.Println("=============================================")
	fmt.Println("  HTTP SCREEN SHARE")
	fmt.Println("=============================================")
	fmt.Printf("  Monitör: %s\n", monitor)
	fmt.Printf("  URL: http://%s:%d\n", ip, port)
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
			frame = data
			frameMu.Unlock()
		}
	}
}

func handleFrame(w http.ResponseWriter, r *http.Request) {
	frameMu.RLock()
	data := frame
	frameMu.RUnlock()

	if len(data) == 0 {
		http.Error(w, "No frame", 503)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Write(data)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html><html><head><title>Screen</title>
<style>*{margin:0;padding:0}body{background:#000}img{width:100vw;height:100vh;object-fit:contain}</style>
</head><body><img id="s"><script>
const s=document.getElementById('s');let n=0;
function f(){const i=new Image();i.onload=()=>{s.src=i.src;n++;requestAnimationFrame(f);};i.onerror=()=>setTimeout(f,50);i.src='/frame.jpg?'+Date.now();}
f();setInterval(()=>{document.title=n+' FPS';n=0;},1000);
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
