package oauth

import (
	"context"
	"crypto"
	"crypto/ecdh"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

var ErrKeysUnavailable = errors.New("oauth: signing keys unavailable")

type JWKS struct {
	url        string
	httpClient *http.Client
	minRefresh time.Duration

	mu        sync.RWMutex
	keys      map[string]crypto.PublicKey
	lastFetch time.Time
}

func NewJWKS(url string) *JWKS {
	return &JWKS{
		url:        url,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		minRefresh: 30 * time.Second,
	}
}

func (j *JWKS) Key(ctx context.Context, kid string) (crypto.PublicKey, error) {
	if k := j.lookup(kid); k != nil {
		return k, nil
	}
	if err := j.refetch(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeysUnavailable, err)
	}
	if k := j.lookup(kid); k != nil {
		return k, nil
	}
	return nil, fmt.Errorf("oauth: unknown kid %q", kid)
}

func (j *JWKS) lookup(kid string) crypto.PublicKey {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.keys[kid]
}

func (j *JWKS) refetch(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.keys != nil && time.Since(j.lastFetch) < j.minRefresh {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.url, nil)
	if err != nil {
		return err
	}
	resp, err := j.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	keys, err := parseJWKSet(raw)
	if err != nil {
		return err
	}
	j.keys = keys
	j.lastFetch = time.Now()
	return nil
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func parseJWKSet(raw []byte) (map[string]crypto.PublicKey, error) {
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, err
	}
	out := make(map[string]crypto.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kid == "" {
			continue
		}
		if pub, err := k.publicKey(); err == nil {
			out[k.Kid] = pub
		}
	}
	return out, nil
}

func (k jwk) publicKey() (crypto.PublicKey, error) {
	b64 := base64.RawURLEncoding
	switch k.Kty {
	case "EC":
		if k.Crv != "P-256" {
			return nil, fmt.Errorf("unsupported curve %q", k.Crv)
		}
		x, err := b64.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		y, err := b64.DecodeString(k.Y)
		if err != nil {
			return nil, err
		}
		point := append(append([]byte{0x04}, x...), y...)
		pub, err := ecdh.P256().NewPublicKey(point)
		if err != nil {
			return nil, err
		}
		der, err := x509.MarshalPKIXPublicKey(pub)
		if err != nil {
			return nil, err
		}
		return x509.ParsePKIXPublicKey(der)
	case "RSA":
		n, err := b64.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
		e, err := b64.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
	}
	return nil, fmt.Errorf("unsupported kty %q", k.Kty)
}
