package plexlabelproxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const hardeningOwner = "OWNERSECRET123"

var hardeningOwnerRev = rev(hardeningOwner)

// rawGet sends an un-normalized request line so dot segments and encoded
// separators reach the proxy exactly as an attacker would send them.
func rawGet(t *testing.T, addr, target string, extraHeaders string) (*http.Response, string) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	fmt.Fprintf(c, "GET %s HTTP/1.1\r\nHost: x\r\n%sConnection: close\r\n\r\n", target, extraHeaders)
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

// echoUpstream reports what it received via X-Seen-* response headers that
// encode the token (reversed) so the response scrubber cannot mask the result.
func echoUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s hdr=%s q=%s", r.URL.Path, rev(r.Header.Get("X-Plex-Token")), rev(r.URL.Query().Get("X-Plex-Token")))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func rev(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func hardeningProxy(t *testing.T, up *httptest.Server, cfg Config) string {
	t.Helper()
	cfg.Upstream = up.URL
	cfg.OwnerToken = hardeningOwner
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	fe := httptest.NewServer(p)
	t.Cleanup(fe.Close)
	return fe.Listener.Addr().String()
}

func TestHardening_TraversalNeverCarriesOwnerToken(t *testing.T) {
	up := echoUpstream(t)
	addr := hardeningProxy(t, up, Config{ElevateLiveTV: true})
	for _, target := range []string{
		"/livetv/../security/token?X-Plex-Token=usertok",
		"/livetv/%2e%2e/security/token?X-Plex-Token=usertok",
		"/livetv/dvrs/%2E%2E/%2E%2E/myplex/account?X-Plex-Token=usertok",
		"/livetv/dvrs/..%2fmyplex/account?X-Plex-Token=usertok",
		"/livetv//dvrs?X-Plex-Token=usertok",
		"/livetv/dvrs/..%5cx?X-Plex-Token=usertok",
	} {
		_, body := rawGet(t, addr, target, "")
		if strings.Contains(body, hardeningOwnerRev) {
			t.Errorf("%s leaked owner token upstream: %s", target, body)
		}
		if !strings.Contains(body, rev("usertok")) {
			t.Errorf("%s: user token should pass through untouched: %s", target, body)
		}
	}
	// control: canonical Live TV path is still elevated
	_, body := rawGet(t, addr, "/livetv/dvrs?X-Plex-Token=usertok", "")
	if !strings.Contains(body, "hdr="+hardeningOwnerRev) {
		t.Errorf("canonical /livetv/dvrs should be elevated: %s", body)
	}
}

func TestHardening_UnknownLiveTVFamilyNotElevated(t *testing.T) {
	up := echoUpstream(t)
	addr := hardeningProxy(t, up, Config{ElevateLiveTV: true})
	_, body := rawGet(t, addr, "/livetv/admin/thing?X-Plex-Token=usertok", "")
	if strings.Contains(body, hardeningOwnerRev) {
		t.Fatalf("unknown /livetv/ family elevated: %s", body)
	}
	for _, ok := range []string{"/livetv/epg/channels", "/livetv/sessions/abc/index.m3u8", "/livetv/hubs/discover"} {
		_, body := rawGet(t, addr, ok+"?X-Plex-Token=usertok", "")
		if !strings.Contains(body, hardeningOwnerRev) {
			t.Errorf("%s should be elevated: %s", ok, body)
		}
	}
}

func TestHardening_RefererOnlyPathOrFragmentCounts(t *testing.T) {
	up := echoUpstream(t)
	addr := hardeningProxy(t, up, Config{ElevateLiveTV: true})
	_, body := rawGet(t, addr, "/?X-Plex-Token=usertok", "Referer: http://x/web/index.html?next=/livetv/dvrs\r\n")
	if strings.Contains(body, hardeningOwnerRev) {
		t.Errorf("livetv text in Referer query must not elevate: %s", body)
	}
	_, body = rawGet(t, addr, "/?X-Plex-Token=usertok", "Referer: http://x/web/index.html#!/media/tv.plex.providers.epg.xmltv:2/x\r\n")
	if !strings.Contains(body, hardeningOwnerRev) {
		t.Errorf("Plex Web fragment route should still elevate: %s", body)
	}
}

func TestHardening_ElevateAllWithholdsOwnerOnlyAreas(t *testing.T) {
	up := echoUpstream(t)
	addr := hardeningProxy(t, up, Config{ElevateAll: true})
	for _, target := range []string{
		"/myplex/account", "/security/token", "/:/prefs", "/status/sessions", "/accounts",
		"/devices", "/%6dyplex/account", "/library/../security/token",
	} {
		_, body := rawGet(t, addr, target+"?X-Plex-Token=usertok", "")
		if strings.Contains(body, hardeningOwnerRev) {
			t.Errorf("%s: -elevate-all lent owner token to owner-only area: %s", target, body)
		}
	}
	_, body := rawGet(t, addr, "/library/sections?X-Plex-Token=usertok", "")
	if !strings.Contains(body, "hdr="+hardeningOwnerRev) {
		t.Errorf("ordinary path should still be elevated in -elevate-all: %s", body)
	}
}

