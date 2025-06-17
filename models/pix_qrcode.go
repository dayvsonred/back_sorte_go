package models

import (
	"time"

	"github.com/google/uuid"
)

type PixQRCode struct {
	ID         uuid.UUID
	IdDoacao   string
	Valor      string
	CPF        string
	Nome       string
	Mensagem   string
	Anonimo    bool
	Visivel    bool
	DataCriacao time.Time
}

type PixQRCodeStatus struct {
	ID              uuid.UUID
	IdPixQRCode     uuid.UUID
	DataCriacao     time.Time
	Expiracao       int
	TipoPagamento   string
	LocID           int
	LocTipoCob      string
	LocCriacao      time.Time
	Location        string
	PixCopiaECola   string
	Chave           string
	IdPix           string
	Status          string
	Buscar          bool
	Finalizado      bool
	DataPago        *time.Time
}
