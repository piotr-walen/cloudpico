package controller

import (
	"bytes"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"cloudpico-server/internal/modules/weather/views"
	"cloudpico-server/internal/utils"
)

func (c *weatherControllerImpl) handleStationsPartial(w http.ResponseWriter, r *http.Request) {
	data := views.DashboardData{}
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("stations partial: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}

	for _, s := range stations {
		latest, err := c.repository.GetLatestReadings(s.ID, 1)
		if err != nil {
			slog.Error("stations partial: get latest reading failed", "station_id", s.ID, "error", err)
			utils.WriteError(w, http.StatusInternalServerError, "failed to load reading")
			return
		}
		if len(latest) != 0 {
			data.Stations = append(data.Stations, views.StationReading{StationID: s.ID, StationName: s.Name, Reading: &latest[0]})
			continue
		}
		data.Stations = append(data.Stations, views.StationReading{StationID: s.ID, StationName: s.Name, Reading: nil})
	}

	var buf bytes.Buffer
	if err := views.RenderStationsPartial(&buf, &data); err != nil {
		slog.Error("stations partial render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("stations partial: write response failed", "error", err)
	}
}

func (c *weatherControllerImpl) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := views.DashboardData{}
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("dashboard: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}

	for _, s := range stations {
		latest, err := c.repository.GetLatestReadings(s.ID, 1)
		if err != nil {
			slog.Error("dashboard: get latest reading failed", "station_id", s.ID, "error", err)
			utils.WriteError(w, http.StatusInternalServerError, "failed to load reading")
			return
		}
		if len(latest) != 0 {
			data.Stations = append(data.Stations, views.StationReading{StationID: s.ID, StationName: s.Name, Reading: &latest[0]})
			continue
		}
		data.Stations = append(data.Stations, views.StationReading{StationID: s.ID, StationName: s.Name, Reading: nil})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderDashboard(w, &data); err != nil {
		slog.Error("dashboard template render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render page")
		return
	}
}

func (c *weatherControllerImpl) handleHistory(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("dashboard: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}
	state := readWeatherStateCookie(r)
	selectedID := r.URL.Query().Get("station_id")
	if selectedID == "" {
		selectedID = state.StationID
	}
	if selectedID == "" && len(stations) > 0 {
		selectedID = stations[0].ID
	}
	selectedRangeKey := r.URL.Query().Get("range")
	if selectedRangeKey == "" {
		selectedRangeKey = state.RangeKey
	}
	if selectedRangeKey == "" {
		selectedRangeKey = defaultHistoryRangeKey
	}
	opts := make([]views.StationOption, 0, len(stations))
	for _, s := range stations {
		opts = append(opts, views.StationOption{ID: s.ID, Name: s.Name})
	}
	data := views.HistoryParams{
		Stations:          opts,
		SelectedStationID: selectedID,
		SelectedRangeKey:  selectedRangeKey,
	}
	writeWeatherStateCookie(w, selectedID, selectedRangeKey, state.Page)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderHistory(w, &data); err != nil {
		slog.Error("history template render failed", "error", err)
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

// buildHistoryPageItems returns page numbers and ellipsis for the pagination bar.
// It only considers {1, totalPages, current±window}, so work is O(1) in totalPages.
func buildHistoryPageItems(totalPages, currentPage int) []views.PaginationItem {
	if totalPages <= 0 {
		return nil
	}
	const window = 2
	// Collect only pages to show: first, last, and current ± window (clamped).
	seen := map[int]bool{1: true, totalPages: true}
	for p := currentPage - window; p <= currentPage+window; p++ {
		if p >= 1 && p <= totalPages {
			seen[p] = true
		}
	}
	sorted := make([]int, 0, len(seen))
	for p := range seen {
		sorted = append(sorted, p)
	}
	sort.Ints(sorted)
	var items []views.PaginationItem
	prev := 0
	for _, p := range sorted {
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

	state := readWeatherStateCookie(r)
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = state.RangeKey
	}
	rangeInfo, ok := resolveHistoryRange(rangeKey)
	if !ok && rangeKey != "" {
		slog.Warn("history: invalid range", "range", rangeKey)
	}
	resolvedRangeKey := rangeKey
	if resolvedRangeKey == "" || !ok {
		resolvedRangeKey = defaultHistoryRangeKey
		rangeInfo, _ = resolveHistoryRange(resolvedRangeKey)
	}

	requestStation := r.URL.Query().Get("station_id")
	if requestStation == "" {
		requestStation = state.StationID
	}

	page := parseHistoryPage(r)
	if r.URL.Query().Get("page") == "" {
		if requestStation != state.StationID || resolvedRangeKey != state.RangeKey {
			page = 1
		} else if state.Page >= 1 {
			page = state.Page
		}
	}

	stationID := requestStation
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
			if err := views.RenderHistoryPartial(&buf, &data); err != nil {
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
	totalPages := 1
	if count > 0 {
		totalPages = (count + historyPageSize - 1) / historyPageSize
	}
	if page < 1 || page > totalPages {
		page = 1
	}
	offset := (page - 1) * historyPageSize

	readings, err := c.repository.GetReadings(stationID, from, now, historyPageSize, offset)
	if err != nil {
		slog.Error("history: get readings failed", "station_id", stationID, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load readings")
		return
	}

	data := views.HistoryData{
		StationName: stationName,
		StationID:   stationID,
		RangeLabel:  rangeInfo.Label,
		RangeKey:    resolvedRangeKey,
		Readings:    readings,
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		PageItems:   buildHistoryPageItems(totalPages, page),
	}
	writeWeatherStateCookie(w, stationID, resolvedRangeKey, page)
	var buf bytes.Buffer
	if err := views.RenderHistoryPartial(&buf, &data); err != nil {
		slog.Error("history partial render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("history: write response failed", "error", err)
	}
}
