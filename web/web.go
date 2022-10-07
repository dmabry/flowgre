// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/utils"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"sync"
)

func RunWebServer(ip string, port int, wg *sync.WaitGroup, ctx context.Context, sc *utils.StatCollector) {
	defer wg.Done()
	listenAddr := ip + ":" + strconv.Itoa(port)
	log.Printf("Starting Web server %s\n", listenAddr)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", IndexHandler)
	router.HandleFunc("/health", HealthHandler)
	router.HandleFunc("/stats", sc.StatsHandler)
	go func() {
		err := http.ListenAndServe(listenAddr, router)
		if err != nil {
			fmt.Errorf("Issue starting web server! %v\n", err)
			log.Println(err.Error())
		}
	}()
	select {
	case <-ctx.Done(): //Caught the signal to be done.... time to wrap it up
		log.Printf("Web server Exiting due to signal\n")
		return
	default:
	}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := models.Health{
		Status:  "OK",
		Message: "Everything is OK!",
	}
	json.NewEncoder(w).Encode(health)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	health := models.Health{
		Status:  "OK",
		Message: "Flowgre is flinging packets!",
	}
	json.NewEncoder(w).Encode(health)
}
