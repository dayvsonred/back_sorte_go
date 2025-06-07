package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"BACK_SORTE_GO/models"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

func UserPasswordChangeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obter o token do cabeçalho Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validar e extrair claims
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Obter o ID do usuário do token
		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			http.Error(w, "ID do usuário inválido no token", http.StatusUnauthorized)
			return
		}

		// Estrutura esperada no JSON
		var req struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}
		if req.OldPassword == "" || req.NewPassword == "" {
			http.Error(w, "As senhas antiga e nova são obrigatórias", http.StatusBadRequest)
			return
		}

		// Obter a senha atual do banco
		var hashedPassword string
		err = db.QueryRow("SELECT password FROM core.user WHERE id = $1", userID).Scan(&hashedPassword)
		if err != nil {
			http.Error(w, "Usuário não encontrado", http.StatusNotFound)
			return
		}

		// Verificar se a senha antiga está correta
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.OldPassword)); err != nil {
			http.Error(w, "Senha antiga incorreta", http.StatusUnauthorized)
			return
		}

		// Gerar hash da nova senha
		newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erro ao criptografar nova senha", http.StatusInternalServerError)
			return
		}

		// Atualizar no banco
		_, err = db.Exec("UPDATE core.user SET password = $1, date_update = NOW() WHERE id = $2", newHashedPassword, userID)
		if err != nil {
			http.Error(w, "Erro ao atualizar a senha", http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Senha atualizada com sucesso",
		})
	}
}
