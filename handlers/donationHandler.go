package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v4"
)

// Chave secreta para validação do token JWT (mesma usada no login)
var jwtSecretKey1 = []byte("SUA_CHAVE_SECRETA")

// Estrutura para a requisição de doação
type DonationRequest struct {
	IDUser string  `json:"id_user"`
	Name   string  `json:"name"`
	Valor  float64 `json:"valor"`
	Texto  string  `json:"texto"`
	Area   string  `json:"area"`
	Img    string  `json:"img"`
}

// DonationHandler lida com o cadastro de doações
func DonationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validar o token JWT no cabeçalho Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey1, nil
		})
		if err != nil {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Decodificar o corpo da requisição
		var donationReq DonationRequest
		if err := json.NewDecoder(r.Body).Decode(&donationReq); err != nil {
			http.Error(w, "Erro ao processar JSON", http.StatusBadRequest)
			return
		}

		// Validar os campos obrigatórios
		if donationReq.IDUser == "" || donationReq.Name == "" || donationReq.Valor <= 0 ||
			donationReq.Texto == "" || donationReq.Area == "" || donationReq.Img == "" {
			http.Error(w, "Todos os campos são obrigatórios", http.StatusBadRequest)
			return
		}

		// Criar IDs para as tabelas
		donationID := uuid.NewString()
		now := time.Now()

		// Inserir na tabela `doacao`
		queryDonation := `
			INSERT INTO core.doacao (id, id_user, name, valor, active, dell, date_start, date_create)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = db.Exec(queryDonation, donationID, donationReq.IDUser, donationReq.Name, donationReq.Valor, true, false, now, now)
		if err != nil {
			http.Error(w, "Erro ao salvar a doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Inserir na tabela `doacao_details`
		queryDetails := `
			INSERT INTO core.doacao_details (id, id_doacao, texto, img_caminho, area)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err = db.Exec(queryDetails, uuid.NewString(), donationID, donationReq.Texto, donationReq.Img, donationReq.Area)
		if err != nil {
			http.Error(w, "Erro ao salvar os detalhes da doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Responder com sucesso
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Doação criada com sucesso",
			"id":      donationID,
		})
	}
}
