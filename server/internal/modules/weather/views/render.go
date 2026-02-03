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
	Value float64
	Time  time.Time
}

// RenderCurrentConditionsPartial executes only the current-conditions partial into w.
// Use for HTMX fragment refresh.
func RenderCurrentConditionsPartial(w io.Writer, data CurrentConditionsData) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}

	return dashboardTmpl.ExecuteTemplate(w, "partials/current-conditions.html", data)
}
