package views

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/fstest"
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

	data := struct {
		Title string
	}{Title: "Custom Title"}

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

type failingWriter struct{ err error }

func (f *failingWriter) Write([]byte) (int, error) { return 0, f.err }
