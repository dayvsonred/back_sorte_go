package models

import "time"

type User struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Email      string    `json:"email" db:"email"`
	Password   string    `json:"password" db:"password"`
	CPF        string    `json:"cpf" db:"cpf"`
	Active     bool      `json:"active" db:"active"`
	Inicial    bool      `json:"inicial" db:"inicial"`
	Dell       bool      `json:"dell" db:"dell"`
	DateCreate time.Time `json:"date_create" db:"date_create"`
	DateUpdate time.Time `json:"date_update" db:"date_update"`
}