func TestHardening_ResponseScrubsOwnerToken(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/livetv/dvrs":
			w.Header().Set("Content-Type", "text/xml")
			w.Header().Set("Location", "/x?X-Plex-Token="+hardeningOwner)
			w.Header().Add("Set-Cookie", "t="+hardeningOwner)
			fmt.Fprintf(w, `<MediaContainer><Part key="/p?X-Plex-Token=%s"/></MediaContainer>`, hardeningOwner)
		case "/livetv/sessions/gz":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			fmt.Fprintf(gz, `{"token":"%s"}`, hardeningOwner)
			gz.Close()
		case "/livetv/sessions/video":
			w.Header().Set("Content-Type", "video/mp2t")
			fmt.Fprintf(w, "TS%sTS", hardeningOwner) // binary media is not scanned
		}
	}))
	defer up.Close()
	addr := hardeningProxy(t, up, Config{ElevateLiveTV: true})

	resp, body := rawGet(t, addr, "/livetv/dvrs?X-Plex-Token=usertok", "")
	if strings.Contains(body, hardeningOwner) || !strings.Contains(body, "X-Plex-Token=usertok") {
		t.Errorf("body not scrubbed to user token: %s", body)
	}
	if strings.Contains(resp.Header.Get("Location"), hardeningOwner) || strings.Contains(strings.Join(resp.Header.Values("Set-Cookie"), ";"), hardeningOwner) {
		t.Errorf("headers not scrubbed: %v", resp.Header)
	}

	resp, body = rawGet(t, addr, "/livetv/sessions/gz?X-Plex-Token=usertok", "Accept-Encoding: gzip\r\n")
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(bytes.NewReader([]byte(body)))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(zr)
		body = string(b)
	}
	if strings.Contains(body, hardeningOwner) || !strings.Contains(body, "usertok") {
		t.Errorf("gzip JSON not scrubbed: %s", body)
	}

	_, body = rawGet(t, addr, "/livetv/sessions/video?X-Plex-Token=usertok", "")
	if !strings.Contains(body, hardeningOwner) {
		t.Errorf("media payloads must stream untouched (control): %s", body)
	}
}

func TestHardening_ScrubRedactsWhenNoUserToken(t *testing.T) {
	p := &Proxy{cfg: Config{OwnerToken: hardeningOwner}}
	req := httptest.NewRequest("GET", "/livetv/dvrs?X-Plex-Token="+hardeningOwner, nil)
	resp := &http.Response{
		StatusCode: 200, Request: req,
		Header: http.Header{"Content-Type": {"text/plain"}},
		Body:   io.NopCloser(strings.NewReader("a " + hardeningOwner + " b")),
	}
	if err := p.scrubOwnerToken(resp); err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "a redacted b" {
		t.Fatalf("got %q", b)
	}
}

func TestHardening_OversizedTextualResponseFailsClosed(t *testing.T) {
	p := &Proxy{cfg: Config{OwnerToken: hardeningOwner}}
	req := httptest.NewRequest("GET", "/x?X-Plex-Token="+hardeningOwner, nil)
	resp := &http.Response{
		StatusCode: 200, Request: req, ContentLength: -1,
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   io.NopCloser(io.LimitReader(zeroReader{}, maxScrubBody+10)),
	}
	if err := p.scrubOwnerToken(resp); err == nil {
		t.Fatal("expected fail-closed error for oversized body")
	}
}

type zeroReader struct{}

func (zeroReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = ' '
	}
	return len(b), nil
}

func TestHardening_AuthorizerCacheBounded(t *testing.T) {
	a := &PlexTokenAuthorizer{cache: map[string]tokenDecision{}, ttl: time.Minute}
	now := time.Now()
	for i := 0; i < maxAuthCacheEntries*3; i++ {
		a.storeLocked(fmt.Sprintf("tok-%d", i), i%7 == 0, now)
		if len(a.cache) > maxAuthCacheEntries {
			t.Fatalf("cache exceeded bound: %d", len(a.cache))
		}
	}
}

func TestHardening_PathSafety(t *testing.T) {
	good := []string{"/livetv/dvrs", "/video/:/transcode/universal/start", "/:/timeline", "/library/metadata/1/thumb/2"}
	bad := []string{"/a/../b", "/a/./b", "/a//b", "/a%2fb", "/a%5Cb", "/a\\b", "/a/%2e%2e/b", "/a/%252e%252e/b", "/a%00"}
	for _, g := range good {
		if !pathSafeForElevation(httptest.NewRequest("GET", g, nil)) {
			t.Errorf("%s should be safe", g)
		}
	}
	for _, b := range bad {
		r := httptest.NewRequest("GET", "/", nil)
		r.URL.Path, r.URL.RawPath = "", ""
		u := *r.URL
		u.Opaque = ""
		r.URL = &u
		r.URL.Path = mustUnescape(b)
		r.URL.RawPath = b
		if pathSafeForElevation(r) {
			t.Errorf("%s should be unsafe", b)
		}
	}
}

func mustUnescape(s string) string {
	r := strings.NewReplacer("%2f", "/", "%2F", "/", "%5C", `\`, "%5c", `\`, "%2e", ".", "%2E", ".", "%252e", "%2e", "%00", "\x00")
	return r.Replace(s)
}
