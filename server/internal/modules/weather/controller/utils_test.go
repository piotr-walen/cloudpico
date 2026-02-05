package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func Test_parseReadingsQuery(t *testing.T) {
	t.Run("no params returns defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings", nil)
		from, to, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		if !from.IsZero() || !to.IsZero() {
			t.Errorf("from.IsZero()=%v to.IsZero()=%v; want both true", from.IsZero(), to.IsZero())
		}
		if limit != 100 {
			t.Errorf("limit = %d; want 100", limit)
		}
	})

	t.Run("valid from only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?from=2025-01-01T00:00:00Z", nil)
		from, to, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		wantFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if !from.Equal(wantFrom) {
			t.Errorf("from = %v; want %v", from, wantFrom)
		}
		if !to.IsZero() {
			t.Errorf("to should be zero; got %v", to)
		}
		if limit != 100 {
			t.Errorf("limit = %d; want 100", limit)
		}
	})

	t.Run("valid to only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?to=2025-12-31T23:59:59Z", nil)
		from, to, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		if !from.IsZero() {
			t.Errorf("from should be zero; got %v", from)
		}
		wantTo := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
		if !to.Equal(wantTo) {
			t.Errorf("to = %v; want %v", to, wantTo)
		}
		if limit != 100 {
			t.Errorf("limit = %d; want 100", limit)
		}
	})

	t.Run("valid from and to", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?from=2025-01-01T00:00:00Z&to=2025-01-31T12:00:00Z", nil)
		from, to, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		wantFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		wantTo := time.Date(2025, 1, 31, 12, 0, 0, 0, time.UTC)
		if !from.Equal(wantFrom) || !to.Equal(wantTo) {
			t.Errorf("from=%v to=%v; want from=%v to=%v", from, to, wantFrom, wantTo)
		}
		if limit != 100 {
			t.Errorf("limit = %d; want 100", limit)
		}
	})

	t.Run("invalid from returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?from=not-a-date", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "invalid 'from' (expected RFC3339)" {
			t.Errorf("err = %q; want invalid 'from' (expected RFC3339)", err.Error())
		}
	})

	t.Run("invalid to returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?to=bad", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "invalid 'to' (expected RFC3339)" {
			t.Errorf("err = %q; want invalid 'to' (expected RFC3339)", err.Error())
		}
	})

	t.Run("from after to returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?from=2025-02-01T00:00:00Z&to=2025-01-01T00:00:00Z", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "'from' must be <= 'to'" {
			t.Errorf("err = %q; want 'from' must be <= 'to'", err.Error())
		}
	})

	t.Run("valid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=50", nil)
		_, _, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		if limit != 50 {
			t.Errorf("limit = %d; want 50", limit)
		}
	})

	t.Run("limit 1 allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=1", nil)
		_, _, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		if limit != 1 {
			t.Errorf("limit = %d; want 1", limit)
		}
	})

	t.Run("limit 1000 allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=1000", nil)
		_, _, limit, err := parseReadingsQuery(req)
		if err != nil {
			t.Fatalf("parseReadingsQuery() err = %v; want nil", err)
		}
		if limit != 1000 {
			t.Errorf("limit = %d; want 1000", limit)
		}
	})

	t.Run("invalid limit (non-integer) returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=abc", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "invalid 'limit' (expected integer)" {
			t.Errorf("err = %q; want invalid 'limit' (expected integer)", err.Error())
		}
	})

	t.Run("limit zero returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=0", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "'limit' must be > 0" {
			t.Errorf("err = %q; want 'limit' must be > 0", err.Error())
		}
	})

	t.Run("limit negative returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=-5", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "'limit' must be > 0" {
			t.Errorf("err = %q; want 'limit' must be > 0", err.Error())
		}
	})

	t.Run("limit over 1000 returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readings?limit=1001", nil)
		_, _, _, err := parseReadingsQuery(req)
		if err == nil {
			t.Fatal("parseReadingsQuery() err = nil; want non-nil")
		}
		if err.Error() != "'limit' must be <= 1000" {
			t.Errorf("err = %q; want 'limit' must be <= 1000", err.Error())
		}
	})
}

