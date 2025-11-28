package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

// RedisPubSub handles Redis pub/sub for distributing messages across multiple instances
type RedisPubSub struct {
	client        *redis.Client
	ctx           context.Context
	cancel        context.CancelFunc
	hub           *Hub
	channelPrefix string
}

// RedisMessage wraps a message with room context for Redis pub/sub
type RedisMessage struct {
	RoomID           string `json:"room_id"`
	Message          []byte `json:"message"`
	ExceptClientID   string `json:"except_client_id,omitempty"`
	SpecificClientID string `json:"specific_client_id,omitempty"`
	ApprovedOnly     bool   `json:"approved_only"`
}

// NewRedisPubSub creates a new Redis pub/sub manager
func NewRedisPubSub(client *redis.Client, hub *Hub) *RedisPubSub {
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisPubSub{
		client:        client,
		ctx:           ctx,
		cancel:        cancel,
		hub:           hub,
		channelPrefix: "goretro:broadcast:",
	}
}

// Start begins listening for Redis pub/sub messages
func (r *RedisPubSub) Start() {
	// Subscribe to all room channels using pattern
	pubsub := r.client.PSubscribe(r.ctx, r.channelPrefix+"*")
	defer pubsub.Close()

	log.Println("Redis pub/sub started, listening for broadcast messages")

	ch := pubsub.Channel()
	for {
		select {
		case <-r.ctx.Done():
			log.Println("Redis pub/sub stopped")
			return
		case msg := <-ch:
			r.handleRedisMessage(msg)
		}
	}
}

// Stop stops the Redis pub/sub listener
func (r *RedisPubSub) Stop() {
	r.cancel()
}

// handleRedisMessage processes incoming Redis messages and broadcasts them locally
func (r *RedisPubSub) handleRedisMessage(msg *redis.Message) {
	var redisMsg RedisMessage
	if err := json.Unmarshal([]byte(msg.Payload), &redisMsg); err != nil {
		log.Printf("Failed to unmarshal Redis message: %v", err)
		return
	}

	// Broadcast to local clients based on the message type
	if redisMsg.SpecificClientID != "" {
		// Send to specific client
		r.hub.sendToClientLocal(redisMsg.RoomID, redisMsg.SpecificClientID, redisMsg.Message)
	} else if redisMsg.ExceptClientID != "" {
		// Broadcast to all except one
		r.hub.broadcastToRoomExceptLocal(redisMsg.RoomID, redisMsg.ExceptClientID, redisMsg.Message)
	} else if redisMsg.ApprovedOnly {
		// Broadcast to approved participants only
		r.hub.broadcastToApprovedParticipantsLocal(redisMsg.RoomID, redisMsg.Message)
	} else {
		// Broadcast to all in room
		r.hub.broadcastToRoomLocal(redisMsg.RoomID, redisMsg.Message)
	}
}

// PublishToRoom publishes a message to Redis for distribution across instances
func (r *RedisPubSub) PublishToRoom(roomID string, msg []byte) error {
	redisMsg := RedisMessage{
		RoomID:  roomID,
		Message: msg,
	}
	return r.publish(roomID, redisMsg)
}

// PublishToRoomExcept publishes a message to Redis excluding one client
func (r *RedisPubSub) PublishToRoomExcept(roomID, exceptClientID string, msg []byte) error {
	redisMsg := RedisMessage{
		RoomID:         roomID,
		Message:        msg,
		ExceptClientID: exceptClientID,
	}
	return r.publish(roomID, redisMsg)
}

// PublishToApprovedParticipants publishes a message to Redis for approved participants only
func (r *RedisPubSub) PublishToApprovedParticipants(roomID string, msg []byte) error {
	redisMsg := RedisMessage{
		RoomID:       roomID,
		Message:      msg,
		ApprovedOnly: true,
	}
	return r.publish(roomID, redisMsg)
}

// PublishToClient publishes a message to Redis for a specific client
func (r *RedisPubSub) PublishToClient(roomID, clientID string, msg []byte) error {
	redisMsg := RedisMessage{
		RoomID:           roomID,
		Message:          msg,
		SpecificClientID: clientID,
	}
	return r.publish(roomID, redisMsg)
}

// publish sends a message to Redis
func (r *RedisPubSub) publish(roomID string, redisMsg RedisMessage) error {
	payload, err := json.Marshal(redisMsg)
	if err != nil {
		return err
	}

	channel := r.channelPrefix + roomID
	return r.client.Publish(r.ctx, channel, payload).Err()
}
