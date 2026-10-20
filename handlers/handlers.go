package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	Postgres  *pgxpool.Pool
	Mongo     *mongo.Database
	JWTSecret string
	Hub       *Hub
}

func New(postgres *pgxpool.Pool, mongoDB *mongo.Database, secret string) *Handler {
	return &Handler{
		Postgres:  postgres,
		Mongo:     mongoDB,
		JWTSecret: secret,
		Hub:       NewHub(),
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) issueToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.JWTSecret))
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

func validID(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	compact := strings.ReplaceAll(id, "-", "")
	_, err := hex.DecodeString(compact)
	return err == nil
}

func validUsername(username string) bool {
	if len(username) < 3 || len(username) > 24 {
		return false
	}
	for _, char := range username {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func currentUserID(c *gin.Context) string {
	userID, _ := c.Get("userID")
	id, _ := userID.(string)
	return id
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
