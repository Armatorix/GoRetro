package models

import (
	"database/sql"
	"encoding/json"

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

// InitSchema initializes the database schema
func (s *RoomStore) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS rooms (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		owner_id VARCHAR(255) NOT NULL,
		phase VARCHAR(50) NOT NULL,
		votes_per_user INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS participants (
		room_id VARCHAR(255) NOT NULL,
		user_id VARCHAR(255) NOT NULL,
		user_email VARCHAR(255) NOT NULL,
		user_name VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL,
		votes_used INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (room_id, user_id),
		FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tickets (
		id VARCHAR(255) PRIMARY KEY,
		room_id VARCHAR(255) NOT NULL,
		content TEXT NOT NULL,
		author_id VARCHAR(255) NOT NULL,
		group_id VARCHAR(255),
		votes INTEGER NOT NULL DEFAULT 0,
		voter_ids JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS ticket_groups (
		id VARCHAR(255) PRIMARY KEY,
		room_id VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		ticket_ids JSONB NOT NULL DEFAULT '[]',
		FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS action_tickets (
		id VARCHAR(255) PRIMARY KEY,
		room_id VARCHAR(255) NOT NULL,
		content TEXT NOT NULL,
		assignee_id VARCHAR(255),
		ticket_id VARCHAR(255) NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_participants_room_id ON participants(room_id);
	CREATE INDEX IF NOT EXISTS idx_tickets_room_id ON tickets(room_id);
	CREATE INDEX IF NOT EXISTS idx_ticket_groups_room_id ON ticket_groups(room_id);
	CREATE INDEX IF NOT EXISTS idx_action_tickets_room_id ON action_tickets(room_id);
	CREATE INDEX IF NOT EXISTS idx_rooms_owner_id ON rooms(owner_id);
	`
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
		INSERT INTO rooms (id, name, owner_id, phase, votes_per_user, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, room.ID, room.Name, room.OwnerID, room.Phase, room.VotesPerUser, room.CreatedAt)
	if err != nil {
		return err
	}

	// Insert participants
	room.RLock()
	for _, participant := range room.Participants {
		_, err = tx.Exec(`
			INSERT INTO participants (room_id, user_id, user_email, user_name, role, votes_used)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, room.ID, participant.User.ID, participant.User.Email, participant.User.Name, participant.Role, participant.VotesUsed)
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
		Participants:  make(map[string]*Participant),
		Tickets:       make(map[string]*Ticket),
		TicketGroups:  make(map[string]*TicketGroup),
		ActionTickets: make(map[string]*ActionTicket),
	}

	// Get room data
	err := s.db.QueryRow(`
		SELECT id, name, owner_id, phase, votes_per_user, created_at
		FROM rooms WHERE id = $1
	`, id).Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.CreatedAt)
	if err != nil {
		return nil, false
	}

	// Get participants
	rows, err := s.db.Query(`
		SELECT user_id, user_email, user_name, role, votes_used
		FROM participants WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	for rows.Next() {
		var p Participant
		err := rows.Scan(&p.User.ID, &p.User.Email, &p.User.Name, &p.Role, &p.VotesUsed)
		if err != nil {
			return nil, false
		}
		room.Participants[p.User.ID] = &p
	}

	// Get tickets
	ticketRows, err := s.db.Query(`
		SELECT id, content, author_id, group_id, votes, voter_ids, created_at
		FROM tickets WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer ticketRows.Close()

	for ticketRows.Next() {
		var t Ticket
		var groupID sql.NullString
		var voterIDsJSON []byte
		err := ticketRows.Scan(&t.ID, &t.Content, &t.AuthorID, &groupID, &t.Votes, &voterIDsJSON, &t.CreatedAt)
		if err != nil {
			return nil, false
		}
		if groupID.Valid {
			t.GroupID = groupID.String
		}
		if err := json.Unmarshal(voterIDsJSON, &t.VoterIDs); err != nil {
			return nil, false
		}
		room.Tickets[t.ID] = &t
	}

	// Get ticket groups
	groupRows, err := s.db.Query(`
		SELECT id, name, ticket_ids
		FROM ticket_groups WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var tg TicketGroup
		var ticketIDsJSON []byte
		err := groupRows.Scan(&tg.ID, &tg.Name, &ticketIDsJSON)
		if err != nil {
			return nil, false
		}
		if err := json.Unmarshal(ticketIDsJSON, &tg.TicketIDs); err != nil {
			return nil, false
		}
		room.TicketGroups[tg.ID] = &tg
	}

	// Get action tickets
	actionRows, err := s.db.Query(`
		SELECT id, content, assignee_id, ticket_id, created_at
		FROM action_tickets WHERE room_id = $1
	`, id)
	if err != nil {
		return nil, false
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var at ActionTicket
		var assigneeID sql.NullString
		err := actionRows.Scan(&at.ID, &at.Content, &assigneeID, &at.TicketID, &at.CreatedAt)
		if err != nil {
			return nil, false
		}
		if assigneeID.Valid {
			at.AssigneeID = assigneeID.String
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
		SELECT id, name, owner_id, phase, votes_per_user, created_at
		FROM rooms
	`)
	if err != nil {
		return []*Room{}
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.CreatedAt)
		if err != nil {
			continue
		}
		// Initialize maps
		room.Participants = make(map[string]*Participant)
		room.Tickets = make(map[string]*Ticket)
		room.TicketGroups = make(map[string]*TicketGroup)
		room.ActionTickets = make(map[string]*ActionTicket)
		rooms = append(rooms, &room)
	}
	return rooms
}

// ListByOwner returns all rooms owned by a user
func (s *RoomStore) ListByOwner(ownerID string) []*Room {
	rows, err := s.db.Query(`
		SELECT id, name, owner_id, phase, votes_per_user, created_at
		FROM rooms WHERE owner_id = $1
	`, ownerID)
	if err != nil {
		return []*Room{}
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.CreatedAt)
		if err != nil {
			continue
		}
		// Initialize maps
		room.Participants = make(map[string]*Participant)
		room.Tickets = make(map[string]*Ticket)
		room.TicketGroups = make(map[string]*TicketGroup)
		room.ActionTickets = make(map[string]*ActionTicket)
		rooms = append(rooms, &room)
	}
	return rooms
}

// ListByParticipant returns all rooms where user is a participant
func (s *RoomStore) ListByParticipant(userID string) []*Room {
	rows, err := s.db.Query(`
		SELECT DISTINCT r.id, r.name, r.owner_id, r.phase, r.votes_per_user, r.created_at
		FROM rooms r
		INNER JOIN participants p ON r.id = p.room_id
		WHERE p.user_id = $1
	`, userID)
	if err != nil {
		return []*Room{}
	}
	defer rows.Close()

	rooms := make([]*Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.Phase, &room.VotesPerUser, &room.CreatedAt)
		if err != nil {
			continue
		}
		// Initialize maps
		room.Participants = make(map[string]*Participant)
		room.Tickets = make(map[string]*Ticket)
		room.TicketGroups = make(map[string]*TicketGroup)
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
		UPDATE rooms SET name = $1, owner_id = $2, phase = $3, votes_per_user = $4
		WHERE id = $5
	`, room.Name, room.OwnerID, room.Phase, room.VotesPerUser, room.ID)
	if err != nil {
		return err
	}

	// Delete existing participants, tickets, groups, and actions
	_, err = tx.Exec(`DELETE FROM participants WHERE room_id = $1`, room.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM tickets WHERE room_id = $1`, room.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM ticket_groups WHERE room_id = $1`, room.ID)
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
			INSERT INTO participants (room_id, user_id, user_email, user_name, role, votes_used)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, room.ID, participant.User.ID, participant.User.Email, participant.User.Name, participant.Role, participant.VotesUsed)
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
		var groupID *string
		if ticket.GroupID != "" {
			groupID = &ticket.GroupID
		}
		_, err = tx.Exec(`
			INSERT INTO tickets (id, room_id, content, author_id, group_id, votes, voter_ids, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, ticket.ID, room.ID, ticket.Content, ticket.AuthorID, groupID, ticket.Votes, voterIDsJSON, ticket.CreatedAt)
		if err != nil {
			room.RUnlock()
			return err
		}
	}

	// Insert ticket groups
	for _, group := range room.TicketGroups {
		ticketIDsJSON, err := json.Marshal(group.TicketIDs)
		if err != nil {
			room.RUnlock()
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO ticket_groups (id, room_id, name, ticket_ids)
			VALUES ($1, $2, $3, $4)
		`, group.ID, room.ID, group.Name, ticketIDsJSON)
		if err != nil {
			room.RUnlock()
			return err
		}
	}

	// Insert action tickets
	for _, action := range room.ActionTickets {
		var assigneeID *string
		if action.AssigneeID != "" {
			assigneeID = &action.AssigneeID
		}
		_, err = tx.Exec(`
			INSERT INTO action_tickets (id, room_id, content, assignee_id, ticket_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, action.ID, room.ID, action.Content, assigneeID, action.TicketID, action.CreatedAt)
		if err != nil {
			room.RUnlock()
			return err
		}
	}
	room.RUnlock()

	return tx.Commit()
}
