package models

import "time"

type Listing struct {
	ID             string    `json:"id"`
	SellerID       string    `json:"-"`
	SellerPublicID int64     `json:"seller_public_id"`
	SellerUsername string    `json:"seller_username"`
	SellerName     string    `json:"seller_name"`
	Title          string    `json:"title"`
	Game           string    `json:"game"`
	Category       string    `json:"category"`
	Description    string    `json:"description"`
	PriceKZT       int64     `json:"price_kzt"`
	ImageURL       string    `json:"image_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ListingInput struct {
	Title       string `json:"title" binding:"required"`
	Game        string `json:"game" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description" binding:"required"`
	PriceKZT    int64  `json:"price_kzt" binding:"required"`
	ImageURL    string `json:"image_url"`
}
