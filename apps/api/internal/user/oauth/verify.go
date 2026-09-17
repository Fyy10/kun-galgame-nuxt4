package oauth

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// sub is the user's UUID; the integer id shared across services is the custom
// id claim. aud is the client's site domain, so client identity is client_id.
type AccessClaims struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email,omitempty"`
	Scope     string   `json:"scope,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	SiteRoles []string `json:"site_roles,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	jwt.RegisteredClaims
}

type KeySource interface {
	Key(ctx context.Context, kid string) (crypto.PublicKey, error)
}

type AccessTokenVerifier struct {
	keys      KeySource
	issuer    string
	clientIDs map[string]struct{}
}

func NewAccessTokenVerifier(keys KeySource, issuer string, clientIDs []string) *AccessTokenVerifier {
	allowed := make(map[string]struct{}, len(clientIDs))
	for _, id := range clientIDs {
		allowed[id] = struct{}{}
	}
	return &AccessTokenVerifier{keys: keys, issuer: issuer, clientIDs: allowed}
}

func (v *AccessTokenVerifier) Verify(ctx context.Context, raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	keyfunc := func(t *jwt.Token) (any, error) {
		typ, _ := t.Header["typ"].(string)
		if !strings.EqualFold(typ, "at+jwt") && !strings.EqualFold(typ, "application/at+jwt") {
			return nil, fmt.Errorf("oauth: token typ %q is not an access token", typ)
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("oauth: token has no kid")
		}
		return v.keys.Key(ctx, kid)
	}
	_, err := jwt.ParseWithClaims(raw, claims, keyfunc,
		jwt.WithValidMethods([]string{"ES256", "RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	if _, ok := v.clientIDs[claims.ClientID]; !ok {
		return nil, fmt.Errorf("oauth: client %q may not call this API", claims.ClientID)
	}
	if claims.ID <= 0 {
		return nil, errors.New("oauth: token carries no user id")
	}
	return claims, nil
}
