package internal

import (
	"net/http"
	"os"
	"strings"
	"sync"
)

// Caddy overwrites X-Real-IP, and the upstream listens only on loopback.
// These caps bound the test service independently of client-origin validation.
func SelfhostGuard(next http.Handler) http.Handler {
	var mu sync.Mutex
	counts := make(map[string]int)
	total := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/signaling" {
			next.ServeHTTP(w, r)
			return
		}
		allowed := false
		for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
			if origin != "" && r.Header.Get("Origin") == origin {
				allowed = true
			}
		}
		if !allowed {
			http.Error(w, "origin forbidden", http.StatusForbidden)
			return
		}
		ip := r.Header.Get("X-Real-IP")
		mu.Lock()
		if total >= 128 || counts[ip] >= 16 {
			mu.Unlock()
			http.Error(w, "connection limit", http.StatusTooManyRequests)
			return
		}
		total++
		counts[ip]++
		mu.Unlock()
		defer func() {
			mu.Lock()
			total--
			counts[ip]--
			if counts[ip] == 0 {
				delete(counts, ip)
			}
			mu.Unlock()
		}()
		next.ServeHTTP(w, r)
	})
}
