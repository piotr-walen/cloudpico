package views

import (
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

// RenderDashboard executes the dashboard page (base layout + dashboard content) into w.
func RenderDashboard(w io.Writer, data any) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}
	return dashboardTmpl.ExecuteTemplate(w, "base.html", data)
}
