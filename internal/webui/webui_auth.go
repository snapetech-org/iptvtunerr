package webui

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxFailedLoginIPs = 1024
const maxLoginFormBytes = 8 << 10
const loginCSRFTokenTTL = 15 * time.Minute

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.renderLogin(w, r, http.StatusOK, "")
	case http.MethodPost:
		if s.loginBlocked(r) {
			s.renderLogin(w, r, http.StatusTooManyRequests, "Too many login attempts. Wait a few minutes and try again.")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxLoginFormBytes)
		if err := r.ParseForm(); err != nil {
			s.renderLogin(w, r, http.StatusBadRequest, "Invalid login form.")
			return
		}
		if !s.validLoginCSRFToken(r.Form.Get("csrf_token")) {
			s.renderLogin(w, r, http.StatusForbidden, "Invalid or expired login form. Refresh the page and try again.")
			return
		}
		user := strings.TrimSpace(r.Form.Get("username"))
		pass := r.Form.Get("password")
		if len(user) > 256 || len(pass) > 1024 {
			s.renderLogin(w, r, http.StatusBadRequest, "Invalid login form.")
			return
		}
		if !s.validCredentials(user, pass) {
			s.noteFailedLogin(r)
			s.recordActivity("auth", "login_failed", "Deck login failed.", map[string]interface{}{"username": user})
			s.renderLogin(w, r, http.StatusUnauthorized, "Wrong username or password.")
			return
		}
		s.clearFailedLogins(r)
		if _, err := s.startSession(w, r); err != nil {
			http.Error(w, "unable to create session", http.StatusInternalServerError)
			return
		}
		s.recordActivity("auth", "login", "Deck session opened.", map[string]interface{}{"username": user})
		http.Redirect(w, r, safeOIDCReturnTo(r.Form.Get("next")), http.StatusSeeOther)
	default:
		writeMethodNotAllowedPlain(w, http.MethodGet, http.MethodHead, http.MethodPost)
	}
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowedPlain(w, http.MethodPost)
		return
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessionMu.Lock()
		delete(s.sessions, cookie.Value)
		s.sessionMu.Unlock()
	}
	s.recordActivity("auth", "logout", "Deck session closed.", map[string]interface{}{})
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secureCookie(r),
	})
	s.clearAllOIDCStateCookies(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) renderLogin(w http.ResponseWriter, r *http.Request, status int, errText string) {
	s.ensureTemplates()
	csrfToken, err := s.newLoginCSRFToken()
	if err != nil {
		http.Error(w, "unable to prepare login form", http.StatusInternalServerError)
		return
	}
	provider := oidcProviderPresentation(s.oidcSettings.Provider)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if status > 0 {
		w.WriteHeader(status)
	}
	_ = s.loginTmpl.Execute(w, map[string]interface{}{
		"Version":               fallbackVersion(s.Version),
		"Now":                   time.Now().UTC().Format(time.RFC3339),
		"Error":                 errText,
		"LoginCSRF":             csrfToken,
		"User":                  s.deckSettingsReport().AuthUser,
		"DefaultPassword":       s.deckSettingsReport().AuthDefaultPassword,
		"GeneratedPassword":     s.generatedPass,
		"ShowGeneratedPassword": s.generatedPass != "" && !s.AllowLAN,
		"OIDCEnabled":           s.oidcSettings.enabled(),
		"OIDCLoginURL":          "/auth/oidc/login?next=" + url.QueryEscape(safeOIDCReturnTo(r.URL.Query().Get("next"))),
		"OIDCProviderLabel":     provider.Label,
		"OIDCProviderIcon":      provider.Icon,
		"OIDCProviderIconsURL":  "/oidc-provider-icons.svg",
		"OIDCProviderSetupURL":  provider.SetupURL,
		"Next":                  safeOIDCReturnTo(r.URL.Query().Get("next")),
	})
}

