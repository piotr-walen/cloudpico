package views

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/fstest"
	"time"
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
	err := RenderDashboard(&buf, nil)
	if err == nil {
		t.Fatal("RenderDashboard() = nil; want error when templates not loaded")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("err = %q; want message containing \"not loaded\"", err.Error())
	}
}

func TestRenderDashboard_nilData(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	var buf bytes.Buffer
	err := RenderDashboard(&buf, nil)
	if err != nil {
		t.Fatalf("RenderDashboard(nil data) = %v; want nil", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("RenderDashboard(nil data) produced empty output")
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

	data := DashboardData{
		Stations:          []StationOption{{ID: "s1", Name: "Station One"}},
		SelectedStationID: "s1",
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
	if !strings.Contains(out, "station-selector") {
		t.Errorf("output missing station selector; got %q", out)
	}
	// Ensure we get HTML layout (base defines structure).
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("output missing DOCTYPE; got %q", out)
	}
	if !strings.Contains(out, "<main") {
		t.Errorf("output missing <main>; got %q", out)
	}
}

// Ensure RenderDashboard propagates write errors (e.g. closed writer).
func TestRenderDashboard_writeError(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}

	w := &failingWriter{err: io.ErrClosedPipe}
	err := RenderDashboard(w, nil)
	if err == nil {
		t.Fatal("RenderDashboard(failingWriter) = nil; want error")
	}
	if err != io.ErrClosedPipe {
		t.Errorf("RenderDashboard() = %v; want %v", err, io.ErrClosedPipe)
	}
}

func TestRenderCurrentConditionsPartial_notLoaded(t *testing.T) {
	prev := dashboardTmpl
	dashboardTmpl = nil
	t.Cleanup(func() { dashboardTmpl = prev })

	var buf bytes.Buffer
	err := RenderCurrentConditionsPartial(&buf, CurrentConditionsData{})
	if err == nil {
		t.Fatal("RenderCurrentConditionsPartial() = nil; want error when templates not loaded")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("err = %q; want message containing \"not loaded\"", err.Error())
	}
}

func TestRenderCurrentConditionsPartial_withReading(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}
	ts := time.Date(2025, 2, 3, 14, 30, 0, 0, time.UTC)
	data := CurrentConditionsData{
		StationName: "Home Station",
		Reading:     &ReadingPartial{Value: 22.5, Time: ts},
	}
	var buf bytes.Buffer
	err := RenderCurrentConditionsPartial(&buf, data)
	if err != nil {
		t.Fatalf("RenderCurrentConditionsPartial() = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Current conditions") {
		t.Errorf("output missing \"Current conditions\"; got %q", out)
	}
	if !strings.Contains(out, "Home Station") {
		t.Errorf("output missing station name; got %q", out)
	}
	if !strings.Contains(out, "22.5") {
		t.Errorf("output missing value; got %q", out)
	}
}

func TestRenderCurrentConditionsPartial_noReading(t *testing.T) {
	if err := LoadTemplates(); err != nil {
		t.Fatalf("LoadTemplates(): %v", err)
	}
	data := CurrentConditionsData{StationName: "Home", Reading: nil}
	var buf bytes.Buffer
	err := RenderCurrentConditionsPartial(&buf, data)
	if err != nil {
		t.Fatalf("RenderCurrentConditionsPartial() = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No recent reading") {
		t.Errorf("output missing \"No recent reading\"; got %q", out)
	}
}

type failingWriter struct{ err error }

func (f *failingWriter) Write([]byte) (int, error) { return 0, f.err }
