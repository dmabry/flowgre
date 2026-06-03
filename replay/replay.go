// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Replay is used to send netflow packets recorded off the wire and stored in a db at a specified target

package replay

import (
	"context"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
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
		log.Printf("Listen: %v\n", err)
		return
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
			_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, payload, false)
			if err != nil {
				log.Printf("Worker [%2d] Issue sending packet: %v\n", id, err)
				return
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
func dbReader(ctx context.Context, wg *sync.WaitGroup, dbdir string, dataChan chan<- []byte, loop bool, updateTS bool, verbose bool) {
	defer wg.Done()
	// Create/Open DB for reading
	options := badger.DefaultOptions(dbdir)
	// Disable badger logging output
	options.Logger = nil
	db, err := badger.Open(options)
	if err != nil {
		log.Printf("Failed to open DB: %v\n", err)
		return
	}
	defer db.Close()
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
						//key := item.Key()
						value, err := item.ValueCopy(nil)
						if err != nil {
							log.Printf("DB Reader issue getting value from db: %v", err)
							return err
						}
						if updateTS {
							newValue, err := netflow.UpdateTimeStamp(value)
							if err != nil {
								log.Printf("DB Reader issue rewriting timestamp: %v", err)
								return err
							}
							value = newValue
						}
						dataChan <- value
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

// RunCtx replays netflow packets from a db with an external context.
// Cancelling ctx stops all workers cleanly. In non-loop mode, the function
// returns when all packets have been sent. Use Run() for CLI usage where
// OS signal handling is desired.
func RunCtx(ctx context.Context, server string, port int, delay int, dbdir string, loop bool, workers int, updateTS bool, verbose bool) {
	wg := &sync.WaitGroup{}
	dataChan := make(chan []byte, 1024)

	// Start dbReader
	wg.Add(1)
	go dbReader(ctx, wg, dbdir, dataChan, loop, updateTS, verbose)

	// Start up the workers
	wg.Add(workers)
	for w := 1; w <= workers; w++ {
		go worker(w, ctx, server, port, delay, wg, loop, dataChan)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Log completion if context was not cancelled (natural completion)
	if ctx.Err() == nil {
		log.Printf("\nReplay complete, shutting down...\n")
	}
}

// Run Replay. Kicks off the replay of netflow packets from a db.
// It sets up OS signal handling (SIGINT/SIGTERM) for clean shutdown.
// Use RunCtx() when you need to control the lifecycle via context.
func Run(server string, port int, delay int, dbdir string, loop bool, workers int, updateTS bool, verbose bool) {
	mgr := lifecycle.New()

	// Setup signal handling via lifecycle manager.
	cleanupDone := mgr.SetupSignalHandler()

	go func() {
		<-cleanupDone
		log.Printf("\rReceived signal, shutting down...\n\n")
		mgr.Cancel()
	}()

	RunCtx(mgr.Context(), server, port, delay, dbdir, loop, workers, updateTS, verbose)
	mgr.Wait()
}
