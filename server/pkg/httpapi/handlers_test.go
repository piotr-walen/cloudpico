package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := NewServer(":0")
	ts := httptest.NewServer(srv.Handler)

	t.Cleanup(ts.Close)
	return ts
}

func mustGetJSON[T any](t *testing.T, client *http.Client, url string, out *T) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	return resp
}

func mustGetRaw(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestHealthz(t *testing.T) {
	ts := newTestServer(t)

	var body map[string]string
	resp := mustGetJSON(t, ts.Client(), ts.URL+"/healthz", &body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}
	if body["status"] != "ok" {
		t.Fatalf("body.status=%q want=%q", body["status"], "ok")
	}
}

func TestStations(t *testing.T) {
	ts := newTestServer(t)

	var stations []Station
	resp := mustGetJSON(t, ts.Client(), ts.URL+"/api/v1/stations", &stations)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}
	if len(stations) < 1 {
		t.Fatalf("expected at least 1 station, got %d", len(stations))
	}
	if stations[0].ID == "" || stations[0].Name == "" {
		t.Fatalf("expected station fields to be non-empty, got %+v", stations[0])
	}
}

func TestLatest(t *testing.T) {
	ts := newTestServer(t)

	var r Reading
	resp := mustGetJSON(t, ts.Client(), ts.URL+"/api/v1/stations/st-001/latest", &r)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}
	if r.StationID != "st-001" {
		t.Fatalf("stationId=%q want=%q", r.StationID, "st-001")
	}
	// Ensure time parses and looks sane (non-zero)
	if r.Time.IsZero() {
		t.Fatalf("expected non-zero time")
	}
}

func TestReadings_Defaults(t *testing.T) {
	ts := newTestServer(t)

	var body map[string]any
	resp := mustGetJSON(t, ts.Client(), ts.URL+"/api/v1/stations/st-001/readings", &body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	if body["stationId"] != "st-001" {
		t.Fatalf("stationId=%v want=%v", body["stationId"], "st-001")
	}

	// default: from/to nil, limit 100
	if body["from"] != nil {
		t.Fatalf("from=%v want=nil", body["from"])
	}
	if body["to"] != nil {
		t.Fatalf("to=%v want=nil", body["to"])
	}

	limitF, ok := body["limit"].(float64) // JSON numbers decode to float64
	if !ok {
		t.Fatalf("limit type=%T want float64", body["limit"])
	}
	if int(limitF) != 100 {
		t.Fatalf("limit=%d want=%d", int(limitF), 100)
	}

	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items type=%T want []any", body["items"])
	}
	if len(items) == 0 {
		t.Fatalf("expected items not empty")
	}
}

func TestReadings_WithWindowAndLimit(t *testing.T) {
	ts := newTestServer(t)

	from := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Second)
	to := time.Now().UTC().Add(-5 * time.Minute).Truncate(time.Second)
	limit := 10

	url := ts.URL + "/api/v1/stations/st-002/readings?from=" + from.Format(time.RFC3339) +
		"&to=" + to.Format(time.RFC3339) +
		"&limit=" + strconv.Itoa(limit)

	var body map[string]any
	resp := mustGetJSON(t, ts.Client(), url, &body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items type=%T want []any", body["items"])
	}
	if len(items) > limit {
		t.Fatalf("items len=%d want <= %d", len(items), limit)
	}
}

func TestReadings_InvalidQueryParams(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid from",
			url:  ts.URL + "/api/v1/stations/st-001/readings?from=not-a-time",
		},
		{
			name: "invalid to",
			url:  ts.URL + "/api/v1/stations/st-001/readings?to=not-a-time",
		},
		{
			name: "from after to",
			url: ts.URL + "/api/v1/stations/st-001/readings?from=2026-02-02T10:00:00Z" +
				"&to=2026-02-02T09:00:00Z",
		},
		{
			name: "invalid limit",
			url:  ts.URL + "/api/v1/stations/st-001/readings?limit=abc",
		},
		{
			name: "limit <= 0",
			url:  ts.URL + "/api/v1/stations/st-001/readings?limit=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]any
			resp := mustGetJSON(t, ts.Client(), tt.url, &body)

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusBadRequest)
			}

			// We return {error, message}
			if _, ok := body["error"]; !ok {
				t.Fatalf("expected error field, got %v", body)
			}
			if _, ok := body["message"]; !ok {
				t.Fatalf("expected message field, got %v", body)
			}
		})
	}
}

func TestRouting_UnknownRoute(t *testing.T) {
	ts := newTestServer(t)

	resp := mustGetRaw(t, ts.Client(), ts.URL+"/does-not-exist")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRouting_WrongMethod(t *testing.T) {
	ts := newTestServer(t)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
