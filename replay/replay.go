// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Replay is used to send netflow packets recorded off the wire and stored in a db at a specificed target

package replay

import (
	"bytes"
	"context"
	"encoding/binary"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/models"
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
func worker(id int, ctx context.Context, server string, port int, delay int, wg *sync.WaitGroup, dataChan <-chan []byte) {
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
		}
	}
}

// dbReader pulls byte payload out of the database and puts it on the data chan
func dbReader(ctx context.Context, wg *sync.WaitGroup, dbdir string, dataChan chan<- []byte, verbose bool) {
	defer wg.Done()
	// Create/Open DB for writing
	options := badger.DefaultOptions(dbdir)
	// Disable badger logging output
	options.Logger = nil
	db, err := badger.Open(options)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	log.Printf("Reading from database %s\n", dbdir)
	// Prep the loop
	count := 0
	itOptions := badger.DefaultIteratorOptions
	itOptions.PrefetchSize = runtime.GOMAXPROCS(0)
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(itOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			// Check to see if context is done and return, otherwise pull payloads and write
			select {
			case <-ctx.Done():
				log.Println("Database Reader exiting due to signal")
				err := db.Close()
				if err != nil {
					log.Printf("Issue closing database: %v", err)
					return err
				}
				return nil
			default:
			}
			item := it.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				buf := bytes.NewReader(key)
				var k uint32
				err := binary.Read(buf, binary.BigEndian, &k)
				if err != nil {
					log.Printf("Issue reading key: %v\n", err)
				}
				v := val
				//log.Printf("key: %v value: %v \n", k, v)
				dataChan <- v
				return nil
			})
			if err != nil {
				log.Printf("Issue getting value from db: %v", err)
			}
			count++
		}
		log.Printf("Read %d payloads from the database\n", count)
		return nil
	})
}

// Ingest pulls byte payload off the data chan and puts them in the badger db
func parseNetflow(ctx context.Context, wg *sync.WaitGroup, parseChan <-chan []byte, dataChan chan<- []byte, verbose bool) {
	defer wg.Done()
	// Prep the loop
	rStats := models.RecordStat{
		ValidCount:   0,
		InvalidCount: 0,
	}
	printStats := false
	startTime := time.Now().UnixNano()
	// Start the loop
	for {
		now := time.Now().UnixNano()
		statsCycle := (now - startTime) / int64(time.Second) % 10
		// Print out basic statistics per worker every 10 seconds
		if statsCycle == 0 {
			if printStats {
				log.Printf("Netflow v9 Packets: %d Ignored Packets: %d ",
					rStats.ValidCount, rStats.InvalidCount)
				printStats = false
			}
		} else {
			printStats = true
		}
		// Check to see if context is done and return, otherwise pull payloads and write
		select {
		case <-ctx.Done():
			log.Println("Netflow parser exiting due to signal")
			return
		case payload := <-parseChan:
			// Decode first uint16 and see if it is a version 9
			buf := bytes.NewReader(payload)
			var nfVersion uint16
			err := binary.Read(buf, binary.BigEndian, &nfVersion)
			if err != nil {
				log.Printf("Skipping packet due to issue parsing: %v", err)
				continue
			}
			if nfVersion == 9 {
				// Netflow v9 Packet send it on
				rStats.ValidCount++
				dataChan <- payload
			} else {
				// Not a Netflow v9 Packet... skip
				rStats.InvalidCount++
			}
		default:
			// Non-blocking
		}
	}
}

// Run Replay. Kicks off the replay of netflow packets from a db.
func Run(server string, port int, delay int, dbdir string, loop bool, workers int, verbose bool) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	dataChan := make(chan []byte, 1024)
	//parseChan := make(chan []byte, 1024)

	// Start netIngest
	//wg.Add(1)
	//go netIngest(ctx, wg, ip, port, parseChan, verbose)

	// Start parseNetflow
	//wg.Add(1)
	//go parseNetflow(ctx, wg, parseChan, dataChan, verbose)

	// Start dbIngest
	wg.Add(1)
	go dbReader(ctx, wg, dbdir, dataChan, verbose)

	// Start up the workers
	wg.Add(workers)
	for w := 1; w <= workers; w++ {
		go worker(w, ctx, server, port, delay, wg, dataChan)
	}

	// Wait for a SIGINT (perhaps triggered by user with CTRL-C)
	// Run cleanup when signal is received
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt, os.Kill, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGHUP))

	go func() {
		for range signalChan {
			log.Printf("\rReceived signal, shutting down...\n\n")
			cancel()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
	wg.Wait()
	close(signalChan)
	os.Exit(0)
}
