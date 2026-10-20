package models

import "time"

type User struct {
	ID        string    `json:"-"`
	PublicID  int64     `json:"public_id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
