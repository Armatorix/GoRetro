package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/Armatorix/GoRetro/internal/models"
	"github.com/Armatorix/GoRetro/internal/websocket"
)

var upgrader = gorillaWS.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin allows all origins for development.
	// TODO: In production, this should validate against a list of allowed origins
	// or use the Origin header to check against the request's Host.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler contains all HTTP handlers
type Handler struct {
	store *models.RoomStore
	hub   *websocket.Hub
}

// NewHandler creates a new handler
func NewHandler(store *models.RoomStore, hub *websocket.Hub) *Handler {
	return &Handler{
		store: store,
		hub:   hub,
	}
}

// getUserFromRequest extracts user information from OAuth2-proxy headers
func getUserFromRequest(c echo.Context) models.User {
	email := c.Request().Header.Get("X-Forwarded-Email")
	if email == "" {
		email = c.Request().Header.Get("X-Auth-Request-Email")
	}

	name := c.Request().Header.Get("X-Forwarded-Preferred-Username")
	if name == "" {
		name = c.Request().Header.Get("X-Auth-Request-Preferred-Username")
	}

	userID := c.Request().Header.Get("X-Forwarded-User")
	if userID == "" {
		userID = c.Request().Header.Get("X-Auth-Request-User")
	}

	// Fallback for development without OAuth2-proxy
	if email == "" {
		email = "dev@example.com"
	}
	if userID == "" {
		userID = email
	}
	if name == "" {
		// If name is missing, try to use the part before @ in email, or userID
		name = userID
	}

	return models.User{
		ID:    userID,
		Email: email,
		Name:  name,
	}
}

// CreateRoomRequest is the request body for creating a room
type CreateRoomRequest struct {
	Name         string `json:"name" form:"name"`
	VotesPerUser int    `json:"votes_per_user" form:"votes_per_user"`
}

// RoomResponse is the response for room endpoints
type RoomResponse struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Phase        models.Phase `json:"phase"`
	VotesPerUser int          `json:"votes_per_user"`
	OwnerID      string       `json:"owner_id"`
	CreatedAt    time.Time    `json:"created_at"`
}

// Index renders the home page
func (h *Handler) Index(c echo.Context) error {
	user := getUserFromRequest(c)
	rooms := h.store.ListByParticipant(user.ID)
	return c.Render(http.StatusOK, "index.html", map[string]any{
		"User":  user,
		"Rooms": rooms,
	})
}

// CreateRoom creates a new room
func (h *Handler) CreateRoom(c echo.Context) error {
	user := getUserFromRequest(c)

	var req CreateRoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.Name == "" {
		req.Name = "Retrospective"
	}
	if req.VotesPerUser <= 0 {
		req.VotesPerUser = 3
	}

	roomID := uuid.New().String()
	room := models.NewRoom(roomID, req.Name, user.ID, req.VotesPerUser)
	room.AddParticipant(user, models.RoleOwner, models.StatusApproved)

	if err := h.store.Create(room); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create room"})
	}

	// Check if it's an AJAX request or form submission
	if c.Request().Header.Get("Accept") == "application/json" {
		return c.JSON(http.StatusCreated, RoomResponse{
			ID:           room.ID,
			Name:         room.Name,
			Phase:        room.Phase,
			VotesPerUser: room.VotesPerUser,
			OwnerID:      room.OwnerID,
			CreatedAt:    room.CreatedAt,
		})
	}

	return c.Redirect(http.StatusSeeOther, "/rooms/"+room.ID)
}

// ListRooms returns all rooms for the user
func (h *Handler) ListRooms(c echo.Context) error {
	user := getUserFromRequest(c)
	rooms := h.store.ListByParticipant(user.ID)

	response := make([]RoomResponse, 0, len(rooms))
	for _, room := range rooms {
		response = append(response, RoomResponse{
			ID:           room.ID,
			Name:         room.Name,
			Phase:        room.Phase,
			VotesPerUser: room.VotesPerUser,
			OwnerID:      room.OwnerID,
			CreatedAt:    room.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, response)
}

// GetRoom renders the room page
func (h *Handler) GetRoom(c echo.Context) error {
	roomID := c.Param("id")
	user := getUserFromRequest(c)

	room, ok := h.store.Get(roomID)
	if !ok {
		return c.String(http.StatusNotFound, "Room not found")
	}

	// Add user as pending participant if not already a participant or pending
	if _, exists := room.GetParticipant(user.ID); !exists {
		if _, pendingExists := room.GetPendingParticipant(user.ID); !pendingExists {
			room.AddParticipant(user, models.RoleParticipant, models.StatusPending)
			// Update room in store
			if err := h.store.Update(room); err != nil {
				return c.String(http.StatusInternalServerError, "Failed to update room")
			}
		}
	}

	return c.Render(http.StatusOK, "room.html", map[string]any{
		"User": user,
		"Room": room,
	})
}

// GetRoomAPI returns room details as JSON
func (h *Handler) GetRoomAPI(c echo.Context) error {
	roomID := c.Param("id")

	room, ok := h.store.Get(roomID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	return c.JSON(http.StatusOK, RoomResponse{
		ID:           room.ID,
		Name:         room.Name,
		Phase:        room.Phase,
		VotesPerUser: room.VotesPerUser,
		OwnerID:      room.OwnerID,
		CreatedAt:    room.CreatedAt,
	})
}

// DeleteRoom deletes a room
func (h *Handler) DeleteRoom(c echo.Context) error {
	roomID := c.Param("id")
	user := getUserFromRequest(c)

	room, ok := h.store.Get(roomID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}

	// Only owner can delete
	if room.OwnerID != user.ID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Only room owner can delete"})
	}

	if err := h.store.Delete(roomID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete room"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Room deleted"})
}

// WebSocket handles WebSocket connections
func (h *Handler) WebSocket(c echo.Context) error {
	roomID := c.Param("id")
	user := getUserFromRequest(c)

	room, ok := h.store.Get(roomID)
	if !ok {
		return c.String(http.StatusNotFound, "Room not found")
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := websocket.NewClient(user.ID, roomID, conn)
	h.hub.Register(client)

	// Check if user is approved participant
	if _, exists := room.GetParticipant(user.ID); exists {
		// User is approved - notify others and send room state
		h.hub.NotifyUserJoined(room, user)
		h.hub.SendRoomState(client, room)
	} else if pendingParticipant, pendingExists := room.GetPendingParticipant(user.ID); pendingExists {
		// User is pending - send room state but notify about pending status
		h.hub.SendRoomState(client, room)
		h.hub.NotifyParticipantPending(room, pendingParticipant)
	} else {
		// User is not yet added - add as pending
		room.AddParticipant(user, models.RoleParticipant, models.StatusPending)
		if err := h.store.Update(room); err != nil {
			return c.String(http.StatusInternalServerError, "Failed to update room")
		}
		pendingParticipant, _ := room.GetPendingParticipant(user.ID)
		h.hub.SendRoomState(client, room)
		h.hub.NotifyParticipantPending(room, pendingParticipant)
	}

	// Start goroutines for reading and writing
	go h.writePump(client)
	go h.readPump(client, room)

	return nil
}

func (h *Handler) writePump(client *websocket.Client) {
	defer func() {
		client.Conn.Close()
	}()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(gorillaWS.TextMessage, message); err != nil {
			return
		}
	}
}

func (h *Handler) readPump(client *websocket.Client, room *models.Room) {
	defer func() {
		h.hub.Unregister(client)
		h.hub.NotifyUserLeft(room, client.ID)
		client.Conn.Close()
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}

		h.hub.HandleMessage(client, message)
	}
}
