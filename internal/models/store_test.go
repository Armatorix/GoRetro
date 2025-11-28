package models

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
// Note: This requires a running PostgreSQL instance for integration tests
// For unit tests, you might want to use a mock or in-memory SQLite
func setupTestDB(t *testing.T) *sql.DB {
	t.Skip("Skipping database tests - requires PostgreSQL instance")

	db, err := sql.Open("postgres", "postgres://goretro:goretro@localhost:5432/goretro_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

func TestRoomStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewRoomStore(db)
	if err := store.InitSchema(); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	room := NewRoom("room-1", "Test Room", "owner-1", 3)

	if err := store.Create(room); err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	got, ok := store.Get("room-1")
	if !ok {
		t.Error("Expected room to be found")
	}
	if got.ID != "room-1" {
		t.Errorf("Expected room ID 'room-1', got '%s'", got.ID)
	}
}

func TestRoomStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewRoomStore(db)
	if err := store.InitSchema(); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	room := NewRoom("room-1", "Test Room", "owner-1", 3)

	if err := store.Create(room); err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}
	if err := store.Delete("room-1"); err != nil {
		t.Fatalf("Failed to delete room: %v", err)
	}

	_, ok := store.Get("room-1")
	if ok {
		t.Error("Expected room to be deleted")
	}
}

func TestRoomStore_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewRoomStore(db)
	if err := store.InitSchema(); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	room1 := NewRoom("room-1", "Test Room 1", "owner-1", 3)
	room2 := NewRoom("room-2", "Test Room 2", "owner-2", 3)

	if err := store.Create(room1); err != nil {
		t.Fatalf("Failed to create room1: %v", err)
	}
	if err := store.Create(room2); err != nil {
		t.Fatalf("Failed to create room2: %v", err)
	}

	rooms := store.List()
	if len(rooms) < 2 {
		t.Errorf("Expected at least 2 rooms, got %d", len(rooms))
	}
}

func TestRoomStore_ListByOwner(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewRoomStore(db)
	if err := store.InitSchema(); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	room1 := NewRoom("room-1", "Test Room 1", "owner-1", 3)
	room2 := NewRoom("room-2", "Test Room 2", "owner-1", 3)
	room3 := NewRoom("room-3", "Test Room 3", "owner-2", 3)

	if err := store.Create(room1); err != nil {
		t.Fatalf("Failed to create room1: %v", err)
	}
	if err := store.Create(room2); err != nil {
		t.Fatalf("Failed to create room2: %v", err)
	}
	if err := store.Create(room3); err != nil {
		t.Fatalf("Failed to create room3: %v", err)
	}

	rooms := store.ListByOwner("owner-1")
	if len(rooms) < 2 {
		t.Errorf("Expected at least 2 rooms for owner-1, got %d", len(rooms))
	}
}

func TestRoomStore_ListByParticipant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewRoomStore(db)
	if err := store.InitSchema(); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	room1 := NewRoom("room-1", "Test Room 1", "owner-1", 3)
	room2 := NewRoom("room-2", "Test Room 2", "owner-2", 3)

	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}
	room1.AddParticipant(user, RoleParticipant, StatusApproved)

	if err := store.Create(room1); err != nil {
		t.Fatalf("Failed to create room1: %v", err)
	}
	if err := store.Create(room2); err != nil {
		t.Fatalf("Failed to create room2: %v", err)
	}

	rooms := store.ListByParticipant("user-1")
	if len(rooms) < 1 {
		t.Errorf("Expected at least 1 room for user-1, got %d", len(rooms))
	}
}
