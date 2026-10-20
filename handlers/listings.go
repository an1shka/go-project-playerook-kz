package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/an1shka/go-project-playerook-kz/models"
)

func (h *Handler) ListListings(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	game := strings.TrimSpace(c.Query("game"))
	category := strings.TrimSpace(c.Query("category"))
	if category != "" && category != "account" && category != "resource" && category != "service" {
		respondError(c, http.StatusBadRequest, "category must be account, resource, or service")
		return
	}
	limit := 40
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 100 {
			respondError(c, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	offset := 0
	if rawOffset := c.Query("offset"); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			respondError(c, http.StatusBadRequest, "offset must be a non-negative integer")
			return
		}
		offset = parsed
	}

	rows, err := h.Postgres.Query(c.Request.Context(), `
		SELECT l.id, l.seller_id, u.public_id, u.username, u.name, l.title, l.game, l.category, l.description,
		       l.price_kzt, l.image_url, l.created_at, l.updated_at
		FROM listings l JOIN users u ON u.id = l.seller_id
		WHERE ($1 = '' OR l.title ILIKE '%' || $1 || '%' OR l.description ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR l.game ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR l.category = $3)
		ORDER BY l.created_at DESC LIMIT $4 OFFSET $5`,
		query, game, category, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not load listings")
		return
	}
	defer rows.Close()

	listings := make([]models.Listing, 0)
	for rows.Next() {
		var listing models.Listing
		if err := rows.Scan(&listing.ID, &listing.SellerID, &listing.SellerPublicID, &listing.SellerUsername, &listing.SellerName, &listing.Title, &listing.Game,
			&listing.Category, &listing.Description, &listing.PriceKZT, &listing.ImageURL, &listing.CreatedAt, &listing.UpdatedAt); err != nil {
			respondError(c, http.StatusInternalServerError, "could not read listings")
			return
		}
		listings = append(listings, listing)
	}
	if err := rows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "could not read listings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": listings, "limit": limit, "offset": offset})
}

func (h *Handler) MyListings(c *gin.Context) {
	rows, err := h.Postgres.Query(c.Request.Context(), `
		SELECT l.id, l.seller_id, u.public_id, u.username, u.name, l.title, l.game, l.category,
		       l.description, l.price_kzt, l.image_url, l.created_at, l.updated_at
		FROM listings l JOIN users u ON u.id = l.seller_id
		WHERE l.seller_id = $1
		ORDER BY l.created_at DESC`, currentUserID(c))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not load your listings")
		return
	}
	defer rows.Close()

	listings := make([]models.Listing, 0)
	for rows.Next() {
		var listing models.Listing
		if err := rows.Scan(&listing.ID, &listing.SellerID, &listing.SellerPublicID, &listing.SellerUsername,
			&listing.SellerName, &listing.Title, &listing.Game, &listing.Category, &listing.Description,
			&listing.PriceKZT, &listing.ImageURL, &listing.CreatedAt, &listing.UpdatedAt); err != nil {
			respondError(c, http.StatusInternalServerError, "could not read your listings")
			return
		}
		listings = append(listings, listing)
	}
	if err := rows.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "could not read your listings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": listings})
}

func (h *Handler) GetListing(c *gin.Context) {
	if !validID(c.Param("id")) {
		respondError(c, http.StatusBadRequest, "listing ID must be a UUID")
		return
	}
	listing, err := h.loadListing(c, c.Param("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		respondError(c, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not load listing")
		return
	}
	c.JSON(http.StatusOK, listing)
}

func (h *Handler) CreateListing(c *gin.Context) {
	input, ok := bindListing(c)
	if !ok {
		return
	}
	id, err := newID()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not create listing")
		return
	}
	var listing models.Listing
	err = h.Postgres.QueryRow(c.Request.Context(), `
		INSERT INTO listings (id, seller_id, title, game, category, description, price_kzt, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, seller_id, (SELECT u.public_id FROM users u WHERE u.id = listings.seller_id),
		          (SELECT u.username FROM users u WHERE u.id = listings.seller_id), title, game, category, description, price_kzt, image_url, created_at, updated_at`,
		id, currentUserID(c), input.Title, input.Game, input.Category, input.Description, input.PriceKZT, input.ImageURL).
		Scan(&listing.ID, &listing.SellerID, &listing.SellerPublicID, &listing.SellerUsername, &listing.Title, &listing.Game, &listing.Category, &listing.Description,
			&listing.PriceKZT, &listing.ImageURL, &listing.CreatedAt, &listing.UpdatedAt)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not create listing")
		return
	}
	c.JSON(http.StatusCreated, listing)
}

