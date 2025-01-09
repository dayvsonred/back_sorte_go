package models

import "time"

type DoacaoQRCode struct {
	ID         string    `json:"id" db:"id"`
	IDDoacao   string    `json:"id_doacao" db:"id_doacao"`
	QRCode     string    `json:"qrcode" db:"qrcode"`
	Valor      float64   `json:"valor" db:"valor"`
	Active     bool      `json:"active" db:"active"`
	Dell       bool      `json:"dell" db:"dell"`
	DateStart  time.Time `json:"date_start" db:"date_start"`
	DateCreate time.Time `json:"date_create" db:"date_create"`
	DateUpdate time.Time `json:"date_update" db:"date_update"`
}
