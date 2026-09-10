package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

type OpenRequest struct {
	BufferID string
}

type Session struct {
	ID       string
	BufferID string
	OK       bool
}

type Registry struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func NewRegistry() *Registry {
	return &Registry{sessions: map[string]Session{}}
}

func (r *Registry) Open(req OpenRequest) Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := Session{ID: randomID(), BufferID: req.BufferID, OK: true}
	r.sessions[session.ID] = session
	return session
}

func (r *Registry) Lookup(id string) Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[id]
	if !ok {
		return Session{}
	}
	session.OK = true
	return session
}

func (r *Registry) Close(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
}

func randomID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(fmt.Errorf("generate org preview session id: %w", err))
	}
	return "session-" + hex.EncodeToString(buf[:])
}
