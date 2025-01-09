package models

import "time"

type UserDetails struct {
	ID         string    `json:"id" db:"id"`
	IDUser     string    `json:"id_user" db:"id_user"`
	CPFValid   bool      `json:"cpf_valid" db:"cpf_valid"`
	EmailValid bool      `json:"email_valid" db:"email_valid"`
	CEP        string    `json:"cep" db:"cep"`
	Telefone   string    `json:"telefone" db:"telefone"`
	Apelido    string    `json:"apelido" db:"apelido"`
	DateCreate time.Time `json:"date_create" db:"date_create"`
	DateUpdate time.Time `json:"date_update" db:"date_update"`
}
