package httpapi

import (
	"cloudpico-server/pkg/config"
	"cloudpico-server/pkg/db"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Station struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Reading struct {
	StationID string    `json:"stationId"`
	Time      time.Time `json:"time"`
	Value     float64   `json:"value"`
}

type weatherAPIImpl struct {
	config config.Config
}

type WeatherAPI interface {
	HandleHealthz(w http.ResponseWriter, r *http.Request)
	HandleStations(w http.ResponseWriter, r *http.Request)
	HandleLatest(w http.ResponseWriter, r *http.Request)
	HandleReadings(w http.ResponseWriter, r *http.Request)
}

func NewWeatherAPI(config config.Config) WeatherAPI {
	return &weatherAPIImpl{config: config}
}

func (weatherAPI *weatherAPIImpl) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	dbConn, err := db.Open(weatherAPI.config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = db.Close(dbConn) }()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (weatherAPI *weatherAPIImpl) HandleStations(w http.ResponseWriter, r *http.Request) {
	// TODO: replace with real data source
	stations := []Station{
		{ID: "st-001", Name: "Central"},
		{ID: "st-002", Name: "North"},
	}
	writeJSON(w, http.StatusOK, stations)
}

func (weatherAPI *weatherAPIImpl) HandleLatest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing station id")
		return
	}

	latest := Reading{
		StationID: id,
		Time:      time.Now().UTC().Truncate(time.Second),
		Value:     12.34,
	}
	writeJSON(w, http.StatusOK, latest)
}

func (weatherAPI *weatherAPIImpl) HandleReadings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing station id")
		return
	}

	from, to, limit, err := parseReadingsQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	items := make([]Reading, 0, limit)

	for i := 0; i < limit; i++ {
		t := now.Add(-time.Duration(i) * time.Minute)

		if !from.IsZero() && t.Before(from) {
			continue
		}
		if !to.IsZero() && t.After(to) {
			continue
		}

		items = append(items, Reading{
			StationID: id,
			Time:      t,
			Value:     10.0 + float64(i)*0.1,
		})
	}

	resp := map[string]any{
		"stationId": id,
		"from":      zeroAsNullTime(from),
		"to":        zeroAsNullTime(to),
		"limit":     limit,
		"items":     items,
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseReadingsQuery(r *http.Request) (from time.Time, to time.Time, limit int, err error) {
	q := r.URL.Query()

	if s := q.Get("from"); s != "" {
		from, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, errors.New("invalid 'from' (expected RFC3339)")
		}
	}
	if s := q.Get("to"); s != "" {
		to, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, errors.New("invalid 'to' (expected RFC3339)")
		}
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return time.Time{}, time.Time{}, 0, errors.New("'from' must be <= 'to'")
	}

	limit = 100
	if s := q.Get("limit"); s != "" {
		n, convErr := strconv.Atoi(s)
		if convErr != nil {
			return time.Time{}, time.Time{}, 0, errors.New("invalid 'limit' (expected integer)")
		}
		if n <= 0 {
			return time.Time{}, time.Time{}, 0, errors.New("'limit' must be > 0")
		}
		if n > 1000 {
			return time.Time{}, time.Time{}, 0, errors.New("'limit' must be <= 1000")
		}
		limit = n
	}

	return from, to, limit, nil
}

func zeroAsNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		slog.Error("failed to write JSON", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error":   http.StatusText(status),
		"message": msg,
	})
}
