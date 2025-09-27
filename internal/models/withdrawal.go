package models

import "time"

type Withdrawal struct {
	OrderID     string    `json:"order" db:"order_id"`
	UID         string    `json:"-" db:"uid"`
	Amount      float64   `json:"sum" db:"amount"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}
