package models

import (
	"encoding/json"
	"time"
)

type OrderStatus string

const (
	StatusNew        OrderStatus = "NEW"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusProcessed  OrderStatus = "PROCESSED"
	StatusInvalid    OrderStatus = "INVALID"
)

type Order struct {
	ID            string      `json:"number" db:"id"`
	UID           string      `json:"-" db:"uid"`
	Accrual       *float64    `json:"accrual,omitempty" db:"accrual"`
	AccrualStatus OrderStatus `json:"status" db:"accrual_status"`
	UploadedAt    time.Time   `json:"uploaded_at" db:"uploaded_at"`
}

// ToJSON converts order to JSON
func (o *Order) ToJSON() ([]byte, error) {
	return json.Marshal(o)
}
