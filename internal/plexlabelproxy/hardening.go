package plexlabelproxy

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// maxScrubBody bounds how much of a textual elevated response is buffered to
// remove the owner token. Larger bodies fail closed rather than pass through
// unscanned.
const maxScrubBody = 16 << 20

type userTokenCtxKey struct{}

// withUserToken remembers the inbound (pre-elevation) client token so the
// response scrubber can hand the user their own token back wherever PMS echoed
// the owner's.
func withUserToken(req *http.Request, token string) {
	*req = *req.WithContext(context.WithValue(req.Context(), userTokenCtxKey{}, token))
}

func userTokenFromContext(ctx context.Context) string {
	s, _ := ctx.Value(userTokenCtxKey{}).(string)
	return s
}

// tokensEqual compares tokens without leaking length-matching prefixes via timing.
func tokensEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(a)), []byte(strings.TrimSpace(b))) == 1
}

// pathSafeForElevation reports whether the request path is in canonical form.
// The owner token must never ride on a path that a classifier matched by prefix
// but that the upstream could resolve elsewhere (dot segments, encoded
// separators, doubled slashes, control characters).
func pathSafeForElevation(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	raw := req.URL.EscapedPath()
	low := strings.ToLower(raw)
	if strings.Contains(low, "%2f") || strings.Contains(low, "%5c") || strings.Contains(low, "%00") {
		return false
	}
	s := raw
	for i := 0; i < 4; i++ {
		if !canonicalPathText(s) {
			return false
		}
		next, err := url.PathUnescape(s)
		if err != nil {
			return false
		}
		if next == s {
			return true
		}
		s = next
	}
	return canonicalPathText(s)
}

func canonicalPathText(s string) bool {
	if strings.Contains(s, "//") || strings.Contains(s, `\`) {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	for _, seg := range strings.Split(s, "/") {
		if seg == ".." || seg == "." {
			return false
		}
	}
	return true
}

// liveTVRoutePath reports whether path is an exact Live TV route family:
// /livetv/{dvrs,epg,sessions,hubs}[/...]. Anything else under /livetv/ is not
// elevated.
func liveTVRoutePath(path string) bool {
	rest, ok := strings.CutPrefix(path, "/livetv/")
	if !ok {
		return false
	}
	seg, _, _ := strings.Cut(rest, "/")
	switch seg {
	case "dvrs", "epg", "sessions", "hubs":
		return true
	}
	return false
}

// ownerOnlyPath lists PMS areas that expose the owner's account, credentials,
// or other users' activity. -elevate-all never lends the owner token to them.
func ownerOnlyPath(escapedPath string) bool {
	p := strings.ToLower(escapedPath)
	for i := 0; i < 4; i++ {
		next, err := url.PathUnescape(p)
		if err != nil || next == p {
			break
		}
		p = next
	}
	for _, prefix := range []string{
		"/myplex", "/security", "/:/prefs", "/:/plugins", "/accounts", "/devices",
		"/status/sessions", "/status/history", "/updater", "/diagnostics", "/butler",
		"/log", "/activities", "/system", "/services", "/servers", "/clients",
		"/library/sections/all/refresh", "/admin",
	} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") || strings.HasPrefix(p, prefix+"?") {
			return true
		}
	}
	return false
}

// refererIsLiveTVStrict inspects only the Referer's path and fragment, where
// Plex Web places its route, not arbitrary query text.
func refererIsLiveTVStrict(ref string) bool {
	u, err := url.Parse(ref)
	if err != nil {
		return false
	}
	return liveTVText(u.Path) || liveTVText(u.Fragment)
}

// scrubOwnerToken removes the owner token from an elevated response so a
// borrowed credential is never handed to the client. Headers are always
// scrubbed; textual bodies are scrubbed when small enough to scan.
func (p *Proxy) scrubOwnerToken(resp *http.Response) error {
	owner := strings.TrimSpace(p.cfg.OwnerToken)
	if owner == "" || resp == nil || resp.Request == nil || !p.requestCarriesOwnerToken(resp.Request) {
		return nil
	}
	replacement := strings.TrimSpace(userTokenFromContext(resp.Request.Context()))
	if replacement != "" && tokensEqual(replacement, owner) {
		return nil // the owner is using the proxy as themselves
	}
	if replacement == "" {
		replacement = "redacted"
	}
	needles := []string{owner}
	if esc := url.QueryEscape(owner); esc != owner {
		needles = append(needles, esc)
	}
	scrub := func(s string) string {
		for _, n := range needles {
			s = strings.ReplaceAll(s, n, replacement)
		}
		return s
	}
	for name, vals := range resp.Header {
		for i, v := range vals {
			resp.Header[name][i] = scrub(v)
		}
	}
	if resp.StatusCode == http.StatusSwitchingProtocols || resp.Body == nil || resp.Body == http.NoBody {
		return nil
	}
	if !scrubbableContentType(resp.Header.Get("Content-Type")) {
		return nil
	}
	if resp.ContentLength > maxScrubBody {
		return errBodyTooLarge
	}
	body, encoding, err := readBodyLimited(resp, maxScrubBody)
	if err != nil {
		return err
	}
	return restoreBody(resp, []byte(scrub(string(body))), encoding)
}

func readBodyLimited(resp *http.Response, limit int64) ([]byte, string, error) {
	resp.Body = limitedReadCloser(resp.Body, limit)
	body, enc, err := readBody(resp)
	if err != nil {
		return nil, "", fmt.Errorf("plexlabelproxy: scrub read: %w", err)
	}
	return body, enc, nil
}

func (p *Proxy) requestCarriesOwnerToken(req *http.Request) bool {
	owner := p.cfg.OwnerToken
	if tokensEqual(req.Header.Get("X-Plex-Token"), owner) {
		return true
	}
	return tokensEqual(req.URL.Query().Get("X-Plex-Token"), owner)
}

// scrubbableContentType is true for response types that can echo URLs/tokens.
// Media payloads, images and opaque binaries are left streaming.
func scrubbableContentType(ct string) bool {
	ct = strings.ToLower(ct)
	if ct == "" {
		return true
	}
	for _, m := range []string{"xml", "json", "text/", "mpegurl", "javascript"} {
		if strings.Contains(ct, m) {
			return true
		}
	}
	return false
}

type limitedBody struct {
	rc        io.ReadCloser
	remaining int64
}

func limitedReadCloser(rc io.ReadCloser, limit int64) io.ReadCloser {
	return &limitedBody{rc: rc, remaining: limit}
}

func (l *limitedBody) Read(b []byte) (int, error) {
	if l.remaining < 0 {
		return 0, errBodyTooLarge
	}
	if int64(len(b)) > l.remaining+1 {
		b = b[:l.remaining+1]
	}
	n, err := l.rc.Read(b)
	l.remaining -= int64(n)
	if l.remaining < 0 {
		return n, errBodyTooLarge
	}
	return n, err
}

func (l *limitedBody) Close() error { return l.rc.Close() }

var errBodyTooLarge = errors.New("plexlabelproxy: elevated textual response too large to scrub")
