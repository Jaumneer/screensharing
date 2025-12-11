package main

import (
"bytes"
"encoding/binary"
"fmt"
"log"
"net"
"os"
"os/exec"
"os/signal"
"syscall"
)

const (
udpPort   = 9999
monitor   = "HEADLESS-2"
quality   = 60
maxPacket = 60000
)

func main() {
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", udpPort))
if err != nil {
log.Fatal(err)
}

conn, err := net.ListenUDP("udp", addr)
if err != nil {
log.Fatal(err)
}
defer conn.Close()

ip := getLocalIP()
fmt.Println("=============================================")
fmt.Println("  UDP SCREEN SHARE")
fmt.Println("=============================================")
fmt.Printf("  Monitör: %s\n", monitor)
fmt.Printf("  UDP: %s:%d\n", ip, udpPort)
fmt.Printf("  Kalite: %d%%\n", quality)
fmt.Println("=============================================")

clients := make(map[string]*net.UDPAddr)

// Client listener
go func() {
buf := make([]byte, 64)
for {
n, clientAddr, err := conn.ReadFromUDP(buf)
if err != nil {
continue
}
msg := string(buf[:n])
if msg == "CONNECT" {
clients[clientAddr.String()] = clientAddr
fmt.Printf("+ Client: %s\n", clientAddr.String())
} else if msg == "DISCONNECT" {
delete(clients, clientAddr.String())
fmt.Printf("- Client: %s\n", clientAddr.String())
}
}
}()

// Capture loop
go func() {
var buf bytes.Buffer
var frameNum uint32 = 0

for {
buf.Reset()

cmd := exec.Command("grim", "-o", monitor, "-t", "jpeg", "-q", fmt.Sprintf("%d", quality), "-")
cmd.Stdout = &buf

if err := cmd.Run(); err != nil || buf.Len() == 0 {
continue
}

frameNum++
data := buf.Bytes()

for _, addr := range clients {
sendFrame(conn, addr, frameNum, data)
}
}
}()

go func() {
<-sigChan
fmt.Println("\nKapatılıyor...")
os.Exit(0)
}()

select {}
}

func sendFrame(conn *net.UDPConn, addr *net.UDPAddr, frameNum uint32, data []byte) {
total := len(data)
packets := (total + maxPacket - 1) / maxPacket

for i := 0; i < packets; i++ {
start := i * maxPacket
end := start + maxPacket
if end > total {
end = total
}

// Header: frame(4) + idx(2) + count(2) + size(4) = 12 bytes
hdr := make([]byte, 12)
binary.BigEndian.PutUint32(hdr[0:4], frameNum)
binary.BigEndian.PutUint16(hdr[4:6], uint16(i))
binary.BigEndian.PutUint16(hdr[6:8], uint16(packets))
binary.BigEndian.PutUint32(hdr[8:12], uint32(total))

pkt := append(hdr, data[start:end]...)
conn.WriteToUDP(pkt, addr)
}
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
