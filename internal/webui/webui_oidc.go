package webui

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcTransactionTTL    = 10 * time.Minute
	maxOIDCTransactions   = 16
	maxOIDCResponseBytes  = 1 << 20
	oidcStateCookiePrefix = "iptvtunerr_oidc_state_"
	maxOIDCReturnToBytes  = 2048
)

// OIDCSettings contains operator-owned OIDC configuration. The client secret
// is never included in a response or browser-visible page.
type OIDCSettings struct {
	Provider             string
	IssuerURL            string
	ClientID             string
	ClientSecret         string
	RedirectURI          string
	AllowedEmails        string
	AllowedEmailDomains  string
	AllowAnyVerifiedUser bool
	AllowPrivateIssuer   bool
}

type oidcProviderDisplay struct {
	Label    string
	Icon     string
	SetupURL string
}

func oidcProviderPresentation(provider string) oidcProviderDisplay {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return oidcProviderDisplay{"Google", "google", "https://developers.google.com/identity/openid-connect/openid-connect"}
	case "microsoft-entra":
		return oidcProviderDisplay{"Microsoft Entra ID", "microsoft", "https://learn.microsoft.com/en-us/entra/identity-platform/v2-protocols-oidc"}
	case "okta":
		return oidcProviderDisplay{"Okta", "okta", "https://developer.okta.com/docs/guides/implement-grant-type/authcodepkce/main/"}
	case "auth0":
		return oidcProviderDisplay{"Auth0", "auth0", "https://auth0.com/docs/get-started/authentication-and-authorization-flow/authorization-code-flow"}
	case "keycloak":
		return oidcProviderDisplay{"Keycloak", "keycloak", "https://www.keycloak.org/docs/latest/server_admin/"}
	case "authentik":
		return oidcProviderDisplay{"authentik", "authentik", "https://docs.goauthentik.io/add-secure-apps/providers/oauth2/create-oauth2-provider/"}
	case "zitadel":
		return oidcProviderDisplay{"ZITADEL", "zitadel", "https://zitadel.com/docs/guides/integrate/login/oidc/oauth-recommended-flows"}
	case "amazon-cognito":
		return oidcProviderDisplay{"Amazon Cognito", "amazon-cognito", "https://docs.aws.amazon.com/cognito/latest/developerguide/federation-endpoints.html"}
	case "apple-sign-in":
		return oidcProviderDisplay{"Apple", "apple", "https://developer.apple.com/documentation/signinwithapple/configuring-your-environment-for-sign-in-with-apple"}
	case "gitlab":
		return oidcProviderDisplay{"GitLab", "gitlab", "https://docs.gitlab.com/integration/openid_connect_provider/"}
	case "fusionauth":
		return oidcProviderDisplay{"FusionAuth", "fusionauth", "https://fusionauth.io/docs/lifecycle/authenticate-users/oauth/"}
	case "onelogin":
		return oidcProviderDisplay{"OneLogin", "onelogin", "https://developers.onelogin.com/docs/openid-connect/connect-to-onelogin/"}
	case "pingone":
		return oidcProviderDisplay{"PingOne", "pingone", "https://docs.pingidentity.com/pingone/applications/p1_edit_application_oidc.html"}
	case "jumpcloud":
		return oidcProviderDisplay{"JumpCloud", "jumpcloud", "https://jumpcloud.com/support/sso-with-oidc"}
	default:
		return oidcProviderDisplay{"identity provider", "generic", "https://openid.net/developers/how-connect-works/"}
	}
}

func oidcStateCookieName(state string) string {
	digest := sha256.Sum256([]byte(state))
	return oidcStateCookiePrefix + hex.EncodeToString(digest[:16])
}

func (c OIDCSettings) enabled() bool {
	return strings.TrimSpace(c.IssuerURL) != "" &&
		strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != "" &&
		strings.TrimSpace(c.RedirectURI) != ""
}

