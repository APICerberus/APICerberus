package admin

import (
	"crypto/rand"
	"encoding/base64"
	"io/fs"
	"net/http"
	"path"
	"strings"

	apicerberus "github.com/APICerberus/APICerebrus"
)

// generateCSPNonce generates a random 16-byte nonce for CSP.
func generateCSPNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func embeddedDashboardFS() (fs.FS, error) {
	return apicerberus.EmbeddedDashboardFS()
}

func (s *Server) newDashboardHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.dashboardFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(strings.TrimSpace(r.URL.Path), "/admin/api/") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		cleanPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		requested := strings.TrimPrefix(cleanPath, "/")

		if requested != "" && dashboardAssetExists(s.dashboardFS, requested) {
			fileServer.ServeHTTP(w, r)
			return
		}

		index, err := fs.ReadFile(s.dashboardFS, "index.html")
		if err != nil {
			http.Error(w, "dashboard assets unavailable", http.StatusServiceUnavailable)
			return
		}
		// M-003: Generate per-request nonce for CSP to replace 'unsafe-inline'.
		// The nonce is injected into the inline script and referenced in the CSP header.
		nonce := generateCSPNonce()
		// Inject nonce into the inline theme flash prevention script.
		// The script tag is: <script>\n      (function() {...
		// We need to replace just the opening <script> with <script nonce="...">
		index = []byte(strings.Replace(
			string(index),
			"<script>\n      ",
			`<script nonce="`+nonce+`">\n      `,
			1,
		))
		setDashboardSecurityHeadersWithNonce(w, nonce)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	})
}

func setDashboardSecurityHeadersWithNonce(w http.ResponseWriter, nonce string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	// M-003: Replaced 'unsafe-inline' with nonce-based script allowlist.
	// Only the inline script with the matching nonce is permitted.
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self' 'nonce-"+nonce+"'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
}

func dashboardAssetExists(fileSystem fs.FS, name string) bool {
	if fileSystem == nil {
		return false
	}
	file, err := fileSystem.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return false
	}
	return !info.IsDir()
}
