// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/stats"
	"golang.org/x/crypto/bcrypt"
)

// BasicAuthMiddleware provides HTTP Basic Authentication for web endpoints.
func BasicAuthMiddleware(username, hashedPassword string, realm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(auth, "Basic ") {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
			if err != nil {
				http.Error(w, "Bad Authorization", http.StatusBadRequest)
				return
			}

			parts := strings.SplitN(string(payload), ":", 2)
			if len(parts) != 2 {
				http.Error(w, "Bad Authorization", http.StatusBadRequest)
				return
			}

			// Constant-time comparison for username
			if !constantTimeCompare(parts[0], username) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// bcrypt comparison for password
			if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(parts[1])); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// constantTimeCompare provides constant-time string comparison to prevent timing attacks.
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// RunWebServer is used to start the web server goroutine.
func RunWebServer(ip string, port int, wg *sync.WaitGroup, ctx context.Context, sc *stats.Collector, username, hashedPassword string) {
	defer wg.Done()
	listenAddr := ip + ":" + strconv.Itoa(port)
	log.Printf("Starting Web server %s\n", listenAddr)

	router := http.NewServeMux()

	// Wrap all handlers with basic auth middleware
	authMiddleware := BasicAuthMiddleware(username, hashedPassword, "FlowGre")

	router.Handle("/", authMiddleware(http.HandlerFunc(IndexHandler)))
	router.Handle("/health", authMiddleware(http.HandlerFunc(HealthHandler)))
	router.Handle("/stats", authMiddleware(http.HandlerFunc(sc.StatsHandler)))
	router.Handle("/stats/history", authMiddleware(http.HandlerFunc(sc.HistoryHandler)))
	router.Handle("/dashboard", authMiddleware(http.HandlerFunc(sc.DashboardHandler)))

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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
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