func (c OIDCSettings) validate() error {
	if !c.enabled() {
		return fmt.Errorf("incomplete OIDC settings")
	}
	if err := validateOIDCIssuer(c.IssuerURL, c.AllowPrivateIssuer); err != nil {
		return err
	}
	redirect, err := url.Parse(strings.TrimSpace(c.RedirectURI))
	if err != nil || redirect.Scheme == "" || redirect.Host == "" || redirect.User != nil || redirect.RawQuery != "" || redirect.Fragment != "" {
		return fmt.Errorf("invalid OIDC redirect URI")
	}
	if redirect.Path != "/auth/oidc/callback" {
		return fmt.Errorf("OIDC redirect URI must use the registered callback path")
	}
	if redirect.Scheme != "https" {
		return fmt.Errorf("OIDC redirect URI must use HTTPS")
	}
	return nil
}

type oidcTransaction struct {
	Nonce        string
	CodeVerifier string
	ReturnTo     string
	ExpiresAt    time.Time
}

type oidcRuntime struct {
	mu           sync.Mutex
	transactions map[string]oidcTransaction
	provider     *oidc.Provider
	providerErr  error
	client       *http.Client
	providerOnce sync.Once
}

func (s *Server) oidcRuntimeState() *oidcRuntime {
	return &s.oidc
}

