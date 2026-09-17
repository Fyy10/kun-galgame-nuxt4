package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer = "https://account.nextmoe.com"
	testKid    = "test-kid"
	appClient  = "kungal-app"
)

type opFixture struct {
	key     *ecdsa.PrivateKey
	fetches atomic.Int32
	url     string
}

func newOP(t *testing.T) *opFixture {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	op := &opFixture{key: key}
	pub, err := key.PublicKey.ECDH()
	if err != nil {
		t.Fatal(err)
	}
	point := pub.Bytes()
	b64 := base64.RawURLEncoding.EncodeToString
	body, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": testKid,
		"x": b64(point[1:33]), "y": b64(point[33:]),
	}}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		op.fetches.Add(1)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	op.url = srv.URL
	return op
}

func validClaims() AccessClaims {
	now := time.Now()
	return AccessClaims{
		ID:        1207,
		Name:      "kun",
		Roles:     []string{"user"},
		SiteRoles: []string{"moderator"},
		ClientID:  appClient,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "8b0c5a9e-uuid",
			Audience:  jwt.ClaimStrings{"www.kungal.com"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}
}

func (op *opFixture) sign(t *testing.T, claims AccessClaims, header map[string]any) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = testKid
	tok.Header["typ"] = "at+jwt"
	for k, v := range header {
		if v == nil {
			delete(tok.Header, k)
		} else {
			tok.Header[k] = v
		}
	}
	raw, err := tok.SignedString(op.key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestVerifyAcceptsAnAllowedAppToken(t *testing.T) {
	op := newOP(t)
	v := NewAccessTokenVerifier(NewJWKS(op.url), testIssuer, []string{"forum", appClient})

	for range 3 {
		claims, err := v.Verify(context.Background(), op.sign(t, validClaims(), nil))
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if claims.ID != 1207 || claims.ClientID != appClient || claims.SiteRoles[0] != "moderator" {
			t.Fatalf("claims = %+v", claims)
		}
	}
	if n := op.fetches.Load(); n != 1 {
		t.Errorf("JWKS fetched %d times for one kid, want 1", n)
	}
}

func TestVerifyRejects(t *testing.T) {
	op := newOP(t)
	v := NewAccessTokenVerifier(NewJWKS(op.url), testIssuer, []string{appClient})

	otherClient := validClaims()
	otherClient.ClientID = "some-third-party"
	wrongIssuer := validClaims()
	wrongIssuer.Issuer = "http://oauth:9277"
	expired := validClaims()
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	noExpiry := validClaims()
	noExpiry.ExpiresAt = nil
	noUser := validClaims()
	noUser.ID = 0

	hs256, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims()).SignedString([]byte("shared-secret"))

	for _, tc := range []struct {
		why string
		raw string
	}{
		{"a client outside the allow-list, though aud names this site", op.sign(t, otherClient, nil)},
		{"the internal issuer instead of the public one", op.sign(t, wrongIssuer, nil)},
		{"expired", op.sign(t, expired, nil)},
		{"no exp", op.sign(t, noExpiry, nil)},
		{"no integer user id", op.sign(t, noUser, nil)},
		{"an id_token-shaped typ", op.sign(t, validClaims(), map[string]any{"typ": "JWT"})},
		{"no kid", op.sign(t, validClaims(), map[string]any{"kid": nil})},
		{"HS256 from before the asymmetric flip", hs256},
		{"garbage", "not.a.jwt"},
	} {
		if _, err := v.Verify(context.Background(), tc.raw); err == nil {
			t.Errorf("%s: accepted", tc.why)
		}
	}
}

func TestVerifyReportsAnUnreachableOP(t *testing.T) {
	op := newOP(t)
	raw := op.sign(t, validClaims(), nil)
	v := NewAccessTokenVerifier(NewJWKS("http://127.0.0.1:1/oauth/jwks"), testIssuer, []string{appClient})

	_, err := v.Verify(context.Background(), raw)
	if !errors.Is(err, ErrKeysUnavailable) {
		t.Fatalf("err = %v, want ErrKeysUnavailable so the caller answers 5xx, not a logout-inducing 401", err)
	}
}
