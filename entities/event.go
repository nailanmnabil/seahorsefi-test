package entities

import (
	"time"
)

type Event struct {
	ID               int       `json:"id"`
	Address          string    `json:"address"`
	BlockNumber      uint64    `json:"block_number"`
	TransactionHash  string    `json:"transaction_hash"`
	EventType        string    `json:"event_type"`
	WalletID         string    `json:"wallet_id"`
	CreatedAt        time.Time `json:"created_at"`
	LastCalculatedAt time.Time `json:"last_calculated_at"`
	ClosedAt         *time.Time `json:"closed_at"`
}

func (Event) Table() string {
	return "events"
}
