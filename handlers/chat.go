package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/an1shka/go-project-playerook-kz/models"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*websocket.Conn]struct{})}
}

func (hub *Hub) add(chatID string, conn *websocket.Conn) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.clients[chatID] == nil {
		hub.clients[chatID] = make(map[*websocket.Conn]struct{})
	}
	hub.clients[chatID][conn] = struct{}{}
}

func (hub *Hub) remove(chatID string, conn *websocket.Conn) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	delete(hub.clients[chatID], conn)
	if len(hub.clients[chatID]) == 0 {
		delete(hub.clients, chatID)
	}
}

func (hub *Hub) broadcast(chatID string, data []byte) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for conn := range hub.clients[chatID] {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			delete(hub.clients[chatID], conn)
			_ = conn.Close()
		}
	}
	if len(hub.clients[chatID]) == 0 {
		delete(hub.clients, chatID)
	}
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host == r.Host
	},
}

type outgoingMessage struct {
	RecipientUsername string `json:"recipient_username"`
	Text              string `json:"text"`
}

func (h *Handler) Chat(c *gin.Context) {
	if c.Query("channel") == "news" {
		h.newsChannel(c)
		return
	}
	recipientUsername := strings.ToLower(strings.TrimSpace(c.Query("with")))
	userID := currentUserID(c)
	if !validUsername(recipientUsername) {
		respondError(c, http.StatusBadRequest, "query parameter 'with' must be a valid username")
		return
	}
	var recipientID, senderUsername string
	if err := h.Postgres.QueryRow(c.Request.Context(),
		`SELECT id, username FROM users WHERE id = $1`, userID).Scan(&userID, &senderUsername); err != nil {
		respondError(c, http.StatusInternalServerError, "could not load your account")
		return
	}
	if err := h.Postgres.QueryRow(c.Request.Context(),
		`SELECT id FROM users WHERE username = $1`, recipientUsername).Scan(&recipientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(c, http.StatusNotFound, "chat recipient not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "could not find chat recipient")
		return
	}
	if recipientID == userID {
		respondError(c, http.StatusBadRequest, "you cannot start a chat with yourself")
		return
	}
	chatID := conversationID(userID, recipientID)
	conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	h.Hub.add(chatID, conn)
	defer func() {
		h.Hub.remove(chatID, conn)
		_ = conn.Close()
	}()

	ctx := c.Request.Context()
	cursor, err := h.Mongo.Collection("messages").Find(ctx, bson.M{"chat_id": chatID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50))
	if err != nil {
		_ = conn.WriteJSON(gin.H{"error": "could not load chat history"})
		return
	}
	messages := make([]models.Message, 0)
	if err := cursor.All(ctx, &messages); err != nil {
		_ = conn.WriteJSON(gin.H{"error": "could not read chat history"})
		return
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	_ = conn.WriteJSON(gin.H{"type": "history", "items": messages})

	for {
		var input outgoingMessage
		if err := conn.ReadJSON(&input); err != nil {
			return
		}
		input.Text = strings.TrimSpace(input.Text)
		input.RecipientUsername = strings.ToLower(strings.TrimSpace(input.RecipientUsername))
		if input.RecipientUsername != recipientUsername || input.Text == "" || len(input.Text) > 4000 {
			_ = conn.WriteJSON(gin.H{"type": "error", "error": "recipient_username must match the chat and text must be 1-4000 characters"})
			continue
		}
		message := models.Message{
			ChatID: chatID, SenderID: userID, SenderUsername: senderUsername,
			RecipientID: recipientID, RecipientUsername: recipientUsername,
			Text: input.Text, CreatedAt: time.Now().UTC(),
		}
		result, err := h.Mongo.Collection("messages").InsertOne(ctx, message)
		if err != nil {
			_ = conn.WriteJSON(gin.H{"type": "error", "error": "could not save message"})
			continue
		}
		objectID, ok := result.InsertedID.(primitive.ObjectID)
		if !ok {
			_ = conn.WriteJSON(gin.H{"type": "error", "error": "saved message has an invalid ID"})
			continue
		}
		message.ID = objectID
		payload, err := json.Marshal(gin.H{"type": "message", "item": message})
		if err != nil {
			_ = conn.WriteJSON(gin.H{"type": "error", "error": "could not encode message"})
			continue
		}
		h.Hub.broadcast(chatID, payload)
	}
}

func (h *Handler) newsChannel(c *gin.Context) {
	conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	h.Hub.add("news", conn)
	defer func() {
		h.Hub.remove("news", conn)
		_ = conn.Close()
	}()

	cursor, err := h.Mongo.Collection("news").Find(c.Request.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50))
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "error": "could not load news"})
		return
	}
	var news []models.News
	if err := cursor.All(c.Request.Context(), &news); err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "error": "could not read news"})
		return
	}
	if err := conn.WriteJSON(gin.H{"type": "history", "items": news}); err != nil {
		return
	}
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func conversationID(first, second string) string {
	users := []string{first, second}
	sort.Strings(users)
	return users[0] + ":" + users[1]
}

func (h *Handler) ListNews(c *gin.Context) {
	cursor, err := h.Mongo.Collection("news").Find(c.Request.Context(), bson.M{},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not load news")
		return
	}
	defer cursor.Close(c.Request.Context())
	news := make([]models.News, 0)
	if err := cursor.All(c.Request.Context(), &news); err != nil {
		respondError(c, http.StatusInternalServerError, "could not read news")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": news})
}

func (h *Handler) CreateNews(c *gin.Context) {
	var input struct {
		Title string `json:"title" binding:"required"`
		Body  string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "title and body are required")
		return
	}
	input.Title, input.Body = strings.TrimSpace(input.Title), strings.TrimSpace(input.Body)
	if input.Title == "" || input.Body == "" || len(input.Title) > 160 || len(input.Body) > 4000 {
		respondError(c, http.StatusBadRequest, "title must be 1-160 characters and body 1-4000 characters")
		return
	}
	news := models.News{Title: input.Title, Body: input.Body, CreatedAt: time.Now().UTC()}
	result, err := h.Mongo.Collection("news").InsertOne(c.Request.Context(), news)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not publish news")
		return
	}
	objectID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		respondError(c, http.StatusInternalServerError, "published news has an invalid ID")
		return
	}
	news.ID = objectID
	payload, err := json.Marshal(gin.H{"type": "news", "item": news})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not encode news")
		return
	}
	h.Hub.broadcast("news", payload)
	c.JSON(http.StatusCreated, news)
}
