package controller

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"cloudpico-server/internal/modules/weather/types"
	"cloudpico-server/internal/modules/weather/views"
	"cloudpico-server/internal/utils"
)

// batteryPercentageFromReading returns SOC from fuel gauge when available (>= 0), else voltage-based estimate.
func batteryPercentageFromReading(r types.Reading) float64 {
	if r.BatterySOCPct >= 0 && r.BatterySOCPct <= 100 {
		return r.BatterySOCPct
	}
	return views.CalculateBatteryPercentage(r.BatteryV)
}

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
			batteryPct := batteryPercentageFromReading(latest[0])
			data.Stations = append(data.Stations, views.StationReading{
				StationID:         s.ID,
				StationName:       s.Name,
				Reading:           &latest[0],
				BatteryPercentage: batteryPct,
			})
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
			batteryPct := batteryPercentageFromReading(latest[0])
			data.Stations = append(data.Stations, views.StationReading{
				StationID:         s.ID,
				StationName:       s.Name,
				Reading:           &latest[0],
				BatteryPercentage: batteryPct,
			})
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
	// Handle show_y_axes: read from query param, fallback to cookie, default to true
	showYAxes := true
	if s := r.URL.Query().Get("show_y_axes"); s != "" {
		if b, err := strconv.ParseBool(s); err == nil {
			showYAxes = b
		} else if state.ShowYAxes != nil {
			showYAxes = *state.ShowYAxes
		}
	} else if state.ShowYAxes != nil {
		showYAxes = *state.ShowYAxes
	}
	opts := make([]views.StationOption, 0, len(stations))
	for _, s := range stations {
		opts = append(opts, views.StationOption{ID: s.ID, Name: s.Name})
	}
	data := views.HistoryParams{
		Stations:          opts,
		SelectedStationID: selectedID,
		SelectedRangeKey:  selectedRangeKey,
		ShowYAxes:         showYAxes,
	}
	showYAxesPtr := &showYAxes
	writeWeatherStateCookie(w, selectedID, selectedRangeKey, state.Page, showYAxesPtr)
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

