package views

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"cloudpico-server/internal/modules/weather/types"
)

func TestLoadTemplates_success(t *testing.T) {
	err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates() = %v; want nil", err)
	}
	if dashboardTmpl == nil {
		t.Fatal("LoadTemplates() left dashboardTmpl nil")
	}
}

func TestLoadTemplates_failure_sub(t *testing.T) {
	// Empty FS has no "templates" directory; fs.Sub fails.
	emptyFS := fstest.MapFS{}
	err := loadTemplatesFromFS(emptyFS, "templates")
	if err == nil {
		t.Fatal("loadTemplatesFromFS(emptyFS, \"templates\") = nil; want error")
	}
}

func TestLoadTemplates_failure_parse(t *testing.T) {
	// FS with invalid template syntax; ParseFS fails.
	badFS := fstest.MapFS{
		"templates/base.html": {Data: []byte("{{ .")},
	}
	err := loadTemplatesFromFS(badFS, "templates")
	if err == nil {
		t.Fatal("loadTemplatesFromFS(badFS, \"templates\") = nil; want error")
	}
}

func TestRenderDashboard_notLoaded(t *testing.T) {
	// Ensure templates are not loaded for this test.
	prev := dashboardTmpl
	dashboardTmpl = nil
	t.Cleanup(func() { dashboardTmpl = prev })

	var buf bytes.Buffer
	err := RenderDashboard(&buf, (*DashboardData)(nil))
	if err == nil {
		t.Fatal("RenderDashboard() = nil; want error when templates not loaded")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("err = %q; want message containing \"not loaded\"", err.Error())
	}
}

func TestRenderDashboard_emptyData(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	var buf bytes.Buffer
	err := RenderDashboard(&buf, &DashboardData{})
	if err != nil {
		t.Fatalf("RenderDashboard(empty data) = %v; want nil", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("RenderDashboard(empty data) produced empty output")
	}
	// Base layout and dashboard content should still render.
	if !strings.Contains(out, "Cloudpico") {
		t.Errorf("output missing \"Cloudpico\"; got %q", out)
	}
	if !strings.Contains(out, "Dashboard") {
		t.Errorf("output missing \"Dashboard\"; got %q", out)
	}
}

func TestRenderDashboard_withData(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	data := &DashboardData{
		Stations: []StationReading{
			{StationName: "Station One", Reading: &types.Reading{Value: 22.5, Time: time.Date(2025, 2, 3, 14, 30, 0, 0, time.UTC)}},
		},
	}

	var buf bytes.Buffer
	err := RenderDashboard(&buf, data)
	if err != nil {
		t.Fatalf("RenderDashboard(data) = %v; want nil", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("RenderDashboard(data) produced empty output")
	}
	if !strings.Contains(out, "Dashboard") {
		t.Errorf("output missing \"Dashboard\"; got %q", out)
	}
	if !strings.Contains(out, "Cloudpico") {
		t.Errorf("output missing \"Cloudpico\"; got %q", out)
	}
	if !strings.Contains(out, "Station One") {
		t.Errorf("output missing station name; got %q", out)
	}
	if !strings.Contains(out, "Current conditions") {
		t.Errorf("output missing current conditions; got %q", out)
	}
	// Ensure we get HTML layout (base defines structure).
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("output missing DOCTYPE; got %q", out)
	}
	if !strings.Contains(out, "<main") {
		t.Errorf("output missing <main>; got %q", out)
	}
}

func TestRenderHistory_notLoaded(t *testing.T) {
	prev := dashboardTmpl
	dashboardTmpl = nil
	t.Cleanup(func() { dashboardTmpl = prev })

	var buf bytes.Buffer
	err := RenderHistory(&buf, &HistoryParams{})
	if err == nil {
		t.Fatal("RenderHistory() = nil; want error when templates not loaded")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("err = %q; want message containing \"not loaded\"", err.Error())
	}
}

func TestRenderHistory_emptyData(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	var buf bytes.Buffer
	err := RenderHistory(&buf, &HistoryParams{})
	if err != nil {
		t.Fatalf("RenderHistory(empty data) = %v; want nil", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("RenderHistory(empty data) produced empty output")
	}
	if !strings.Contains(out, "Cloudpico") {
		t.Errorf("output missing \"Cloudpico\"; got %q", out)
	}
	if !strings.Contains(out, "History") {
		t.Errorf("output missing \"History\"; got %q", out)
	}
}

func TestRenderHistory_withData(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	data := &HistoryParams{
		Stations:          []StationOption{{ID: "s1", Name: "Station One"}},
		SelectedStationID: "s1",
		SelectedRangeKey:  "24h",
	}

	var buf bytes.Buffer
	err := RenderHistory(&buf, data)
	if err != nil {
		t.Fatalf("RenderHistory(data) = %v; want nil", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("RenderHistory(data) produced empty output")
	}
	if !strings.Contains(out, "History") {
		t.Errorf("output missing \"History\"; got %q", out)
	}
	if !strings.Contains(out, "Cloudpico") {
		t.Errorf("output missing \"Cloudpico\"; got %q", out)
	}
	if !strings.Contains(out, "Station One") {
		t.Errorf("output missing station name; got %q", out)
	}
	if !strings.Contains(out, "station-selector") {
		t.Errorf("output missing station selector; got %q", out)
	}
	if !strings.Contains(out, "history-range") {
		t.Errorf("output missing history range selector; got %q", out)
	}
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("output missing DOCTYPE; got %q", out)
	}
}

// Ensure RenderHistory propagates write errors (e.g. closed writer).
func TestRenderHistory_writeError(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	w := &failingWriter{err: io.ErrClosedPipe}
	err := RenderHistory(w, &HistoryParams{})
	if err == nil {
		t.Fatal("RenderHistory(failingWriter) = nil; want error")
	}
	if err != io.ErrClosedPipe {
		t.Errorf("RenderHistory() = %v; want %v", err, io.ErrClosedPipe)
	}
}

// Ensure RenderDashboard propagates write errors (e.g. closed writer).
func TestRenderDashboard_writeError(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	w := &failingWriter{err: io.ErrClosedPipe}
	err := RenderDashboard(w, &DashboardData{})
	if err == nil {
		t.Fatal("RenderDashboard(failingWriter) = nil; want error")
	}
	if err != io.ErrClosedPipe {
		t.Errorf("RenderDashboard() = %v; want %v", err, io.ErrClosedPipe)
	}
}

type failingWriter struct{ err error }

func (f *failingWriter) Write([]byte) (int, error) { return 0, f.err }
