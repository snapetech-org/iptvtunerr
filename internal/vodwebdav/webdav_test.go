package vodwebdav

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snapetech/iptvtunerr/internal/catalog"
	"github.com/snapetech/iptvtunerr/internal/vodfs"
)

type testMat struct {
	path    string
	byAsset map[string]string
}

func (t testMat) Materialize(_ context.Context, assetID string, _ string) (string, error) {
	if t.byAsset != nil {
		if path := strings.TrimSpace(t.byAsset[assetID]); path != "" {
			return path, nil
		}
	}
	return t.path, nil
}

func TestDAVFSStatAndReadMovie(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "movie.mp4")
	if err := os.WriteFile(local, []byte("movie-bytes"), 0o600); err != nil {
		t.Fatalf("write local movie: %v", err)
	}
	fs := &davFS{
		tree: NewTestTree(),
		mat:  testMat{path: local},
	}
	info, err := fs.Stat(context.Background(), "/Movies")
	if err != nil {
		t.Fatalf("stat movies: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected /Movies to be a directory")
	}
	file, err := fs.OpenFile(context.Background(), "/Movies/Live: Movie (2024)/Live: Movie (2024).mp4", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open movie: %v", err)
	}
	defer file.Close()
	buf := make([]byte, 32)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("read movie: %v", err)
	}
	if got := string(buf[:n]); got != "movie-bytes" {
		t.Fatalf("unexpected movie bytes %q", got)
	}
}

func TestMountHint(t *testing.T) {
	if got := MountHint("darwin", "127.0.0.1:8123"); got == "" {
		t.Fatal("expected darwin mount hint")
	}
	if got := MountHint("windows", "127.0.0.1:8123"); got == "" {
		t.Fatal("expected windows mount hint")
	}
}

func TestMountCommand(t *testing.T) {
	if got := MountCommand("darwin", "127.0.0.1:8123", "/Volumes/Test"); got == "" {
		t.Fatal("expected darwin mount command")
	}
	if got := MountCommand("windows", "127.0.0.1:8123", "Z:"); got == "" {
		t.Fatal("expected windows mount command")
	}
}

func TestMountHintsSupportHTTPS(t *testing.T) {
	if got := MountHintForScheme("darwin", "media.example:58188", "https"); !strings.Contains(got, "https://media.example:58188/") {
		t.Fatalf("HTTPS mount hint = %q", got)
	}
	if got := MountCommandForScheme("linux", "media.example:58188", "/mnt/vod", "https"); !strings.Contains(got, "davs://media.example:58188/") {
		t.Fatalf("HTTPS mount command = %q", got)
	}
}

func TestNewServerRestrictsNetworkListeners(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") })
	for _, addr := range []string{"127.0.0.1:58188", "[::1]:58188", "localhost:58188"} {
		if _, err := NewServer(addr, "", "", "", "", handler); err != nil {
			t.Errorf("loopback %q rejected: %v", addr, err)
		}
	}
	for _, addr := range []string{":58188", "0.0.0.0:58188", "192.0.2.10:58188"} {
		if _, err := NewServer(addr, "", "", "", "", handler); err == nil {
			t.Errorf("plaintext network listener %q was accepted", addr)
		}
		if _, err := NewServer(addr, "cert.pem", "key.pem", "", "", handler); err == nil {
			t.Errorf("unauthenticated TLS network listener %q was accepted", addr)
		}
		server, err := NewServer(addr, "cert.pem", "key.pem", "operator", "sixteen-byte-secret", handler)
		if err != nil {
			t.Errorf("TLS and authenticated listener %q rejected: %v", addr, err)
			continue
		}
		if server.TLSConfig == nil || server.TLSConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("TLS minimum version for %q is not TLS 1.2", addr)
		}
	}
	if _, err := NewServer("0.0.0.0:58188", "cert.pem", "key.pem", "operator", "short", handler); err == nil {
		t.Fatal("weak password accepted for a network listener")
	}
}

