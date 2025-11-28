package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Armatorix/GoRetro/internal/chatcompletion"
	"github.com/Armatorix/GoRetro/internal/models"
	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Room ID -> Client ID -> Client
	rooms          map[string]map[string]*Client
	store          *models.RoomStore
	register       chan *Client
	unregister     chan *Client
	mu             sync.RWMutex
	redisPubSub    *RedisPubSub
	chatCompletion *chatcompletion.Service
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

// SetRedisPubSub sets the Redis pub/sub manager (optional for distributed mode)
func (h *Hub) SetRedisPubSub(redisPubSub *RedisPubSub) {
	h.redisPubSub = redisPubSub
}

// SetChatCompletion sets the chat completion service (optional for auto-merge feature)
func (h *Hub) SetChatCompletion(chatCompletion *chatcompletion.Service) {
	h.chatCompletion = chatCompletion
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

// broadcastToRoomLocal sends a message to all local clients in a room
func (h *Hub) broadcastToRoomLocal(roomID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	for _, client := range clients {
		client.SendMessage(msg)
	}
}

// BroadcastToRoom sends a message to all clients in a room (local + Redis)
func (h *Hub) BroadcastToRoom(roomID string, msg []byte) {
	// Broadcast locally
	h.broadcastToRoomLocal(roomID, msg)

	// Publish to Redis for other instances
	if h.redisPubSub != nil {
		if err := h.redisPubSub.PublishToRoom(roomID, msg); err != nil {
			log.Printf("Failed to publish to Redis: %v", err)
		}
	}
}

// broadcastToRoomExceptLocal sends a message to all local clients except one
func (h *Hub) broadcastToRoomExceptLocal(roomID, exceptClientID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	for id, client := range clients {
		if id != exceptClientID {
			client.SendMessage(msg)
		}
	}
}

// BroadcastToRoomExcept sends a message to all clients except one (local + Redis)
func (h *Hub) BroadcastToRoomExcept(roomID, exceptClientID string, msg []byte) {
	// Broadcast locally
	h.broadcastToRoomExceptLocal(roomID, exceptClientID, msg)

	// Publish to Redis for other instances
	if h.redisPubSub != nil {
		if err := h.redisPubSub.PublishToRoomExcept(roomID, exceptClientID, msg); err != nil {
			log.Printf("Failed to publish to Redis: %v", err)
		}
	}
}

// broadcastToApprovedParticipantsLocal sends a message only to approved local participants in a room
func (h *Hub) broadcastToApprovedParticipantsLocal(roomID string, msg []byte) {
	room, ok := h.store.Get(roomID)
	if !ok {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	for clientID, client := range clients {
		// Only send to approved participants
		if _, isApproved := room.GetParticipant(clientID); isApproved {
			client.SendMessage(msg)
		}
	}
}

// BroadcastToApprovedParticipants sends a message only to approved participants in a room (local + Redis)
func (h *Hub) BroadcastToApprovedParticipants(roomID string, msg []byte) {
	// Broadcast locally
	h.broadcastToApprovedParticipantsLocal(roomID, msg)

	// Publish to Redis for other instances
	if h.redisPubSub != nil {
		if err := h.redisPubSub.PublishToApprovedParticipants(roomID, msg); err != nil {
			log.Printf("Failed to publish to Redis: %v", err)
		}
	}
}

// sendToClientLocal sends a message to a specific local client
func (h *Hub) sendToClientLocal(roomID, clientID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[roomID]; ok {
		if client, ok := clients[clientID]; ok {
			client.SendMessage(msg)
		}
	}
}

// SendToClient sends a message to a specific client (local + Redis)
func (h *Hub) SendToClient(roomID, clientID string, msg []byte) {
	// Send locally
	h.sendToClientLocal(roomID, clientID, msg)

	// Publish to Redis for other instances
	if h.redisPubSub != nil {
		if err := h.redisPubSub.PublishToClient(roomID, clientID, msg); err != nil {
			log.Printf("Failed to publish to Redis: %v", err)
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

	// Check if user is approved (not pending) before allowing any actions
	_, isApproved := room.GetParticipant(client.ID)
	if !isApproved {
		h.sendError(client, "You must be approved to perform actions")
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
	case MsgApproveParticipant:
		h.handleApproveParticipant(client, room, message.Payload)
	case MsgRejectParticipant:
		h.handleRejectParticipant(client, room, message.Payload)
	case MsgSetAutoApprove:
		h.handleSetAutoApprove(client, room, message.Payload)
	case MsgAutoMergeTickets:
		h.handleAutoMergeTickets(client, room, message.Payload)
	case MsgAutoProposeActions:
		h.handleAutoProposeActions(client, room, message.Payload)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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
		models.PhaseMerging,
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
	h.BroadcastToApprovedParticipants(room.ID, responseBytes)
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

func (h *Hub) handleApproveParticipant(client *Client, room *models.Room, payload map[string]any) {
	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderator or owner can approve participants")
		return
	}

	userID, _ := payload["user_id"].(string)

	if !room.ApproveParticipant(userID) {
		h.sendError(client, "Participant not found in pending list")
		return
	}

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to approve participant")
		return
	}

	participant, _ := room.GetParticipant(userID)

	response := Message{
		Type: MsgParticipantApproved,
		Payload: map[string]any{
			"user_id":     userID,
			"participant": participant,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)

	// Send full room state to the newly approved participant
	h.SendToClient(room.ID, userID, func() []byte {
		room.RLock()
		defer room.RUnlock()
		stateMsg := Message{
			Type: MsgRoomState,
			Payload: map[string]any{
				"id":                   room.ID,
				"name":                 room.Name,
				"phase":                room.Phase,
				"votes_per_user":       room.VotesPerUser,
				"participants":         room.Participants,
				"pending_participants": room.PendingParticipants,
				"tickets":              room.Tickets,
				"action_tickets":       room.ActionTickets,
			},
		}
		bytes, _ := json.Marshal(stateMsg)
		return bytes
	}())
}

func (h *Hub) handleRejectParticipant(client *Client, room *models.Room, payload map[string]any) {
	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderator or owner can reject participants")
		return
	}

	userID, _ := payload["user_id"].(string)

	if !room.RejectParticipant(userID) {
		h.sendError(client, "Participant not found in pending list")
		return
	}

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to reject participant")
		return
	}

	response := Message{
		Type: MsgParticipantRejected,
		Payload: map[string]any{
			"user_id": userID,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleSetAutoApprove(client *Client, room *models.Room, payload map[string]any) {
	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderator or owner can change auto-approve setting")
		return
	}

	autoApprove, ok := payload["auto_approve"].(bool)
	if !ok {
		h.sendError(client, "Invalid auto_approve value")
		return
	}

	room.SetAutoApprove(autoApprove)

	// Persist to database
	if err := h.store.Update(room); err != nil {
		h.sendError(client, "Failed to update auto-approve setting")
		return
	}

	response := Message{
		Type: MsgAutoApproveChanged,
		Payload: map[string]any{
			"auto_approve": autoApprove,
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
			"id":                   room.ID,
			"name":                 room.Name,
			"phase":                room.Phase,
			"votes_per_user":       room.VotesPerUser,
			"auto_approve":         room.AutoApprove,
			"participants":         room.Participants,
			"pending_participants": room.PendingParticipants,
			"tickets":              room.Tickets,
			"action_tickets":       room.ActionTickets,
		},
	}
	responseBytes, _ := json.Marshal(response)
	client.SendMessage(responseBytes)
}

// SendPendingRoomState sends a limited room state to a pending participant
func (h *Hub) SendPendingRoomState(client *Client, room *models.Room) {
	room.RLock()
	defer room.RUnlock()

	response := Message{
		Type: MsgRoomState,
		Payload: map[string]any{
			"id":                   room.ID,
			"name":                 room.Name,
			"phase":                room.Phase,
			"votes_per_user":       room.VotesPerUser,
			"participants":         make(map[string]*models.Participant),
			"pending_participants": make(map[string]*models.Participant),
			"tickets":              make(map[string]*models.Ticket),
			"action_tickets":       make(map[string]*models.ActionTicket),
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

// NotifyParticipantPending notifies all clients in a room that a user is pending approval
func (h *Hub) NotifyParticipantPending(room *models.Room, participant *models.Participant) {
	response := Message{
		Type: MsgParticipantPending,
		Payload: map[string]any{
			"participant": participant,
		},
	}
	responseBytes, _ := json.Marshal(response)
	h.BroadcastToRoom(room.ID, responseBytes)
}

func (h *Hub) handleAutoMergeTickets(client *Client, room *models.Room, payload map[string]any) {
	// Only moderators/owners can trigger auto-merge
	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderators can trigger auto-merge")
		return
	}

	// Only available in DISCUSSION phase
	if room.Phase != models.PhaseMerging {
		h.sendError(client, "Auto-merge is only available during discussion phase")
		return
	}

	// Check if chat completion service is configured
	if h.chatCompletion == nil || !h.chatCompletion.IsConfigured() {
		h.sendError(client, "Chat completion service not configured")
		return
	}

	// Send progress message
	progressMsg := Message{
		Type: MsgAutoMergeProgress,
		Payload: map[string]any{
			"status": "analyzing",
		},
	}
	progressBytes, _ := json.Marshal(progressMsg)
	h.SendToClient(room.ID, client.ID, progressBytes)

	// Get all tickets
	room.RLock()
	tickets := make(map[string]*models.Ticket)
	for id, ticket := range room.Tickets {
		tickets[id] = ticket
	}
	room.RUnlock()

	// Call AI service to get merge suggestions
	mergeResponse, err := h.chatCompletion.SuggestMerges(tickets)
	if err != nil {
		log.Printf("Auto-merge failed: %v", err)
		h.sendError(client, fmt.Sprintf("Auto-merge failed: %v", err))
		return
	}

	// Apply the suggested merges
	mergesApplied := 0
	for _, group := range mergeResponse.MergeGroups {
		// Validate that parent ticket exists
		parentTicket, ok := room.GetTicket(group.ParentTicketID)
		if !ok {
			log.Printf("Parent ticket %s not found, skipping group", group.ParentTicketID)
			continue
		}

		// Skip if parent is already a child
		if parentTicket.DeduplicationTicketID != nil {
			log.Printf("Parent ticket %s is already a child, skipping group", group.ParentTicketID)
			continue
		}

		// Apply merges for this group
		for _, childID := range group.ChildTicketIDs {
			childTicket, ok := room.GetTicket(childID)
			if !ok {
				log.Printf("Child ticket %s not found, skipping", childID)
				continue
			}

			// Skip if already merged
			if childTicket.DeduplicationTicketID != nil {
				continue
			}

			// Skip if trying to merge with itself
			if childID == group.ParentTicketID {
				continue
			}

			// Merge the child into the parent by setting deduplication_ticket_id
			room.Lock()
			childTicket.DeduplicationTicketID = &group.ParentTicketID
			room.Unlock()
			mergesApplied++

			// Broadcast the ticket update
			response := Message{
				Type: MsgTicketUpdated,
				Payload: map[string]any{
					"ticket": childTicket,
				},
			}
			responseBytes, _ := json.Marshal(response)
			h.BroadcastToApprovedParticipants(room.ID, responseBytes)
		}
	}

	// Persist changes to database
	if err := h.store.Update(room); err != nil {
		log.Printf("Failed to save auto-merge changes: %v", err)
		h.sendError(client, "Failed to save changes")
		return
	}

	// Send completion message
	completeMsg := Message{
		Type: MsgAutoMergeComplete,
		Payload: map[string]any{
			"merges_applied": mergesApplied,
			"groups_count":   len(mergeResponse.MergeGroups),
		},
	}
	completeBytes, _ := json.Marshal(completeMsg)
	h.SendToClient(room.ID, client.ID, completeBytes)
}

func (h *Hub) handleAutoProposeActions(client *Client, room *models.Room, payload map[string]any) {
	// Only moderators/owners can trigger auto-propose
	if !room.IsModeratorOrOwner(client.ID) {
		h.sendError(client, "Only moderators can trigger auto-propose actions")
		return
	}

	// Only available in DISCUSSION phase
	if room.Phase != models.PhaseDiscussion {
		h.sendError(client, "Auto-propose actions is only available during summary phase")
		return
	}

	// Check if chat completion service is configured
	if h.chatCompletion == nil || !h.chatCompletion.IsConfigured() {
		h.sendError(client, "Chat completion service not configured")
		return
	}

	// Get team context from payload (optional)
	teamContext := ""
	if ctx, ok := payload["team_context"].(string); ok {
		teamContext = ctx
	}

	// Send progress message
	progressMsg := Message{
		Type: MsgAutoProposeProgress,
		Payload: map[string]any{
			"status": "analyzing",
		},
	}
	progressBytes, _ := json.Marshal(progressMsg)
	h.SendToClient(room.ID, client.ID, progressBytes)

	// Get all tickets
	room.RLock()
	tickets := make(map[string]*models.Ticket)
	for id, ticket := range room.Tickets {
		tickets[id] = ticket
	}
	room.RUnlock()

	// Call AI service to get action suggestions
	actionResponse, err := h.chatCompletion.ProposeActions(tickets, teamContext)
	if err != nil {
		log.Printf("Auto-propose actions failed: %v", err)
		h.sendError(client, fmt.Sprintf("Auto-propose actions failed: %v", err))
		return
	}

	// Create the suggested actions with robot icon prefix
	actionsCreated := 0
	for _, suggestion := range actionResponse.Actions {
		action := &models.ActionTicket{
			ID:          uuid.New().String(),
			Content:     "🤖 " + suggestion.Content,
			TicketID:    suggestion.TicketID,
			AssigneeIDs: []string{},
			CreatedAt:   time.Now(),
		}

		room.AddActionTicket(action)
		actionsCreated++

		// Broadcast the new action
		response := Message{
			Type: MsgActionAdded,
			Payload: map[string]any{
				"action": action,
			},
		}
		responseBytes, _ := json.Marshal(response)
		h.BroadcastToApprovedParticipants(room.ID, responseBytes)
	}

	// Persist changes to database
	if err := h.store.Update(room); err != nil {
		log.Printf("Failed to save auto-proposed actions: %v", err)
		h.sendError(client, "Failed to save actions")
		return
	}

	// Send completion message
	completeMsg := Message{
		Type: MsgAutoProposeComplete,
		Payload: map[string]any{
			"actions_created": actionsCreated,
		},
	}
	completeBytes, _ := json.Marshal(completeMsg)
	h.SendToClient(room.ID, client.ID, completeBytes)
}