func (h *Handler) UpdateListing(c *gin.Context) {
	if !validID(c.Param("id")) {
		respondError(c, http.StatusBadRequest, "listing ID must be a UUID")
		return
	}
	input, ok := bindListing(c)
	if !ok {
		return
	}
	var listing models.Listing
	err := h.Postgres.QueryRow(c.Request.Context(), `
		UPDATE listings
		SET title = $1, game = $2, category = $3, description = $4, price_kzt = $5, image_url = $6, updated_at = NOW()
		WHERE id = $7 AND seller_id = $8
		RETURNING id, seller_id, (SELECT u.public_id FROM users u WHERE u.id = listings.seller_id),
		          (SELECT u.username FROM users u WHERE u.id = listings.seller_id), title, game, category, description, price_kzt, image_url, created_at, updated_at`,
		input.Title, input.Game, input.Category, input.Description, input.PriceKZT, input.ImageURL, c.Param("id"), currentUserID(c)).
		Scan(&listing.ID, &listing.SellerID, &listing.SellerPublicID, &listing.SellerUsername, &listing.Title, &listing.Game, &listing.Category, &listing.Description,
			&listing.PriceKZT, &listing.ImageURL, &listing.CreatedAt, &listing.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		h.listingMutationNotFound(c)
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not update listing")
		return
	}
	c.JSON(http.StatusOK, listing)
}

func (h *Handler) DeleteListing(c *gin.Context) {
	if !validID(c.Param("id")) {
		respondError(c, http.StatusBadRequest, "listing ID must be a UUID")
		return
	}
	result, err := h.Postgres.Exec(c.Request.Context(),
		`DELETE FROM listings WHERE id = $1 AND seller_id = $2`, c.Param("id"), currentUserID(c))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not delete listing")
		return
	}
	if result.RowsAffected() == 0 {
		h.listingMutationNotFound(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindListing(c *gin.Context) (models.ListingInput, bool) {
	var input models.ListingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "title, game, category, description, and positive price_kzt are required")
		return input, false
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Game = strings.TrimSpace(input.Game)
	input.Category = strings.TrimSpace(input.Category)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	if input.Title == "" || input.Game == "" || input.Description == "" || input.PriceKZT <= 0 {
		respondError(c, http.StatusBadRequest, "title, game, description, and positive price_kzt are required")
		return input, false
	}
	if len(input.Title) > 120 || len(input.Game) > 80 || len(input.Description) > 2000 || len(input.ImageURL) > 2048 {
		respondError(c, http.StatusBadRequest, "title, game, description, or image_url exceeds the allowed length")
		return input, false
	}
	if input.Category != "account" && input.Category != "resource" && input.Category != "service" {
		respondError(c, http.StatusBadRequest, "category must be account, resource, or service")
		return input, false
	}
	if input.ImageURL != "" {
		parsed, err := url.ParseRequestURI(input.ImageURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			respondError(c, http.StatusBadRequest, "image_url must be an http or https URL")
			return input, false
		}
	}
	return input, true
}

func (h *Handler) loadListing(c *gin.Context, id string) (models.Listing, error) {
	var listing models.Listing
	err := h.Postgres.QueryRow(c.Request.Context(), `
		SELECT l.id, l.seller_id, u.public_id, u.username, u.name, l.title, l.game, l.category, l.description,
		       l.price_kzt, l.image_url, l.created_at, l.updated_at
		FROM listings l JOIN users u ON u.id = l.seller_id WHERE l.id = $1`, id).
		Scan(&listing.ID, &listing.SellerID, &listing.SellerPublicID, &listing.SellerUsername, &listing.SellerName, &listing.Title, &listing.Game, &listing.Category,
			&listing.Description, &listing.PriceKZT, &listing.ImageURL, &listing.CreatedAt, &listing.UpdatedAt)
	return listing, err
}

func (h *Handler) listingMutationNotFound(c *gin.Context) {
	_, err := h.loadListing(c, c.Param("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		respondError(c, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "could not verify listing")
		return
	}
	respondError(c, http.StatusForbidden, "you can only change your own listings")
}
