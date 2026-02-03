package httpapi

import (
	"database/sql"
	"net/http"
	"os"
)

func NewMux(db *sql.DB, staticDir string) *http.ServeMux {
	mux := http.NewServeMux()
	registerHealthcheck(mux, db)
	if staticDir != "" {
		if _, err := os.Stat(staticDir); err == nil {
			mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
		}
	}
	return mux
}
