// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Replay is used to send netflow packets recorded off the wire and stored in a db at a specified target

package replay

import (
	"bytes"
	"context"
	"encoding/binary"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/utils"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, delay int, wg *sync.WaitGroup, loop bool, dataChan <-chan []byte) {
	defer wg.Done()
	// Sent limiter to given delay
	limiter := time.Tick(time.Millisecond * time.Duration(delay))
	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	srcPort := utils.RandomNum(10000, 15000)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(server)
	log.Printf("Worker [%2d] Slinging packets at %s:%d with delay of %dms \n",
		id, server, port, delay)
	//Infinite loop to keep slinging until we receive context done.
	for {
		select {
		case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
			log.Printf("Worker [%2d] Exiting due to signal\n", id)
			return
		case payload := <-dataChan:
			length := len(payload)
			log.Printf("Worker [%2d] sending packet with length: %d\n", id, length)
			// send packet here.
			var buf bytes.Buffer
			err := binary.Write(&buf, binary.BigEndian, &payload)
			if err != nil {
				log.Printf("Worker [%2d] Issue reading data: %v\n", id, err)
			}
			_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, buf, false)
			if err != nil {
				log.Fatalf("Worker [%2d] Issue sending packet %v\n", id, err)
			}
			<-limiter
		case <-time.After(time.Second * 1):
			if !loop {
				log.Printf("Worker [%2d] Exiting due to empty channel\n", id)
				return
			}
		}
	}
}

// dbReader pulls byte payload out of the database and puts it on the data chan
func dbReader(ctx context.Context, wg *sync.WaitGroup, dbdir string, dataChan chan<- []byte, loop bool, verbose bool) {
	defer wg.Done()
	// Create/Open DB for writing
	options := badger.DefaultOptions(dbdir)
	// Disable badger logging output
	options.Logger = nil
	db, err := badger.Open(options)
	defer db.Close()
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	log.Printf("Reading from database %s\n", dbdir)
	// Prep the loop
	count := 0
	itOptions := badger.DefaultIteratorOptions
	itOptions.PrefetchSize = runtime.GOMAXPROCS(0)
	for {
		select {
		case <-ctx.Done():
			log.Println("DB Reader exiting due to signal")
			return
		default:
			err = db.View(func(txn *badger.Txn) error {
				it := txn.NewIterator(itOptions)
				defer it.Close()
				for it.Rewind(); it.Valid(); it.Next() {
					// Check to see if context is done and return, otherwise pull payloads and write
					select {
					case <-ctx.Done():
						log.Println("DB Reader exiting due to signal, finishing read")
						return nil
					default:
						item := it.Item()
						key := item.Key()
						err := item.Value(func(val []byte) error {
							buf := bytes.NewReader(key)
							var k uint32
							err := binary.Read(buf, binary.BigEndian, &k)
							if err != nil {
								log.Printf("DB Reader issue reading key: %v\n", err)
								return err
							}
							v := val
							dataChan <- v
							return nil
						})
						if err != nil {
							log.Printf("DB Reader issue getting value from db: %v", err)
							return err
						}
						count++
					}
				}
				return nil
			})
			if err != nil {
				log.Printf("DB Reader had an issue: %v", err)
			}
		}
		// only run once if not a loop
		if !loop {
			break
		}
	}
	log.Printf("DB Reader read %d payloads from the database\n", count)
	log.Printf("DB Reader done.")
	return
}

// Run Replay. Kicks off the replay of netflow packets from a db.
func Run(server string, port int, delay int, dbdir string, loop bool, workers int, verbose bool) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	dataChan := make(chan []byte, 1024)

	// Start dbReader
	wg.Add(1)
	go dbReader(ctx, wg, dbdir, dataChan, loop, verbose)

	// Start up the workers
	wg.Add(workers)
	for w := 1; w <= workers; w++ {
		go worker(w, ctx, server, port, delay, wg, loop, dataChan)
	}

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt, os.Kill, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGHUP))

	go func() {
		for {
			select {
			case <-signalChan:
				log.Printf("\rReceived signal, shutting down...\n\n")
				cancel()
				cleanupDone <- true
			case <-ctx.Done():
				cleanupDone <- true
			case <-time.After(time.Second * 1):
				if !loop {
					if len(dataChan) == 0 {
						log.Printf("Replay complete, shutting down...\n\n")
						cancel()
						cleanupDone <- true
					}
				}
			}
		}
	}()
	<-cleanupDone
	wg.Wait()
	close(signalChan)
	close(cleanupDone)
	os.Exit(0)
}
