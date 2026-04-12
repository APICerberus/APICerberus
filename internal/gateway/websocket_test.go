package gateway

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestProxyForwardWebSocketTunnel(t *testing.T) {
	t.Parallel()

	pathCh := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWebSocketUpgrade(r) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pathCh <- r.URL.Path

		h, ok := w.(http.Hijacker)
		if !ok {
			t.Errorf("upstream writer is not hijackable")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, rw, err := h.Hijack()
		if err != nil {
			t.Errorf("upstream hijack failed: %v", err)
			return
		}
		defer conn.Close()

		accept := websocketAccept(r.Header.Get("Sec-WebSocket-Key"))
		_, _ = fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\n")
		_, _ = fmt.Fprintf(rw, "Upgrade: websocket\r\n")
		_, _ = fmt.Fprintf(rw, "Connection: Upgrade\r\n")
		_, _ = fmt.Fprintf(rw, "Sec-WebSocket-Accept: %s\r\n", accept)
		_, _ = fmt.Fprintf(rw, "\r\n")
		_ = rw.Flush()

		_, _ = io.Copy(conn, conn)
	}))
	defer upstream.Close()

	proxy := NewProxy()
	target := &config.UpstreamTarget{Address: upstream.URL}

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := proxy.ForwardWebSocket(&RequestContext{
			Request:        r,
			ResponseWriter: w,
			Route: &config.Route{
				StripPath: true,
				Paths:     []string{"/ws/*"},
			},
		}, target)
		if err != nil {
			// In test flow this should not happen after successful upgrade.
			t.Errorf("ForwardWebSocket failed: %v", err)
		}
	}))
	defer gateway.Close()

	gatewayURL, err := url.Parse(gateway.URL)
	if err != nil {
		t.Fatalf("parse gateway url: %v", err)
	}

	conn, err := net.DialTimeout("tcp", gatewayURL.Host, 3*time.Second)
	if err != nil {
		t.Fatalf("dial gateway: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	key := "dGhlIHNhbXBsZSBub25jZQ=="
	reqText := strings.Join([]string{
		"GET /ws/echo HTTP/1.1",
		"Host: gateway.local",
		"Upgrade: websocket",
		"Connection: Upgrade",
		"Sec-WebSocket-Key: " + key,
		"Sec-WebSocket-Version: 13",
		"",
		"",
	}, "\r\n")
	if _, err := conn.Write([]byte(reqText)); err != nil {
		t.Fatalf("write websocket request: %v", err)
	}

	reader := bufio.NewReader(conn)
	dummyReq, _ := http.NewRequest(http.MethodGet, "http://gateway.local/ws/echo", nil)
	resp, err := http.ReadResponse(reader, dummyReq)
	if err != nil {
		t.Fatalf("read handshake response: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101 switching protocols got %d", resp.StatusCode)
	}

	if got := <-pathCh; got != "/echo" {
		t.Fatalf("expected strip_path to forward /echo got %q", got)
	}

	payload := []byte("hello-websocket-tunnel")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	got := make([]byte, len(payload))
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatalf("read echoed payload: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("unexpected echoed payload: %q", string(got))
	}
}

func websocketAccept(key string) string {
	const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	sum := sha1.Sum([]byte(strings.TrimSpace(key) + guid))
	return base64.StdEncoding.EncodeToString(sum[:])
}
