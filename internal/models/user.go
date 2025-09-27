package models

import (
	"time"

	"github.com/go-ozzo/ozzo-validation/v4"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Login     string    `json:"login" db:"login"`
	Password  string    `json:"password" db:"password"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ValidateCredentials validates user login and password
func (u *User) ValidateCredentials() error {
	return validation.ValidateStruct(u,
		validation.Field(&u.Login, validation.Required),
		validation.Field(&u.Password, validation.Required),
	)
}
