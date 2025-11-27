package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Client to server messages
	MsgJoinRoom     MessageType = "join_room"
	MsgLeaveRoom    MessageType = "leave_room"
	MsgAddTicket    MessageType = "add_ticket"
	MsgEditTicket   MessageType = "edit_ticket"
	MsgDeleteTicket MessageType = "delete_ticket"
	MsgCreateGroup  MessageType = "create_group"
	MsgMergeTickets MessageType = "merge_tickets"
	MsgVote         MessageType = "vote"
	MsgUnvote       MessageType = "unvote"
	MsgAddAction    MessageType = "add_action"
	MsgSetPhase     MessageType = "set_phase"
	MsgSetRole      MessageType = "set_role"
	MsgRemoveUser   MessageType = "remove_user"

	// Server to client messages
	MsgRoomState     MessageType = "room_state"
	MsgUserJoined    MessageType = "user_joined"
	MsgUserLeft      MessageType = "user_left"
	MsgTicketAdded   MessageType = "ticket_added"
	MsgTicketUpdated MessageType = "ticket_updated"
	MsgTicketDeleted MessageType = "ticket_deleted"
	MsgGroupCreated  MessageType = "group_created"
	MsgTicketsMerged MessageType = "tickets_merged"
	MsgVoteUpdated   MessageType = "vote_updated"
	MsgActionAdded   MessageType = "action_added"
	MsgPhaseChanged  MessageType = "phase_changed"
	MsgRoleChanged   MessageType = "role_changed"
	MsgUserRemoved   MessageType = "user_removed"
	MsgError         MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType    `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID     string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
	mu     sync.Mutex
}

// NewClient creates a new WebSocket client
func NewClient(id, roomID string, conn *websocket.Conn) *Client {
	return &Client{
		ID:     id,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case c.Send <- msg:
	default:
		// Channel full, message dropped
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	close(c.Send)
}
