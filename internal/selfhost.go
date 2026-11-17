package internal

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

// The connection caps unless SIGNALING_MAX_CONNECTIONS and SIGNALING_MAX_PER_ADDRESS say otherwise.
const (
	defaultMaxConnections = 128
	defaultMaxPerAddress  = 16
)

// originRule is one ALLOWED_ORIGINS entry: an exact origin ("https://example.com:8443"), or "https://*.example.com",
// which allows https on example.com and every host under it, on the default port (a portal's per-game frames:
// *.crazygames.com, whose sitelock guide checks the same).
type originRule struct {
	exact  string
	suffix string // the host after "*.", for a wildcard rule
}

func originRules(list string) []originRule {
	var rules []originRule
	for _, entry := range strings.Split(list, ",") {
		entry = strings.TrimSpace(entry)
		if host, ok := strings.CutPrefix(entry, "https://*."); ok {
			if host != "" && !strings.ContainsAny(host, "*/:") {
				rules = append(rules, originRule{suffix: strings.ToLower(host)})
			}
		} else if entry != "" {
			rules = append(rules, originRule{exact: entry})
		}
	}
	return rules
}

func (rule originRule) allows(origin string) bool {
	if rule.suffix == "" {
		return origin == rule.exact
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Port() != "" || u.Path != "" || u.User != nil || u.RawQuery != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == rule.suffix || strings.HasSuffix(host, "."+rule.suffix)
}

func envCap(name string, fallback int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return fallback
}

// Caddy overwrites X-Real-IP, and the upstream listens only on loopback.
// These caps bound the service independently of client-origin validation.
func SelfhostGuard(next http.Handler) http.Handler {
	rules := originRules(os.Getenv("ALLOWED_ORIGINS"))
	maxTotal := envCap("SIGNALING_MAX_CONNECTIONS", defaultMaxConnections)
	maxPerAddress := envCap("SIGNALING_MAX_PER_ADDRESS", defaultMaxPerAddress)
	var mu sync.Mutex
	counts := make(map[string]int)
	total := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/signaling" {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		allowed := false
		for _, rule := range rules {
			if rule.allows(origin) {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "origin forbidden", http.StatusForbidden)
			return
		}
		ip := r.Header.Get("X-Real-IP")
		mu.Lock()
		if total >= maxTotal || counts[ip] >= maxPerAddress {
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
