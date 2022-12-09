// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package web

import (
	"context"
	"encoding/json"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/utils"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TestRun runs a test to verify functionality. TODO: Need to determine a sane test for this
func TestRun(t *testing.T) {
	t.Parallel()
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	// configure web server
	webIP := "127.0.0.1"
	webPort := 8080
	statusURL := "http://" + webIP + ":" + strconv.Itoa(webPort) + "/"
	statusExpected := "OK"
	buffer := 20
	// Start the StatsCollector
	sc := &utils.StatCollector{}
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
	go RunWebServer(webIP, webPort, wg, ctx, sc)
	// check that it is serving up status page
	time.Sleep(time.Second * 2)
	// do check
	resp, err := http.Get(statusURL)
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
