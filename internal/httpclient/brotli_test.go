package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestTransportWithOptionalBrotli_decompresses(t *testing.T) {
	t.Setenv("IPTV_TUNERR_HTTP_ACCEPT_BROTLI", "1")
	var enc bytes.Buffer
	w := brotli.NewWriter(&enc)
	_, _ = w.Write([]byte("hello-tunerr-br"))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	payload := enc.Bytes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !bytes.Contains([]byte(r.Header.Get("Accept-Encoding")), []byte("br")) {
			t.Errorf("Accept-Encoding should mention br: %q", r.Header.Get("Accept-Encoding"))
		}
		w.Header().Set("Content-Encoding", "br")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	tr := TransportWithOptionalBrotli(CloneDefaultTransport())
	c := &http.Client{Transport: tr}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello-tunerr-br" {
		t.Fatalf("body=%q", body)
	}
}

func TestTransportWithOptionalBrotli_disabledPassesThrough(t *testing.T) {
	t.Setenv("IPTV_TUNERR_HTTP_ACCEPT_BROTLI", "0")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain"))
	}))
	defer srv.Close()
	tr := TransportWithOptionalBrotli(CloneDefaultTransport())
	c := &http.Client{Transport: tr}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "plain" {
		t.Fatalf("body=%q", body)
	}
}

func TestHTTPCompressorWildcardNegotiation(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		want           string
	}{
		{
			name:           "wildcard applies to unlisted encoding",
			acceptEncoding: "gzip;q=0.2, *;q=0.8",
			want:           "br",
		},
		{
			name:           "explicit refusal overrides earlier wildcard",
			acceptEncoding: "*, br;q=0",
			want:           "gzip",
		},
		{
			name:           "wildcard applies after explicit refusal",
			acceptEncoding: "br;q=0, *;q=0.5",
			want:           "gzip",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			compressor := brotli.HTTPCompressor(recorder, req)
			if err := compressor.Close(); err != nil {
				t.Fatal(err)
			}
			if got := recorder.Header().Get("Content-Encoding"); got != tt.want {
				t.Fatalf("Content-Encoding = %q, want %q", got, tt.want)
			}
		})
	}
}
