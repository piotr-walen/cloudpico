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
	stations    []types.Station
	stationsErr error
	latest      []types.Reading
	latestErr   error
	readings    []types.Reading
	readingsErr error
}

func (m *mockRepo) GetStations() ([]types.Station, error) {
	return m.stations, m.stationsErr
}

func (m *mockRepo) GetLatestReadings(stationID string, limit int) ([]types.Reading, error) {
	return m.latest, m.latestErr
}

func (m *mockRepo) GetReadings(stationID string, from, to time.Time, limit int) ([]types.Reading, error) {
	return m.readings, m.readingsErr
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

func Test_handleCurrentConditionsPartial(t *testing.T) {
	if err := views.LoadTemplates(); err != nil {
		t.Skipf("LoadTemplates failed: %v", err)
	}

	t.Run("returns 200 and HTML partial with reading when station has latest", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		latest := []types.Reading{
			{StationID: "st-1", Time: time.Date(2025, 2, 3, 12, 0, 0, 0, time.UTC), Value: 18.5},
		}
		ctrl := NewWeatherController(&mockRepo{stations: stations, latest: latest}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/current-conditions", nil)
		rec := httptest.NewRecorder()

		ctrl.handleCurrentConditionsPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q; want text/html; charset=utf-8", ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Current conditions") {
			t.Errorf("body missing \"Current conditions\"; got %q", body)
		}
		if !strings.Contains(body, "Station One") {
			t.Errorf("body missing station name; got %q", body)
		}
		if !strings.Contains(body, "18.5") {
			t.Errorf("body missing value; got %q", body)
		}
	})

	t.Run("returns 200 and no recent reading when no latest", func(t *testing.T) {
		stations := []types.Station{{ID: "st-1", Name: "Station One"}}
		ctrl := NewWeatherController(&mockRepo{stations: stations, latest: nil}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/current-conditions", nil)
		rec := httptest.NewRecorder()

		ctrl.handleCurrentConditionsPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "No recent reading") {
			t.Errorf("body missing \"No recent reading\"; got %q", body)
		}
	})

	t.Run("uses first station when no station_id query", func(t *testing.T) {
		stations := []types.Station{{ID: "first", Name: "First Station"}, {ID: "second", Name: "Second"}}
		latest := []types.Reading{{StationID: "first", Time: time.Now(), Value: 20}}
		ctrl := NewWeatherController(&mockRepo{stations: stations, latest: latest}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/current-conditions", nil)
		rec := httptest.NewRecorder()

		ctrl.handleCurrentConditionsPartial(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "First Station") {
			t.Errorf("body should use first station; got %q", body)
		}
	})

	t.Run("returns 500 when GetStations fails", func(t *testing.T) {
		ctrl := NewWeatherController(&mockRepo{stationsErr: errors.New("db error")}).(*weatherControllerImpl)
		req := httptest.NewRequest(http.MethodGet, "/partials/current-conditions", nil)
		rec := httptest.NewRecorder()

		ctrl.handleCurrentConditionsPartial(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d; want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
