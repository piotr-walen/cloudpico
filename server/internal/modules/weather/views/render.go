package views

import (
	"cloudpico-server/internal/modules/weather/types"
	"errors"
	"html/template"
	"io"
	"io/fs"
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

func RenderDashboard(w io.Writer, data *DashboardData) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}
	return dashboardTmpl.ExecuteTemplate(w, "dashboard.html", data)
}

type HistoryParams struct {
	Stations          []StationOption
	SelectedStationID string
	SelectedRangeKey  string
}

func RenderHistory(w io.Writer, data *HistoryParams) error {
	if dashboardTmpl == nil {
		return errors.New("history template not loaded: call views.LoadTemplates during startup")
	}
	return dashboardTmpl.ExecuteTemplate(w, "history.html", data)
}

type StationReading struct {
	StationID   string
	StationName string
	Reading     *types.Reading
}
type DashboardData struct {
	Stations []StationReading
}

// PaginationItem is one entry in the pagination bar: either a page number or an ellipsis.
type PaginationItem struct {
	Page     int
	Ellipsis bool
}

// HistoryData is the view model for the history partial.
type HistoryData struct {
	StationName string
	StationID   string // for pagination links
	RangeLabel  string
	RangeKey    string // for pagination links, e.g. "24h"
	Readings    []types.Reading
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
	PageItems   []PaginationItem // page numbers and ellipsis for the pagination bar
}

// RenderHistoryPartial executes only the history partial into w.
// Use for HTMX fragment refresh.
func RenderHistoryPartial(w io.Writer, data *HistoryData) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}

	return dashboardTmpl.ExecuteTemplate(w, "partials/history.html", data)
}

// RenderStationsPartial executes only the stations partial into w.
// Use for HTMX fragment refresh (e.g. dashboard auto-refresh).
func RenderStationsPartial(w io.Writer, data *DashboardData) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}
	return dashboardTmpl.ExecuteTemplate(w, "partials/stations.html", data)
}
