package httpapi

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
)

func NewMux(db *sql.DB, staticDir string, mqttStatus MQTTConnectedChecker) *http.ServeMux {
	mux := http.NewServeMux()
	registerHealthcheck(mux, db, mqttStatus)
	if staticDir != "" {
		if _, err := os.Stat(staticDir); err == nil {
			mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
		} else {
			slog.Warn("static directory not found or not readable; /static/ routes will not be served", "dir", staticDir, "err", err)
		}
	}
	return mux
}
