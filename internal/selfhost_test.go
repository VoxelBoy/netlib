package internal

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func signalingRequest(origin, ip string) *http.Request {
	r := probeRequest(origin, ip)
	r.Header.Set("Upgrade", "websocket")
	return r
}

// The browser's plain request asking whether a connection would be turned away.
func probeRequest(origin, ip string) *http.Request {
	r := httptest.NewRequest("GET", "/v0/signaling", nil)
	r.Header.Set("Origin", origin)
	r.Header.Set("X-Real-IP", ip)
	return r
}

func TestSelfhostOriginAndConnectionLimit(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://saltedcaramelgames.com:8443,http://127.0.0.1:5180")
	entered := make(chan struct{}, 16)
	release := make(chan struct{})
	handler := SelfhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release }))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, signalingRequest("https://another.example", "192.0.2.1"))
	if w.Code != http.StatusForbidden {
		t.Fatal("wrong origin accepted")
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.ServeHTTP(httptest.NewRecorder(), signalingRequest("https://saltedcaramelgames.com:8443", "192.0.2.1"))
		}()
		<-entered
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, signalingRequest("https://saltedcaramelgames.com:8443", "192.0.2.1"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("per-address limit not enforced")
	}
	close(release)
	wg.Wait()
}

func TestSelfhostWildcardOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://saltedcaramelgames.com:8443, https://*.crazygames.com")
	handler := SelfhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for origin, want := range map[string]int{
		"https://loot-hole-hjh.game-files.crazygames.com": http.StatusOK,
		"https://games.crazygames.com":                    http.StatusOK,
		"https://crazygames.com":                          http.StatusOK,
		"https://Loot-Hole.Game-Files.CrazyGames.com":     http.StatusOK,
		"https://saltedcaramelgames.com:8443":             http.StatusOK,
		"http://loot-hole.game-files.crazygames.com":      http.StatusForbidden,
		"https://loot-hole.crazygames.com:8443":           http.StatusForbidden,
		"https://evilcrazygames.com":                      http.StatusForbidden,
		"https://crazygames.com.example":                  http.StatusForbidden,
		"https://crazygames.com.evil.example":             http.StatusForbidden,
		"capacitor://app.crazygames.com":                  http.StatusForbidden,
		"https://saltedcaramelgames.com":                  http.StatusForbidden,
		"":                                                http.StatusForbidden,
		"null":                                            http.StatusForbidden,
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, signalingRequest(origin, "192.0.2.1"))
		if w.Code != want {
			t.Errorf("%q: %d, want %d", origin, w.Code, want)
		}
	}
	// A wildcard written anywhere but right after https:// allows nothing.
	t.Setenv("ALLOWED_ORIGINS", "https://games.*.com,*.crazygames.com,https://*.")
	handler = SelfhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, signalingRequest("https://games.crazygames.com", "192.0.2.1"))
	if w.Code != http.StatusForbidden {
		t.Fatal("a malformed wildcard allowed an origin")
	}
}

func TestSelfhostCapsFromEnvironment(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://*.crazygames.com")
	t.Setenv("SIGNALING_MAX_CONNECTIONS", "3")
	t.Setenv("SIGNALING_MAX_PER_ADDRESS", "2")
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	handler := SelfhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release }))
	origin := "https://loot-hole-hjh.game-files.crazygames.com"
	var wg sync.WaitGroup
	hold := func(ip string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.ServeHTTP(httptest.NewRecorder(), signalingRequest(origin, ip))
		}()
		<-entered
	}
	hold("192.0.2.1")
	hold("192.0.2.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, signalingRequest(origin, "192.0.2.1"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("per-address cap from the environment not enforced")
	}
	hold("192.0.2.2")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, signalingRequest(origin, "192.0.2.3"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("total cap from the environment not enforced")
	}
	close(release)
	wg.Wait()
}

func TestSelfhostProbeSaysWhetherAConnectionWouldBeTurnedAway(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://*.crazygames.com")
	t.Setenv("SIGNALING_MAX_CONNECTIONS", "1")
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	handler := SelfhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release }))
	origin := "https://loot-hole-hjh.game-files.crazygames.com"
	probe := func(origin string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, probeRequest(origin, "192.0.2.9"))
		return w
	}
	w := probe(origin)
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("room to spare: %d %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
	// Probes take no place: the one connection allowed still gets in.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.ServeHTTP(httptest.NewRecorder(), signalingRequest(origin, "192.0.2.1"))
	}()
	<-entered
	w = probe(origin)
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("full: %d %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w = probe("https://another.example"); w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("another origin: %d %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
	close(release)
	wg.Wait()
}
