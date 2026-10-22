package vodwebdav

import (
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// NewServer creates the VOD WebDAV listener. Non-loopback listeners must use
// TLS and HTTP Basic authentication; loopback remains available for native
// local mounts and can opt into either control independently.
func NewServer(addr, certFile, keyFile, username, password string, handler http.Handler) (*http.Server, error) {
	certFile = strings.TrimSpace(certFile)
	keyFile = strings.TrimSpace(keyFile)
	if strings.TrimSpace(username) == "" {
		username = ""
	}
	hasTLS := certFile != "" || keyFile != ""
	if hasTLS && (certFile == "" || keyFile == "") {
		return nil, fmt.Errorf("both TLS certificate and key are required")
	}
	if (username == "") != (password == "") {
		return nil, fmt.Errorf("both WebDAV username and password are required")
	}
	if !loopbackListenAddress(addr) {
		if !hasTLS {
			return nil, fmt.Errorf("non-loopback VOD WebDAV listeners require TLS")
		}
		if username == "" {
			return nil, fmt.Errorf("non-loopback VOD WebDAV listeners require authentication")
		}
		if len(password) < 16 {
			return nil, fmt.Errorf("non-loopback VOD WebDAV passwords must contain at least 16 bytes")
		}
	}
	if handler == nil {
		return nil, fmt.Errorf("WebDAV handler is required")
	}
	if username != "" {
		handler = basicAuthHandler(username, password, handler)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if hasTLS {
		server.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return server, nil
}

func loopbackListenAddress(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func basicAuthHandler(username, password string, next http.Handler) http.Handler {
	wantUser := []byte(username)
	wantPassword := []byte(password)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPassword, ok := r.BasicAuth()
		userMatch := ok && subtle.ConstantTimeCompare([]byte(gotUser), wantUser) == 1
		passwordMatch := ok && subtle.ConstantTimeCompare([]byte(gotPassword), wantPassword) == 1
		if !userMatch || !passwordMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="IPTV Tunerr VOD", charset="UTF-8"`)
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
