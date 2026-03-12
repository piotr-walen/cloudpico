package controller

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"cloudpico-api/internal/modules/weather/types"
	"cloudpico-api/internal/utils"
)

const (
	defaultHistoryRangeKey = "24h"
	historyPageSize        = 20
)

type historyRange struct {
	Duration time.Duration
	Label    string
}

var historyRanges = map[string]historyRange{
	"1h":  {Duration: time.Hour, Label: "Last 1 hour"},
	"6h":  {Duration: 6 * time.Hour, Label: "Last 6 hours"},
	"24h": {Duration: 24 * time.Hour, Label: "Last 24 hours"},
	"3d":  {Duration: 3 * 24 * time.Hour, Label: "Last 3 days"},
	"7d":  {Duration: 7 * 24 * time.Hour, Label: "Last 7 days"},
	"30d": {Duration: 30 * 24 * time.Hour, Label: "Last 30 days"},
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

func parseLatestQuery(r *http.Request) (limit int, err error) {
	q := r.URL.Query()
	limit = 100
	if s := q.Get("limit"); s != "" {
		n, convErr := strconv.Atoi(s)
		if convErr != nil {
			return 0, errors.New("invalid 'limit' (expected integer)")
		}
		if n <= 0 {
			return 0, errors.New("'limit' must be > 0")
		}
		if n > 1000 {
			return 0, errors.New("'limit' must be <= 1000")
		}
		limit = n
	}
	return limit, nil
}

func resolveHistoryRange(key string) (historyRange, bool) {
	if key == "" {
		return historyRanges[defaultHistoryRangeKey], true
	}
	info, ok := historyRanges[key]
	if ok {
		return info, true
	}
	return historyRanges[defaultHistoryRangeKey], false
}

func zeroAsNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// stationReading is the API response type for stations with latest reading.
type stationReading struct {
	StationID         string         `json:"stationId"`
	StationName       string         `json:"stationName"`
	Reading           *types.Reading `json:"reading"`
	BatteryPercentage float64        `json:"batteryPercentage"`
}

// batteryPercentageFromReading returns SOC from fuel gauge when available (>= 0), else voltage-based estimate.
func batteryPercentageFromReading(r types.Reading) float64 {
	if r.BatterySOCPct >= 0 && r.BatterySOCPct <= 100 {
		return r.BatterySOCPct
	}
	return calculateBatteryPercentage(r.BatteryV)
}

// calculateBatteryPercentage calculates battery percentage from voltage (3.3V = 0%, 4.2V = 100%).
func calculateBatteryPercentage(voltage float64) float64 {
	if voltage <= 0.0 || voltage <= 3.3 {
		return 0.0
	}
	if voltage >= 4.2 {
		return 100.0
	}
	percentage := ((voltage - 3.3) / 0.9) * 100.0
	return math.Round(percentage*10) / 10
}

func (c *weatherControllerImpl) handleStations(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, stations)
}

func (c *weatherControllerImpl) handleStationsWithLatest(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("stations with latest: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}

	result := make([]stationReading, 0, len(stations))
	for _, s := range stations {
		latest, err := c.repository.GetLatestReadings(s.ID, 1)
		if err != nil {
			slog.Error("stations with latest: get latest reading failed", "station_id", s.ID, "error", err)
			result = append(result, stationReading{StationID: s.ID, StationName: s.Name, Reading: nil})
			continue
		}
		if len(latest) != 0 {
			batteryPct := batteryPercentageFromReading(latest[0])
			result = append(result, stationReading{
				StationID:         s.ID,
				StationName:       s.Name,
				Reading:           &latest[0],
				BatteryPercentage: batteryPct,
			})
			continue
		}
		result = append(result, stationReading{StationID: s.ID, StationName: s.Name, Reading: nil})
	}

	utils.WriteJSON(w, http.StatusOK, result)
}

func (c *weatherControllerImpl) handleLatest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	limit, err := parseLatestQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	latest, err := c.repository.GetLatestReadings(id, limit)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, latest)
}

func (c *weatherControllerImpl) handleReadings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	from, to, limit, err := parseReadingsQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	readings, err := c.repository.GetReadings(id, from, to, limit)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, readings)
}

// parseHistoryQuery parses query parameters for history endpoints: range, page, limit.
// Returns from time, to time, page (1-based), limit, and error.
func parseHistoryQuery(r *http.Request) (from time.Time, to time.Time, page int, limit int, err error) {
	q := r.URL.Query()

	// Parse range
	rangeKey := q.Get("range")
	if rangeKey == "" {
		rangeKey = defaultHistoryRangeKey
	}
	rangeInfo, ok := resolveHistoryRange(rangeKey)
	if !ok {
		rangeInfo, _ = resolveHistoryRange(defaultHistoryRangeKey)
	}

	now := time.Now().UTC()
	from = now.Add(-rangeInfo.Duration)
	to = now

	// Parse page (1-based)
	page = 1
	if s := q.Get("page"); s != "" {
		if n, convErr := strconv.Atoi(s); convErr == nil && n >= 1 {
			page = n
		}
	}

	// Parse limit
	limit = historyPageSize
	if s := q.Get("limit"); s != "" {
		if n, convErr := strconv.Atoi(s); convErr == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	return from, to, page, limit, nil
}

func (c *weatherControllerImpl) handleHistoryReadings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	from, to, _, limit, err := parseHistoryQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	readings, err := c.repository.GetReadings(id, from, to, limit)
	if err != nil {
		slog.Error("history readings: get readings failed", "station_id", id, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load readings")
		return
	}

	if readings == nil {
		readings = []types.Reading{}
	}

	total := len(readings)
	response := map[string]interface{}{
		"readings":   readings,
		"page":       1,
		"limit":      limit,
		"total":      total,
		"totalPages": 1,
		"hasPrev":    false,
		"hasNext":    false,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func (c *weatherControllerImpl) handleHistoryChart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	from, to, _, _, err := parseHistoryQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch all readings for the chart (up to 10000 to avoid performance issues)
	chartReadings, err := c.repository.GetReadings(id, from, to, 10000)
	if err != nil {
		slog.Error("history chart: get chart readings failed", "station_id", id, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load chart data")
		return
	}

	if chartReadings == nil {
		chartReadings = []types.Reading{}
	}

	utils.WriteJSON(w, http.StatusOK, chartReadings)
}

func (c *weatherControllerImpl) handleHistoryCount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	from, to, _, _, err := parseHistoryQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	count, err := c.repository.GetReadingsCount(id, from, to)
	if err != nil {
		slog.Error("history count: get count failed", "station_id", id, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load count")
		return
	}

	response := map[string]interface{}{
		"count": count,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}
