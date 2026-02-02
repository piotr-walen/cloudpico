package httpapi

import (
	"cloudpico-server/internal/utils"
	"database/sql"
	"log/slog"
	"net/http"
)

type healthchecker interface {
	handleHealthz(w http.ResponseWriter, r *http.Request)
}

type healthcheckerImpl struct {
	db *sql.DB
}

func NewHealthchecker(db *sql.DB) healthchecker {
	return &healthcheckerImpl{db: db}
}

func (h *healthcheckerImpl) handleHealthz(w http.ResponseWriter, r *http.Request) {
	var ok int
	if err := h.db.QueryRow(`SELECT 1`).Scan(&ok); err != nil {
		slog.Error("failed to check database connectivity", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to check database connectivity")
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func registerHealthcheck(mux *http.ServeMux, db *sql.DB) {
	healthchecker := NewHealthchecker(db)
	mux.HandleFunc("GET /healthz", healthchecker.handleHealthz)
}
