// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Replay is used to send netflow packets recorded off the wire and stored in a db at a specified target

package replay

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"runtime"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
	"golang.org/x/sync/errgroup"
)

// netflowVersion returns the version field from a flow packet header.
// Returns 0 if the payload is too short.
func netflowVersion(payload []byte) uint16 {
	if len(payload) < 2 {
		return 0
	}
	return binary.BigEndian.Uint16(payload[0:2])
}

// updateTimestamp updates the timestamp in a flow packet, dispatching to the
// correct protocol-specific updater based on the version field.
func updateTimestamp(payload []byte) ([]byte, error) {
	version := netflowVersion(payload)
	switch version {
	case 9:
		return netflow.UpdateTimeStamp(payload)
	case 10:
		return ipfix.UpdateTimeStamp(payload)
	default:
		return nil, fmt.Errorf("unsupported flow version %d for timestamp update", version)
	}
}

// Worker is the goroutine used to create workers
func worker(id int, ctx context.Context, server string, port int, delay int, loop bool, dataChan <-chan []byte) error {
	limiter := time.NewTicker(time.Millisecond * time.Duration(delay))
	defer limiter.Stop()

	srcPort, err := utils.RandomNum(10000, 15000)
	if err != nil {
		return fmt.Errorf("replay worker %d generate source port: %w", id, err)
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		return fmt.Errorf("replay worker %d listen: %w", id, err)
	}
	defer conn.Close()

	destIP := net.ParseIP(server)
	log.Printf("Worker [%2d] Slinging packets at %s:%d with delay of %dms \n",
		id, server, port, delay)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker [%2d] Exiting due to signal\n", id)
			return nil
		case payload, ok := <-dataChan:
			if !ok {
				log.Printf("Worker [%2d] Exiting due to closed channel\n", id)
				return nil
			}
			length := len(payload)
			log.Printf("Worker [%2d] sending packet with length: %d\n", id, length)
			_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: port}, payload, false)
			if err != nil {
				return fmt.Errorf("replay worker %d send: %w", id, err)
			}
			select {
			case <-limiter.C:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// dbReader pulls byte payload out of the database and puts it on the data chan.
// In non-loop mode, it closes dataChan after the final pass to signal workers.
func dbReader(ctx context.Context, dbdir string, dataChan chan<- []byte, loop bool, updateTS bool, verbose bool) error {
	if !loop {
		defer close(dataChan)
	}

	options := badger.DefaultOptions(dbdir)
	options.Logger = nil
	db, err := badger.Open(options)
	if err != nil {
		return fmt.Errorf("open DB %s: %w", dbdir, err)
	}
	defer db.Close()
	log.Printf("Reading from database %s\n", dbdir)

	count := 0
	itOptions := badger.DefaultIteratorOptions
	itOptions.PrefetchSize = runtime.GOMAXPROCS(0)
	for {
		recordsThisPass := 0
		select {
		case <-ctx.Done():
			log.Println("DB Reader exiting due to signal")
			return nil
		default:
			err = db.View(func(txn *badger.Txn) error {
				it := txn.NewIterator(itOptions)
				defer it.Close()
				for it.Rewind(); it.Valid(); it.Next() {
					select {
					case <-ctx.Done():
						log.Println("DB Reader exiting due to signal, finishing read")
						return nil
					default:
						item := it.Item()
						value, verr := item.ValueCopy(nil)
						if verr != nil {
							return fmt.Errorf("read value: %w", verr)
						}
						if updateTS {
							newValue, verr := updateTimestamp(value)
							if verr != nil {
								return fmt.Errorf("update timestamp: %w", verr)
							}
							value = newValue
						}
						select {
						case dataChan <- value:
						case <-ctx.Done():
							return nil
						}
						count++
						recordsThisPass++
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("DB view: %w", err)
			}
		}
		if !loop {
			break
		}
		if recordsThisPass == 0 {
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return nil
			case <-timer.C:
			}
		}
	}
	log.Printf("DB Reader read %d payloads from the database\n", count)
	return nil
}

// RunCtx replays netflow packets from a db with an external context.
// Cancelling ctx stops all workers cleanly. In non-loop mode, the function
// returns when all packets have been sent. Use Run() for CLI usage where
// OS signal handling is desired.
func RunCtx(ctx context.Context, server string, port int, delay int, dbdir string, loop bool, workers int, updateTS bool, verbose bool) error {
	dataChan := make(chan []byte, 1024)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return dbReader(egCtx, dbdir, dataChan, loop, updateTS, verbose)
	})

	for w := 1; w <= workers; w++ {
		eg.Go(func() error {
			return worker(w, egCtx, server, port, delay, loop, dataChan)
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("replay: %w", err)
	}

	if ctx.Err() == nil {
		log.Printf("\nReplay complete, shutting down...\n")
	}
	return nil
}

// Run Replay. Kicks off the replay of netflow packets from a db.
// It sets up OS signal handling (SIGINT/SIGTERM) for clean shutdown.
// Use RunCtx() when you need to control the lifecycle via context.
func Run(server string, port int, delay int, dbdir string, loop bool, workers int, updateTS bool, verbose bool) {
	mgr := lifecycle.New()
	defer mgr.Cancel()

	_ = mgr.SetupSignalHandler()

	if err := RunCtx(mgr.Context(), server, port, delay, dbdir, loop, workers, updateTS, verbose); err != nil {
		log.Printf("Replay error: %v", err)
	}
}