func Test_parseLatestQuery(t *testing.T) {
	t.Run("no limit returns default 100", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest", nil)
		limit, err := parseLatestQuery(req)
		if err != nil {
			t.Fatalf("parseLatestQuery() err = %v; want nil", err)
		}
		if limit != 100 {
			t.Errorf("limit = %d; want 100", limit)
		}
	})
	t.Run("valid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=50", nil)
		limit, err := parseLatestQuery(req)
		if err != nil {
			t.Fatalf("parseLatestQuery() err = %v; want nil", err)
		}
		if limit != 50 {
			t.Errorf("limit = %d; want 50", limit)
		}
	})
	t.Run("limit 1 allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=1", nil)
		limit, err := parseLatestQuery(req)
		if err != nil {
			t.Fatalf("parseLatestQuery() err = %v; want nil", err)
		}
		if limit != 1 {
			t.Errorf("limit = %d; want 1", limit)
		}
	})
	t.Run("limit 1000 allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=1000", nil)
		limit, err := parseLatestQuery(req)
		if err != nil {
			t.Fatalf("parseLatestQuery() err = %v; want nil", err)
		}
		if limit != 1000 {
			t.Errorf("limit = %d; want 1000", limit)
		}
	})
	t.Run("invalid limit (non-integer) returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=abc", nil)
		_, err := parseLatestQuery(req)
		if err == nil {
			t.Fatal("parseLatestQuery() err = nil; want non-nil")
		}
		if err.Error() != "invalid 'limit' (expected integer)" {
			t.Errorf("err = %q; want invalid 'limit' (expected integer)", err.Error())
		}
	})
	t.Run("limit zero returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=0", nil)
		_, err := parseLatestQuery(req)
		if err == nil {
			t.Fatal("parseLatestQuery() err = nil; want non-nil")
		}
		if err.Error() != "'limit' must be > 0" {
			t.Errorf("err = %q; want 'limit' must be > 0", err.Error())
		}
	})
	t.Run("limit negative returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=-5", nil)
		_, err := parseLatestQuery(req)
		if err == nil {
			t.Fatal("parseLatestQuery() err = nil; want non-nil")
		}
		if err.Error() != "'limit' must be > 0" {
			t.Errorf("err = %q; want 'limit' must be > 0", err.Error())
		}
	})
	t.Run("limit over 1000 returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/latest?limit=1001", nil)
		_, err := parseLatestQuery(req)
		if err == nil {
			t.Fatal("parseLatestQuery() err = nil; want non-nil")
		}
		if err.Error() != "'limit' must be <= 1000" {
			t.Errorf("err = %q; want 'limit' must be <= 1000", err.Error())
		}
	})
}

func Test_zeroAsNullTime(t *testing.T) {
	t.Run("zero time returns nil", func(t *testing.T) {
		got := zeroAsNullTime(time.Time{})
		if got != nil {
			t.Errorf("zeroAsNullTime(zero) = %v; want nil", got)
		}
	})

	t.Run("non-zero time returns time", func(t *testing.T) {
		ts := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
		got := zeroAsNullTime(ts)
		if got == nil {
			t.Fatal("zeroAsNullTime(non-zero) = nil; want time")
		}
		if tVal, ok := got.(time.Time); !ok || !tVal.Equal(ts) {
			t.Errorf("zeroAsNullTime(non-zero) = %v (%T); want %v", got, got, ts)
		}
	})
}

