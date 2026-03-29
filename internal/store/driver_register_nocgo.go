//go:build !cgo

package store

import (
	"database/sql"
	"sync"

	sqlite "modernc.org/sqlite"
)

var registerOnce sync.Once

func registerDriver() {
	registerOnce.Do(func() {
		sql.Register(sqliteDriverName, &sqlite.Driver{})
	})
}