func (s *Server) newLoginCSRFToken() (string, error) {
	secret, err := s.loginCSRFKey()
	if err != nil {
		return "", err
	}
	nonce, err := newSessionToken()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%d.%s", time.Now().Add(loginCSRFTokenTTL).Unix(), nonce)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *Server) validLoginCSRFToken(token string) bool {
	if len(token) == 0 || len(token) > 256 {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(parts[1]) != 64 {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	if expiresAt < now || expiresAt > now+int64(loginCSRFTokenTTL.Seconds())+60 {
		return false
	}
	nonce, err := hex.DecodeString(parts[1])
	if err != nil || len(nonce) != 32 {
		return false
	}
	providedMAC, err := hex.DecodeString(parts[2])
	if err != nil || len(providedMAC) != sha256.Size {
		return false
	}
	secret, err := s.loginCSRFKey()
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	return hmac.Equal(providedMAC, mac.Sum(nil))
}

func (s *Server) loginCSRFKey() ([]byte, error) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if len(s.loginCSRFSecret) == 0 {
		s.loginCSRFSecret = make([]byte, 32)
		if _, err := rand.Read(s.loginCSRFSecret); err != nil {
			s.loginCSRFSecret = nil
			return nil, fmt.Errorf("create login CSRF key: %w", err)
		}
	}
	return append([]byte(nil), s.loginCSRFSecret...), nil
}

func (s *Server) sessionAuthOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" || r.URL.Path == "/auth/oidc/login" || r.URL.Path == "/auth/oidc/callback" {
			h.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/logout" {
			if s.hasValidSession(r) && !s.requireCSRF(w, r) {
				return
			}
			h.ServeHTTP(w, r)
			return
		}
		if token, ok := s.validSessionToken(r); ok {
			if requiresCSRF(r.Method) && !s.requireCSRFForToken(w, r, token) {
				return
			}
			h.ServeHTTP(w, withDeckSessionToken(r, token))
			return
		}
		user, pass, ok := r.BasicAuth()
		if ok && s.validCredentials(user, pass) {
			s.clearFailedLogins(r)
			if requiresCSRF(r.Method) && !basicAuthMutationAllowed(w, r) {
				return
			}
			if !isScriptableDeckPath(r.URL.Path) {
				token, err := s.startSession(w, r)
				if err != nil {
					http.Error(w, "unable to create session", http.StatusInternalServerError)
					return
				}
				// The response cookie is not visible to this request. Carry the new,
				// server-created session into page rendering so its CSRF bootstrap is
				// available on the very first Basic-authenticated browser navigation.
				r = withDeckSessionToken(r, token)
				s.recordActivity("auth", "basic_auth", "Deck session opened via HTTP Basic auth.", map[string]interface{}{"username": user})
			}
			h.ServeHTTP(w, r)
			return
		}
		if ok {
			s.noteFailedLogin(r)
		}
		s.handleUnauthorized(w, r)
	})
}

func (s *Server) handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	if s.loginBlocked(r) {
		w.Header().Set("Retry-After", strconv.Itoa(int(failedLoginWindow/time.Second)))
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/deck/") {
			writeJSONError(w, http.StatusTooManyRequests, "too many login attempts")
			return
		}
		code := http.StatusSeeOther
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			code = http.StatusTemporaryRedirect
		}
		http.Redirect(w, r, "/login", code)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/deck/") {
		w.Header().Set("WWW-Authenticate", `Basic realm="IPTV Tunerr Deck"`)
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	code := http.StatusSeeOther
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		code = http.StatusTemporaryRedirect
	}
	http.Redirect(w, r, "/login", code)
}

func (s *Server) validCredentials(user, pass string) bool {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return subtle.ConstantTimeCompare([]byte(user), []byte(s.settings.AuthUser)) == 1 &&
		subtle.ConstantTimeCompare([]byte(pass), []byte(s.settings.AuthPass)) == 1
}

// basicAuthMutationAllowed protects browser-cached HTTP Basic credentials from
// cross-site form submissions while keeping command-line clients scriptable.
// Requests without browser origin metadata are treated as non-browser clients.
func basicAuthMutationAllowed(w http.ResponseWriter, r *http.Request) bool {
	site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")))
	if site != "" && site != "same-origin" {
		writeJSONError(w, http.StatusForbidden, "cross-origin mutation rejected")
		return false
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		origin = strings.TrimSpace(r.Header.Get("Referer"))
	}
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		writeJSONError(w, http.StatusForbidden, "cross-origin mutation rejected")
		return false
	}

	expectedScheme := "http"
	if r.TLS != nil || (isLoopback(remoteHost(r)) && strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")) {
		expectedScheme = "https"
	}
	if !strings.EqualFold(u.Scheme, expectedScheme) || !strings.EqualFold(u.Host, r.Host) {
		writeJSONError(w, http.StatusForbidden, "cross-origin mutation rejected")
		return false
	}
	return true
}

