package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ChatID            string             `bson:"chat_id" json:"chat_id"`
	SenderID          string             `bson:"sender_id" json:"-"`
	SenderUsername    string             `bson:"sender_username" json:"sender_username"`
	RecipientID       string             `bson:"recipient_id" json:"-"`
	RecipientUsername string             `bson:"recipient_username" json:"recipient_username"`
	Text              string             `bson:"text" json:"text"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
}

type News struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Slug      string             `bson:"slug,omitempty" json:"-"`
	Title     string             `bson:"title" json:"title"`
	Body      string             `bson:"body" json:"body"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
