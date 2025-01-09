package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"BACK_SORTE_GO/models"
)

// CreateUserHandler lida com a criação de um novo usuário
func CreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
			CPF      string `json:"cpf"`
		}

		// Decodificar o corpo da requisição JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar o JSON", http.StatusBadRequest)
			return
		}

		// Validar os campos obrigatórios
		if req.Name == "" || req.Email == "" || req.Password == "" || req.CPF == "" {
			http.Error(w, "Todos os campos (name, email, password, cpf) são obrigatórios", http.StatusBadRequest)
			return
		}

		// Gerar um UUID para o usuário (se necessário)
		userID := uuid.NewString()

		// Obter a data atual para `DateCreate`
		now := time.Now()

		// Criar o modelo do usuário
		user := models.User{
			ID:         userID,
			Name:       req.Name,
			Email:      req.Email,
			Password:   req.Password,
			CPF:        req.CPF,
			Active:     true,
			Inicial:    false,
			Dell:       false,
			DateCreate: now,
			DateUpdate: now, // Define como NULL no banco
		}

		// Query de inserção
		query := `
			INSERT INTO core.user (id, name, email, password, cpf, active, inicial, dell, date_create, date_update)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		// Executar a query
		_, err := db.Exec(query, user.ID, user.Name, user.Email, user.Password, user.CPF, user.Active, user.Inicial, user.Dell, user.DateCreate, nil)
		if err != nil {
			http.Error(w, "Erro ao criar o usuário: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Retornar resposta de sucesso
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Usuário criado com sucesso",
			"id":      user.ID,
		})
	}
}
