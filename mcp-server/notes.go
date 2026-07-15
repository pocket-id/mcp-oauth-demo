package main

import (
	"sync"
	"time"
)

const maxNotesPerUser = 100

type note struct {
	ID        uint64    `json:"id" jsonschema:"the note identifier"`
	Text      string    `json:"text" jsonschema:"the note text"`
	CreatedAt time.Time `json:"createdAt" jsonschema:"when the note was created"`
}

type userNotes struct {
	NextID uint64
	Notes  []note
}

type noteStore struct {
	mu    sync.RWMutex
	users map[string]*userNotes
	now   func() time.Time
}

func newNoteStore() *noteStore {
	return &noteStore{
		users: make(map[string]*userNotes),
		now:   time.Now,
	}
}

func (s *noteStore) add(userID, text string) (note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[userID]
	if user == nil {
		user = &userNotes{NextID: 1}
		s.users[userID] = user
	}
	if len(user.Notes) >= maxNotesPerUser {
		return note{}, false
	}

	created := note{
		ID:        user.NextID,
		Text:      text,
		CreatedAt: s.now().UTC(),
	}
	user.NextID++
	user.Notes = append(user.Notes, created)
	return created, true
}

func (s *noteStore) list(userID string) []note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user := s.users[userID]
	if user == nil {
		return []note{}
	}
	return append([]note(nil), user.Notes...)
}

func (s *noteStore) clear(userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[userID]
	if user == nil {
		return 0
	}
	count := len(user.Notes)
	delete(s.users, userID)
	return count
}
