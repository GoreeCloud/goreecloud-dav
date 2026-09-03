package auth

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

type Principal struct {
	ID string
}

type Provider interface {
	Authenticate(*http.Request) (Principal, bool)
	Challenge(http.ResponseWriter)
}

type DevelopmentProvider struct {
	Username string
	Password string
}

func (p DevelopmentProvider) Authenticate(r *http.Request) (Principal, bool) {
	if p.Username != "" || p.Password != "" {
		username, password, ok := r.BasicAuth()
		if !ok || !constantTimeEqual(username, p.Username) || !constantTimeEqual(password, p.Password) {
			return Principal{}, false
		}
		return Principal{ID: p.Username}, true
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return Principal{}, false
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return Principal{}, false
	}
	return Principal{ID: "local"}, true
}

func (DevelopmentProvider) Challenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="GoreeCloud DAV Development", charset="UTF-8"`)
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type StaticProvider struct {
	PrincipalID string
}

func (p StaticProvider) Authenticate(*http.Request) (Principal, bool) {
	if strings.TrimSpace(p.PrincipalID) == "" {
		return Principal{}, false
	}
	return Principal{ID: p.PrincipalID}, true
}

func (StaticProvider) Challenge(http.ResponseWriter) {}
