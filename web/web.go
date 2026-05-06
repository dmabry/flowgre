// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/stats"
	"github.com/gorilla/mux"
)

// RunWebServer is used to start the web server goroutine.
func RunWebServer(ip string, port int, wg *sync.WaitGroup, ctx context.Context, sc *stats.Collector) {
	defer wg.Done()
	listenAddr := ip + ":" + strconv.Itoa(port)
	log.Printf("Starting Web server %s\n", listenAddr)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/health", HealthHandler)
	router.HandleFunc("/stats", sc.StatsHandler)
	router.HandleFunc("/dashboard", sc.DashboardHandler)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           router,
		ReadTimeout:       time.Second * 5,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 5,
		IdleTimeout:       time.Second * 5,
	}
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Issue starting web server! %v\n", err)
			return
		}
	}()
	<-ctx.Done() // Block until context is cancelled
	log.Printf("Web server Exiting due to signal\n")
	_ = srv.Shutdown(context.Background())
}

// HealthHandler is used to generate json payload for health.  static for now.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := models.Health{
		Status:  "OK",
		Message: "Everything is OK!",
	}
	err := json.NewEncoder(w).Encode(health)
	if err != nil {
		log.Printf("Web server had an issue: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// IndexHandler is used to produce a similar health payload.  static for now.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	health := models.Health{
		Status:  "OK",
		Message: "Flowgre is flinging packets!",
	}
	err := json.NewEncoder(w).Encode(health)
	if err != nil {
		log.Printf("Web server had an issue: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
