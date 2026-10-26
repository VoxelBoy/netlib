package cloudflare

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCoturnCredentials(t *testing.T) {
	t.Setenv("TURN_SECRET", strings.Repeat("a", 64))
	t.Setenv("TURN_URLS", "turn:saltedcaramelgames.com:3478?transport=udp,turns:saltedcaramelgames.com:5349?transport=tcp")
	c := NewCredentialsClient("", "", 2*time.Hour)
	first, err := c.GetCredentials(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.GetCredentials(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Username == second.Username {
		t.Fatal("credentials must have independent allocation quotas")
	}
	expires, err := strconv.ParseInt(strings.Split(first.Username, ":")[0], 10, 64)
	if err != nil || expires < time.Now().Unix()+7198 || expires > time.Now().Unix()+7201 {
		t.Fatal("wrong credential expiry")
	}
	mac := hmac.New(sha1.New, []byte(strings.Repeat("a", 64)))
	mac.Write([]byte(first.Username))
	if first.Credential != base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatal("coturn HMAC mismatch")
	}
	if first.Lifetime != 7200 || len(first.URL.([]string)) != 2 {
		t.Fatal("missing lifetime or transport fallback")
	}
	t.Setenv("TURN_SECRET", "")
	if _, err = NewCredentialsClient("", "", time.Hour).GetCredentials(context.Background()); err == nil {
		t.Fatal("missing secret must fail closed")
	}
}
