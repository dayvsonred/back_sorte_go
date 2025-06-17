package handlers

import (
	"BACK_SORTE_GO/config"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/efipay/sdk-go-apis-efi/src/efipay/pix"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// PixChargeRequest define a estrutura do JSON recebido na requisição
type PixChargeRequest struct {
	Valor    string `json:"valor"`
	CNPJ     string `json:"cnpj"`
	Nome     string `json:"nome"`
	Chave    string `json:"chave"`
	Mensagem string `json:"mensagem"`
	Anonimo	 bool  `json:"anonimo"`
	IdDoacao string `json:"id"`
}


// parseTime faz parse de string ISO para time.Time
func parseTimeISO(v interface{}) time.Time {
	if v == nil {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, v.(string))
	if err != nil {
		return time.Now()
	}
	return t
}

// TestPixTokenHandler cria uma cobrança PIX ao receber uma requisição HTTP
func CreatePixTokenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}

		var req PixChargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		efi := pix.NewEfiPay(config.GetCredentials())

		body := map[string]interface{}{
			"calendario": map[string]interface{}{"expiracao": 3600},
			"devedor": map[string]interface{}{
				"cpf":  req.CNPJ,
				"nome": req.Nome,
			},
			"valor":              map[string]interface{}{"original": req.Valor},
			"chave":              req.Chave,
			"solicitacaoPagador": "pagamento de doação",
		}

		// Chamada da API
		resStr, err := efi.CreateImmediateCharge(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao criar cobrança PIX: %v", err), http.StatusInternalServerError)
			return
		}

		// Converte resposta em map
		var resMap map[string]interface{}
		if err := json.Unmarshal([]byte(resStr), &resMap); err != nil {
			http.Error(w, "Erro ao decodificar resposta do PIX: "+err.Error(), http.StatusInternalServerError)
			return
		}

		txid, ok := resMap["txid"].(string)
		if !ok || txid == "" {
			http.Error(w, "Resposta inválida da API (txid ausente)", http.StatusInternalServerError)
			return
		}

		// Inicia transação
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Erro ao iniciar transação: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		idPixQRCode := uuid.New()

		// Insert pix_qrcode
		_, err = tx.Exec(`
			INSERT INTO core.pix_qrcode 
			(id, id_doacao, valor, cpf, nome, mensagem, anonimo, visivel, data_criacao)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		`,
			idPixQRCode,
			req.IdDoacao,
			req.Valor,
			req.CNPJ,
			req.Nome,
			req.Mensagem,
			req.Anonimo,
			false,
		)
		if err != nil {
			http.Error(w, "Erro ao salvar pix_qrcode: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Salvando status para id_pix_qrcode: %v\n", idPixQRCode)

		// Insert pix_qrcode_status
		_, err = tx.Exec(`
			INSERT INTO core.pix_qrcode_status (
				id, id_pix_qrcode, data_criacao, expiracao, tipo_pagamento,
				loc_id, loc_tipo_cob, loc_criacao, location, pix_copia_e_cola,
				chave, id_pix, status, buscar, finalizado, data_pago
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16
			)
		`,
			uuid.NewString(),
			idPixQRCode,
			parseTimeISO(resMap["calendario"].(map[string]interface{})["criacao"]),
			int(resMap["calendario"].(map[string]interface{})["expiracao"].(float64)),
			"v1",
			int(resMap["loc"].(map[string]interface{})["id"].(float64)),
			resMap["loc"].(map[string]interface{})["tipoCob"],
			parseTimeISO(resMap["loc"].(map[string]interface{})["criacao"]),
			resMap["loc"].(map[string]interface{})["location"],
			resMap["loc"].(map[string]interface{})["location"], // ou outro campo
			req.Chave,
			txid,
			resMap["status"],
			true,
			false,
			nil,
		)
		if err != nil {
			http.Error(w, "Erro ao salvar pix_qrcode_status: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Commit transação
		if err := tx.Commit(); err != nil {
			http.Error(w, "Erro ao commitar transação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Retorno da API
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resStr))
	}
}

func PixChargeStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		txid := vars["txid"]
		if txid == "" {
			http.Error(w, "txid é obrigatório", http.StatusBadRequest)
			return
		}

		// Obtém as credenciais do config.go
		//credentials := Credentials
		credentials := config.GetCredentials()
		fmt.Printf("Credentials: %+v\n", credentials)
		efi := pix.NewEfiPay(credentials)

		//efi := pix.NewEfiPay(config.GetCredentials())

		// Consulta o status da cobrança PIX
		res, err := efi.DetailCharge(txid)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao consultar status do PIX: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(res)) // Retorna a resposta original da API
	}
}
