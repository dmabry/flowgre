// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dmabry/flowgre/models"
	"github.com/dmabry/flowgre/stats"
	"golang.org/x/crypto/bcrypt"
)

const passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

// GenerateRandomPassword generates a cryptographically secure random password of the given length.
func GenerateRandomPassword(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordCharset))))
		if err != nil {
			return "", fmt.Errorf("generate random byte: %w", err)
		}
		result[i] = passwordCharset[idx.Int64()]
	}
	return string(result), nil
}

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
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// isLoopback returns true if the IP is a loopback address (127.0.0.1, ::1).
// Wildcard addresses (0.0.0.0, ::) are NOT considered loopback.
func isLoopback(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback()
}

// ValidateWebBinding checks that the web server binding is safe.
// Non-loopback addresses require TLS certificate and key files.
func ValidateWebBinding(ip, tlsCert, tlsKey string) error {
	if (tlsCert == "") != (tlsKey == "") {
		return fmt.Errorf("TLS certificate and key must be provided together")
	}
	useTLS := tlsCert != "" && tlsKey != ""
	if !useTLS && !isLoopback(ip) {
		return fmt.Errorf("binding web server to non-loopback address %s requires TLS (--tls-cert and --tls-key)", ip)
	}
	if useTLS {
		if _, err := os.Stat(tlsCert); err != nil {
			return fmt.Errorf("TLS certificate file: %w", err)
		}
		if _, err := os.Stat(tlsKey); err != nil {
			return fmt.Errorf("TLS key file: %w", err)
		}
	}
	return nil
}

// RunWebServer is used to start the web server goroutine.
// If tlsCert and tlsKey are empty, the server uses plain HTTP and requires
// a loopback bind address. Non-loopback addresses require TLS to protect
// credentials in transit.
func RunWebServer(ip string, port int, wg *sync.WaitGroup, ctx context.Context, sc *stats.Collector, username, hashedPassword, tlsCert, tlsKey string) {
	defer wg.Done()

	// Enforce loopback-only for non-TLS to protect credentials
	useTLS := tlsCert != "" && tlsKey != ""
	if !useTLS && !isLoopback(ip) {
		log.Printf("Refusing to start web server on non-loopback address %s without TLS. "+
			"Provide --tls-cert and --tls-key, or bind to 127.0.0.1.\n", ip)
		return
	}

	listenAddr := ip + ":" + strconv.Itoa(port)
	if useTLS {
		log.Printf("Starting Web server with TLS on %s\n", listenAddr)
	} else {
		log.Printf("Starting Web server on %s\n", listenAddr)
	}

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

	serverErr := make(chan error, 1)
	go func() {
		if useTLS {
			srv.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
			serverErr <- srv.ListenAndServeTLS(tlsCert, tlsKey)
		} else {
			serverErr <- srv.ListenAndServe()
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Printf("Web server Exiting due to signal\n")
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Web server error: %v\n", err)
		}
	}

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
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Web server had an issue: %v\n", err)
	}
}

// IndexHandler is used to produce a similar health payload.  static for now.
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	health := models.Health{
		Status:  "OK",
		Message: "Flowgre is flinging packets!",
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Web server had an issue: %v\n", err)
	}
}
