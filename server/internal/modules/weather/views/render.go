package views

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"time"
)

var dashboardTmpl *template.Template

// loadTemplatesFromFS loads dashboard templates from the given fs and dir.
// Used by LoadTemplates and by tests to simulate failure scenarios.
func loadTemplatesFromFS(fsys fs.FS, dir string) error {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		return err
	}
	dashboardTmpl, err = template.ParseFS(sub, "*.html", "partials/*.html")
	if err != nil {
		return err
	}
	return nil
}

// LoadTemplates loads embedded dashboard templates. Call during startup before
// serving requests; if it returns an error, do not start the server.
func LoadTemplates() error {
	return loadTemplatesFromFS(viewsFS, "templates")
}

// StationOption is the view model for a station in the dashboard selector.
type StationOption struct {
	ID   string
	Name string
}

// DashboardData is the view model for the dashboard page.
type DashboardData struct {
	Stations          []StationOption
	SelectedStationID string
}

// RenderDashboard executes the dashboard page (base layout + dashboard content) into w.
func RenderDashboard(w io.Writer, data any) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}
	return dashboardTmpl.ExecuteTemplate(w, "base.html", data)
}

// CurrentConditionsData is the view model for the current-conditions partial.
type CurrentConditionsData struct {
	StationName string
	Reading     *ReadingPartial // nil when no recent reading
}

// ReadingPartial exposes reading fields for the template (avoids importing types in views).
type ReadingPartial struct {
	Value       float64
	Time        time.Time
	HumidityPct float64
	PressureHpa float64
}

// PaginationItem is one entry in the pagination bar: either a page number or an ellipsis.
type PaginationItem struct {
	Page     int  // page number (1-based); only valid when Ellipsis is false
	Ellipsis bool // when true, render "..." instead of a link
}

// HistoryData is the view model for the history partial.
type HistoryData struct {
	StationName string
	StationID   string // for pagination links
	RangeLabel  string
	RangeKey    string // for pagination links, e.g. "24h"
	Readings    []ReadingPartial
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
	PageItems   []PaginationItem // page numbers and ellipsis for the pagination bar
}

// RenderCurrentConditionsPartial executes only the current-conditions partial into w.
// Use for HTMX fragment refresh.
func RenderCurrentConditionsPartial(w io.Writer, data CurrentConditionsData) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}

	return dashboardTmpl.ExecuteTemplate(w, "partials/current-conditions.html", data)
}

// RenderHistoryPartial executes only the history partial into w.
// Use for HTMX fragment refresh.
func RenderHistoryPartial(w io.Writer, data HistoryData) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}

	return dashboardTmpl.ExecuteTemplate(w, "partials/history.html", data)
}
