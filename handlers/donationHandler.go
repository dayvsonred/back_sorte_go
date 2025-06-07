package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
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

// Estrutura de resposta de listagem de doações
type DonationResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Valor     float64 `json:"valor"`
	Texto     string  `json:"texto"`
	Area      string  `json:"area"`
	Img       string  `json:"img"`
	DateStart string  `json:"date_start"`
}

// DonationListByIDUserHandler retorna doações de um usuário com paginação
func DonationListByIDUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idUser := r.URL.Query().Get("id_user")
		if idUser == "" {
			http.Error(w, "Parâmetro 'id_user' é obrigatório", http.StatusBadRequest)
			return
		}

		// Padrões para paginação
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		page := 1
		limit := 10

		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100 // valor máximo permitido
			} else {
				limit = l
			}
		}

		offset := (page - 1) * limit

		// Consulta principal com paginação
		query := `
			SELECT d.id, d.name, d.valor, d.date_create, dd.texto, dd.img_caminho, dd.area
			FROM core.doacao d
			JOIN core.doacao_details dd ON d.id = dd.id_doacao
			WHERE d.id_user = $1
			ORDER BY d.date_create DESC
			LIMIT $2 OFFSET $3
		`

		rows, err := db.Query(query, idUser, limit, offset)
		if err != nil {
			http.Error(w, "Erro ao buscar doações: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var donations []map[string]interface{}
		for rows.Next() {
			var id, name, texto, img, area string
			var valor float64
			var dateCreate time.Time

			if err := rows.Scan(&id, &name, &valor, &dateCreate, &texto, &img, &area); err != nil {
				http.Error(w, "Erro ao processar dados: "+err.Error(), http.StatusInternalServerError)
				return
			}

			donations = append(donations, map[string]interface{}{
				"id":           id,
				"name":         name,
				"valor":        valor,
				"date_create":  dateCreate,
				"texto":        texto,
				"img":          img,
				"area":         area,
			})
		}

		// Verifica se há mais registros
		var total int
		countQuery := `
			SELECT COUNT(*)
			FROM core.doacao d
			WHERE d.id_user = $1
		`
		err = db.QueryRow(countQuery, idUser).Scan(&total)
		if err != nil {
			http.Error(w, "Erro ao contar doações: "+err.Error(), http.StatusInternalServerError)
			return
		}

		hasNext := (offset + limit) < total

		// Resposta
		response := map[string]interface{}{
			"items":         donations,
			"page":          page,
			"limit":         limit,
			"total":         total,
			"has_next_page": hasNext,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}