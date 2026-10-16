package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// do sends one request to a fresh in-process server and returns the response.
func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeTask(t *testing.T, rec *httptest.ResponseRecorder) Task {
	t.Helper()
	var task Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatalf("response was not a task: %v (%s)", err, rec.Body.String())
	}
	return task
}

func TestCreateAndGet(t *testing.T) {
	h := NewServer(NewStore())

	rec := do(t, h, "POST", "/tasks", `{"title":"  Write README  "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201", rec.Code)
	}
	created := decodeTask(t, rec)
	if created.Title != "Write README" {
		t.Errorf("title not trimmed: %q", created.Title)
	}

	rec = do(t, h, "GET", "/tasks/1", "")
	if rec.Code != http.StatusOK || decodeTask(t, rec).ID != created.ID {
		t.Errorf("get: got %d %s", rec.Code, rec.Body.String())
	}
}

func TestCreateValidation(t *testing.T) {
	h := NewServer(NewStore())
	cases := map[string]string{
		"empty title":   `{"title":"   "}`,
		"missing title": `{}`,
		"too long":      `{"title":"` + strings.Repeat("a", 201) + `"}`,
		"bad json":      `{not json`,
		"unknown field": `{"title":"x","colour":"red"}`,
	}
	for name, body := range cases {
		if rec := do(t, h, "POST", "/tasks", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
	}
}

func TestListFilterByDone(t *testing.T) {
	h := NewServer(NewStore())
	do(t, h, "POST", "/tasks", `{"title":"one"}`)
	do(t, h, "POST", "/tasks", `{"title":"two"}`)
	do(t, h, "PATCH", "/tasks/2", `{"done":true}`)

	count := func(path string) int {
		rec := do(t, h, "GET", path, "")
		var tasks []Task
		json.Unmarshal(rec.Body.Bytes(), &tasks)
		return len(tasks)
	}
	if got := count("/tasks"); got != 2 {
		t.Errorf("all: got %d, want 2", got)
	}
	if got := count("/tasks?done=true"); got != 1 {
		t.Errorf("done=true: got %d, want 1", got)
	}
	if got := count("/tasks?done=false"); got != 1 {
		t.Errorf("done=false: got %d, want 1", got)
	}
	if rec := do(t, h, "GET", "/tasks?done=maybe", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad done value: got %d, want 400", rec.Code)
	}
}

func TestUpdate(t *testing.T) {
	h := NewServer(NewStore())
	do(t, h, "POST", "/tasks", `{"title":"old"}`)

	rec := do(t, h, "PATCH", "/tasks/1", `{"title":"new","done":true}`)
	got := decodeTask(t, rec)
	if rec.Code != http.StatusOK || got.Title != "new" || !got.Done {
		t.Errorf("update failed: %d %+v", rec.Code, got)
	}

	// Sending only done must not wipe the title.
	rec = do(t, h, "PATCH", "/tasks/1", `{"done":false}`)
	got = decodeTask(t, rec)
	if got.Title != "new" || got.Done {
		t.Errorf("partial update wrong: %+v", got)
	}

	if rec := do(t, h, "PATCH", "/tasks/1", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty update: got %d, want 400", rec.Code)
	}
	if rec := do(t, h, "PATCH", "/tasks/1", `{"title":""}`); rec.Code != http.StatusBadRequest {
		t.Errorf("blank title: got %d, want 400", rec.Code)
	}
}

func TestNotFoundAndBadID(t *testing.T) {
	h := NewServer(NewStore())
	if rec := do(t, h, "GET", "/tasks/99", ""); rec.Code != http.StatusNotFound {
		t.Errorf("get missing: got %d, want 404", rec.Code)
	}
	if rec := do(t, h, "PATCH", "/tasks/99", `{"done":true}`); rec.Code != http.StatusNotFound {
		t.Errorf("patch missing: got %d, want 404", rec.Code)
	}
	if rec := do(t, h, "GET", "/tasks/abc", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("bad id: got %d, want 400", rec.Code)
	}
}

func TestDelete(t *testing.T) {
	h := NewServer(NewStore())
	do(t, h, "POST", "/tasks", `{"title":"temp"}`)
	if rec := do(t, h, "DELETE", "/tasks/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", rec.Code)
	}
	if rec := do(t, h, "DELETE", "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("second delete: got %d, want 404", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	if rec := do(t, NewServer(NewStore()), "GET", "/health", ""); rec.Code != http.StatusOK {
		t.Errorf("health: got %d, want 200", rec.Code)
	}
}

// Run with `go test -race` to check the mutex really protects the store.
func TestConcurrentCreates(t *testing.T) {
	store := NewStore()
	h := NewServer(store)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			do(t, h, "POST", "/tasks", `{"title":"parallel"}`)
		}()
	}
	wg.Wait()

	if got := len(store.List(nil, "")); got != 50 {
		t.Errorf("got %d tasks, want 50", got)
	}
}

func TestSearch(t *testing.T) {
	h := NewServer(NewStore())
	do(t, h, "POST", "/tasks", `{"title":"Buy milk"}`)
	do(t, h, "POST", "/tasks", `{"title":"Write report"}`)
	do(t, h, "POST", "/tasks", `{"title":"buy stamps"}`)
	do(t, h, "PATCH", "/tasks/3", `{"done":true}`)

	count := func(path string) int {
		rec := do(t, h, "GET", path, "")
		var tasks []Task
		json.Unmarshal(rec.Body.Bytes(), &tasks)
		return len(tasks)
	}
	if got := count("/tasks?q=buy"); got != 2 {
		t.Errorf("q=buy (case-insensitive): got %d, want 2", got)
	}
	if got := count("/tasks?q=REPORT"); got != 1 {
		t.Errorf("q=REPORT: got %d, want 1", got)
	}
	if got := count("/tasks?q=nothing"); got != 0 {
		t.Errorf("q=nothing: got %d, want 0", got)
	}
	if got := count("/tasks?q=buy&done=true"); got != 1 {
		t.Errorf("q=buy&done=true: got %d, want 1", got)
	}
	if got := count("/tasks?q="); got != 3 {
		t.Errorf("empty q should return everything: got %d, want 3", got)
	}
}
