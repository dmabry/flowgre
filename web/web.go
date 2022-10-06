// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type Worker struct {
	Name      string
	Completed bool
	Due       time.Time
}

type Workers []Worker

type Health struct {
	Status  string
	Message string
}

func RunWebServer(ip string, port int, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	listenAddr := ip + ":" + strconv.Itoa(port)
	log.Printf("Starting Web server %s\n", listenAddr)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", HealthHandler)
	router.HandleFunc("/health", HealthHandler)
	//router.HandleFunc("/workers", TodoIndex)
	//router.HandleFunc("/workers/{workerId}", TodoShow)
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
	health := Health{
		Status:  "OK",
		Message: "Everything is OK!",
	}

	json.NewEncoder(w).Encode(health)
}

/*
func TodoIndex(w http.ResponseWriter, r *http.Request) {
	todos := Todos{
		Todo{Name: "Write presentation"},
		Todo{Name: "Host meetup"},
	}

	json.NewEncoder(w).Encode(todos)
}

func TodoShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	todoId := vars["todoId"]
	fmt.Fprintf(w, "Todo show: %s\n", todoId)
}
*/
