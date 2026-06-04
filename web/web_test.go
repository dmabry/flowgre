// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/stats"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// pickPort tries 8080 first, then falls back to a random port > 1024.
func pickPort() int {
	// Try the default port first
	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err == nil {
		listener.Close()
		return 8080
	}
	// Pick a random port in the ephemeral range
	for i := 0; i < 5; i++ {
		port := 1024 + rand.Intn(64512)
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			listener.Close()
			return port
		}
	}
	// Last resort: let the OS pick
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("cannot find an available port")
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TestRun verifies that the web server starts, serves the dashboard, and returns valid JSON.
func TestRun(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	// configure web server — pick an available port
	webIP := "127.0.0.1"
	webPort := pickPort()
	statusURL := "http://" + webIP + ":" + strconv.Itoa(webPort) + "/"
	statusExpected := "OK"
	buffer := 20
	// Start the StatsCollector
	sc := &stats.Collector{}
	sc.StatsChan = make(chan models.WorkerStat, buffer)
	sc.StatsMap = make(map[int]models.WorkerStat)
	sc.StatsTotals = models.StatTotals{
		FlowsSent: 0,
		Cycles:    0,
		BytesSent: 0,
	}
	//sc.Config = config
	wg.Add(1)
	go sc.Run(wg, ctx)

	// run web server
	wg.Add(1)
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
	go RunWebServer(webIP, webPort, wg, ctx, sc, "admin", string(hashedPassword))
	// check that it is serving up status page
	time.Sleep(time.Second * 2)
	// do check with auth
	req, _ := http.NewRequest("GET", statusURL, nil)
	req.SetBasicAuth("admin", "testpass")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Response: %s", string(body))
	var result Response
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		t.Errorf("Error unmarshaling JSON from %s", statusURL)
	}
	if result.Status != statusExpected {
		t.Errorf("Error parsing status. got: %s expected: %s", result.Status, statusExpected)
	}
	cancel()
	wg.Wait()
	sc.Stop()
}
