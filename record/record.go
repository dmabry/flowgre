// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Record is used to take netflow packets off the wire and store them in a bbolt database

package record

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/dmabry/flowgre/ipfix"
	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/stats"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/sync/errgroup"
)

const udpMaxBufferSize = 65507

// netIngest is used to pull packets off the wire and put the byte payload on the data chan
func netIngest(ctx context.Context, wg *sync.WaitGroup, ip string, port int, data chan<- []byte, verbose bool) {
	defer wg.Done()
	if err := runNetIngest(ctx, ip, port, data, verbose); err != nil {
		log.Printf("Packet ingest error: %v", err)
	}
}

func runNetIngest(ctx context.Context, ip string, port int, data chan<- []byte, verbose bool) error {
	// Create UDP listener and setup db to catch files
	listenIP := net.ParseIP(ip)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: listenIP, Port: port})
	if err != nil {
		return fmt.Errorf("listen on %s:%d: %w", ip, port, err)
	}
	log.Printf("Listening on %s:%d", ip, port)
	defer conn.Close()
	stopCancelWakeup := context.AfterFunc(ctx, func() {
		_ = conn.SetReadDeadline(time.Now())
	})
	defer stopCancelWakeup()
	// Start the loop and check context for done, otherwise listen for packets
	for {
		select {
		case <-ctx.Done():
			log.Println("Packet ingest exiting due to signal")
			return nil
		default:
			payload := make([]byte, udpMaxBufferSize)
			timeout := time.Now().Add(5 * time.Second)
			err := conn.SetReadDeadline(timeout)
			if err != nil {
				return fmt.Errorf("set read deadline: %w", err)
			}
			length, fromIP, err := conn.ReadFromUDP(payload)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("read UDP packet: %w", err)
			}
			payload = payload[:length]
			if verbose {
				log.Printf("Packet Received from %s with size of %d", fromIP.String(), length)
			}
			// Send payload to the data channel
			select {
			case data <- payload:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// dbIngest pulls byte payload off the data chan and puts them in the bbolt database
func dbIngest(ctx context.Context, wg *sync.WaitGroup, dbdir string, data <-chan []byte, verbose bool) {
	defer wg.Done()
	if err := runDBIngest(ctx, dbdir, data, verbose); err != nil {
		log.Printf("Database ingest error: %v", err)
	}
}

// flowsBucket is the bbolt bucket holding recorded flow payloads. Records are
// keyed by big-endian uint32 ID so cursor order matches record order.
var flowsBucket = []byte("flows")

// openRecordDB opens the recorded-flows database file at path, creating it if
// needed, and ensures the flows bucket exists.
func openRecordDB(path string) (*bolt.DB, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil, fmt.Errorf("%s is a directory: flowgre stores recorded flows in a single bbolt database file (older releases used a badger directory)", path)
	}
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(flowsBucket)
		return err
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("create flows bucket: %w", err)
	}
	return db, nil
}

func nextRecordID(db *bolt.DB) (uint32, error) {
	var next uint32 = 1
	err := db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(flowsBucket).Cursor()
		if k, _ := c.Last(); k != nil {
			if len(k) != 4 {
				return fmt.Errorf("unexpected record key length %d", len(k))
			}
			last := binary.BigEndian.Uint32(k)
			if last == ^uint32(0) {
				return fmt.Errorf("record database key space exhausted")
			}
			next = last + 1
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("find last record key: %w", err)
	}
	return next, nil
}

func runDBIngest(ctx context.Context, dbdir string, data <-chan []byte, verbose bool) (retErr error) {
	db, err := openRecordDB(dbdir)
	if err != nil {
		return fmt.Errorf("open database %s: %w", dbdir, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close database: %w", err))
		}
	}()
	log.Printf("Writing to database %s\n", dbdir)
	nextID, err := nextRecordID(db)
	if err != nil {
		return err
	}
	// Start the loop
	for {
		// Check to see if context is done and return, otherwise pull payloads and write
		select {
		case <-ctx.Done():
			log.Println("Database ingest exiting due to signal")
			return nil
		case payload, ok := <-data:
			if !ok {
				return nil
			}
			key := make([]byte, 4)
			binary.BigEndian.PutUint32(key, nextID)
			err := db.Update(func(tx *bolt.Tx) error {
				return tx.Bucket(flowsBucket).Put(key, payload)
			})
			if err != nil {
				return fmt.Errorf("write record %d: %w", nextID, err)
			}
			if nextID == ^uint32(0) {
				return fmt.Errorf("record database key space exhausted")
			}
			nextID++
		}
	}
}

// parseFlow validates that the payload received is valid NetFlow v9 or IPFIX v10
func parseFlow(ctx context.Context, wg *sync.WaitGroup, parseChan <-chan []byte, dataChan chan<- []byte, verbose bool) {
	defer wg.Done()
	_ = runParseFlow(ctx, parseChan, dataChan, verbose)
}

func runParseFlow(ctx context.Context, parseChan <-chan []byte, dataChan chan<- []byte, verbose bool) error {
	// Prep the loop
	rStats := stats.RecordStat{
		ValidCount:   0,
		InvalidCount: 0,
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	// Start the loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Flow parser exiting due to signal")
			return nil
		case payload, ok := <-parseChan:
			if !ok {
				return nil
			}
			// Decode first uint16 and see if it is a version 9 (NetFlow) or 10 (IPFIX)
			ok, err := netflow.IsValidNetFlow(payload, 9)
			if err != nil {
				// Try IPFIX
				ok, err = ipfix.IsValidIPFIX(payload)
				if err != nil {
					if verbose {
						log.Printf("Skipping packet due to issue parsing: %v", err)
					}
					rStats.IncrInvalid()
					continue
				}
			}
			if ok {
				// Valid NetFlow v9 or IPFIX v10 Packet send it on
				rStats.IncrValid()
				select {
				case dataChan <- payload:
				case <-ctx.Done():
					return nil
				}
			} else {
				// Not a valid flow Packet... skip
				rStats.IncrInvalid()
			}
		case <-ticker.C:
			log.Printf("Flow Packets: %d Ignored Packets: %d ",
				rStats.LoadValid(), rStats.LoadInvalid())
		}
	}
}

// RunCtx starts the recording process with an external context.
// Cancelling ctx stops all workers cleanly. Use Run() for CLI usage
// where OS signal handling is desired.
func RunCtx(ctx context.Context, ip string, port int, dbdir string, verbose bool) error {
	dataChan := make(chan []byte, 1024)
	parseChan := make(chan []byte, 1024)
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error { return runNetIngest(egCtx, ip, port, parseChan, verbose) })
	eg.Go(func() error { return runParseFlow(egCtx, parseChan, dataChan, verbose) })
	eg.Go(func() error { return runDBIngest(egCtx, dbdir, dataChan, verbose) })
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("record: %w", err)
	}
	return nil
}

// Run Record. Kicks off the recording process.
// It sets up OS signal handling (SIGINT/SIGTERM) for clean shutdown.
// Use RunCtx() when you need to control the lifecycle via context.
func Run(ip string, port int, dbdir string, verbose bool) {
	mgr := lifecycle.New()
	defer mgr.Cancel()

	// Setup signal handling BEFORE starting workers to avoid race
	_ = mgr.SetupSignalHandler()

	if err := RunCtx(mgr.Context(), ip, port, dbdir, verbose); err != nil {
		log.Printf("Record error: %v", err)
	}

	mgr.Wait()
}
