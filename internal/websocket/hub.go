package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Armatorix/GoRetro/internal/models"
	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Room ID -> Client ID -> Client
	rooms      map[string]map[string]*Client
	store      *models.RoomStore
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub
func NewHub(store *models.RoomStore) *Hub {
	return &Hub{
		rooms:      make(map[string]map[string]*Client),
		store:      store,
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.rooms[client.RoomID]; !ok {
				h.rooms[client.RoomID] = make(map[string]*Client)
			}
			h.rooms[client.RoomID][client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.RoomID]; ok {
				if _, ok := clients[client.ID]; ok {
					delete(clients, client.ID)
					if len(clients) == 0 {
						delete(h.rooms, client.RoomID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(roomID string, msg []byte) {
	h.mu.RLock()
	clients, ok := h.rooms[roomID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range clients {
		client.SendMessage(msg)
	}
}

// BroadcastToRoomExcept sends a message to all clients except one
func (h *Hub) BroadcastToRoomExcept(roomID, exceptClientID string, msg []byte) {
	h.mu.RLock()
	clients, ok := h.rooms[roomID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, client := range clients {
		if id != exceptClientID {
			client.SendMessage(msg)
		}
	}
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(roomID, clientID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[roomID]; ok {
		if client, ok := clients[clientID]; ok {
			client.SendMessage(msg)
		}
	}
}

// HandleMessage processes incoming WebSocket messages
func (h *Hub) HandleMessage(client *Client, msg []byte) {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		h.sendError(client, "Invalid message format")
		return
	}

	room, ok := h.store.Get(client.RoomID)
	if !ok {
		h.sendError(client, "Room not found")
		return
	}

	switch message.Type {
	case MsgAddTicket:
		h.handleAddTicket(client, room, message.Payload)
	case MsgEditTicket:
		h.handleEditTicket(client, room, message.Payload)
	case MsgDeleteTicket:
		h.handleDeleteTicket(client, room, message.Payload)
	case MsgVote:
		h.handleVote(client, room, message.Payload)
	case MsgUnvote:
		h.handleUnvote(client, room, message.Payload)
	case MsgAddAction:
		h.handleAddAction(client, room, message.Payload)
	case MsgDeleteAction:
		h.handleDeleteAction(client, room, message.Payload)
	case MsgMarkCovered:
		h.handleMarkCovered(client, room, message.Payload)
	case MsgSetPhase:
		h.handleSetPhase(client, room, message.Payload)
	case MsgSetRole:
		h.handleSetRole(client, room, message.Payload)
	case MsgRemoveUser:
		h.handleRemoveUser(client, room, message.Payload)
	default:
		h.sendError(client, "Unknown message type")
	}
}

func (h *Hub) handleAddTicket(client *Client, room *models.Room, payload map[string]any) {
	if room.Phase != models.PhaseTicketing {
		h.sendError(client, "Can only add tickets during ticketing phase")
		return
	}

	content, ok := payload["content"].(string)
	if !ok || content == "" {
		h.sendError(client, "Content is required")
		return
	}

	ticket := &models.Ticket{
		ID:        uuid.New().String(),
		Content:   content,
		AuthorID:  client.ID,
		Votes:     0,
		VoterIDs:  []string{},
		CreatedAt: time.Now(),
	}

	room.AddTicket(ticket)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to save ticket")
		return
	}

	response := Message{
		Type: MsgTicketAdded,
		Payload: map[string]any{
			"ticket": ticket,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleEditTicket(client *Client, room *models.Room, payload map[string]any) {
	ticketID, _ := payload["ticket_id"].(string)
	content, hasContent := payload["content"].(string)

	ticket, ok := room.GetTicket(ticketID)
	if !ok {
		h.sendError(client, "Ticket not found")
		return
	}

	// Only author or moderator can edit their ticket
	if ticket.AuthorID != client.ID && !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Not authorized to edit this ticket")
		return
	}

	room.Lock()

	// Update content if provided
	if hasContent {
		ticket.Content = content
	}

	// Update deduplication_ticket_id if provided in payload
	if deduplicationID, exists := payload["deduplication_ticket_id"]; exists {
		if deduplicationID == nil {
			// Remove deduplication
			ticket.DeduplicationTicketID = nil
		} else if dedupStr, ok := deduplicationID.(string); ok {
			// Set deduplication to parent ticket
			ticket.DeduplicationTicketID = &dedupStr
		}
	}

	room.Unlock()

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to update ticket")
		return
	}

	response := Message{
		Type: MsgTicketUpdated,
		Payload: map[string]any{
			"ticket": ticket,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleDeleteTicket(client *Client, room *models.Room, payload map[string]any) {
	ticketID, _ := payload["ticket_id"].(string)

	ticket, ok := room.GetTicket(ticketID)
	if !ok {
		h.sendError(client, "Ticket not found")
		return
	}

	// Only author or moderator can delete
	if ticket.AuthorID != client.ID && !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Not authorized to delete this ticket")
		return
	}

	room.RemoveTicket(ticketID)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to delete ticket")
		return
	}

	response := Message{
		Type: MsgTicketDeleted,
		Payload: map[string]any{
			"ticket_id": ticketID,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleVote(client *Client, room *models.Room, payload map[string]any) {
	if room.Phase != models.PhaseVoting {
		h.sendError(client, "Can only vote during voting phase")
		return
	}

	ticketID, _ := payload["ticket_id"].(string)

	if !room.Vote(client.ID, ticketID) {
		h.sendError(client, "Could not vote (no votes left or already voted)")
		return
	}

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to save vote")
		return
	}

	ticket, _ := room.GetTicket(ticketID)
	participant, _ := room.GetParticipant(client.ID)

	response := Message{
		Type: MsgVoteUpdated,
		Payload: map[string]any{
			"ticket_id":  ticketID,
			"votes":      ticket.Votes,
			"voter_ids":  ticket.VoterIDs,
			"user_id":    client.ID,
			"votes_used": participant.VotesUsed,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleUnvote(client *Client, room *models.Room, payload map[string]any) {
	if room.Phase != models.PhaseVoting {
		h.sendError(client, "Can only unvote during voting phase")
		return
	}

	ticketID, _ := payload["ticket_id"].(string)

	if !room.Unvote(client.ID, ticketID) {
		h.sendError(client, "Could not unvote")
		return
	}

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to save unvote")
		return
	}

	ticket, _ := room.GetTicket(ticketID)
	participant, _ := room.GetParticipant(client.ID)

	response := Message{
		Type: MsgVoteUpdated,
		Payload: map[string]any{
			"ticket_id":  ticketID,
			"votes":      ticket.Votes,
			"voter_ids":  ticket.VoterIDs,
			"user_id":    client.ID,
			"votes_used": participant.VotesUsed,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleAddAction(client *Client, room *models.Room, payload map[string]any) {
	if room.Phase != models.PhaseDiscussion {
		h.sendError(client, "Can only add actions during discussion phase")
		return
	}

	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderators can add actions")
		return
	}

	content, _ := payload["content"].(string)
	ticketID, _ := payload["ticket_id"].(string)

	// Handle assignee_ids as array
	var assigneeIDs []string
	if assigneeIDsRaw, ok := payload["assignee_ids"].([]interface{}); ok {
		for _, id := range assigneeIDsRaw {
			if idStr, ok := id.(string); ok {
				assigneeIDs = append(assigneeIDs, idStr)
			}
		}
	}

	action := &models.ActionTicket{
		ID:          uuid.New().String(),
		Content:     content,
		TicketID:    ticketID,
		AssigneeIDs: assigneeIDs,
		CreatedAt:   time.Now(),
	}

	room.AddActionTicket(action)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to save action")
		return
	}

	response := Message{
		Type: MsgActionAdded,
		Payload: map[string]any{
			"action": action,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleDeleteAction(client *Client, room *models.Room, payload map[string]any) {
	if room.Phase != models.PhaseDiscussion {
		h.sendError(client, "Can only delete actions during discussion phase")
		return
	}

	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderators can delete actions")
		return
	}

	actionID, ok := payload["action_id"].(string)
	if !ok || actionID == "" {
		h.sendError(client, "Action ID is required")
		return
	}

	// Check if action exists
	if _, exists := room.GetActionTicket(actionID); !exists {
		h.sendError(client, "Action not found")
		return
	}

	room.RemoveActionTicket(actionID)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to delete action")
		return
	}

	response := Message{
		Type: MsgActionDeleted,
		Payload: map[string]any{
			"action_id": actionID,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleMarkCovered(client *Client, room *models.Room, payload map[string]any) {
	if room.Phase != models.PhaseDiscussion && room.Phase != models.PhaseSummary {
		h.sendError(client, "Can only mark tickets as covered during discussion or summary phase")
		return
	}

	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderators can mark tickets as covered")
		return
	}

	ticketID, ok := payload["ticket_id"].(string)
	if !ok || ticketID == "" {
		h.sendError(client, "Ticket ID is required")
		return
	}

	covered, ok := payload["covered"].(bool)
	if !ok {
		h.sendError(client, "Covered status is required")
		return
	}

	ticket, exists := room.GetTicket(ticketID)
	if !exists {
		h.sendError(client, "Ticket not found")
		return
	}

	room.Lock()
	ticket.Covered = covered
	room.Unlock()

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to update ticket covered status")
		return
	}

	response := Message{
		Type: MsgTicketUpdated,
		Payload: map[string]any{
			"ticket": ticket,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleSetPhase(client *Client, room *models.Room, payload map[string]any) {
	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderators can change phase")
		return
	}

	phaseStr, _ := payload["phase"].(string)
	phase := models.Phase(phaseStr)

	// Validate phase transition
	validPhases := []models.Phase{
		models.PhaseTicketing,
		models.PhaseBrainstorm,
		models.PhaseVoting,
		models.PhaseDiscussion,
		models.PhaseSummary,
	}

	valid := false
	for _, p := range validPhases {
		if phase == p {
			valid = true
			break
		}
	}

	if !valid {
		h.sendError(client, "Invalid phase")
		return
	}

	room.SetPhase(phase)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to save phase change")
		return
	}

	response := Message{
		Type: MsgPhaseChanged,
		Payload: map[string]any{
			"phase": phase,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleSetRole(client *Client, room *models.Room, payload map[string]any) {
	if room.OwnerID != client.ID {
		h.sendError(client, "Only room owner can change roles")
		return
	}

	userID, _ := payload["user_id"].(string)
	roleStr, _ := payload["role"].(string)
	role := models.Role(roleStr)

	if role != models.RoleModerator && role != models.RoleParticipant {
		h.sendError(client, "Invalid role")
		return
	}

	if !room.SetParticipantRole(userID, role) {
		h.sendError(client, "User not found")
		return
	}

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to save role change")
		return
	}

	response := Message{
		Type: MsgRoleChanged,
		Payload: map[string]any{
			"user_id": userID,
			"role":    role,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleRemoveUser(client *Client, room *models.Room, payload map[string]any) {
	if room.OwnerID != client.ID && !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only owner or moderator can remove users")
		return
	}

	userID, _ := payload["user_id"].(string)

	// Cannot remove the owner
	if userID == room.OwnerID {
		h.sendError(client, "Cannot remove room owner")
		return
	}

	room.RemoveParticipant(userID)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to remove user")
		return
	}

	response := Message{
		Type: MsgUserRemoved,
		Payload: map[string]any{
			"user_id": userID,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) sendError(client *Client, message string) {
	response := Message{
		Type: MsgError,
		Payload: map[string]any{
			"message": message,
		},
	}
	responseBytes, _ := json.Marshal(response)
	client.SendMessage(responseBytes)
}

// SendRoomState sends the current room state to a client
func (h *Hub) SendRoomState(client *Client, room *models.Room) {
	room.RLock()
	defer room.RUnlock()

	response := Message{
		Type: MsgRoomState,
		Payload: map[string]any{
			"id":             room.ID,
			"name":           room.Name,
			"phase":          room.Phase,
			"votes_per_user": room.VotesPerUser,
			"participants":   room.Participants,
			"tickets":        room.Tickets,
			"action_tickets": room.ActionTickets,
		},
	}
	responseBytes, _ := json.Marshal(response)
	client.SendMessage(responseBytes)
}

// NotifyUserJoined notifies all clients in a room that a user joined
func (h *Hub) NotifyUserJoined(room *models.Room, user models.User) {
	response := Message{
		Type: MsgUserJoined,
		Payload: map[string]any{
			"user": user,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

// NotifyUserLeft notifies all clients in a room that a user left
func (h *Hub) NotifyUserLeft(room *models.Room, userID string) {
	response := Message{
		Type: MsgUserLeft,
		Payload: map[string]any{
			"user_id": userID,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}
