package controller

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"cloudpico-server/internal/modules/weather/views"
	"cloudpico-server/internal/utils"
)

func (c *weatherControllerImpl) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("dashboard: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}
	selectedID := r.URL.Query().Get("station_id")
	if selectedID == "" && len(stations) > 0 {
		selectedID = stations[0].ID
	}
	opts := make([]views.StationOption, 0, len(stations))
	for _, s := range stations {
		opts = append(opts, views.StationOption{ID: s.ID, Name: s.Name})
	}
	data := views.DashboardData{Stations: opts, SelectedStationID: selectedID}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderDashboard(w, data); err != nil {
		slog.Error("dashboard template render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render page")
		return
	}
}

func (c *weatherControllerImpl) handleStations(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, stations)
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

	readings, err := c.repository.GetReadings(id, from, to, limit, 0)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, readings)
}

func (c *weatherControllerImpl) handleCurrentConditionsPartial(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("current conditions: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}

	stationID := r.URL.Query().Get("station_id")
	var stationName string
	if stationID == "" {
		if len(stations) == 0 {
			var buf bytes.Buffer
			if err := views.RenderCurrentConditionsPartial(&buf, views.CurrentConditionsData{StationName: "", Reading: nil}); err != nil {
				slog.Error("current conditions partial render failed", "error", err)
				utils.WriteError(w, http.StatusInternalServerError, "failed to render")
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err := w.Write(buf.Bytes()); err != nil {
				slog.Error("current conditions: write response failed", "error", err)
			}
			return
		}
		stationID = stations[0].ID
		stationName = stations[0].Name
	} else {
		for _, s := range stations {
			if s.ID == stationID {
				stationName = s.Name
				break
			}
		}
		if stationName == "" {
			slog.Warn("current conditions: unknown station_id", "station_id", stationID)
			stationName = "Unknown Station"
		}
	}

	latest, err := c.repository.GetLatestReadings(stationID, 1)
	if err != nil {
		slog.Error("current conditions: get latest failed", "station_id", stationID, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load reading")
		return
	}

	var reading *views.ReadingPartial
	if len(latest) > 0 {
		reading = &views.ReadingPartial{
			Value:       latest[0].Value,
			Time:        latest[0].Time,
			HumidityPct: latest[0].HumidityPct,
			PressureHpa: latest[0].PressureHpa,
		}
	}

	data := views.CurrentConditionsData{StationName: stationName, Reading: reading}
	var buf bytes.Buffer
	if err := views.RenderCurrentConditionsPartial(&buf, data); err != nil {
		slog.Error("current conditions partial render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("current conditions: write response failed", "error", err)
	}
}

// buildHistoryPageItems returns page numbers and ellipsis for the pagination bar.
func buildHistoryPageItems(totalPages, currentPage int) []views.PaginationItem {
	if totalPages <= 0 {
		return nil
	}
	const window = 2
	show := map[int]bool{1: true, totalPages: true}
	for p := currentPage - window; p <= currentPage+window; p++ {
		if p >= 1 && p <= totalPages {
			show[p] = true
		}
	}
	var items []views.PaginationItem
	prev := 0
	for p := 1; p <= totalPages; p++ {
		if !show[p] {
			continue
		}
		if prev != 0 && p > prev+1 {
			items = append(items, views.PaginationItem{Ellipsis: true})
		}
		items = append(items, views.PaginationItem{Page: p, Ellipsis: false})
		prev = p
	}
	return items
}

func (c *weatherControllerImpl) handleHistoryPartial(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("history: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}

	rangeKey := r.URL.Query().Get("range")
	rangeInfo, ok := resolveHistoryRange(rangeKey)
	if !ok && rangeKey != "" {
		slog.Warn("history: invalid range", "range", rangeKey)
	}
	resolvedRangeKey := rangeKey
	if resolvedRangeKey == "" || !ok {
		resolvedRangeKey = defaultHistoryRangeKey
	}

	page := parseHistoryPage(r)
	offset := (page - 1) * historyPageSize

	stationID := r.URL.Query().Get("station_id")
	var stationName string
	if stationID == "" {
		if len(stations) == 0 {
			data := views.HistoryData{
				StationName: "",
				StationID:   "",
				RangeLabel:  rangeInfo.Label,
				RangeKey:    resolvedRangeKey,
				Readings:    nil,
				CurrentPage: 1,
				TotalPages:  1,
				HasPrev:     false,
				HasNext:     false,
				PrevPage:    1,
				NextPage:    2,
				PageItems:   []views.PaginationItem{{Page: 1, Ellipsis: false}},
			}
			var buf bytes.Buffer
			if err := views.RenderHistoryPartial(&buf, data); err != nil {
				slog.Error("history partial render failed", "error", err)
				utils.WriteError(w, http.StatusInternalServerError, "failed to render")
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err := w.Write(buf.Bytes()); err != nil {
				slog.Error("history: write response failed", "error", err)
			}
			return
		}
		stationID = stations[0].ID
		stationName = stations[0].Name
	} else {
		for _, s := range stations {
			if s.ID == stationID {
				stationName = s.Name
				break
			}
		}
		if stationName == "" {
			slog.Warn("history: unknown station_id", "station_id", stationID)
			stationName = "Unknown Station"
		}
	}

	now := time.Now().UTC()
	from := now.Add(-rangeInfo.Duration)

	count, err := c.repository.GetReadingsCount(stationID, from, now)
	if err != nil {
		slog.Error("history: get readings count failed", "station_id", stationID, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load readings")
		return
	}
	totalPages := (count + historyPageSize - 1) / historyPageSize
	if totalPages < 1 {
		totalPages = 1
	}

	limit := historyPageSize + 1
	readings, err := c.repository.GetReadings(stationID, from, now, limit, offset)
	if err != nil {
		slog.Error("history: get readings failed", "station_id", stationID, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load readings")
		return
	}

	hasNext := len(readings) > historyPageSize
	if hasNext {
		readings = readings[:historyPageSize]
	}
	partials := make([]views.ReadingPartial, 0, len(readings))
	for _, reading := range readings {
		partials = append(partials, views.ReadingPartial{
			Value:       reading.Value,
			Time:        reading.Time,
			HumidityPct: reading.HumidityPct,
			PressureHpa: reading.PressureHpa,
		})
	}

	data := views.HistoryData{
		StationName: stationName,
		StationID:   stationID,
		RangeLabel:  rangeInfo.Label,
		RangeKey:    resolvedRangeKey,
		Readings:    partials,
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		PageItems:   buildHistoryPageItems(totalPages, page),
	}
	var buf bytes.Buffer
	if err := views.RenderHistoryPartial(&buf, data); err != nil {
		slog.Error("history partial render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("history: write response failed", "error", err)
	}
}
