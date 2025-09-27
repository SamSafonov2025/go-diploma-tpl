package models

type Balance struct {
	UID       string  `json:"-" db:"uid"`
	Current   float64 `json:"current" db:"current_balance"`
	Withdrawn float64 `json:"withdrawn" db:"withdrawn"`
}
