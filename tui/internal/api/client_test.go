package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entries" || r.Method != "GET" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("authorization") != "Bearer tok" {
			t.Errorf("missing bearer")
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode([]EntryMeta{{ID: "a", Title: "t", CreatedAt: time.Now()}})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	got, err := c.ListEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("got %+v", got)
	}
}

func TestCreateEntrySendsJSON(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"xyz","title":"hello","created_at":"2026-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	meta, err := c.CreateEntry("hello", "body text", "work/journal")
	if err != nil {
		t.Fatal(err)
	}
	if meta.ID != "xyz" {
		t.Fatalf("id = %q", meta.ID)
	}
	if !strings.Contains(receivedBody, `"title":"hello"`) ||
		!strings.Contains(receivedBody, `"body":"body text"`) ||
		!strings.Contains(receivedBody, `"folder":"work/journal"`) {
		t.Fatalf("unexpected request body: %s", receivedBody)
	}
}

func TestUpdateEntryPatchesTitleAndBody(t *testing.T) {
	var method, path, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{"id":"abc","title":"new","folder":"work","created_at":"2026-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	meta, err := c.UpdateEntry("abc", "new", "new body")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPatch || path != "/entries/abc" {
		t.Fatalf("unexpected %s %s", method, path)
	}
	if !strings.Contains(body, `"title":"new"`) || !strings.Contains(body, `"body":"new body"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	if meta.Title != "new" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestMoveEntryPatchesFolderOnly(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/entries/abc" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{"id":"abc","title":"t","folder":"archive","created_at":"2026-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	meta, err := c.MoveEntry("abc", "archive")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, `"folder":"archive"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	if strings.Contains(body, `"body"`) || strings.Contains(body, `"title"`) {
		t.Fatalf("move must send folder only, got: %s", body)
	}
	if meta.Folder != "archive" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestRefreshesAndRetriesOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("authorization") == "Bearer fresh" {
			_ = json.NewEncoder(w).Encode([]EntryMeta{{ID: "a"}})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "stale")
	refreshed := 0
	c.Refresh = func() (string, error) {
		refreshed++
		return "fresh", nil
	}

	got, err := c.ListEntries()
	if err != nil {
		t.Fatalf("expected the retry to succeed, got %v", err)
	}
	if refreshed != 1 {
		t.Fatalf("expected exactly one refresh, got %d", refreshed)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestUnauthorizedReturnsSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	_, err := c.ListEntries()
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestDelete204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/entries/abc" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := New(srv.URL, "tok").DeleteEntry("abc"); err != nil {
		t.Fatal(err)
	}
}
