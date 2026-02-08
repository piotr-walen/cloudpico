package httpapi

import (
	"cloudpico-server/internal/utils"
	"database/sql"
	"log/slog"
	"net/http"
)

// MQTTConnectedChecker is implemented by *mqtt.Subscriber for healthcheck.
type MQTTConnectedChecker interface {
	Connected() bool
}

type healthchecker interface {
	handleHealthz(w http.ResponseWriter, r *http.Request)
}

type healthcheckerImpl struct {
	db         *sql.DB
	mqttStatus MQTTConnectedChecker
}

func NewHealthchecker(db *sql.DB, mqttStatus MQTTConnectedChecker) healthchecker {
	return &healthcheckerImpl{db: db, mqttStatus: mqttStatus}
}

func (h *healthcheckerImpl) handleHealthz(w http.ResponseWriter, r *http.Request) {
	var ok int
	if err := h.db.QueryRow(`SELECT 1`).Scan(&ok); err != nil {
		slog.Error("failed to check database connectivity", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to check database connectivity")
		return
	}
	body := map[string]any{"status": "ok"}
	if h.mqttStatus != nil {
		if h.mqttStatus.Connected() {
			body["mqtt"] = "connected"
		} else {
			body["mqtt"] = "disconnected"
		}
	}
	utils.WriteJSON(w, http.StatusOK, body)
}

func registerHealthcheck(mux *http.ServeMux, db *sql.DB, mqttStatus MQTTConnectedChecker) {
	healthchecker := NewHealthchecker(db, mqttStatus)
	mux.HandleFunc("GET /healthz", healthchecker.handleHealthz)
}
