package entities

import "time"

type Wallet struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Points    uint64    `json:"points"`
	Events    []Event   `json:"events,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Wallet) Table() string {
	return "wallets"
}