func TestNewServerRequiresCompleteTLSAndAuthPairs(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, tc := range []struct {
		name, cert, key, user, password string
	}{
		{name: "certificate without key", cert: "cert.pem"},
		{name: "username without password", user: "operator"},
		{name: "password without username", password: "secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewServer("127.0.0.1:0", tc.cert, tc.key, tc.user, tc.password, handler); err == nil {
				t.Fatal("incomplete security configuration was accepted")
			}
		})
	}
}

func TestWebDAVBasicAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") })
	server, err := NewServer("127.0.0.1:58188", "", "", "operator", "secret", next)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, user, password string
		wantStatus           int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong password", user: "operator", password: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid", user: "operator", password: "secret", wantStatus: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://local/", nil)
			if tc.user != "" {
				req.SetBasicAuth(tc.user, tc.password)
			}
			w := httptest.NewRecorder()
			server.Handler.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("unauthorized response has no Basic challenge")
			}
		})
	}
}

func TestHandler_OPTIONSAndPROPFIND_ClientShapes(t *testing.T) {
	tmp := t.TempDir()
	localMovie := filepath.Join(tmp, "movie.mp4")
	if err := os.WriteFile(localMovie, []byte("movie-bytes"), 0o600); err != nil {
		t.Fatalf("write local movie: %v", err)
	}
	localEpisode := filepath.Join(tmp, "episode.mp4")
	if err := os.WriteFile(localEpisode, []byte("episode-bytes"), 0o600); err != nil {
		t.Fatalf("write local episode: %v", err)
	}
	h := NewHandler(
		[]catalog.Movie{{ID: "m1", Title: "Movie", Year: 2024, StreamURL: "http://example.com/movie.mp4"}},
		[]catalog.Series{{
			ID:    "s1",
			Title: "Show",
			Year:  2020,
			Seasons: []catalog.Season{{
				Number: 1,
				Episodes: []catalog.Episode{{
					ID:         "e1",
					SeasonNum:  1,
					EpisodeNum: 1,
					Title:      "",
					StreamURL:  "http://example.com/episode.mp4",
				}},
			}},
		}},
		testMat{byAsset: map[string]string{"m1": localMovie, "e1": localEpisode}},
	)

	t.Run("options", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "http://local/", nil)
		req.Header.Set("User-Agent", "Microsoft-WebDAV-MiniRedir/10.0.19045")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		allow := w.Header().Get("Allow")
		if allow != readOnlyDAVAllow {
			t.Fatalf("Allow=%q", allow)
		}
		for _, forbidden := range []string{"PUT", "DELETE", "LOCK", "MOVE", "COPY", "POST"} {
			if strings.Contains(allow, forbidden) {
				t.Fatalf("Allow unexpectedly contains %q: %q", forbidden, allow)
			}
		}
		if dav := w.Header().Get("DAV"); dav == "" {
			t.Fatal("missing DAV header")
		}
		if via := w.Header().Get("MS-Author-Via"); via != "DAV" {
			t.Fatalf("MS-Author-Via=%q", via)
		}
	})

	t.Run("propfind-root-darwin", func(t *testing.T) {
		req := httptest.NewRequest(methodPROPFIND, "http://local/", strings.NewReader(`<propfind xmlns="DAV:"><allprop/></propfind>`))
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "text/xml")
		req.Header.Set("User-Agent", "WebDAVFS/3.0 (03008000) Darwin/24.0.0")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMultiStatus {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, "Movies") || !strings.Contains(body, "TV") {
			t.Fatalf("body missing collections: %q", body)
		}
	})

	t.Run("propfind-movies-windows", func(t *testing.T) {
		req := httptest.NewRequest(methodPROPFIND, "http://local/Movies", strings.NewReader(`<a:propfind xmlns:a="DAV:"><a:allprop/></a:propfind>`))
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "text/xml")
		req.Header.Set("User-Agent", "Microsoft-WebDAV-MiniRedir/10.0.19045")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMultiStatus {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Live: Movie (2024)") {
			t.Fatalf("body=%q", w.Body.String())
		}
	})

	t.Run("head-movie-file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "http://local/Movies/Live:%20Movie%20%282024%29/Live:%20Movie%20%282024%29.mp4", nil)
		req.Header.Set("User-Agent", "WebDAVFS/3.0 (03008000) Darwin/24.0.0")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if got := w.Header().Get("Accept-Ranges"); got != "bytes" {
			t.Fatalf("Accept-Ranges=%q", got)
		}
		if got := w.Header().Get("Content-Length"); got == "" {
			t.Fatal("missing Content-Length")
		}
	})

	t.Run("get-movie-file-range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://local/Movies/Live:%20Movie%20%282024%29/Live:%20Movie%20%282024%29.mp4", nil)
		req.Header.Set("Range", "bytes=0-4")
		req.Header.Set("User-Agent", "Microsoft-WebDAV-MiniRedir/10.0.19045")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusPartialContent {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if body := w.Body.String(); body != "movie" {
			t.Fatalf("body=%q", body)
		}
	})

	t.Run("propfind-episode-file-depth-zero", func(t *testing.T) {
		req := httptest.NewRequest(methodPROPFIND, "http://local/TV/Live:%20Show%20%282020%29/Season%2001/Live:%20Show%20%282020%29%20-%20s01e01.mp4", strings.NewReader(`<a:propfind xmlns:a="DAV:"><a:allprop/></a:propfind>`))
		req.Header.Set("Depth", "0")
		req.Header.Set("Content-Type", "text/xml")
		req.Header.Set("User-Agent", "WebDAVFS/3.0 (03008000) Darwin/24.0.0")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMultiStatus {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "s01e01.mp4") {
			t.Fatalf("body=%q", w.Body.String())
		}
	})

	t.Run("head-episode-file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "http://local/TV/Live:%20Show%20%282020%29/Season%2001/Live:%20Show%20%282020%29%20-%20s01e01.mp4", nil)
		req.Header.Set("User-Agent", "Microsoft-WebDAV-MiniRedir/10.0.19045")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if got := w.Header().Get("Accept-Ranges"); got != "bytes" {
			t.Fatalf("Accept-Ranges=%q", got)
		}
	})

	t.Run("range-episode-file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://local/TV/Live:%20Show%20%282020%29/Season%2001/Live:%20Show%20%282020%29%20-%20s01e01.mp4", nil)
		req.Header.Set("Range", "bytes=0-6")
		req.Header.Set("User-Agent", "Microsoft-WebDAV-MiniRedir/10.0.19045")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusPartialContent {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if body := w.Body.String(); body != "episode" {
			t.Fatalf("body=%q", body)
		}
	})

	t.Run("put-rejected-cleanly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "http://local/Movies/Live:%20Movie%20%282024%29/Live:%20Movie%20%282024%29.mp4", strings.NewReader("bad"))
		req.Header.Set("User-Agent", "Microsoft-WebDAV-MiniRedir/10.0.19045")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
		}
		if got := w.Header().Get("Allow"); got != readOnlyDAVAllow {
			t.Fatalf("Allow=%q", got)
		}
	})
}

func NewTestTree() *vodfs.Tree {
	return vodfs.NewTree(
		[]catalog.Movie{{ID: "m1", Title: "Movie", Year: 2024, StreamURL: "http://example.com/movie.mp4"}},
		[]catalog.Series{{
			ID:    "s1",
			Title: "Show",
			Year:  2020,
			Seasons: []catalog.Season{{
				Number: 1,
				Episodes: []catalog.Episode{{
					ID:         "e1",
					SeasonNum:  1,
					EpisodeNum: 1,
					Title:      "",
					StreamURL:  "http://example.com/episode.mp4",
				}},
			}},
		}},
	)
}
