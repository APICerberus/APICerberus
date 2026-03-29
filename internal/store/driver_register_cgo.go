//go:build cgo

package store

/*
#cgo CFLAGS: -I${SRCDIR}
#include "sqlite3.h"
*/
import "C"

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
