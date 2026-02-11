package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloudpico-server/internal/modules/weather/types"
	"cloudpico-server/internal/modules/weather/views"
)

type mockRepo struct {
	stations              []types.Station
	stationsErr           error
	latest                []types.Reading
	latestErr             error
	readings              []types.Reading
	readingsErr           error
	readingsCount         int // returned by GetReadingsCount; 0 means no count set
	countErr              error
	lastReadingsStationID string
	lastReadingsFrom      time.Time
	lastReadingsTo        time.Time
	lastReadingsLimit     int
	lastReadingsOffset    int
	insertErr             error
}

func (m *mockRepo) GetStations() ([]types.Station, error) {
	return m.stations, m.stationsErr
}

func (m *mockRepo) GetLatestReadings(stationID string, limit int) ([]types.Reading, error) {
	return m.latest, m.latestErr
}

func (m *mockRepo) GetReadings(stationID string, from, to time.Time, limit int, offset int) ([]types.Reading, error) {
	m.lastReadingsStationID = stationID
	m.lastReadingsFrom = from
	m.lastReadingsTo = to
	m.lastReadingsLimit = limit
	m.lastReadingsOffset = offset
	return m.readings, m.readingsErr
}

func (m *mockRepo) GetReadingsCount(stationID string, from, to time.Time) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	if m.readingsCount != 0 {
		return m.readingsCount, nil
	}
	return len(m.readings), nil
}

func (m *mockRepo) InsertReading(stationID string, ts time.Time, temperature *float64, humidity *float64, pressure *float64) error {
	return m.insertErr
}

