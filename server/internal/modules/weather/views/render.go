package views

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
)

var dashboardTmpl *template.Template

// LoadTemplates loads embedded dashboard templates. Call during startup before
// serving requests; if it returns an error, do not start the server.
func LoadTemplates() error {
	sub, err := fs.Sub(viewsFS, "templates")
	if err != nil {
		return err
	}
	dashboardTmpl, err = template.ParseFS(sub, "*.html", "partials/*.html")
	if err != nil {
		return err
	}
	return nil
}

// RenderDashboard executes the dashboard page (base layout + dashboard content) into w.
func RenderDashboard(w io.Writer, data any) error {
	if dashboardTmpl == nil {
		return errors.New("dashboard template not loaded: call views.LoadTemplates during startup")
	}
	return dashboardTmpl.ExecuteTemplate(w, "base.html", data)
}
