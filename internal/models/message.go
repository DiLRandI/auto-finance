package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID      uuid.UUID
	From    string    `json:"from"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}
