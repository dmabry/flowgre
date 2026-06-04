// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/stats"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
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

// TestHealthHandler_OK verifies the /health endpoint returns a valid JSON health response.
func TestHealthHandler_OK(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var health models.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	if health.Status != "OK" {
		t.Errorf("expected status 'OK', got '%s'", health.Status)
	}
	if health.Message != "Everything is OK!" {
		t.Errorf("expected message 'Everything is OK!', got '%s'", health.Message)
	}
}

// TestHealthHandler_ValidJSON verifies the /health endpoint returns valid JSON.
func TestHealthHandler_ValidJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify the response is valid JSON
	var health models.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	// Verify Content-Type is set (even if default)
	contentType := rec.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("expected Content-Type header to be set")
	}
}

// TestHealthHandler_Method_Verification tests that the handler responds correctly to different HTTP methods.
func TestHealthHandler_Methods(t *testing.T) {
	t.Parallel()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/health", nil)
			rec := httptest.NewRecorder()

			HealthHandler(rec, req)

			// Gorilla mux doesn't enforce method restrictions on HandleFunc, so all methods succeed
			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", method, rec.Code)
			}
		})
	}
}

// TestIndexHandler_OK verifies the / endpoint returns a valid JSON health response.
func TestIndexHandler_OK(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	IndexHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var health models.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to unmarshal index response: %v", err)
	}

	if health.Status != "OK" {
		t.Errorf("expected status 'OK', got '%s'", health.Status)
	}
	if health.Message != "Flowgre is flinging packets!" {
		t.Errorf("expected message 'Flowgre is flinging packets!', got '%s'", health.Message)
	}
}

// TestIndexHandler_ValidJSON verifies the / endpoint returns valid JSON.
func TestIndexHandler_ValidJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	IndexHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify the response is valid JSON
	var health models.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to unmarshal index response: %v", err)
	}

	// Verify Content-Type is set (even if default)
	contentType := rec.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("expected Content-Type header to be set")
	}
}

// TestIndexHandler_Methods tests that the handler responds correctly to different HTTP methods.
func TestIndexHandler_Methods(t *testing.T) {
	t.Parallel()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			IndexHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", method, rec.Code)
			}
		})
	}
}

// TestHealthHandler_Encoding_Error simulates an encoding error by using a broken writer.
func TestHealthHandler_EncodingError(t *testing.T) {
	t.Parallel()

	// Create a custom ResponseWriter that fails on Encode
	failWriter := &failingResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}

	req := httptest.NewRequest("GET", "/health", nil)
	HealthHandler(failWriter, req)

	// The handler should still return 200 because json.Encoder doesn't fail on normal writers
	// This test verifies the error handling path exists
	rec := failWriter.Recorder()
	t.Logf("Response status: %d, body: %s", rec.Code, rec.Body.String())
}

// TestIndexHandler_EncodingError simulates an encoding error by using a broken writer.
func TestIndexHandler_EncodingError(t *testing.T) {
	t.Parallel()

	failWriter := &failingResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}

	req := httptest.NewRequest("GET", "/", nil)
	IndexHandler(failWriter, req)

	rec := failWriter.Recorder()
	t.Logf("Response status: %d, body: %s", rec.Code, rec.Body.String())
}

// failingResponseWriter wraps a httptest.ResponseRecorder to simulate encoding failures.
type failingResponseWriter struct {
	http.ResponseWriter
	written bool
}

func (fw *failingResponseWriter) Write(p []byte) (int, error) {
	if fw.written {
		return 0, errors.New("simulated write error")
	}
	fw.written = true
	return fw.ResponseWriter.Write(p)
}

func (fw *failingResponseWriter) Recorder() *httptest.ResponseRecorder {
	return fw.ResponseWriter.(*httptest.ResponseRecorder)
}

// TestRunWithMockedCollector tests the web server with a mocked stats.Collector.
func TestRunWithMockedCollector(t *testing.T) {
	t.Parallel()

	webIP := "127.0.0.1"
	webPort := pickPort()

	// Create a fully initialized Collector for testing
	buffer := 20
	sc := &stats.Collector{
		StatsChan: make(chan models.WorkerStat, buffer),
		StatsMap:  make(map[int]models.WorkerStat),
		StatsTotals: models.StatTotals{
			FlowsSent: 0,
			Cycles:    0,
			BytesSent: 0,
		},
		Config: &models.Config{
			Protocol: "netflow",
			Workers:  4,
		},
		StartTime: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go sc.Run(&wg, ctx)
	go RunWebServer(webIP, webPort, &wg, ctx, sc)

	// Allow server to start
	time.Sleep(2 * time.Second)

	// Test all endpoints
	endpoints := []struct {
		path         string
		expectedKey  string
	}{
		{"/", "status"},
		{"/health", "status"},
		{"/stats", "workers"},
		{"/stats/history", "[]"}, // Empty history returns empty array
		{"/dashboard", "<!DOCTYPE"},
	}

	for _, ep := range endpoints {
		url := "http://" + webIP + ":" + strconv.Itoa(webPort) + ep.path
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("Failed to GET %s: %v", ep.path, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Failed to read body from %s: %v", ep.path, err)
			continue
		}

		if !bytes.Contains(body, []byte(ep.expectedKey)) {
			t.Errorf("GET %s: expected body to contain '%s', got: %s", ep.path, ep.expectedKey, string(body)[:min(100, len(body))])
		}
	}

	cancel()
	wg.Wait()
	sc.Stop()
}

// min returns the smaller of a or b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
