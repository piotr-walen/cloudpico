package controller

import (
	"net/http"
	"net/http/httptest"
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
