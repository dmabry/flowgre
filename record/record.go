// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Record is used to take netflow packets off the wire and store them in a badger db

package record

import (
	"context"
	"encoding/binary"
	badger "github.com/dgraph-io/badger/v3"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// netIngest is used to pull packets off the wire and put the byte payload on the data chan
func netIngest(ctx context.Context, wg *sync.WaitGroup, ip string, port int, data chan<- []byte) {
	defer wg.Done()
	// Create UDP listener and setup db to catch files
	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		log.Fatalf("Listening on %s:%d failed! Got: %v", ip, port, err)
	}
	log.Printf("Listening on %s:%d", ip, port)
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("Error closing listener: %v", err)
		}
	}(conn)
	// Start the loop and check context for done, otherwise listen for packets
	for {
		select {
		case <-ctx.Done():
			log.Println("Packet ingest exiting due to signal")
			return
		default:
			payload := make([]byte, 4096)
			timeout := time.Now().Add(5 * time.Second)
			err := conn.SetReadDeadline(timeout)
			if err != nil {
				log.Printf("Issue setting deadline: %v", err)
				return
			}
			length, fromIP, err := conn.ReadFromUDP(payload)
			if err != nil {
				// log.Fatalf("Error reading packet from %s: %v", fromIP, err)
				log.Println("No packets received")
				continue
			}
			payload = payload[:length]
			log.Printf("Packet Recieved from %s with size of %d", fromIP.String(), length)
			// Send payload to the data channel
			data <- payload
		}
	}
}

// dbIngest pulls byte payload off the data chan and puts them in the badger db
func dbIngest(ctx context.Context, wg *sync.WaitGroup, dbdir string, data <-chan []byte) {
	defer wg.Done()
	// Create/Open DB for writing
	options := badger.DefaultOptions(dbdir)
	// Disable badger logging output
	options.Logger = nil
	db, err := badger.Open(options)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	log.Printf("Writing to database %s\n", dbdir)
	// Prep the loop
	var count uint32 = 0
	// Start the loop
	for {
		// Check to see if context is done and return, otherwise pull payloads and write
		select {
		case <-ctx.Done():
			log.Println("Database ingest exiting due to signal")
			err := db.Close()
			if err != nil {
				log.Printf("Issue closing database: %v", err)
				return
			}
			return
		case payload := <-data:
			count++
			key := make([]byte, 4)
			binary.LittleEndian.PutUint32(key, count)
			err := db.Update(func(txn *badger.Txn) error {
				entry := badger.NewEntry(key, payload)
				terr := txn.SetEntry(entry)
				return terr
			})
			if err != nil {
				log.Fatalf("Error writing to db: %v", err)
			}
		}
	}
}

// Run Record. Kicks off the recording process.
func Run(ip string, port int, dbdir string) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	dataChan := make(chan []byte, 1024)

	// Start netIngest
	wg.Add(1)
	go netIngest(ctx, wg, ip, port, dataChan)

	// Start dbIngest
	wg.Add(1)
	go dbIngest(ctx, wg, dbdir, dataChan)

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt, os.Kill, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGHUP))

	go func() {
		for range signalChan {
			log.Println("Received signal, shutting down...")
			cancel()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
	wg.Wait()
	close(signalChan)
	os.Exit(0)
}
