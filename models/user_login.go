package models

import "time"

type UserLogin struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	IDUser    *string   `json:"id_user,omitempty" db:"id_user"`
	PassValid bool      `json:"pass_valid" db:"pass_valid"`
	Date      time.Time `json:"date" db:"date"`
}
