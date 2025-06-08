package models

import "time"

type DoacaoLink struct {
	ID        string    `json:"id"`
	IDDoacao  string    `json:"id_doacao"`
	NomeLink  string    `json:"nome_link"`
	CreatedAt time.Time `json:"created_at"` // Se quiser timestamp no futuro
}
