package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"BACK_SORTE_GO/models"
)

func CreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
			CPF      string `json:"cpf"`
		}

		// Decodificar JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar o JSON", http.StatusBadRequest)
			return
		}

		// Validar campos obrigatórios
		if req.Name == "" || req.Email == "" || req.Password == "" || req.CPF == "" {
			http.Error(w, "Todos os campos (name, email, password, cpf) são obrigatórios", http.StatusBadRequest)
			return
		}

		// Verificar duplicação de email
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM core.user WHERE email = $1)", req.Email).Scan(&exists)
		if err != nil {
			http.Error(w, "Erro ao verificar duplicação de email: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, "O email já está em uso", http.StatusBadRequest)
			return
		}

		// Hash da senha
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erro ao processar a senha", http.StatusInternalServerError)
			return
		}

		// Gerar UUID e data atual
		userID := uuid.NewString()
		now := time.Now()

		// Criar o modelo do usuário
		user := models.User{
			ID:         userID,
			Name:       req.Name,
			Email:      req.Email,
			Password:   string(hashedPassword),
			CPF:        req.CPF,
			Active:     true,
			Inicial:    false,
			Dell:       false,
			DateCreate: now,
		}

		// Query de inserção
		query := `
			INSERT INTO core.user (id, name, email, password, cpf, active, inicial, dell, date_create, date_update)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`
		_, err = db.Exec(query, user.ID, user.Name, user.Email, user.Password, user.CPF, user.Active, user.Inicial, user.Dell, user.DateCreate, nil)
		if err != nil {
			http.Error(w, "Erro ao criar o usuário: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Resposta de sucesso
		jsonResponse(w, http.StatusCreated, map[string]string{
			"message": "Usuário criado com sucesso",
			"id":      user.ID,
		})
	}
}

// Função auxiliar para resposta JSON
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
