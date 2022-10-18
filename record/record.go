// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Barrage is used to setup a continuous stream of netflow packets towards a single collector

package record

import (
	"encoding/binary"
	badger "github.com/dgraph-io/badger/v3"
	"log"
	"net"
	"os"
)

// Run Record
func Run(ip string, port int, dbdir string) {
	// Create UDP listener and setup db to catch files
	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		log.Fatalf("Listening on %s:%d failed! Got: %v", ip, port, err)
	}
	defer conn.Close()
	log.Printf("Listening on %s:%d recording to %s.", ip, port, dbdir)
	// Create/Open DB for writing
	db, err := badger.Open(badger.DefaultOptions(dbdir))
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()
	var count uint32 = 0
	for {
		payload := make([]byte, 4096)
		length, fromIP, perr := conn.ReadFromUDP(payload)
		if err != nil {
			log.Fatalf("Error reading packet from %s: %v", fromIP, perr)
		}
		payload = payload[:length]
		log.Printf("Packet Recieved from %s with size of %d", fromIP.String(), length)
		count++
		key := make([]byte, 4)
		binary.LittleEndian.PutUint32(key, count)
		derr := db.Update(func(txn *badger.Txn) error {
			entry := badger.NewEntry(key, payload)
			terr := txn.SetEntry(entry)
			return terr
		})
		if derr != nil {
			log.Fatalf("Error writing to db: %v", derr)
		}
	}
	os.Exit(0)
}
