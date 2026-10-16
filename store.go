package main

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Task is one item in the to-do list. The `json:"..."` tags control the
// field names used in the API's JSON.
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrNotFound is returned when a task ID does not exist.
var ErrNotFound = errors.New("task not found")

// Store keeps tasks in memory. The mutex makes it safe when several
// requests arrive at the same time (Go's HTTP server handles each request
// in its own goroutine).
type Store struct {
	mu     sync.Mutex
	nextID int
	tasks  map[int]Task
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{nextID: 1, tasks: make(map[int]Task)}
}

// Create adds a new, not-yet-done task and returns it.
func (s *Store) Create(title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{ID: s.nextID, Title: title, CreatedAt: time.Now().UTC()}
	s.tasks[t.ID] = t
	s.nextID++
	return t
}

// List returns tasks filtered by completion status and keyword search, ordered by ID.
func (s *Store) List(done *bool, search string) []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		matchesDone := done == nil || t.Done == *done

		// Check if the search term matches any part of the title (case-insensitive)
		matchesSearch := search == "" || strings.Contains(strings.ToLower(t.Title), strings.ToLower(search))

		if matchesDone && matchesSearch {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns one task or ErrNotFound.
func (s *Store) Get(id int) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// Update changes the title and/or done flag. A nil pointer means
// "leave this field as it is".
func (s *Store) Update(id int, title *string, done *bool) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if title != nil {
		t.Title = *title
	}
	if done != nil {
		t.Done = *done
	}
	s.tasks[id] = t
	return t, nil
}

// Delete removes a task or returns ErrNotFound.
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
