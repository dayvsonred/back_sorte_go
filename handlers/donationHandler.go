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
	"github.com/gorilla/mux"
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

		// Paginação com limite máximo de 100
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		page := 1
		limit := 10

		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}

		offset := (page - 1) * limit

		// Consulta com os campos extras
		query := `
			SELECT 
				d.id, d.name, d.valor, d.date_create, d.date_start, d.active, d.dell,
				dd.texto, dd.img_caminho, dd.area
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
			var (
				id, name, texto, img, area string
				valor                      float64
				dateCreate, dateStart      time.Time
				active, dell               bool
			)

			if err := rows.Scan(&id, &name, &valor, &dateCreate, &dateStart, &active, &dell, &texto, &img, &area); err != nil {
				http.Error(w, "Erro ao processar dados: "+err.Error(), http.StatusInternalServerError)
				return
			}

			donations = append(donations, map[string]interface{}{
				"id":          id,
				"name":        name,
				"valor":       valor,
				"date_create": dateCreate,
				"date_start":  dateStart,
				"active":      active,
				"dell":        dell,
				"texto":       texto,
				"img":         img,
				"area":        area,
			})
		}

		// Verificar total de registros
		var total int
		err = db.QueryRow(`SELECT COUNT(*) FROM core.doacao WHERE id_user = $1`, idUser).Scan(&total)
		if err != nil {
			http.Error(w, "Erro ao contar doações: "+err.Error(), http.StatusInternalServerError)
			return
		}

		hasNext := (offset + limit) < total

		// Retornar resposta JSON
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

func DonationDellHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obter o ID da doação da URL (ex: /donation/{id})
		vars := mux.Vars(r)
		donationID := vars["id"]
		if donationID == "" {
			http.Error(w, "ID da doação é obrigatório na URL", http.StatusBadRequest)
			return
		}

		// Extrair token JWT do cabeçalho
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey1, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Obter o ID do usuário do token
		userID := claims["sub"]
		if userID == nil {
			http.Error(w, "Token sem id_user", http.StatusUnauthorized)
			return
		}

		// Verificar se a doação pertence ao usuário
		var dbUserID string
		err = db.QueryRow(`SELECT id_user FROM core.doacao WHERE id = $1`, donationID).Scan(&dbUserID)
		if err != nil {
			http.Error(w, "Doação não encontrada", http.StatusNotFound)
			return
		}
		if dbUserID != userID {
			http.Error(w, "Usuário não autorizado a deletar esta doação", http.StatusForbidden)
			return
		}

		// Atualizar o campo dell para true
		_, err = db.Exec(`UPDATE core.doacao SET dell = true WHERE id = $1`, donationID)
		if err != nil {
			http.Error(w, "Erro ao deletar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Sucesso
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Doação deletada com sucesso",
		})
	}
}
