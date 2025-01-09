package models

import "time"

type Doacao struct {
	ID         string    `json:"id" db:"id"`
	IDUser     string    `json:"id_user" db:"id_user"`
	Name       string    `json:"name" db:"name"`
	Valor      float64   `json:"valor" db:"valor"`
	Active     bool      `json:"active" db:"active"`
	Dell       bool      `json:"dell" db:"dell"`
	DateStart  time.Time `json:"date_start" db:"date_start"`
	DateCreate time.Time `json:"date_create" db:"date_create"`
	DateUpdate time.Time `json:"date_update" db:"date_update"`
}
