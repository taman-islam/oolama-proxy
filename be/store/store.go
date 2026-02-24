package store

import (
	"lb/users"
	"sync"
)

// ModelUsage tracks token usage for one model.
type ModelUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// Store is a thread-safe in-memory usage store.
type Store struct {
	mu   sync.Mutex
	data map[string]map[string]*ModelUsage // user -> model -> usage
}

func New() *Store {
	return &Store{data: make(map[string]map[string]*ModelUsage)}
}

// Add increments token counts for the given user + model.
func (s *Store) Add(user, model string, prompt, completion int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[user] == nil {
		s.data[user] = make(map[string]*ModelUsage)
	}
	u := s.data[user][model]
	if u == nil {
		u = &ModelUsage{}
		s.data[user][model] = u
	}
	u.PromptTokens += prompt
	u.CompletionTokens += completion
}

// Get returns a copy of usage for the given user, keyed by model.
func (s *Store) Get(user string) map[string]ModelUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]ModelUsage)
	for model, u := range s.data[user] {
		out[model] = *u
	}
	return out
}

// GetAll returns usage for every user (for admin UI).
func (s *Store) GetAll() map[string]map[string]ModelUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]map[string]ModelUsage)
	for _, user := range users.All() {
		out[user.ID] = make(map[string]ModelUsage)
	}
	for user, models := range s.data {
		out[user] = make(map[string]ModelUsage)
		for model, u := range models {
			out[user][model] = *u
		}
	}
	return out
}