func Test_resolveHistoryRange(t *testing.T) {
	defaultRange := historyRanges[defaultHistoryRangeKey]

	t.Run("empty key returns default and true", func(t *testing.T) {
		got, ok := resolveHistoryRange("")
		if !ok {
			t.Error("resolveHistoryRange(\"\") ok = false; want true")
		}
		if got.Duration != defaultRange.Duration || got.Label != defaultRange.Label {
			t.Errorf("resolveHistoryRange(\"\") = %+v; want %+v", got, defaultRange)
		}
	})

	t.Run("valid key 1h returns that range and true", func(t *testing.T) {
		got, ok := resolveHistoryRange("1h")
		if !ok {
			t.Error("resolveHistoryRange(\"1h\") ok = false; want true")
		}
		want := historyRanges["1h"]
		if got.Duration != want.Duration || got.Label != want.Label {
			t.Errorf("resolveHistoryRange(\"1h\") = %+v; want %+v", got, want)
		}
	})

	t.Run("valid key 24h returns that range and true", func(t *testing.T) {
		got, ok := resolveHistoryRange("24h")
		if !ok {
			t.Error("resolveHistoryRange(\"24h\") ok = false; want true")
		}
		want := historyRanges["24h"]
		if got.Duration != want.Duration || got.Label != want.Label {
			t.Errorf("resolveHistoryRange(\"24h\") = %+v; want %+v", got, want)
		}
	})

	t.Run("valid key 7d returns that range and true", func(t *testing.T) {
		got, ok := resolveHistoryRange("7d")
		if !ok {
			t.Error("resolveHistoryRange(\"7d\") ok = false; want true")
		}
		want := historyRanges["7d"]
		if got.Duration != want.Duration || got.Label != want.Label {
			t.Errorf("resolveHistoryRange(\"7d\") = %+v; want %+v", got, want)
		}
	})

	t.Run("invalid key returns default and false", func(t *testing.T) {
		got, ok := resolveHistoryRange("invalid")
		if ok {
			t.Error("resolveHistoryRange(\"invalid\") ok = true; want false")
		}
		if got.Duration != defaultRange.Duration || got.Label != defaultRange.Label {
			t.Errorf("resolveHistoryRange(\"invalid\") = %+v; want default %+v", got, defaultRange)
		}
	})

	t.Run("unknown key returns default and false", func(t *testing.T) {
		got, ok := resolveHistoryRange("30d")
		if ok {
			t.Error("resolveHistoryRange(\"30d\") ok = true; want false")
		}
		if got.Duration != defaultRange.Duration {
			t.Errorf("resolveHistoryRange(\"30d\") duration = %v; want default %v", got.Duration, defaultRange.Duration)
		}
	})
}

func Test_parseHistoryPage(t *testing.T) {
	t.Run("no page param returns 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history", nil)
		got := parseHistoryPage(req)
		if got != 1 {
			t.Errorf("parseHistoryPage() = %d; want 1", got)
		}
	})

	t.Run("valid page returns that page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history?page=5", nil)
		got := parseHistoryPage(req)
		if got != 5 {
			t.Errorf("parseHistoryPage() = %d; want 5", got)
		}
	})

	t.Run("page=1 returns 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history?page=1", nil)
		got := parseHistoryPage(req)
		if got != 1 {
			t.Errorf("parseHistoryPage() = %d; want 1", got)
		}
	})

	t.Run("invalid page (non-integer) returns 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history?page=abc", nil)
		got := parseHistoryPage(req)
		if got != 1 {
			t.Errorf("parseHistoryPage() = %d; want 1", got)
		}
	})

	t.Run("page zero returns 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history?page=0", nil)
		got := parseHistoryPage(req)
		if got != 1 {
			t.Errorf("parseHistoryPage() = %d; want 1", got)
		}
	})

	t.Run("negative page returns 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/history?page=-3", nil)
		got := parseHistoryPage(req)
		if got != 1 {
			t.Errorf("parseHistoryPage() = %d; want 1", got)
		}
	})
}

