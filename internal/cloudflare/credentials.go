// Keep the signaling credential API wire compatible while minting coturn TURN REST
// credentials locally, without a Cloudflare account or request.
package cloudflare

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type CredentialsClient struct {
	secret   string
	urls     []string
	lifetime time.Duration
}

func NewCredentialsClient(_, _ string, lifetime time.Duration) *CredentialsClient {
	return &CredentialsClient{secret: os.Getenv("TURN_SECRET"), urls: strings.Split(os.Getenv("TURN_URLS"), ","), lifetime: lifetime}
}

func (c *CredentialsClient) Run(context.Context) {}

func (c *CredentialsClient) GetCredentials(context.Context) (*Credentials, error) {
	if len(c.secret) < 32 || c.lifetime < time.Minute || len(c.urls) == 0 {
		return nil, errors.New("coturn credentials are not configured")
	}
	for _, url := range c.urls {
		if !strings.HasPrefix(url, "turn:") && !strings.HasPrefix(url, "turns:") {
			return nil, errors.New("invalid coturn URL")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	username := strconv.FormatInt(time.Now().Add(c.lifetime).Unix(), 10) + ":" + hex.EncodeToString(nonce[:])
	// TURN REST authentication specifies HMAC-SHA1 (coturn's use-auth-secret).
	mac := hmac.New(sha1.New, []byte(c.secret))
	mac.Write([]byte(username))
	return &Credentials{URL: c.urls, Username: username, Credential: base64.StdEncoding.EncodeToString(mac.Sum(nil)), Lifetime: int(c.lifetime / time.Second)}, nil
}
