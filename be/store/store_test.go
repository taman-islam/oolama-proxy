package store_test

import (
	"lb/store"
	"testing"
)

func TestAdd_And_Get(t *testing.T) {
	s := store.New()
	s.Add("user-a", "llama3.2:1b", 10, 20)
	s.Add("user-a", "llama3.2:1b", 5, 3)

	usage := s.Get("user-a")
	u, ok := usage["llama3.2:1b"]
	if !ok {
		t.Fatal("expected usage for llama3.2:1b")
	}
	if u.PromptTokens != 15 {
		t.Errorf("prompt tokens: got %d, want 15", u.PromptTokens)
	}
	if u.CompletionTokens != 23 {
		t.Errorf("completion tokens: got %d, want 23", u.CompletionTokens)
	}
}

func TestGet_UnknownUser(t *testing.T) {
	s := store.New()
	usage := s.Get("nobody")
	if len(usage) != 0 {
		t.Errorf("expected empty map for unknown user, got %v", usage)
	}
}

func TestAdd_MultipleModels(t *testing.T) {
	s := store.New()
	s.Add("user-b", "llama3.2:1b", 10, 5)
	s.Add("user-b", "moondream", 30, 2)

	usage := s.Get("user-b")
	if len(usage) != 2 {
		t.Errorf("expected 2 models, got %d", len(usage))
	}
	if usage["moondream"].PromptTokens != 30 {
		t.Errorf("moondream prompt: got %d, want 30", usage["moondream"].PromptTokens)
	}
}

func TestGetAll(t *testing.T) {
	s := store.New()
	s.Add("user-c", "llama3.2:1b", 1, 1)
	s.Add("user-d", "moondream", 2, 2)

	all := s.GetAll()
	if all["user-c"]["llama3.2:1b"].PromptTokens != 1 {
		t.Errorf("expected user-c to have 1 prompt token for llama3.2:1b")
	}
	if all["user-d"]["moondream"].PromptTokens != 2 {
		t.Errorf("expected user-d to have 2 prompt tokens for moondream")
	}
	// Verify that statically registered users are also seeded.
	if _, ok := all["alice"]; !ok {
		t.Errorf("expected static user 'alice' to be seeded in GetAll")
	}
}

func TestAdd_ThreadSafe(t *testing.T) {
	s := store.New()
	done := make(chan struct{})
	// Fire 100 concurrent writers
	for i := 0; i < 100; i++ {
		go func() {
			s.Add("shared-user", "llama3.2:1b", 1, 1)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	usage := s.Get("shared-user")
	if usage["llama3.2:1b"].PromptTokens != 100 {
		t.Errorf("expected 100 prompt tokens, got %d", usage["llama3.2:1b"].PromptTokens)
	}
}