func Test_handleDashboard(t *testing.T) {
	ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)

	t.Run("returns 404 when path is not /", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		req.URL.Path = "/dashboard"
		rec := httptest.NewRecorder()

		ctrl.handleDashboard(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("returns 404 when path is not exactly /", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.URL.Path = "//"
		rec := httptest.NewRecorder()

		ctrl.handleDashboard(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d; want %d for path %q", rec.Code, http.StatusNotFound, req.URL.Path)
		}
	})

	t.Run("returns 500 and error body when GetStations fails", func(t *testing.T) {
		ctrlErr := NewWeatherController(&mockRepo{stationsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		ctrlErr.handleDashboard(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "failed to load stations") {
			t.Errorf("body = %q; expected 'failed to load stations'", body)
		}
		if !strings.Contains(body, "error") {
			t.Errorf("body = %q; expected error JSON", body)
		}
	})

	t.Run("returns 500 and error body when render fails", func(t *testing.T) {
		// Render fails when templates are not loaded (dashboardTmpl is nil)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		ctrl.handleDashboard(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "failed to render page") {
			t.Errorf("body = %q; expected 'failed to render page'", body)
		}
		if !strings.Contains(body, "error") {
			t.Errorf("body = %q; expected error JSON", body)
		}
	})

	t.Run("sets Content-Type and returns 200 with HTML when path is / and templates loaded", func(t *testing.T) {
		if err := views.LoadTemplates(); err != nil {
			t.Skipf("LoadTemplates failed (embed not available?): %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		ctrl.handleDashboard(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q; want text/html; charset=utf-8", ct)
		}
		body := rec.Body.String()
		if body == "" || !strings.Contains(body, "<!") || !strings.Contains(body, "html") {
			t.Errorf("body should be HTML; got %q", body)
		}
	})

	stations := []types.Station{
		{ID: "st-1", Name: "Station One"},
		{ID: "st-2", Name: "Station Two"},
	}

	t.Run("shows all stations when stations present", func(t *testing.T) {
		if err := views.LoadTemplates(); err != nil {
			t.Skipf("LoadTemplates failed (embed not available?): %v", err)
		}
		ctrlWithStations := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		ctrlWithStations.handleDashboard(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Dashboard shows a current-conditions card per station (no selector).
		if !strings.Contains(body, "Station One") {
			t.Errorf("body should include first station name; got %q", body)
		}
		if !strings.Contains(body, "Station Two") {
			t.Errorf("body should include second station name; got %q", body)
		}
		if !strings.Contains(body, "Current conditions") {
			t.Errorf("body should include current conditions section; got %q", body)
		}
	})
}

func Test_handleStations(t *testing.T) {
	t.Run("returns stations on success", func(t *testing.T) {
		stations := []types.Station{
			{ID: "st-1", Name: "Station One"},
			{ID: "st-2", Name: "Station Two"},
		}
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations", nil)
		rec := httptest.NewRecorder()

		ctrl.handleStations(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q; want application/json", ct)
		}
		body := strings.TrimSpace(rec.Body.String())
		if body == "" || !strings.Contains(body, "st-1") || !strings.Contains(body, "Station One") {
			t.Errorf("body = %q; expected JSON with stations", body)
		}
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stationsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations", nil)
		rec := httptest.NewRecorder()

		ctrl.handleStations(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "error") || !strings.Contains(body, "db error") {
			t.Errorf("body = %q; expected error JSON", body)
		}
	})
}

func Test_handleLatest(t *testing.T) {
	t.Run("returns latest readings on success", func(t *testing.T) {
		readings := []types.Reading{
			{StationID: "st-1", Time: time.Now(), Value: 12.5},
		}
		ctrl := NewWeatherController(&mockRepo{latest: readings}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/latest", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleLatest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "st-1") || !strings.Contains(body, "12.5") {
			t.Errorf("body = %q; expected readings JSON", body)
		}
	})

	t.Run("returns 400 when station id is missing", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations//latest", nil)
		req.SetPathValue("id", "")
		rec := httptest.NewRecorder()

		ctrl.handleLatest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "missing station id") {
			t.Errorf("body = %q; expected missing station id", body)
		}
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{latestErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/latest", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleLatest(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("returns 400 when limit is invalid", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/latest?limit=abc", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleLatest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func Test_handleReadings(t *testing.T) {
	t.Run("returns readings on success", func(t *testing.T) {
		readings := []types.Reading{
			{StationID: "st-1", Time: time.Now(), Value: 10.0},
		}
		ctrl := NewWeatherController(&mockRepo{readings: readings}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/readings?from=2025-01-01T00:00:00Z&to=2025-01-02T00:00:00Z&limit=10", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "st-1") {
			t.Errorf("body = %q; expected readings JSON", body)
		}
	})

	t.Run("returns 400 when station id is missing", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations//readings", nil)
		req.SetPathValue("id", "")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rec.Body.String(), "missing station id") {
			t.Errorf("expected missing station id in body")
		}
	})

	t.Run("returns 400 when from is invalid", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/readings?from=not-a-date", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "from") && !strings.Contains(body, "RFC3339") {
			t.Errorf("body = %q; expected invalid from error", body)
		}
	})

	t.Run("returns 400 when to is invalid", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/readings?to=not-a-date", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 400 when from is after to", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/readings?from=2025-01-02T00:00:00Z&to=2025-01-01T00:00:00Z", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "from") || !strings.Contains(body, "to") {
			t.Errorf("body = %q; expected from <= to error", body)
		}
	})

	t.Run("returns 400 when limit is invalid", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/readings?limit=abc", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 500 when repository fails", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{readingsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stations/st-1/readings", nil)
		req.SetPathValue("id", "st-1")
		rec := httptest.NewRecorder()

		ctrl.handleReadings(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

// paginationItemsEqual compares two PaginationItem slices for equality.
func paginationItemsEqual(a, b []views.PaginationItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Ellipsis != b[i].Ellipsis || a[i].Page != b[i].Page {
			return false
		}
	}
	return true
}

func Test_buildHistoryPageItems(t *testing.T) {
	t.Run("returns nil when totalPages <= 0", func(t *testing.T) {
		got := buildHistoryPageItems(0, 1)
		if got != nil {
			t.Errorf("buildHistoryPageItems(0, 1) = %v; want nil", got)
		}
		got = buildHistoryPageItems(-1, 1)
		if got != nil {
			t.Errorf("buildHistoryPageItems(-1, 1) = %v; want nil", got)
		}
	})

	t.Run("single page returns one item", func(t *testing.T) {
		want := []views.PaginationItem{{Page: 1, Ellipsis: false}}
		got := buildHistoryPageItems(1, 1)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(1, 1) = %v; want %v", got, want)
		}
	})

	t.Run("two pages with currentPage 1 returns both pages no ellipsis", func(t *testing.T) {
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Page: 2, Ellipsis: false},
		}
		got := buildHistoryPageItems(2, 1)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(2, 1) = %v; want %v", got, want)
		}
	})

	t.Run("currentPage at first page with many pages", func(t *testing.T) {
		// totalPages=5, current=1 → window 1±2 gives 1,2,3; plus last 5 → [1,2,3,...,5]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Page: 2, Ellipsis: false},
			{Page: 3, Ellipsis: false},
			{Ellipsis: true},
			{Page: 5, Ellipsis: false},
		}
		got := buildHistoryPageItems(5, 1)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(5, 1) = %v; want %v", got, want)
		}
	})

	t.Run("currentPage at last page with many pages", func(t *testing.T) {
		// totalPages=5, current=5 → window 3,4,5; plus first 1 → [1,...,3,4,5]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Ellipsis: true},
			{Page: 3, Ellipsis: false},
			{Page: 4, Ellipsis: false},
			{Page: 5, Ellipsis: false},
		}
		got := buildHistoryPageItems(5, 5)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(5, 5) = %v; want %v", got, want)
		}
	})

	t.Run("currentPage in middle with window and ellipsis both sides", func(t *testing.T) {
		// totalPages=10, current=5 → 1,10 and 3..7 → [1,...,3,4,5,6,7,...,10]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Ellipsis: true},
			{Page: 3, Ellipsis: false},
			{Page: 4, Ellipsis: false},
			{Page: 5, Ellipsis: false},
			{Page: 6, Ellipsis: false},
			{Page: 7, Ellipsis: false},
			{Ellipsis: true},
			{Page: 10, Ellipsis: false},
		}
		got := buildHistoryPageItems(10, 5)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(10, 5) = %v; want %v", got, want)
		}
	})

	t.Run("currentPage 2 with many pages", func(t *testing.T) {
		// totalPages=10, current=2 → 1,10 and 0,1,2,3,4 clamped → 1,2,3,4,10 → [1,2,3,4,...,10]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Page: 2, Ellipsis: false},
			{Page: 3, Ellipsis: false},
			{Page: 4, Ellipsis: false},
			{Ellipsis: true},
			{Page: 10, Ellipsis: false},
		}
		got := buildHistoryPageItems(10, 2)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(10, 2) = %v; want %v", got, want)
		}
	})

	t.Run("currentPage second to last with many pages", func(t *testing.T) {
		// totalPages=10, current=9 → 1,10 and 7,8,9,10 → [1,...,7,8,9,10]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Ellipsis: true},
			{Page: 7, Ellipsis: false},
			{Page: 8, Ellipsis: false},
			{Page: 9, Ellipsis: false},
			{Page: 10, Ellipsis: false},
		}
		got := buildHistoryPageItems(10, 9)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(10, 9) = %v; want %v", got, want)
		}
	})

	t.Run("all pages fit in window no ellipsis", func(t *testing.T) {
		// totalPages=5, current=3 → 1,2,3,4,5 all in window → [1,2,3,4,5]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Page: 2, Ellipsis: false},
			{Page: 3, Ellipsis: false},
			{Page: 4, Ellipsis: false},
			{Page: 5, Ellipsis: false},
		}
		got := buildHistoryPageItems(5, 3)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(5, 3) = %v; want %v", got, want)
		}
	})

	t.Run("currentPage out of range still produces valid items", func(t *testing.T) {
		// Function does not validate currentPage; currentPage=100 with totalPages=5
		// window 98..102 adds no pages in [1,5], so only first and last → [1,...,5]
		want := []views.PaginationItem{
			{Page: 1, Ellipsis: false},
			{Ellipsis: true},
			{Page: 5, Ellipsis: false},
		}
		got := buildHistoryPageItems(5, 100)
		if !paginationItemsEqual(got, want) {
			t.Errorf("buildHistoryPageItems(5, 100) = %v; want %v", got, want)
		}
	})
}

