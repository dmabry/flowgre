// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to setup a continuous stream of netflow packets towards a single collector

package record

import (
	"log"
	"net"
	"os"
)

// Run Record
func Run(ip string, port int, file string) {
	//var buf bytes.Buffer
	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		log.Fatalf("Listening on %s:%d failed! Got: %v", ip, port, err)
	}
	defer conn.Close()
	log.Printf("Listening on %s:%d recording to %s.", ip, port, file)
	for {
		payload := make([]byte, 4096)
		length, fromIP, perr := conn.ReadFromUDP(payload)
		if err != nil {
			log.Fatalf("Error reading packet from %s: %v", fromIP, perr)
		}
		payload = payload[:length]
		log.Println(payload)
	}
	os.Exit(0)
}
