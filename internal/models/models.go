package models

import (
	"sync"
	"time"
)

// Phase represents the current phase of a retrospective room
type Phase string

const (
	PhaseTicketing  Phase = "TICKETING"
	PhaseMerging    Phase = "MERGING"
	PhaseVoting     Phase = "VOTING"
	PhaseDiscussion Phase = "DISCUSSION"
	PhaseSummary    Phase = "SUMMARY"
)

// Role represents a user's role in a room
type Role string

const (
	RoleOwner       Role = "owner"
	RoleModerator   Role = "moderator"
	RoleParticipant Role = "participant"
)

// ParticipantStatus represents the approval status of a participant
type ParticipantStatus string

const (
	StatusPending  ParticipantStatus = "pending"
	StatusApproved ParticipantStatus = "approved"
)

// User represents a participant in the retrospective
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Participant represents a user's participation in a room
type Participant struct {
	User      User              `json:"user"`
	Role      Role              `json:"role"`
	Status    ParticipantStatus `json:"status"`
	VotesUsed int               `json:"votes_used"`
}

// Ticket represents a retrospective item
type Ticket struct {
	ID                    string    `json:"id"`
	Content               string    `json:"content"`
	AuthorID              string    `json:"author_id"`
	DeduplicationTicketID *string   `json:"deduplication_ticket_id,omitempty"`
	Votes                 int       `json:"votes"`
	VoterIDs              []string  `json:"voter_ids"`
	Covered               bool      `json:"covered"`
	CreatedAt             time.Time `json:"created_at"`
}

// ActionTicket represents an action item from the discussion phase
type ActionTicket struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	AssigneeIDs []string  `json:"assignee_ids,omitempty"`
	TicketID    string    `json:"ticket_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Room represents a retrospective room
type Room struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	OwnerID             string                   `json:"owner_id"`
	Phase               Phase                    `json:"phase"`
	VotesPerUser        int                      `json:"votes_per_user"`
	AutoApprove         bool                     `json:"auto_approve"`
	Participants        map[string]*Participant  `json:"participants"`
	PendingParticipants map[string]*Participant  `json:"pending_participants"`
	Tickets             map[string]*Ticket       `json:"tickets"`
	ActionTickets       map[string]*ActionTicket `json:"action_tickets"`
	CreatedAt           time.Time                `json:"created_at"`
	mu                  sync.RWMutex
}

// NewRoom creates a new room with the given settings
func NewRoom(id, name, ownerID string, votesPerUser int) *Room {
	return &Room{
		ID:                  id,
		Name:                name,
		OwnerID:             ownerID,
		Phase:               PhaseTicketing,
		VotesPerUser:        votesPerUser,
		AutoApprove:         false,
		Participants:        make(map[string]*Participant),
		PendingParticipants: make(map[string]*Participant),
		Tickets:             make(map[string]*Ticket),
		ActionTickets:       make(map[string]*ActionTicket),
		CreatedAt:           time.Now(),
	}
}

// Lock acquires write lock
func (r *Room) Lock() {
	r.mu.Lock()
}

// Unlock releases write lock
func (r *Room) Unlock() {
	r.mu.Unlock()
}

// RLock acquires read lock
func (r *Room) RLock() {
	r.mu.RLock()
}

// RUnlock releases read lock
func (r *Room) RUnlock() {
	r.mu.RUnlock()
}

// AddParticipant adds a user to the room
func (r *Room) AddParticipant(user User, role Role, status ParticipantStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	participant := &Participant{
		User:      user,
		Role:      role,
		Status:    status,
		VotesUsed: 0,
	}
	if status == StatusPending {
		r.PendingParticipants[user.ID] = participant
	} else {
		r.Participants[user.ID] = participant
	}
}

// RemoveParticipant removes a user from the room
func (r *Room) RemoveParticipant(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Participants, userID)
	delete(r.PendingParticipants, userID)
}

// ApproveParticipant moves a pending participant to approved participants
func (r *Room) ApproveParticipant(userID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.PendingParticipants[userID]; ok {
		p.Status = StatusApproved
		r.Participants[userID] = p
		delete(r.PendingParticipants, userID)
		return true
	}
	return false
}

// RejectParticipant removes a pending participant from the room
func (r *Room) RejectParticipant(userID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.PendingParticipants[userID]; ok {
		delete(r.PendingParticipants, userID)
		return true
	}
	return false
}

// GetPendingParticipant returns a pending participant by ID
func (r *Room) GetPendingParticipant(userID string) (*Participant, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.PendingParticipants[userID]
	return p, ok
}

// GetParticipant returns a participant by ID
func (r *Room) GetParticipant(userID string) (*Participant, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.Participants[userID]
	return p, ok
}

// SetParticipantRole changes a participant's role
func (r *Room) SetParticipantRole(userID string, role Role) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.Participants[userID]; ok {
		p.Role = role
		return true
	}
	return false
}

// IsModeratorOrOwner checks if a user has moderator privileges
func (r *Room) IsModeratorOrOwner(userID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.Participants[userID]; ok {
		return p.Role == RoleOwner || p.Role == RoleModerator
	}
	return false
}

// AddTicket adds a new ticket to the room
func (r *Room) AddTicket(ticket *Ticket) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Tickets[ticket.ID] = ticket
}

// RemoveTicket removes a ticket from the room
func (r *Room) RemoveTicket(ticketID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Tickets, ticketID)
}

// GetTicket returns a ticket by ID
func (r *Room) GetTicket(ticketID string) (*Ticket, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.Tickets[ticketID]
	return t, ok
}

// AddActionTicket adds an action item
func (r *Room) AddActionTicket(action *ActionTicket) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ActionTickets[action.ID] = action
}

// RemoveActionTicket removes an action item from the room
func (r *Room) RemoveActionTicket(actionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.ActionTickets, actionID)
}

// GetActionTicket returns an action ticket by ID
func (r *Room) GetActionTicket(actionID string) (*ActionTicket, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.ActionTickets[actionID]
	return a, ok
}

// SetPhase changes the room's phase
func (r *Room) SetPhase(phase Phase) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Phase = phase
}

// Vote adds a vote to a ticket
func (r *Room) Vote(userID, ticketID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, pok := r.Participants[userID]
	t, tok := r.Tickets[ticketID]

	if !pok || !tok {
		return false
	}

	if p.VotesUsed >= r.VotesPerUser {
		return false
	}

	// Check if user already voted on this ticket
	for _, vid := range t.VoterIDs {
		if vid == userID {
			return false
		}
	}

	t.Votes++
	t.VoterIDs = append(t.VoterIDs, userID)
	p.VotesUsed++
	return true
}

// Unvote removes a vote from a ticket
func (r *Room) Unvote(userID, ticketID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, pok := r.Participants[userID]
	t, tok := r.Tickets[ticketID]

	if !pok || !tok {
		return false
	}

	// Find and remove user's vote
	for i, vid := range t.VoterIDs {
		if vid == userID {
			t.VoterIDs = append(t.VoterIDs[:i], t.VoterIDs[i+1:]...)
			t.Votes--
			p.VotesUsed--
			return true
		}
	}
	return false
}

// SetAutoApprove sets the auto-approve setting for the room
func (r *Room) SetAutoApprove(autoApprove bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.AutoApprove = autoApprove
}
