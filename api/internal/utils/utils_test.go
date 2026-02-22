package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Run("sets content-type and status", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]string{"key": "value"}
		WriteJSON(w, http.StatusOK, body)

		if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q; want application/json; charset=utf-8", got)
		}
		if w.Code != http.StatusOK {
			t.Errorf("Code = %d; want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("encodes body as JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]string{"foo": "bar"}
		WriteJSON(w, http.StatusCreated, body)

		var got map[string]string
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("body is not valid JSON: %v", err)
		}
		if got["foo"] != "bar" {
			t.Errorf("body[foo] = %q; want bar", got["foo"])
		}
	})
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	status := http.StatusBadRequest
	msg := "invalid input"
	WriteError(w, status, msg)

	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q; want application/json; charset=utf-8", got)
	}
	if w.Code != status {
		t.Errorf("Code = %d; want %d", w.Code, status)
	}

	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got["error"] != http.StatusText(status) {
		t.Errorf("error = %q; want %q", got["error"], http.StatusText(status))
	}
	if got["message"] != msg {
		t.Errorf("message = %q; want %q", got["message"], msg)
	}
}
