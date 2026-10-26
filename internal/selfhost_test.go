package internal

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestSelfhostOriginAndConnectionLimit(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://saltedcaramelgames.com:8443,http://127.0.0.1:5180")
	entered := make(chan struct{}, 16)
	release := make(chan struct{})
	handler := SelfhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release }))
	request := func(origin string) *http.Request {
		r := httptest.NewRequest("GET", "/v0/signaling", nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("X-Real-IP", "192.0.2.1")
		return r
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request("https://another.example"))
	if w.Code != http.StatusForbidden {
		t.Fatal("wrong origin accepted")
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.ServeHTTP(httptest.NewRecorder(), request("https://saltedcaramelgames.com:8443"))
		}()
		<-entered
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, request("https://saltedcaramelgames.com:8443"))
	if w.Code != http.StatusTooManyRequests {
		t.Fatal("per-address limit not enforced")
	}
	close(release)
	wg.Wait()
}
