package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/an1shka/go-project-playerook-kz/models"
)

type credentials struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "name, valid email, and password are required")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Username = strings.ToLower(strings.TrimSpace(input.Username))
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Password = strings.TrimSpace(input.Password)
	if input.Name == "" || len(input.Name) > 80 || !validUsername(input.Username) || len(input.Password) < 8 || len(input.Password) > 72 {
		respondError(c, http.StatusBadRequest, "name must be 1-80 characters, username 3-24 letters/digits/underscores, and password 8-72 characters")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not secure password")
		return
	}
	id, err := newID()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not create user")
		return
	}
	user := models.User{ID: id, Username: input.Username, Name: input.Name, Email: input.Email, Role: "user", CreatedAt: time.Now().UTC()}
	err = h.Postgres.QueryRow(c.Request.Context(),
		`INSERT INTO users (id, username, name, email, password_hash) VALUES ($1, $2, $3, $4, $5) RETURNING public_id, role, created_at`,
		user.ID, user.Username, user.Name, user.Email, string(passwordHash)).Scan(&user.PublicID, &user.Role, &user.CreatedAt)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			respondError(c, http.StatusConflict, "email or username is already registered")
			return
		}
		respondError(c, http.StatusInternalServerError, "could not create user")
		return
	}

	token, err := h.issueToken(user.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not create access token")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

func (h *Handler) Login(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "valid email and password are required")
		return
	}

	var user models.User
	var passwordHash string
	err := h.Postgres.QueryRow(c.Request.Context(),
		`SELECT id, public_id, username, name, email, role, password_hash, created_at FROM users WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(input.Email))).Scan(&user.ID, &user.PublicID, &user.Username, &user.Name, &user.Email, &user.Role, &passwordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil) {
		respondError(c, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not authenticate user")
		return
	}

	token, err := h.issueToken(user.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not create access token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

func (h *Handler) Profile(c *gin.Context) {
	var user models.User
	err := h.Postgres.QueryRow(c.Request.Context(),
		`SELECT id, public_id, username, name, email, role, created_at FROM users WHERE id = $1`, currentUserID(c)).
		Scan(&user.ID, &user.PublicID, &user.Username, &user.Name, &user.Email, &user.Role, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not load profile")
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) GetPublicProfile(c *gin.Context) {
	username := strings.ToLower(strings.TrimSpace(c.Param("username")))
	if !validUsername(username) {
		respondError(c, http.StatusBadRequest, "invalid username")
		return
	}

	var user models.User
	err := h.Postgres.QueryRow(c.Request.Context(),
		`SELECT id, public_id, username, name, role, created_at FROM users WHERE username = $1`, username).
		Scan(&user.ID, &user.PublicID, &user.Username, &user.Name, &user.Role, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not load user")
		return
	}
	c.JSON(http.StatusOK, user)
}