func mustGenerateDeckPassword(length int) string {
	if length < 12 {
		length = 12
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		panic("webui: cryptographic randomness unavailable for generated password")
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out)
}

func writeMethodNotAllowedJSON(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeMethodNotAllowedPlain(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf("{\"error\":%q}\n", msg)))
}

func (s *Server) hasValidSession(r *http.Request) bool {
	_, ok := s.validSessionToken(r)
	return ok
}

func (s *Server) validSessionToken(r *http.Request) (string, bool) {
	token, _ := r.Context().Value(deckSessionTokenContextKey{}).(string)
	if strings.TrimSpace(token) == "" {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			return "", false
		}
		token = cookie.Value
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.pruneSessionsLocked()
	session, ok := s.sessions[token]
	if !ok || time.Now().After(session.ExpiresAt) {
		delete(s.sessions, token)
		return "", false
	}
	session.ExpiresAt = time.Now().Add(sessionTTL)
	s.sessions[token] = session
	return token, true
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) (string, error) {
	token, err := newSessionToken()
	if err != nil {
		return "", fmt.Errorf("create session token: %w", err)
	}
	csrfToken, err := newSessionToken()
	if err != nil {
		return "", fmt.Errorf("create CSRF token: %w", err)
	}
	s.sessionMu.Lock()
	s.ensureStateMaps()
	s.pruneSessionsLocked()
	s.sessions[token] = deckSession{
		ExpiresAt: time.Now().Add(sessionTTL),
		CSRFToken: csrfToken,
	}
	s.sessionMu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
		Secure:   s.secureCookie(r),
	})
	return token, nil
}

type deckSessionTokenContextKey struct{}

func withDeckSessionToken(r *http.Request, token string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), deckSessionTokenContextKey{}, token))
}

func (s *Server) pruneSessionsLocked() {
	now := time.Now()
	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, token)
		}
	}
}

