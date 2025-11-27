package models

import (
	"sync"
)

// RoomStore is an in-memory store for rooms
type RoomStore struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

// NewRoomStore creates a new room store
func NewRoomStore() *RoomStore {
	return &RoomStore{
		rooms: make(map[string]*Room),
	}
}

// Create adds a new room to the store
func (s *RoomStore) Create(room *Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = room
}

// Get retrieves a room by ID
func (s *RoomStore) Get(id string) (*Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[id]
	return room, ok
}

// Delete removes a room from the store
func (s *RoomStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, id)
}

// List returns all rooms
func (s *RoomStore) List() []*Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]*Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// ListByOwner returns all rooms owned by a user
func (s *RoomStore) ListByOwner(ownerID string) []*Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]*Room, 0)
	for _, room := range s.rooms {
		if room.OwnerID == ownerID {
			rooms = append(rooms, room)
		}
	}
	return rooms
}

// ListByParticipant returns all rooms where user is a participant
func (s *RoomStore) ListByParticipant(userID string) []*Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rooms := make([]*Room, 0)
	for _, room := range s.rooms {
		room.RLock()
		if _, ok := room.Participants[userID]; ok {
			rooms = append(rooms, room)
		}
		room.RUnlock()
	}
	return rooms
}
