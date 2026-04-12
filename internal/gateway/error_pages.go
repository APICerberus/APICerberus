package gateway

import (
	"fmt"
	"net/http"
	"strings"
)

// htmlErrorPage renders a minimal HTML error response.
func htmlErrorPage(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>%d %s</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;margin:0;display:flex;align-items:center;justify-content:center;min-height:100vh;background:#f5f5f5;color:#333}
.error-box{text-align:center;padding:48px;background:#fff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.1);max-width:480px}
.error-code{font-size:64px;font-weight:700;color:#e74c3c;margin:0;line-height:1}
.error-message{font-size:18px;margin:16px 0 8px}
.error-detail{font-size:14px;color:#666;font-family:monospace}
</style>
</head>
<body>
<div class="error-box">
<p class="error-code">%d</p>
<p class="error-message">%s</p>
<p class="error-detail">%s</p>
</div>
</body>
</html>`, status, http.StatusText(status), status, http.StatusText(status), escapeHTML(message))
	_, _ = w.Write([]byte(html))
}

// escapeHTML escapes basic HTML entities to prevent XSS in error messages.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
