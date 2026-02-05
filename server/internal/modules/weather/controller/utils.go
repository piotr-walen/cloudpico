package controller

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	weatherStateCookieName   = "weather_state"
	weatherStateCookieMaxAge = 365 * 24 * 60 * 60 // 1 year in seconds
)

const (
	defaultHistoryRangeKey = "24h"
	historyPageSize        = 20
)

type historyRange struct {
	Duration time.Duration
	Label    string
}

var historyRanges = map[string]historyRange{
	"1h":  {Duration: time.Hour, Label: "Last 1 hour"},
	"6h":  {Duration: 6 * time.Hour, Label: "Last 6 hours"},
	"24h": {Duration: 24 * time.Hour, Label: "Last 24 hours"},
	"7d":  {Duration: 7 * 24 * time.Hour, Label: "Last 7 days"},
}

func parseReadingsQuery(r *http.Request) (from time.Time, to time.Time, limit int, err error) {
	q := r.URL.Query()

	if s := q.Get("from"); s != "" {
		from, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, errors.New("invalid 'from' (expected RFC3339)")
		}
	}
	if s := q.Get("to"); s != "" {
		to, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, time.Time{}, 0, errors.New("invalid 'to' (expected RFC3339)")
		}
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return time.Time{}, time.Time{}, 0, errors.New("'from' must be <= 'to'")
	}

	limit = 100
	if s := q.Get("limit"); s != "" {
		n, convErr := strconv.Atoi(s)
		if convErr != nil {
			return time.Time{}, time.Time{}, 0, errors.New("invalid 'limit' (expected integer)")
		}
		if n <= 0 {
			return time.Time{}, time.Time{}, 0, errors.New("'limit' must be > 0")
		}
		if n > 1000 {
			return time.Time{}, time.Time{}, 0, errors.New("'limit' must be <= 1000")
		}
		limit = n
	}

	return from, to, limit, nil
}

func parseLatestQuery(r *http.Request) (limit int, err error) {
	q := r.URL.Query()
	limit = 100
	if s := q.Get("limit"); s != "" {
		n, convErr := strconv.Atoi(s)
		if convErr != nil {
			return 0, errors.New("invalid 'limit' (expected integer)")
		}
		if n <= 0 {
			return 0, errors.New("'limit' must be > 0")
		}
		if n > 1000 {
			return 0, errors.New("'limit' must be <= 1000")
		}
		limit = n
	}
	return limit, nil
}

func resolveHistoryRange(key string) (historyRange, bool) {
	if key == "" {
		return historyRanges[defaultHistoryRangeKey], true
	}
	info, ok := historyRanges[key]
	if ok {
		return info, true
	}
	return historyRanges[defaultHistoryRangeKey], false
}

// parseHistoryPage returns the 1-based page number from the request (default 1, min 1).
func parseHistoryPage(r *http.Request) int {
	s := r.URL.Query().Get("page")
	if s == "" {
		return 1
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func zeroAsNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// weatherState holds persisted dashboard/history state from the cookie.
type weatherState struct {
	StationID string
	RangeKey  string
	Page      int
}

// readWeatherStateCookie parses the weather_state cookie and returns station_id, range key, and page.
// Returns zero values when the cookie is missing or invalid. Range and page are validated.
func readWeatherStateCookie(r *http.Request) weatherState {
	c, err := r.Cookie(weatherStateCookieName)
	if err != nil {
		return weatherState{}
	}
	vals, err := url.ParseQuery(c.Value)
	if err != nil {
		return weatherState{}
	}
	stationID := vals.Get("station_id")
	rangeKey := vals.Get("range")
	if _, ok := historyRanges[rangeKey]; !ok {
		rangeKey = ""
	}
	page := 1
	if s := vals.Get("page"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 1 {
			page = n
		}
	}
	return weatherState{StationID: stationID, RangeKey: rangeKey, Page: page}
}

// writeWeatherStateCookie sets the weather_state cookie with the given state.
// rangeKey must be a valid history range key (use defaultHistoryRangeKey if unsure).
func writeWeatherStateCookie(w http.ResponseWriter, stationID, rangeKey string, page int) {
	if _, ok := historyRanges[rangeKey]; !ok {
		rangeKey = defaultHistoryRangeKey
	}
	if page < 1 {
		page = 1
	}
	val := url.Values{}
	val.Set("station_id", stationID)
	val.Set("range", rangeKey)
	val.Set("page", strconv.Itoa(page))
	http.SetCookie(w, &http.Cookie{
		Name:     weatherStateCookieName,
		Value:    val.Encode(),
		Path:     "/",
		MaxAge:   weatherStateCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true if you serve over HTTPS only
	})
}
