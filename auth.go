package clinicapi

import (
	"errors"
	"net/http"
	"strings"
)

// AuthProvider applies authentication to an outgoing Clinic API request.
type AuthProvider interface {
	Apply(req *http.Request) error
}

// AuthProviderFunc turns a function into an AuthProvider.
type AuthProviderFunc func(req *http.Request) error

// Apply applies the wrapped function to req.
func (f AuthProviderFunc) Apply(req *http.Request) error {
	if f == nil {
		return nil
	}
	return f(req)
}

// BearerTokenAuthProvider applies a static bearer token to requests.
type BearerTokenAuthProvider struct {
	Token string
}

// Apply sets the Authorization header on req.
func (p BearerTokenAuthProvider) Apply(req *http.Request) error {
	if req == nil {
		return errors.New("request is nil")
	}
	token := strings.TrimSpace(p.Token)
	if token == "" {
		return errors.New("bearer token is empty")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// StaticBearerToken returns an AuthProvider backed by a static bearer token.
func StaticBearerToken(token string) AuthProvider {
	return BearerTokenAuthProvider{Token: token}
}
