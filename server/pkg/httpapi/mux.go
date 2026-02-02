package httpapi

import (
	"database/sql"
	"net/http"
)

func NewMux(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()
	registerHealthcheck(mux, db)
	return mux
}
