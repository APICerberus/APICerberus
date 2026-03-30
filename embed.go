package apicerberus

import (
	"embed"
	"io/fs"
)

var (
	// dashboardDistFS embeds the built admin dashboard assets.
	//
	//go:embed web/dist/*
	dashboardDistFS embed.FS
)

// EmbeddedDashboardFS returns the embedded admin dashboard filesystem rooted at web/dist.
func EmbeddedDashboardFS() (fs.FS, error) {
	return fs.Sub(dashboardDistFS, "web/dist")
}
