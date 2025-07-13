package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Estrutura para ler os dados do corpo da requisição
type ContactRequest struct {
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Mensagem string `json:"mensagem"`
	IP       string `json:"ip"`
	Location string `json:"location"`
	Token    string `json:"token"`
}

// Função para lidar com a rota POST /contact/mensagem
func ContactMensagemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ContactRequest

		// Decodifica o JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Valida campos obrigatórios
		if strings.TrimSpace(req.Nome) == "" || strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Mensagem) == "" {
			http.Error(w, "Campos nome, email e mensagem são obrigatórios", http.StatusBadRequest)
			return
		}

		if len(req.Mensagem) > 200 {
			http.Error(w, "A mensagem deve ter no máximo 200 caracteres", http.StatusBadRequest)
			return
		}

		// Prepara os dados
		id := uuid.NewString()
		dataCreate := time.Now()

		// Insere no banco de dados
		_, err := db.Exec(`
			INSERT INTO core.contact_us (
				id, nome, email, mensagem, ip, location, token, view, data_create
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, false, $8
			)
		`, id, req.Nome, req.Email, req.Mensagem, req.IP, req.Location, req.Token, dataCreate)

		if err != nil {
			http.Error(w, "Erro ao salvar mensagem: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Sucesso
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Mensagem enviada com sucesso",
			"id":      id,
		})
	}
}
