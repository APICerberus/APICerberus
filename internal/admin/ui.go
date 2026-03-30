package admin

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	apicerberus "github.com/APICerberus/APICerebrus"
)

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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	})
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