func (s *Server) csrfTokenForRequest(r *http.Request) string {
	token, ok := s.validSessionToken(r)
	if !ok {
		return ""
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	session, ok := s.sessions[token]
	if !ok {
		return ""
	}
	return session.CSRFToken
}

func requiresCSRF(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (s *Server) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	token, ok := s.validSessionToken(r)
	if !ok {
		s.handleUnauthorized(w, r)
		return false
	}
	return s.requireCSRFForToken(w, r, token)
}

func (s *Server) requireCSRFForToken(w http.ResponseWriter, r *http.Request, token string) bool {
	if !requiresCSRF(r.Method) {
		return true
	}
	header := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	if header == "" {
		header = strings.TrimSpace(r.Header.Get(legacyCSRFHeaderName))
	}
	if header == "" {
		writeJSONError(w, http.StatusForbidden, "missing csrf token")
		return false
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	session, ok := s.sessions[token]
	if !ok || strings.TrimSpace(session.CSRFToken) == "" {
		writeJSONError(w, http.StatusUnauthorized, "invalid session")
		return false
	}
	if subtle.ConstantTimeCompare([]byte(header), []byte(session.CSRFToken)) != 1 {
		writeJSONError(w, http.StatusForbidden, "invalid csrf token")
		return false
	}
	return true
}

func requestCookieSecure(r *http.Request) bool {
	return r != nil && r.TLS != nil
}

func isScriptableDeckPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || path == "/api" || strings.HasPrefix(path, "/deck/")
}

func newSessionToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func (s *Server) loginBlocked(r *http.Request) bool {
	ip := remoteHost(r)
	if ip == "" {
		return false
	}
	s.failedLoginMu.Lock()
	defer s.failedLoginMu.Unlock()
	s.ensureStateMaps()
	s.trimFailedLoginsLocked(ip)
	if _, tracked := s.failedLoginByIP[ip]; tracked {
		return len(s.failedLoginByIP[ip]) >= failedLoginLimit
	}
	if len(s.failedLoginByIP) >= maxFailedLoginIPs {
		now := time.Now()
		s.pruneFailedLoginsLocked(now)
		if len(s.failedLoginByIP) >= maxFailedLoginIPs {
			s.trimFailedLoginOverflowLocked(now)
			return len(s.failedLoginOverflow) >= failedLoginLimit
		}
	}
	return false
}

func (s *Server) noteFailedLogin(r *http.Request) {
	ip := remoteHost(r)
	if ip == "" {
		return
	}
	s.failedLoginMu.Lock()
	defer s.failedLoginMu.Unlock()
	s.ensureStateMaps()
	now := time.Now()
	s.trimFailedLoginsAtLocked(ip, now)
	if _, exists := s.failedLoginByIP[ip]; !exists && len(s.failedLoginByIP) >= maxFailedLoginIPs {
		s.pruneFailedLoginsLocked(now)
		if len(s.failedLoginByIP) >= maxFailedLoginIPs {
			s.trimFailedLoginOverflowLocked(now)
			if len(s.failedLoginOverflow) < failedLoginLimit {
				s.failedLoginOverflow = append(s.failedLoginOverflow, now)
			}
			return
		}
	}
	if len(s.failedLoginByIP[ip]) >= failedLoginLimit {
		return
	}
	s.failedLoginByIP[ip] = append(s.failedLoginByIP[ip], now)
}

func (s *Server) clearFailedLogins(r *http.Request) {
	ip := remoteHost(r)
	if ip == "" {
		return
	}
	s.failedLoginMu.Lock()
	s.ensureStateMaps()
	delete(s.failedLoginByIP, ip)
	s.failedLoginOverflow = nil
	s.failedLoginMu.Unlock()
}

func (s *Server) trimFailedLoginOverflowLocked(now time.Time) {
	cutoff := now.Add(-failedLoginWindow)
	entries := s.failedLoginOverflow
	kept := entries[:0]
	for _, at := range entries {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	s.failedLoginOverflow = kept
}

func (s *Server) trimFailedLoginsLocked(ip string) {
	s.trimFailedLoginsAtLocked(ip, time.Now())
}

func (s *Server) trimFailedLoginsAtLocked(ip string, now time.Time) {
	cutoff := now.Add(-failedLoginWindow)
	entries := s.failedLoginByIP[ip]
	kept := entries[:0]
	for _, at := range entries {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) == 0 {
		delete(s.failedLoginByIP, ip)
		return
	}
	s.failedLoginByIP[ip] = kept
}

func (s *Server) pruneFailedLoginsLocked(now time.Time) {
	for ip := range s.failedLoginByIP {
		s.trimFailedLoginsAtLocked(ip, now)
	}
}

func remoteHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func fallbackVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "dev"
	}
	return v
}

func proxyBase(addr string) string {
	addr = strings.TrimSpace(addr)
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		switch {
		case strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") && !strings.Contains(addr, "]:"):
			host = strings.TrimSuffix(strings.TrimPrefix(addr, "["), "]")
		case err != nil:
			host = addr
		}
		port = "5004"
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return "http://" + host + ":" + port
}

func localhostOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || !isLoopback(host) {
			msg := "forbidden: webui is localhost-only (set IPTV_TUNERR_WEBUI_ALLOW_LAN=1)"
			path := strings.ToLower(strings.TrimSpace(r.URL.Path))
			if path == "/api" || strings.HasPrefix(path, "/api/") || strings.HasSuffix(path, ".json") {
				writeJSONError(w, http.StatusForbidden, msg)
				return
			}
			http.Error(w, msg, http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'",
			"base-uri 'none'",
			"frame-ancestors 'none'",
			"form-action 'self'",
			"img-src 'self' data:",
			"style-src 'self' 'unsafe-inline'",
			"script-src 'self'",
			"connect-src 'self'",
		}, "; "))
		h.ServeHTTP(w, r)
	})
}

func isLoopback(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
