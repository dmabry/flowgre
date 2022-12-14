// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package web

import (
	"context"
	"encoding/json"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/utils"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RunWebServer is used to start the web server goroutine
func RunWebServer(ip string, port int, wg *sync.WaitGroup, ctx context.Context, sc *utils.StatCollector) {
	defer wg.Done()
	listenAddr := ip + ":" + strconv.Itoa(port)
	log.Printf("Starting Web server %s\n", listenAddr)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/health", HealthHandler)
	router.HandleFunc("/stats", sc.StatsHandler)
	router.HandleFunc("/dashboard", sc.DashboardHandler)

	go func() {
		s := &http.Server{
			Addr:              listenAddr,
			Handler:           router,
			ReadTimeout:       time.Second * 5,
			ReadHeaderTimeout: time.Second * 5,
			WriteTimeout:      time.Second * 5,
			IdleTimeout:       time.Second * 5,
		}
		err := s.ListenAndServe()
		if err != nil {
			log.Fatalf("Issue starting web server! %v\n", err)
		}
	}()
	select {
	case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
		log.Printf("Web server Exiting due to signal\n")
		return
	default:
	}
}

// HealthHandler is used to generate json payload for health.  static for now.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := models.Health{
		Status:  "OK",
		Message: "Everything is OK!",
	}
	err := json.NewEncoder(w).Encode(health)
	if err != nil {
		log.Fatalf("Web server had an issue: %v\n", err)
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
		log.Fatalf("Web server had an issue: %v\n", err)
	}
}
