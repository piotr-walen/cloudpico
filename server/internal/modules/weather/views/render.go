package views

import (
	"html/template"
	"io"
	"io/fs"
	"log/slog"
)

var dashboardTmpl *template.Template

func init() {
	sub, err := fs.Sub(viewsFS, "templates")
	if err != nil {
		slog.Error("views fs sub", "error", err)
		return
	}
	dashboardTmpl, err = template.ParseFS(sub, "*.html", "partials/*.html")
	if err != nil {
		slog.Error("parse dashboard templates", "error", err)
		return
	}
}

// RenderDashboard executes the dashboard page (base layout + dashboard content) into w.
func RenderDashboard(w io.Writer, data any) error {
	if dashboardTmpl == nil {
		panic("dashboard template not loaded")
	}
	return dashboardTmpl.ExecuteTemplate(w, "base.html", data)
}