func Test_handleHistoryPartial(t *testing.T) {
	if err := views.LoadTemplates(); err != nil {
		t.Skipf("LoadTemplates failed: %v", err)
	}

	t.Run("returns 200 with readings and selected range", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		readings := []types.Reading{
			{StationID: "st-1", Time: time.Date(2025, 2, 3, 10, 0, 0, 0, time.UTC), Value: 12.5},
		}
		repo := &mockRepo{stations: stations, readings: readings}
		ctrl := NewWeatherController(repo).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history?station_id=st-1&range=1h", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q; want text/html; charset=utf-8", ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "History") && !strings.Contains(body, "Last 1 hour") {
			t.Errorf("body should include history label; got %q", body)
		}
		if !strings.Contains(body, "Station One") {
			t.Errorf("body missing station name; got %q", body)
		}
		if !strings.Contains(body, "12.5") {
			t.Errorf("body missing reading value; got %q", body)
		}
		if repo.lastReadingsStationID != "st-1" {
			t.Errorf("station id = %q; want st-1", repo.lastReadingsStationID)
		}
		wantLimit := historyPageSize
		if repo.lastReadingsLimit != wantLimit {
			t.Errorf("limit = %d; want %d", repo.lastReadingsLimit, wantLimit)
		}
		if repo.lastReadingsOffset != 0 {
			t.Errorf("offset = %d; want 0", repo.lastReadingsOffset)
		}
	})

	t.Run("defaults to first station and default range", func(t *testing.T) {
		stations := []types.Station{{ID: "first", Name: "First Station"}, {ID: "second", Name: "Second"}}
		repo := &mockRepo{stations: stations, readings: nil}
		ctrl := NewWeatherController(repo).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "First Station") {
			t.Errorf("body should use first station; got %q", body)
		}
		if !strings.Contains(body, "Last 24 hours") {
			t.Errorf("body should use default range label; got %q", body)
		}
		if !strings.Contains(body, "No readings in selected range") {
			t.Errorf("body should include empty state; got %q", body)
		}
	})

	t.Run("uses Unknown Station when station_id is invalid", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		repo := &mockRepo{stations: stations, readings: nil}
		ctrl := NewWeatherController(repo).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history?station_id=missing", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Unknown Station") {
			t.Errorf("body should show Unknown Station; got %q", body)
		}
	})

	t.Run("falls back to default range when range is invalid", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		repo := &mockRepo{stations: stations, readings: nil}
		ctrl := NewWeatherController(repo).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history?range=bad", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Last 24 hours") {
			t.Errorf("body should use default range label; got %q", body)
		}
	})

	t.Run("returns 500 when GetStations fails", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stationsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("returns 500 when GetReadingsCount fails", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		ctrl := NewWeatherController(&mockRepo{stations: stations, countErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("returns 500 when GetReadings fails", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		ctrl := NewWeatherController(&mockRepo{stations: stations, readingsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("passes page and offset to GetReadings for page 2", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		readings := make([]types.Reading, 12) // more than one page
		for i := range readings {
			readings[i] = types.Reading{StationID: "st-1", Time: time.Now().Add(-time.Duration(i) * time.Hour), Value: float64(i)}
		}
		repo := &mockRepo{stations: stations, readings: readings, readingsCount: 25} // totalPages=2
		ctrl := NewWeatherController(repo).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/history?station_id=st-1&range=24h&page=2", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistoryPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if repo.lastReadingsOffset != historyPageSize {
			t.Errorf("offset = %d; want %d", repo.lastReadingsOffset, historyPageSize)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "aria-current=\"page\">2</span>") {
			t.Errorf("body should show current page 2 in pagination; got %q", body)
		}
		if !strings.Contains(body, "Previous") {
			t.Errorf("body should show Previous link on page 2; got %q", body)
		}
		if !strings.Contains(body, "First") {
			t.Errorf("body should show First link on page 2; got %q", body)
		}
	})
}

func Test_handleHistory(t *testing.T) {
	if err := views.LoadTemplates(); err != nil {
		t.Skipf("LoadTemplates failed: %v", err)
	}

	stations := []types.Station{
		{ID: "st-1", Name: "Station One"},
		{ID: "st-2", Name: "Station Two"},
	}

	t.Run("defaults to first station and default range when no params or cookies", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q; want text/html; charset=utf-8", ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "History") {
			t.Errorf("body should include History heading; got %q", body)
		}
		if !strings.Contains(body, "station-selector") {
			t.Errorf("body should include station selector; got %q", body)
		}
		if !strings.Contains(body, "history-range") {
			t.Errorf("body should include range selector; got %q", body)
		}
		// First station should be selected
		if !strings.Contains(body, `value="st-1" selected`) {
			t.Errorf("body should have first station selected; got %q", body)
		}
		// Default range (24h) should be selected
		if !strings.Contains(body, `value="24h" selected`) {
			t.Errorf("body should have default range (24h) selected; got %q", body)
		}
		// Check that both stations are in the selector
		if !strings.Contains(body, "Station One") {
			t.Errorf("body should include first station name; got %q", body)
		}
		if !strings.Contains(body, "Station Two") {
			t.Errorf("body should include second station name; got %q", body)
		}
	})

	t.Run("honors station_id query param", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history?station_id=st-2", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Second station should be selected
		if !strings.Contains(body, `value="st-2" selected`) {
			t.Errorf("body should have second station selected; got %q", body)
		}
		// First station should not be selected
		if strings.Contains(body, `value="st-1" selected`) {
			t.Errorf("body should not have first station selected; got %q", body)
		}
	})

	t.Run("honors range query param", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history?range=7d", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// 7d range should be selected
		if !strings.Contains(body, `value="7d" selected`) {
			t.Errorf("body should have 7d range selected; got %q", body)
		}
		// Default range (24h) should not be selected
		if strings.Contains(body, `value="24h" selected`) {
			t.Errorf("body should not have default range selected; got %q", body)
		}
	})

	t.Run("honors both station_id and range query params", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history?station_id=st-2&range=1h", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Second station should be selected
		if !strings.Contains(body, `value="st-2" selected`) {
			t.Errorf("body should have second station selected; got %q", body)
		}
		// 1h range should be selected
		if !strings.Contains(body, `value="1h" selected`) {
			t.Errorf("body should have 1h range selected; got %q", body)
		}
	})

	t.Run("falls back to cookie state when query params not provided", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		// Set cookie with station_id=st-2 and range=6h
		cookie := &http.Cookie{
			Name:  "weather_state",
			Value: "station_id=st-2&range=6h&page=1",
		}
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Second station from cookie should be selected
		if !strings.Contains(body, `value="st-2" selected`) {
			t.Errorf("body should have second station from cookie selected; got %q", body)
		}
		// 6h range from cookie should be selected
		if !strings.Contains(body, `value="6h" selected`) {
			t.Errorf("body should have 6h range from cookie selected; got %q", body)
		}
	})

	t.Run("query params override cookie state", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history?station_id=st-1&range=7d", nil)
		// Set cookie with different values
		cookie := &http.Cookie{
			Name:  "weather_state",
			Value: "station_id=st-2&range=6h&page=1",
		}
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Query param station should be selected (not cookie)
		if !strings.Contains(body, `value="st-1" selected`) {
			t.Errorf("body should have station from query param selected; got %q", body)
		}
		// Query param range should be selected (not cookie)
		if !strings.Contains(body, `value="7d" selected`) {
			t.Errorf("body should have range from query param selected; got %q", body)
		}
	})

	t.Run("rendered HTML includes station selector with all stations", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Check for station selector element
		if !strings.Contains(body, `id="station-selector"`) {
			t.Errorf("body should include station-selector id; got %q", body)
		}
		if !strings.Contains(body, `name="station_id"`) {
			t.Errorf("body should include station_id name attribute; got %q", body)
		}
		// Check for both station options
		if !strings.Contains(body, `<option value="st-1"`) {
			t.Errorf("body should include first station option; got %q", body)
		}
		if !strings.Contains(body, `<option value="st-2"`) {
			t.Errorf("body should include second station option; got %q", body)
		}
	})

	t.Run("rendered HTML includes range selector with all options", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Check for range selector element
		if !strings.Contains(body, `id="history-range"`) {
			t.Errorf("body should include history-range id; got %q", body)
		}
		if !strings.Contains(body, `name="range"`) {
			t.Errorf("body should include range name attribute; got %q", body)
		}
		// Check for all range options
		if !strings.Contains(body, `<option value="1h"`) {
			t.Errorf("body should include 1h option; got %q", body)
		}
		if !strings.Contains(body, `<option value="6h"`) {
			t.Errorf("body should include 6h option; got %q", body)
		}
		if !strings.Contains(body, `<option value="24h"`) {
			t.Errorf("body should include 24h option; got %q", body)
		}
		if !strings.Contains(body, `<option value="7d"`) {
			t.Errorf("body should include 7d option; got %q", body)
		}
	})

	t.Run("returns 500 when GetStations fails", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stationsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "failed to load stations") {
			t.Errorf("body = %q; expected 'failed to load stations'", body)
		}
	})

	t.Run("renders HTML successfully when templates are loaded", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Verify HTML structure
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Errorf("body should include DOCTYPE; got %q", body)
		}
		if !strings.Contains(body, "<html") {
			t.Errorf("body should include html tag; got %q", body)
		}
	})

	t.Run("sets cookie with selected station and range", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: stations}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history?station_id=st-2&range=7d", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		// Check that cookie was set
		cookies := rec.Result().Cookies()
		var weatherCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "weather_state" {
				weatherCookie = c
				break
			}
		}
		if weatherCookie == nil {
			t.Fatal("weather_state cookie not set")
		}
		if !strings.Contains(weatherCookie.Value, "station_id=st-2") {
			t.Errorf("cookie should contain station_id=st-2; got %q", weatherCookie.Value)
		}
		if !strings.Contains(weatherCookie.Value, "range=7d") {
			t.Errorf("cookie should contain range=7d; got %q", weatherCookie.Value)
		}
	})

	t.Run("handles empty stations list gracefully", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stations: []types.Station{}}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		rec := httptest.NewRecorder()

		ctrl.handleHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		// Should still render the page with empty station selector
		if !strings.Contains(body, "History") {
			t.Errorf("body should include History heading; got %q", body)
		}
		if !strings.Contains(body, "station-selector") {
			t.Errorf("body should include station selector; got %q", body)
		}
	})
}