func (s *Server) oidcLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowedPlain(w, http.MethodGet)
		return
	}
	if !s.oidcSettings.enabled() {
		http.NotFound(w, r)
		return
	}
	if err := s.oidcSettings.validate(); err != nil {
		log.Printf("webui: OIDC configuration is invalid")
		s.renderLogin(w, r, http.StatusServiceUnavailable, "Single sign-on is not configured correctly.")
		return
	}

	provider, err := s.oidcProvider()
	if err != nil {
		log.Printf("webui: OIDC provider discovery failed")
		s.renderLogin(w, r, http.StatusServiceUnavailable, "Single sign-on is temporarily unavailable.")
		return
	}

	state := oauth2.GenerateVerifier()
	apple := isAppleOIDCIssuer(s.oidcSettings.IssuerURL)
	transaction := oidcTransaction{
		Nonce:     oauth2.GenerateVerifier(),
		ReturnTo:  safeOIDCReturnTo(r.URL.Query().Get("next")),
		ExpiresAt: time.Now().Add(oidcTransactionTTL),
	}
	if !apple {
		transaction.CodeVerifier = oauth2.GenerateVerifier()
	}
	if !s.storeOIDCTransaction(state, transaction) {
		http.Error(w, "too many sign-in attempts; try again shortly", http.StatusTooManyRequests)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName(state),
		Value:    state,
		Path:     "/auth/oidc/",
		Expires:  transaction.ExpiresAt,
		MaxAge:   int(oidcTransactionTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie(r),
		SameSite: oidcCookieSameSite(s.oidcSettings.IssuerURL),
	})

	config := s.oidcOAuthConfig(provider)
	options := []oauth2.AuthCodeOption{oidc.Nonce(transaction.Nonce)}
	if apple {
		options = append(options, oauth2.SetAuthURLParam("response_mode", "form_post"))
	} else {
		options = append(options, oauth2.S256ChallengeOption(transaction.CodeVerifier))
	}
	url := config.AuthCodeURL(state, options...)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	apple := isAppleOIDCIssuer(s.oidcSettings.IssuerURL)
	if r.Method != http.MethodGet && !(apple && r.Method == http.MethodPost) {
		writeMethodNotAllowedPlain(w, http.MethodGet, http.MethodPost)
		return
	}
	if !s.oidcSettings.enabled() {
		http.NotFound(w, r)
		return
	}

	params := r.URL.Query()
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		if err := r.ParseForm(); err != nil {
			s.failOIDC(w, r, "", "invalid_form_post")
			return
		}
		params = r.PostForm
	}
	state, stateOK := oneOIDCFormValue(params, "state")
	cookie, cookieErr := r.Cookie(oidcStateCookieName(state))
	if !stateOK || state == "" || len(state) > 256 || cookieErr != nil || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		s.failOIDC(w, r, state, "invalid_transaction")
		return
	}
	transaction, ok := s.consumeOIDCTransaction(state)
	if !ok || time.Now().After(transaction.ExpiresAt) {
		s.failOIDC(w, r, state, "expired_transaction")
		return
	}
	if errorCode, _ := oneOIDCFormValue(params, "error"); errorCode != "" {
		s.failOIDC(w, r, state, "provider_rejected")
		return
	}
	code, codeOK := oneOIDCFormValue(params, "code")
	if !codeOK || code == "" || len(code) > 8192 {
		s.failOIDC(w, r, state, "missing_code")
		return
	}

	provider, err := s.oidcProvider()
	if err != nil {
		s.failOIDC(w, r, state, "provider_unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	ctx = oidc.ClientContext(ctx, s.oidc.client)
	var token *oauth2.Token
	if transaction.CodeVerifier != "" {
		token, err = s.oidcOAuthConfig(provider).Exchange(ctx, code, oauth2.VerifierOption(transaction.CodeVerifier))
	} else {
		token, err = s.oidcOAuthConfig(provider).Exchange(ctx, code)
	}
	if err != nil {
		s.failOIDC(w, r, state, "code_exchange_failed")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		s.failOIDC(w, r, state, "missing_id_token")
		return
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: s.oidcSettings.ClientID}).Verify(ctx, rawIDToken)
	if err != nil || idToken.Subject == "" {
		s.failOIDC(w, r, state, "invalid_id_token")
		return
	}
	var claims struct {
		Nonce         string          `json:"nonce"`
		Email         *string         `json:"email"`
		EmailVerified json.RawMessage `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil || claims.Nonce == "" || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(transaction.Nonce)) != 1 {
		s.failOIDC(w, r, state, "invalid_nonce")
		return
	}
	verified, hasVerification := oidcBooleanClaim(claims.EmailVerified)
	if hasVerification && !verified {
		s.failOIDC(w, r, state, "identity_not_allowed")
		return
	}
	email := ""
	if claims.Email != nil && verified {
		email = *claims.Email
	} else {
		userInfo, userInfoErr := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if userInfoErr != nil || userInfo.Subject != idToken.Subject {
			s.failOIDC(w, r, state, "userinfo_unavailable")
			return
		}
		var userInfoClaims struct {
			Email         string          `json:"email"`
			EmailVerified json.RawMessage `json:"email_verified"`
		}
		if userInfo.Claims(&userInfoClaims) != nil {
			s.failOIDC(w, r, state, "identity_not_allowed")
			return
		}
		userInfoVerified, _ := oidcBooleanClaim(userInfoClaims.EmailVerified)
		if !userInfoVerified {
			s.failOIDC(w, r, state, "identity_not_allowed")
			return
		}
		email = userInfoClaims.Email
	}
	if email == "" || !s.oidcEmailAllowed(email) {
		s.failOIDC(w, r, state, "identity_not_allowed")
		return
	}

	if err := s.startSession(w, r); err != nil {
		s.failOIDC(w, r, state, "session_failed")
		return
	}
	s.clearOIDCStateCookie(w, r, state)
	s.recordActivityWithEntry(DeckActivityEntry{
		Kind:    "auth",
		Title:   "OIDC login",
		Message: "Control Deck session opened through the identity provider.",
		Actor:   email,
	})
	http.Redirect(w, r, transaction.ReturnTo, http.StatusSeeOther)
}

func (s *Server) failOIDC(w http.ResponseWriter, r *http.Request, state string, code string) {
	s.clearOIDCStateCookie(w, r, state)
	log.Printf("webui: OIDC sign-in failed (%s)", code)
	s.renderLogin(w, r, http.StatusUnauthorized, "Single sign-on failed. Check the identity provider and allowed-account settings.")
}

func (s *Server) clearOIDCStateCookie(w http.ResponseWriter, r *http.Request, state string) {
	if state == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName(state),
		Value:    "",
		Path:     "/auth/oidc/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookie(r),
		SameSite: oidcCookieSameSite(s.oidcSettings.IssuerURL),
	})
}

func (s *Server) clearAllOIDCStateCookies(w http.ResponseWriter, r *http.Request) {
	// The state cookies are scoped to the callback path, so a logout request
	// cannot receive them. Expire them from the matching pending server flows.
	runtime := s.oidcRuntimeState()
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	for state := range runtime.transactions {
		s.clearOIDCStateCookie(w, r, state)
		delete(runtime.transactions, state)
	}
}

// A configured HTTPS callback URL is the operator-owned public URL hint for
// deployments where TLS terminates at a reverse proxy before this listener.
func (s *Server) secureCookie(r *http.Request) bool {
	return requestCookieSecure(r) || strings.HasPrefix(strings.ToLower(strings.TrimSpace(s.oidcSettings.RedirectURI)), "https://")
}

func (s *Server) storeOIDCTransaction(state string, transaction oidcTransaction) bool {
	runtime := s.oidcRuntimeState()
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.transactions == nil {
		runtime.transactions = make(map[string]oidcTransaction)
	}
	now := time.Now()
	for key, value := range runtime.transactions {
		if now.After(value.ExpiresAt) {
			delete(runtime.transactions, key)
		}
	}
	if len(runtime.transactions) >= maxOIDCTransactions {
		return false
	}
	runtime.transactions[state] = transaction
	return true
}

func (s *Server) consumeOIDCTransaction(state string) (oidcTransaction, bool) {
	runtime := s.oidcRuntimeState()
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	transaction, ok := runtime.transactions[state]
	delete(runtime.transactions, state)
	return transaction, ok
}

func (s *Server) oidcProvider() (*oidc.Provider, error) {
	runtime := s.oidcRuntimeState()
	runtime.providerOnce.Do(func() {
		client, err := newOIDCHTTPClient(s.oidcSettings.IssuerURL, s.oidcSettings.AllowPrivateIssuer)
		if err != nil {
			runtime.providerErr = err
			return
		}
		runtime.client = client
		discoveryCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		discoveryCtx = oidc.ClientContext(discoveryCtx, client)
		runtime.provider, runtime.providerErr = oidc.NewProvider(discoveryCtx, strings.TrimSpace(s.oidcSettings.IssuerURL))
	})
	return runtime.provider, runtime.providerErr
}

func (s *Server) oidcOAuthConfig(provider *oidc.Provider) *oauth2.Config {
	endpoint := provider.Endpoint()
	apple := isAppleOIDCIssuer(s.oidcSettings.IssuerURL)
	if apple {
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	}
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if apple {
		scopes = []string{oidc.ScopeOpenID, "email"}
	}
	return &oauth2.Config{
		ClientID:     strings.TrimSpace(s.oidcSettings.ClientID),
		ClientSecret: s.oidcSettings.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  strings.TrimSpace(s.oidcSettings.RedirectURI),
		Scopes:       scopes,
	}
}

func isAppleOIDCIssuer(issuer string) bool {
	return strings.TrimSpace(issuer) == "https://appleid.apple.com"
}

func oidcCookieSameSite(issuer string) http.SameSite {
	if isAppleOIDCIssuer(issuer) {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func oneOIDCFormValue(values url.Values, key string) (string, bool) {
	items, exists := values[key]
	if !exists || len(items) != 1 {
		return "", false
	}
	return items[0], true
}

func oidcBooleanClaim(raw json.RawMessage) (bool, bool) {
	if len(raw) == 0 {
		return false, false
	}
	var value bool
	if json.Unmarshal(raw, &value) == nil {
		return value, true
	}
	var text string
	if json.Unmarshal(raw, &text) == nil && (text == "true" || text == "false") {
		return text == "true", true
	}
	return false, true
}

func (s *Server) oidcEmailAllowed(rawEmail string) bool {
	email := strings.ToLower(strings.TrimSpace(rawEmail))
	if email == "" {
		return false
	}
	allowedEmails := parseOIDCList(s.oidcSettings.AllowedEmails)
	allowedDomains := parseOIDCList(s.oidcSettings.AllowedEmailDomains)
	for _, allowed := range allowedEmails {
		if email == allowed {
			return true
		}
	}
	if at := strings.LastIndexByte(email, '@'); at >= 0 && at < len(email)-1 {
		domain := email[at+1:]
		for _, allowed := range allowedDomains {
			if domain == allowed {
				return true
			}
		}
	}
	return s.oidcSettings.AllowAnyVerifiedUser
}

func parseOIDCList(raw string) []string {
	items := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\r' || r == '\n'
	})
	return items
}

func safeOIDCReturnTo(raw string) string {
	if len(raw) == 0 || len(raw) > maxOIDCReturnToBytes || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return "/"
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") {
		return "/"
	}
	return u.RequestURI()
}

func newOIDCHTTPClient(issuer string, allowPrivate bool) (*http.Client, error) {
	if err := validateOIDCIssuer(issuer, allowPrivate); err != nil {
		return nil, err
	}
	transport := &http.Transport{
		DialContext:           oidcDialContext(allowPrivate),
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          8,
		MaxIdleConnsPerHost:   2,
	}
	return &http.Client{
		Transport: oidcPolicyTransport{next: transport, allowPrivate: allowPrivate},
		Timeout:   15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func validateOIDCIssuer(issuer string, allowPrivate bool) error {
	u, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid OIDC issuer URL")
	}
	if u.Scheme != "https" && !(allowPrivate && u.Scheme == "http") {
		return fmt.Errorf("OIDC issuer must use HTTPS")
	}
	if u.Scheme == "http" {
		if !allowPrivate || !hostResolvesOnlyToPrivateAddresses(u.Hostname()) {
			return fmt.Errorf("HTTP OIDC issuers are limited to explicitly enabled private addresses")
		}
	}
	return nil
}

type oidcPolicyTransport struct {
	next         http.RoundTripper
	allowPrivate bool
}

func (t oidcPolicyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, fmt.Errorf("invalid OIDC backchannel request")
	}
	if request.URL.Scheme != "https" && (request.URL.Scheme != "http" || !t.allowPrivate || !hostResolvesOnlyToPrivateAddresses(request.URL.Hostname())) {
		return nil, fmt.Errorf("OIDC backchannel must use HTTPS except for explicitly enabled private endpoints")
	}
	response, err := t.next.RoundTrip(request)
	if err != nil || response == nil || response.Body == nil {
		return response, err
	}
	response.Body = &boundedOIDCResponseBody{
		body:   response.Body,
		reader: io.LimitReader(response.Body, maxOIDCResponseBytes+1),
	}
	return response, nil
}

type boundedOIDCResponseBody struct {
	body   io.ReadCloser
	reader io.Reader
	read   int64
}

func (b *boundedOIDCResponseBody) Read(p []byte) (int, error) {
	if b.read > maxOIDCResponseBytes {
		return 0, fmt.Errorf("OIDC response exceeds size limit")
	}
	if remaining := maxOIDCResponseBytes + 1 - b.read; int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := b.reader.Read(p)
	b.read += int64(n)
	if b.read > maxOIDCResponseBytes {
		return n, fmt.Errorf("OIDC response exceeds size limit")
	}
	return n, err
}

func (b *boundedOIDCResponseBody) Close() error {
	return b.body.Close()
}

func hostResolvesOnlyToPrivateAddresses(host string) bool {
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return isNonPublicOIDCAddress(ip)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if !isNonPublicOIDCAddress(ip.IP) {
			return false
		}
	}
	return true
}

func oidcDialContext(allowPrivate bool) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("OIDC host did not resolve")
		}
		if !allowPrivate {
			for _, ip := range ips {
				if isNonPublicOIDCAddress(ip.IP) {
					return nil, fmt.Errorf("OIDC host resolves to a private address")
				}
			}
		}
		var lastErr error
		for _, ip := range ips {
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, lastErr
	}
}

func isNonPublicOIDCAddress(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
