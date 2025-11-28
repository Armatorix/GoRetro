package models

import (
	"database/sql"
	"encoding/json"

	_ "embed"

	_ "github.com/lib/pq"
)

// RoomStore is a PostgreSQL store for rooms
type RoomStore struct {
	db *sql.DB
}

// NewRoomStore creates a new room store
func NewRoomStore(db *sql.DB) *RoomStore {
	return &RoomStore{
		db: db,
	}
}

//go:embed schema.sql
var schema string

// InitSchema initializes the database schema
func (s *RoomStore) InitSchema() error {
	_, err := s.db.Exec(schema)
	return err
}

// Create adds a new room to the store
func (s *RoomStore) Create(room *Room) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert room
	_, err = tx.Exec(`
		INSERT INTO rooms (id, name, owner_id, phase, votes_per_user, auto_approve, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, room.ID, room.Name, room.OwnerID, room.Phase, room.VotesPerUser, room.AutoApprove, room.CreatedAt)
	if err != nil {
		return err
	}

	// Insert participants
	room.RLock()
	for _, participant := range room.Participants {
		_, err = tx.Exec(`
			INSERT INTO participants (room_id, user_id, user_email, user_name, role, status, votes_used)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, room.ID, participant.User.ID, participant.User.Email, participant.User.Name, participant.Role, participant.Status, participant.VotesUsed)
		if err != nil {
			room.RUnlock()
			return err
		}
	}
	// Insert pending participants
	for _, participant := range room.PendingParticipants {
		_, err = tx.Exec(`
			INSERT INTO participants (room_id, user_id, user_email, user_name, role, status, votes_used)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, room.ID, participant.User.ID, participant.User.Email, participant.User.Name, participant.Role, participant.Status, participant.VotesUsed)
		if err != nil {
			room.RUnlock()
			return err
		}
	}
	room.RUnlock()

	return tx.Commit()
}

// Get retrieves a room by ID
func (s *RoomStore) Get(id string) (*Room, bool) {
	room := &Room{
		Participants:        make(map[string]*Participant),
		PendingParticipants: make(map[string]*Participant),
		Tickets:             make(map[string]*Ticket),
		ActionTickets:       make(map[string]*ActionTicket),
	}

	// Get room data
	err := s.db.QueryRow(`
		SELECT id, name, owner_id, phase, votes_per_user, auto_approve, created_at
		FROM rooms WHERE id = $1
	`, id).Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.AutoApprove, &room.CreatedAt)
	if err != nil {
		return nil, false
	}

	// Get participants
	rows, err := s.db.Query(`
		SELECT user_id, user_email, user_name, role, status, votes_used
		FROM participants WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	for rows.Next() {
		var p Participant
		err := rows.Scan(&p.User.ID, &p.User.Email, &p.User.Name, &p.Role, &p.Status, &p.VotesUsed)
		if err != nil {
			return nil, false
		}
		if p.Status == StatusPending {
			room.PendingParticipants[p.User.ID] = &p
		} else {
			room.Participants[p.User.ID] = &p
		}
	}

	// Get tickets
	ticketRows, err := s.db.Query(`
		SELECT id, content, author_id, deduplication_ticket_id, votes, voter_ids, covered, created_at
		FROM tickets WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer ticketRows.Close()

	for ticketRows.Next() {
		var t Ticket
		var deduplicationTicketID sql.NullString
		var voterIDsJSON []byte
		err := ticketRows.Scan(&t.ID, &t.Content, &t.AuthorID, &deduplicationTicketID, &t.Votes, &voterIDsJSON, &t.Covered, &t.CreatedAt)
		if err != nil {
			return nil, false
		}
		if deduplicationTicketID.Valid {
			t.DeduplicationTicketID = &deduplicationTicketID.String
		}
		if err := json.Unmarshal(voterIDsJSON, &t.VoterIDs); err != nil {
			return nil, false
		}
		room.Tickets[t.ID] = &t
	}

	// Get action tickets
	actionRows, err := s.db.Query(`
		SELECT id, content, assignee_ids, ticket_id, created_at
		FROM action_tickets WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var at ActionTicket
		var assigneeIDsJSON []byte
		err := actionRows.Scan(&at.ID, &at.Content, &assigneeIDsJSON, &at.TicketID, &at.CreatedAt)
		if err != nil {
			return nil, false
		}
		if err := json.Unmarshal(assigneeIDsJSON, &at.AssigneeIDs); err != nil {
			return nil, false
		}
		room.ActionTickets[at.ID] = &at
	}

	return room, true
}

// Delete removes a room from the store
func (s *RoomStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM rooms WHERE id = $1`, id)
	return err
}

// List returns all rooms
func (s *RoomStore) List() []*Room {
	rows, err := s.db.Query(`
		SELECT id, name, owner_id, phase, votes_per_user, auto_approve, created_at
		FROM rooms
	`)
	if err != nil {
		return []*Room{}
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.AutoApprove, &room.CreatedAt)
		if err != nil {
			continue
		}
		// Initialize maps
		room.Participants = make(map[string]*Participant)
		room.PendingParticipants = make(map[string]*Participant)
		room.Tickets = make(map[string]*Ticket)
		room.ActionTickets = make(map[string]*ActionTicket)
		rooms = append(rooms, &room)
	}
	return rooms
}

// ListByOwner returns all rooms owned by a user
func (s *RoomStore) ListByOwner(ownerID string) []*Room {
	rows, err := s.db.Query(`
		SELECT id, name, owner_id, phase, votes_per_user, auto_approve, created_at
		FROM rooms WHERE owner_id = $1
	`, ownerID)
	if err != nil {
		return []*Room{}
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.AutoApprove, &room.CreatedAt)
		if err != nil {
			continue
		}
		// Initialize maps
		room.Participants = make(map[string]*Participant)
		room.PendingParticipants = make(map[string]*Participant)
		room.Tickets = make(map[string]*Ticket)
		room.ActionTickets = make(map[string]*ActionTicket)
		rooms = append(rooms, &room)
	}
	return rooms
}

// ListByParticipant returns all rooms where user is a participant
func (s *RoomStore) ListByParticipant(userID string) []*Room {
	rows, err := s.db.Query(`
		SELECT DISTINCT r.id, r.name, r.owner_id, r.phase, r.votes_per_user, r.auto_approve, r.created_at
		FROM rooms r
		INNER JOIN participants p ON r.id = p.room_id
		WHERE p.user_id = $1 AND p.status = 'approved'
	`, userID)
	if err != nil {
		return []*Room{}
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.AutoApprove, &room.CreatedAt)
		if err != nil {
			continue
		}
		// Initialize maps
		room.Participants = make(map[string]*Participant)
		room.PendingParticipants = make(map[string]*Participant)
		room.Tickets = make(map[string]*Ticket)
		room.ActionTickets = make(map[string]*ActionTicket)
		rooms = append(rooms, &room)
	}
	return rooms
}

// Update updates a room in the database
func (s *RoomStore) Update(room *Room) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update room
	_, err = tx.Exec(`
		UPDATE rooms SET name = $1, owner_id = $2, phase = $3, votes_per_user = $4, auto_approve = $5
		WHERE id = $6
	`, room.Name, room.OwnerID, room.Phase, room.VotesPerUser, room.AutoApprove, room.ID)
	if err != nil {
		return err
	}

	// Delete existing participants, tickets, and actions
	_, err = tx.Exec(`DELETE FROM participants WHERE room_id = $1`, room.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM tickets WHERE room_id = $1`, room.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM action_tickets WHERE room_id = $1`, room.ID)
	if err != nil {
		return err
	}

	// Insert participants
	room.RLock()
	for _, participant := range room.Participants {
		_, err = tx.Exec(`
			INSERT INTO participants (room_id, user_id, user_email, user_name, role, status, votes_used)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, room.ID, participant.User.ID, participant.User.Email, participant.User.Name, participant.Role, participant.Status, participant.VotesUsed)
		if err != nil {
			room.RUnlock()
			return err
		}
	}

	// Insert pending participants
	for _, participant := range room.PendingParticipants {
		_, err = tx.Exec(`
			INSERT INTO participants (room_id, user_id, user_email, user_name, role, status, votes_used)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, room.ID, participant.User.ID, participant.User.Email, participant.User.Name, participant.Role, participant.Status, participant.VotesUsed)
		if err != nil {
			room.RUnlock()
			return err
		}
	}

	// Insert tickets
	for _, ticket := range room.Tickets {
		voterIDsJSON, err := json.Marshal(ticket.VoterIDs)
		if err != nil {
			room.RUnlock()
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO tickets (id, room_id, content, author_id, deduplication_ticket_id, votes, voter_ids, covered, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, ticket.ID, room.ID, ticket.Content, ticket.AuthorID, ticket.DeduplicationTicketID, ticket.Votes, voterIDsJSON, ticket.Covered, ticket.CreatedAt)
		if err != nil {
			room.RUnlock()
			return err
		}
	}

	// Insert action tickets
	for _, action := range room.ActionTickets {
		assigneeIDsJSON, err := json.Marshal(action.AssigneeIDs)
		if err != nil {
			room.RUnlock()
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO action_tickets (id, room_id, content, assignee_ids, ticket_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, action.ID, room.ID, action.Content, assigneeIDsJSON, action.TicketID, action.CreatedAt)
		if err != nil {
			room.RUnlock()
			return err
		}
	}
	room.RUnlock()

	return tx.Commit()
}
