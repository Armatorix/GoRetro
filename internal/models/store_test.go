package models

import (
	"testing"
)

func TestRoomStore_Create(t *testing.T) {
	store := NewRoomStore()
	room := NewRoom("room-1", "Test Room", "owner-1", 3)

	store.Create(room)

	got, ok := store.Get("room-1")
	if !ok {
		t.Error("Expected room to be found")
	}
	if got.ID != "room-1" {
		t.Errorf("Expected room ID 'room-1', got '%s'", got.ID)
	}
}

func TestRoomStore_Delete(t *testing.T) {
	store := NewRoomStore()
	room := NewRoom("room-1", "Test Room", "owner-1", 3)

	store.Create(room)
	store.Delete("room-1")

	_, ok := store.Get("room-1")
	if ok {
		t.Error("Expected room to be deleted")
	}
}

func TestRoomStore_List(t *testing.T) {
	store := NewRoomStore()
	room1 := NewRoom("room-1", "Test Room 1", "owner-1", 3)
	room2 := NewRoom("room-2", "Test Room 2", "owner-2", 3)

	store.Create(room1)
	store.Create(room2)

	rooms := store.List()
	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}
}

func TestRoomStore_ListByOwner(t *testing.T) {
	store := NewRoomStore()
	room1 := NewRoom("room-1", "Test Room 1", "owner-1", 3)
	room2 := NewRoom("room-2", "Test Room 2", "owner-1", 3)
	room3 := NewRoom("room-3", "Test Room 3", "owner-2", 3)

	store.Create(room1)
	store.Create(room2)
	store.Create(room3)

	rooms := store.ListByOwner("owner-1")
	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms for owner-1, got %d", len(rooms))
	}
}

func TestRoomStore_ListByParticipant(t *testing.T) {
	store := NewRoomStore()
	room1 := NewRoom("room-1", "Test Room 1", "owner-1", 3)
	room2 := NewRoom("room-2", "Test Room 2", "owner-2", 3)

	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}
	room1.AddParticipant(user, RoleParticipant)

	store.Create(room1)
	store.Create(room2)

	rooms := store.ListByParticipant("user-1")
	if len(rooms) != 1 {
		t.Errorf("Expected 1 room for user-1, got %d", len(rooms))
	}
}