func (c *weatherControllerImpl) handleStationsWithLatest(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("stations with latest: get stations failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
		return
	}

	result := make([]views.StationReading, 0, len(stations))
	for _, s := range stations {
		latest, err := c.repository.GetLatestReadings(s.ID, 1)
		if err != nil {
			slog.Error("stations with latest: get latest reading failed", "station_id", s.ID, "error", err)
			// Continue with nil reading instead of failing completely
			result = append(result, views.StationReading{
				StationID:   s.ID,
				StationName: s.Name,
				Reading:     nil,
			})
			continue
		}
		if len(latest) != 0 {
			batteryPct := batteryPercentageFromReading(latest[0])
			result = append(result, views.StationReading{
				StationID:         s.ID,
				StationName:       s.Name,
				Reading:           &latest[0],
				BatteryPercentage: batteryPct,
			})
			continue
		}
		result = append(result, views.StationReading{
			StationID:   s.ID,
			StationName: s.Name,
			Reading:     nil,
		})
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

	readings, err := c.repository.GetReadings(id, from, to, limit, 0)
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
	
	from, to, page, limit, err := parseHistoryQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	offset := (page - 1) * limit
	readings, err := c.repository.GetReadings(id, from, to, limit, offset)
	if err != nil {
		slog.Error("history readings: get readings failed", "station_id", id, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load readings")
		return
	}
	
	// Ensure readings is never nil (empty slice instead)
	if readings == nil {
		readings = []types.Reading{}
	}
	
	count, err := c.repository.GetReadingsCount(id, from, to)
	if err != nil {
		slog.Error("history readings: get count failed", "station_id", id, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load count")
		return
	}
	
	totalPages := 1
	if count > 0 {
		totalPages = (count + limit - 1) / limit
	}
	
	response := map[string]interface{}{
		"readings":   readings,
		"page":       page,
		"limit":      limit,
		"total":      count,
		"totalPages": totalPages,
		"hasPrev":    page > 1,
		"hasNext":    page < totalPages,
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
	chartReadings, err := c.repository.GetReadings(id, from, to, 10000, 0)
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

	// Handle show_y_axes: read from query param, fallback to cookie, default to true
	showYAxes := true
	if s := r.URL.Query().Get("show_y_axes"); s != "" {
		if b, err := strconv.ParseBool(s); err == nil {
			showYAxes = b
		} else if state.ShowYAxes != nil {
			showYAxes = *state.ShowYAxes
		}
	} else if state.ShowYAxes != nil {
		showYAxes = *state.ShowYAxes
	}

	stationID := requestStation
	var stationName string
	if stationID == "" {
		if len(stations) == 0 {
			data := views.HistoryData{
				StationName:   "",
				StationID:     "",
				RangeLabel:    rangeInfo.Label,
				RangeKey:      resolvedRangeKey,
				Readings:      nil,
				ChartReadings: nil,
				CurrentPage:   1,
				TotalPages:    1,
				HasPrev:       false,
				HasNext:       false,
				PrevPage:      1,
				NextPage:      2,
				PageItems:     []views.PaginationItem{{Page: 1, Ellipsis: false}},
				ShowYAxes:     showYAxes,
			}
			if err := data.SetChartReadingsJSON(); err != nil {
				slog.Error("history: marshal chart readings JSON failed", "error", err)
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

	// Fetch all readings for the chart (up to 10000 to avoid performance issues)
	chartReadings, err := c.repository.GetReadings(stationID, from, now, 10000, 0)
	if err != nil {
		slog.Error("history: get chart readings failed", "station_id", stationID, "error", err)
		// Continue with empty chart data
		chartReadings = []types.Reading{}
	} else {
		slog.Debug("history: fetched chart readings", "station_id", stationID, "count", len(chartReadings), "from", from, "to", now)
	}

	data := views.HistoryData{
		StationName:   stationName,
		StationID:     stationID,
		RangeLabel:    rangeInfo.Label,
		RangeKey:      resolvedRangeKey,
		Readings:      readings,
		ChartReadings: chartReadings,
		CurrentPage:   page,
		TotalPages:    totalPages,
		HasPrev:       page > 1,
		HasNext:       page < totalPages,
		PrevPage:      page - 1,
		NextPage:      page + 1,
		PageItems:     buildHistoryPageItems(totalPages, page),
		ShowYAxes:     showYAxes,
	}
	if err := data.SetChartReadingsJSON(); err != nil {
		slog.Error("history: marshal chart readings JSON failed", "error", err)
		// Continue anyway, chart just won't work
	} else {
		slog.Debug("history: chart readings JSON prepared", "readings_count", len(chartReadings), "json_length", len(data.ChartReadingsJSON))
	}
	showYAxesPtr := &showYAxes
	writeWeatherStateCookie(w, stationID, resolvedRangeKey, page, showYAxesPtr)
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

func (c *weatherControllerImpl) handleHistoryChartData(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		slog.Error("history-chart-data: get stations failed", "error", err)
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
		slog.Warn("history-chart-data: invalid range", "range", rangeKey)
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

	// Handle show_y_axes: read from query param, fallback to cookie, default to true
	showYAxes := true
	if s := r.URL.Query().Get("show_y_axes"); s != "" {
		if b, err := strconv.ParseBool(s); err == nil {
			showYAxes = b
		} else if state.ShowYAxes != nil {
			showYAxes = *state.ShowYAxes
		}
	} else if state.ShowYAxes != nil {
		showYAxes = *state.ShowYAxes
	}

	stationID := requestStation
	if stationID == "" {
		if len(stations) == 0 {
			// No stations available, return empty data
			data := views.ChartDataParams{
				ChartReadingsJSON: template.JS("[]"),
				ShowYAxes:         showYAxes,
			}
			var buf bytes.Buffer
			if err := views.RenderHistoryChartDataPartial(&buf, &data); err != nil {
				slog.Error("history-chart-data partial render failed", "error", err)
				utils.WriteError(w, http.StatusInternalServerError, "failed to render")
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if _, err := w.Write(buf.Bytes()); err != nil {
				slog.Error("history-chart-data: write response failed", "error", err)
			}
			return
		}
		stationID = stations[0].ID
	}

	now := time.Now().UTC()
	from := now.Add(-rangeInfo.Duration)

	// Fetch all readings for the chart (up to 10000 to avoid performance issues)
	chartReadings, err := c.repository.GetReadings(stationID, from, now, 10000, 0)
	if err != nil {
		slog.Error("history-chart-data: get chart readings failed", "station_id", stationID, "error", err)
		// Continue with empty chart data
		chartReadings = []types.Reading{}
	} else {
		slog.Debug("history-chart-data: fetched chart readings", "station_id", stationID, "count", len(chartReadings), "from", from, "to", now)
	}

	// Prepare chart data
	chartData := views.ChartDataParams{
		ShowYAxes: showYAxes,
	}
	
	// Marshal readings to JSON
	if chartReadings == nil {
		chartReadings = []types.Reading{}
	}
	jsonBytes, err := json.Marshal(chartReadings)
	if err != nil {
		slog.Error("history-chart-data: marshal chart readings JSON failed", "error", err)
		chartData.ChartReadingsJSON = template.JS("[]")
	} else {
		chartData.ChartReadingsJSON = template.JS(jsonBytes)
		slog.Debug("history-chart-data: chart readings JSON prepared", "readings_count", len(chartReadings), "json_length", len(chartData.ChartReadingsJSON))
	}

	var buf bytes.Buffer
	if err := views.RenderHistoryChartDataPartial(&buf, &chartData); err != nil {
		slog.Error("history-chart-data partial render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("history-chart-data: write response failed", "error", err)
	}
}
