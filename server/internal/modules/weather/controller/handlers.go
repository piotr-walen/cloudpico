package controller

import (
	"log/slog"
	"net/http"

	"cloudpico-server/internal/modules/weather/views"
	"cloudpico-server/internal/utils"
)

func (c *weatherControllerImpl) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderDashboard(w, nil); err != nil {
		slog.Error("dashboard template render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render page")
		return
	}
}

func (c *weatherControllerImpl) handleStations(w http.ResponseWriter, r *http.Request) {
	stations, err := c.repository.GetStations()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, stations)
}

func (c *weatherControllerImpl) handleLatest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	limit, err := parseLatestQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	latest, err := c.repository.GetLatestReadings(id, limit)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJSON(w, http.StatusOK, latest)
}

func (c *weatherControllerImpl) handleReadings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing station id")
		return
	}

	from, to, limit, err := parseReadingsQuery(r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	readings, err := c.repository.GetReadings(id, from, to, limit)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, readings)
}

func (c *weatherControllerImpl) handleCurrentConditionsPartial(w http.ResponseWriter, r *http.Request) {
	stationID := r.URL.Query().Get("station_id")
	var stationName string
	if stationID == "" {
		stations, err := c.repository.GetStations()
		if err != nil {
			slog.Error("current conditions: get stations failed", "error", err)
			utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
			return
		}
		if len(stations) == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := views.RenderCurrentConditionsPartial(w, views.CurrentConditionsData{StationName: "", Reading: nil}); err != nil {
				slog.Error("current conditions partial render failed", "error", err)
				utils.WriteError(w, http.StatusInternalServerError, "failed to render")
			}
			return
		}
		stationID = stations[0].ID
		stationName = stations[0].Name
	} else {
		stations, err := c.repository.GetStations()
		if err != nil {
			slog.Error("current conditions: get stations failed", "error", err)
			utils.WriteError(w, http.StatusInternalServerError, "failed to load stations")
			return
		}
		for _, s := range stations {
			if s.ID == stationID {
				stationName = s.Name
				break
			}
		}
	}

	latest, err := c.repository.GetLatestReadings(stationID, 1)
	if err != nil {
		slog.Error("current conditions: get latest failed", "station_id", stationID, "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to load reading")
		return
	}

	var reading *views.ReadingPartial
	if len(latest) > 0 {
		reading = &views.ReadingPartial{Value: latest[0].Value, Time: latest[0].Time}
	}

	data := views.CurrentConditionsData{StationName: stationName, Reading: reading}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.RenderCurrentConditionsPartial(w, data); err != nil {
		slog.Error("current conditions partial render failed", "error", err)
		utils.WriteError(w, http.StatusInternalServerError, "failed to render")
		return
	}
}
