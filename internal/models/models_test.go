package models

import (
	"testing"
)

func TestNewRoom(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 5)

	if room.ID != "room-1" {
		t.Errorf("Expected ID 'room-1', got '%s'", room.ID)
	}
	if room.Name != "Test Room" {
		t.Errorf("Expected Name 'Test Room', got '%s'", room.Name)
	}
	if room.OwnerID != "owner-1" {
		t.Errorf("Expected OwnerID 'owner-1', got '%s'", room.OwnerID)
	}
	if room.VotesPerUser != 5 {
		t.Errorf("Expected VotesPerUser 5, got %d", room.VotesPerUser)
	}
	if room.Phase != PhaseTicketing {
		t.Errorf("Expected Phase TICKETING, got '%s'", room.Phase)
	}
}

func TestRoom_AddParticipant(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}

	room.AddParticipant(user, RoleParticipant, StatusApproved)

	p, ok := room.GetParticipant("user-1")
	if !ok {
		t.Error("Expected participant to be added")
	}
	if p.User.ID != "user-1" {
		t.Errorf("Expected user ID 'user-1', got '%s'", p.User.ID)
	}
	if p.Role != RoleParticipant {
		t.Errorf("Expected role 'participant', got '%s'", p.Role)
	}
}

func TestRoom_RemoveParticipant(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}

	room.AddParticipant(user, RoleParticipant, StatusApproved)
	room.RemoveParticipant("user-1")

	_, ok := room.GetParticipant("user-1")
	if ok {
		t.Error("Expected participant to be removed")
	}
}

func TestRoom_SetParticipantRole(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}

	room.AddParticipant(user, RoleParticipant, StatusApproved)
	result := room.SetParticipantRole("user-1", RoleModerator)

	if !result {
		t.Error("Expected SetParticipantRole to return true")
	}

	p, _ := room.GetParticipant("user-1")
	if p.Role != RoleModerator {
		t.Errorf("Expected role 'moderator', got '%s'", p.Role)
	}
}

func TestRoom_IsModeratorOrOwner(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	owner := User{ID: "owner-1", Email: "owner@example.com", Name: "Owner"}
	moderator := User{ID: "mod-1", Email: "mod@example.com", Name: "Moderator"}
	participant := User{ID: "user-1", Email: "user@example.com", Name: "User"}

	room.AddParticipant(owner, RoleOwner, StatusApproved)
	room.AddParticipant(moderator, RoleModerator, StatusApproved)
	room.AddParticipant(participant, RoleParticipant, StatusApproved)

	if !room.IsModeratorOrOwner("owner-1") {
		t.Error("Expected owner to be moderator or owner")
	}
	if !room.IsModeratorOrOwner("mod-1") {
		t.Error("Expected moderator to be moderator or owner")
	}
	if room.IsModeratorOrOwner("user-1") {
		t.Error("Expected participant not to be moderator or owner")
	}
}

func TestRoom_AddAndGetTicket(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	ticket := &Ticket{
		ID:       "ticket-1",
		Content:  "Test ticket",
		AuthorID: "user-1",
		VoterIDs: []string{},
	}

	room.AddTicket(ticket)

	got, ok := room.GetTicket("ticket-1")
	if !ok {
		t.Error("Expected ticket to be found")
	}
	if got.Content != "Test ticket" {
		t.Errorf("Expected content 'Test ticket', got '%s'", got.Content)
	}
}

func TestRoom_Vote(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}
	room.AddParticipant(user, RoleParticipant, StatusApproved)

	ticket := &Ticket{
		ID:       "ticket-1",
		Content:  "Test ticket",
		AuthorID: "owner-1",
		VoterIDs: []string{},
	}
	room.AddTicket(ticket)

	// First vote should succeed
	if !room.Vote("user-1", "ticket-1") {
		t.Error("Expected first vote to succeed")
	}

	got, _ := room.GetTicket("ticket-1")
	if got.Votes != 1 {
		t.Errorf("Expected 1 vote, got %d", got.Votes)
	}

	// Voting again on same ticket should fail
	if room.Vote("user-1", "ticket-1") {
		t.Error("Expected second vote on same ticket to fail")
	}

	// Vote on same ticket with same user shouldn't increase votes
	got, _ = room.GetTicket("ticket-1")
	if got.Votes != 1 {
		t.Errorf("Expected still 1 vote, got %d", got.Votes)
	}
}

func TestRoom_Unvote(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)
	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}
	room.AddParticipant(user, RoleParticipant, StatusApproved)

	ticket := &Ticket{
		ID:       "ticket-1",
		Content:  "Test ticket",
		AuthorID: "owner-1",
		VoterIDs: []string{},
	}
	room.AddTicket(ticket)

	room.Vote("user-1", "ticket-1")

	if !room.Unvote("user-1", "ticket-1") {
		t.Error("Expected unvote to succeed")
	}

	got, _ := room.GetTicket("ticket-1")
	if got.Votes != 0 {
		t.Errorf("Expected 0 votes after unvote, got %d", got.Votes)
	}
}

func TestRoom_VotesPerUserLimit(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 2)
	user := User{ID: "user-1", Email: "test@example.com", Name: "Test User"}
	room.AddParticipant(user, RoleParticipant, StatusApproved)

	ticket1 := &Ticket{ID: "ticket-1", Content: "Ticket 1", AuthorID: "owner-1", VoterIDs: []string{}}
	ticket2 := &Ticket{ID: "ticket-2", Content: "Ticket 2", AuthorID: "owner-1", VoterIDs: []string{}}
	ticket3 := &Ticket{ID: "ticket-3", Content: "Ticket 3", AuthorID: "owner-1", VoterIDs: []string{}}
	room.AddTicket(ticket1)
	room.AddTicket(ticket2)
	room.AddTicket(ticket3)

	room.Vote("user-1", "ticket-1")
	room.Vote("user-1", "ticket-2")

	// Third vote should fail (limit is 2)
	if room.Vote("user-1", "ticket-3") {
		t.Error("Expected third vote to fail due to limit")
	}
}

func TestRoom_SetPhase(t *testing.T) {
	room := NewRoom("room-1", "Test Room", "owner-1", 3)

	room.SetPhase(PhaseBrainstorm)
	if room.Phase != PhaseBrainstorm {
		t.Errorf("Expected phase BRAINSTORMING, got '%s'", room.Phase)
	}

	room.SetPhase(PhaseVoting)
	if room.Phase != PhaseVoting {
		t.Errorf("Expected phase VOTING, got '%s'", room.Phase)
	}
}
