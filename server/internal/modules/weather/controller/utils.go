package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

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

func zeroAsNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