func Test_readWeatherStateCookie(t *testing.T) {
	t.Run("no cookie returns zero state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		got := readWeatherStateCookie(req)
		if got.StationID != "" || got.RangeKey != "" || got.Page != 0 {
			t.Errorf("readWeatherStateCookie() = %+v; want zero weatherState", got)
		}
	})

	t.Run("malformed cookie value returns zero state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "not-valid-query%%"})
		got := readWeatherStateCookie(req)
		if got.StationID != "" || got.RangeKey != "" || got.Page != 0 {
			t.Errorf("readWeatherStateCookie(malformed) = %+v; want zero weatherState", got)
		}
	})

	t.Run("valid cookie parses all fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "station_id=st1&range=7d&page=3"})
		got := readWeatherStateCookie(req)
		if got.StationID != "st1" || got.RangeKey != "7d" || got.Page != 3 {
			t.Errorf("readWeatherStateCookie() = %+v; want StationID=st1 RangeKey=7d Page=3", got)
		}
	})

	t.Run("invalid range key in cookie yields empty RangeKey", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "station_id=st1&range=badkey&page=2"})
		got := readWeatherStateCookie(req)
		if got.RangeKey != "" {
			t.Errorf("readWeatherStateCookie(invalid range) RangeKey = %q; want \"\"", got.RangeKey)
		}
		if got.StationID != "st1" || got.Page != 2 {
			t.Errorf("readWeatherStateCookie() = %+v; want StationID=st1 Page=2 RangeKey empty", got)
		}
	})

	t.Run("negative page in cookie yields page 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "station_id=x&range=24h&page=-1"})
		got := readWeatherStateCookie(req)
		if got.Page != 1 {
			t.Errorf("readWeatherStateCookie(negative page) Page = %d; want 1", got.Page)
		}
	})

	t.Run("zero page in cookie yields page 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "station_id=x&range=24h&page=0"})
		got := readWeatherStateCookie(req)
		if got.Page != 1 {
			t.Errorf("readWeatherStateCookie(page=0) Page = %d; want 1", got.Page)
		}
	})

	t.Run("non-integer page in cookie yields page 1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "station_id=x&range=24h&page=abc"})
		got := readWeatherStateCookie(req)
		if got.Page != 1 {
			t.Errorf("readWeatherStateCookie(page=abc) Page = %d; want 1", got.Page)
		}
	})

	t.Run("missing optional fields use defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: weatherStateCookieName, Value: "station_id=only"})
		got := readWeatherStateCookie(req)
		if got.StationID != "only" {
			t.Errorf("StationID = %q; want \"only\"", got.StationID)
		}
		if got.RangeKey != "" {
			t.Errorf("RangeKey = %q; want \"\" (not in cookie)", got.RangeKey)
		}
		if got.Page != 1 {
			t.Errorf("Page = %d; want 1 (default)", got.Page)
		}
	})
}

func parseCookieValue(value string) (stationID, rangeKey string, page int) {
	vals, err := url.ParseQuery(value)
	if err != nil {
		return "", "", 0
	}
	stationID = vals.Get("station_id")
	rangeKey = vals.Get("range")
	page, _ = strconv.Atoi(vals.Get("page"))
	return stationID, rangeKey, page
}

func Test_writeWeatherStateCookie(t *testing.T) {
	t.Run("writes cookie with correct name and encoded value", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeWeatherStateCookie(w, "st1", "24h", 2)
		header := w.Header().Get("Set-Cookie")
		if header == "" {
			t.Fatal("Set-Cookie header missing")
		}
		if len(w.Result().Cookies()) != 1 {
			t.Fatalf("expected 1 cookie; got %d", len(w.Result().Cookies()))
		}
		c := w.Result().Cookies()[0]
		if c.Name != weatherStateCookieName {
			t.Errorf("cookie Name = %q; want %q", c.Name, weatherStateCookieName)
		}
		stationID, rangeKey, page := parseCookieValue(c.Value)
		if stationID != "st1" || rangeKey != "24h" || page != 2 {
			t.Errorf("cookie Value parsed: station_id=%q range=%q page=%d; want st1, 24h, 2", stationID, rangeKey, page)
		}
		if c.Path != "/" {
			t.Errorf("cookie Path = %q; want \"/\"", c.Path)
		}
		if c.MaxAge != weatherStateCookieMaxAge {
			t.Errorf("cookie MaxAge = %d; want %d", c.MaxAge, weatherStateCookieMaxAge)
		}
		if !c.HttpOnly {
			t.Error("cookie HttpOnly = false; want true")
		}
	})

	t.Run("invalid range key uses default", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeWeatherStateCookie(w, "st1", "invalid", 1)
		c := w.Result().Cookies()[0]
		_, rangeKey, page := parseCookieValue(c.Value)
		if rangeKey != defaultHistoryRangeKey {
			t.Errorf("range = %q; want default %q", rangeKey, defaultHistoryRangeKey)
		}
		if page != 1 {
			t.Errorf("page = %d; want 1", page)
		}
	})

	t.Run("page less than 1 uses 1", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeWeatherStateCookie(w, "st1", "24h", 0)
		c := w.Result().Cookies()[0]
		_, _, page := parseCookieValue(c.Value)
		if page != 1 {
			t.Errorf("page = %d; want 1", page)
		}
	})

	t.Run("negative page uses 1", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeWeatherStateCookie(w, "x", "1h", -5)
		c := w.Result().Cookies()[0]
		_, _, page := parseCookieValue(c.Value)
		if page != 1 {
			t.Errorf("page = %d; want 1", page)
		}
	})
}
