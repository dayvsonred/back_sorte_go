package models

import "time"

type DoacaoPagamento struct {
	ID              string    `json:"id" db:"id"`
	Identificador   string    `json:"identificador" db:"identificador"`
	IDDoacao        string    `json:"id_doacao" db:"id_doacao"`
	IDDoacaoQRCode  string    `json:"id_doacao_qrcode" db:"id_doacao_qrcode"`
	Texto           string    `json:"texto" db:"texto"`
	Valor           float64   `json:"valor" db:"valor"`
	DateCreate      time.Time `json:"date_create" db:"date_create"`
	DateUpdate      time.Time `json:"date_update" db:"date_update"`
}
