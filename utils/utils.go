// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Util funcs used throughout Flowgre

package utils

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dmabry/flowgre/models"
	"io"
	"log"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

// Constant used for generating random strings.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Constants used for calculating byte sizing for output
const (
	sizeKB = uint64(1 << (10 * 1))
	sizeMB = uint64(1 << (10 * 2))
	sizeGB = uint64(1 << (10 * 3))
)

// RandStringBytes Generates a random string of given length
func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[CryptoRandomNumber(int64(len(letterBytes)))]
	}
	return string(b)
}

func CryptoRandomNumber(max int64) int64 {
	n, err := crand.Int(crand.Reader, big.NewInt(max))
	if err != nil {
		panic(fmt.Errorf("crypto number failed to read bytes %v", err))
	}
	return n.Int64()
}

// BinaryDecoder decodes the given payload from a binary stream and puts it in dest
func BinaryDecoder(payload io.Reader, dests ...interface{}) error {
	for _, dest := range dests {
		err := binary.Read(payload, binary.BigEndian, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateRand16 Generates random uint16 num within the given max
func GenerateRand16(max int) uint16 {
	return uint16(CryptoRandomNumber(int64(max)))
}

// IPto32 Converts given IP string to uint32 representation
func IPto32(s string) uint32 {
	ip := net.ParseIP(s)
	return binary.BigEndian.Uint32(ip.To4())
}

// GenerateRand32 Generates a random uint32 within the given max
func GenerateRand32(max int) uint32 {
	// return uint32(rand.Intn(max))
	return uint32(CryptoRandomNumber(int64(max)))
}

// RandomNum Generates a random number between the given min and max
func RandomNum(min, max int) int {
	return int(CryptoRandomNumber(int64(max-min))) + min
}

// ToBytes Converts a given interface to a byte stream.
// Not used currently, but handy to have for later maybe.  Did not work for encoding Netflow packets as it
// encoded field names.
func ToBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RandomIP Picks a random IP from the given CIDR
// TODO: Better error handling needed
func RandomIP(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Println("[ERROR] Parsing CIDR", cidr, " failed. error: ", err)
	}
	ipMin := ipNet.IP
	ipMax, _ := GetLastIP(ipNet)
	ipMinNum := IPToNum(ipMin)
	ipMaxNum := IPToNum(ipMax)
	rand.Seed(time.Now().UnixNano())
	randIPNum := uint32(rand.Int31n(int32(ipMaxNum-ipMinNum)) + int32(ipMinNum)) //#nosec This just used for IP generation
	randIP := NumToIP(randIPNum)
	//check if in range
	if ipNet.Contains(randIP) {
		return randIP, nil
	}
	return nil, errors.New("random IP broken")
}

// GetLastIP Gets the last IP of a given Network
func GetLastIP(ipNet *net.IPNet) (net.IP, error) {
	ip := make(net.IP, len(ipNet.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipNet.IP.To4())|^binary.BigEndian.Uint32(ipNet.Mask))
	return ip, nil
}

// IPToNum Converts given IP to uint32
func IPToNum(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// NumToIP Converts given uint32 to IP
func NumToIP(num uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, num)
	return ip
}

// SendPacket Takes a given byte stream and puts on the wire towards the given host
func SendPacket(conn *net.UDPConn, addr *net.UDPAddr, data bytes.Buffer, verbose bool) (int, error) {
	n, err := conn.WriteTo(data.Bytes(), addr)
	if err != nil {
		log.Fatal("Write:", err)
		return 0, err
	}
	if verbose {
		fmt.Println("Sent", n, "bytes", conn.LocalAddr(), "->", addr)
	}
	return n, err
}

// GatherStats is the function that is called to read all items on the statsChan
func GatherStats(statsChan <-chan models.WorkerStat) (stats models.WorkerStats) {
	var workerStats models.WorkerStats
	select {
	case stat, ok := <-statsChan:
		if ok {
			workerStats = append(workerStats, stat)
			// pull all items off the channel
			for item := range statsChan {
				workerStats = append(workerStats, item)
			}
		} else {
			log.Println("Stats Channel Closed!")
		}
	default:
		// nothing on channel
	}
	return workerStats
}

type StatCollector struct {
	StatsMap  map[int]models.WorkerStat
	StatsChan chan models.WorkerStat
}

func (sc *StatCollector) Run(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	// check the stats channel every 5 seconds
	limiter := time.Tick(time.Second * time.Duration(5))
	// map for aggregated for web output
	//statsMap := make(map[int]models.WorkerStat)
	log.Println("Stats Collector started")
	sizeLabel := "bytes"
	var sizeOut uint64 = 0
	for {
		select {
		case stat, ok := <-sc.StatsChan:
			if ok {
				switch {
				case stat.BytesSent >= sizeKB && stat.BytesSent <= sizeMB:
					sizeLabel = "KB"
					sizeOut = stat.BytesSent / sizeKB
				case stat.BytesSent >= sizeMB && stat.BytesSent <= sizeGB:
					sizeLabel = "MB"
					sizeOut = stat.BytesSent / sizeMB
				case stat.BytesSent > sizeGB:
					sizeLabel = "GB"
					sizeOut = stat.BytesSent / sizeGB
				default:
					sizeOut = stat.BytesSent
				}
				log.Printf("Worker [%d] SourceID: %4d Cycles: %d Flows Sent: %d Bytes Sent: %d %s\n", stat.WorkerID, stat.SourceID, stat.Cycles, stat.FlowsSent, sizeOut, sizeLabel)
				sc.StatsMap[stat.WorkerID] = stat
			} else {
				log.Println("Stats Channel Closed!")
			}
		case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
			log.Printf("Stats Collector Exiting due to signal\n")
			return
		default:
			// nothing on channel
			<-limiter
		}
	}
}

func (sc *StatCollector) StatsHandler(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(sc.StatsMap)
	if err != nil {
		log.Fatalf("Web server had an issue: %v\n", err)
	}
}

func (sc *StatCollector) Stop() {
	close(sc.StatsChan)
}
