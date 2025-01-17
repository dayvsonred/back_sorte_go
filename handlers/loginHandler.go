package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v4"
)

// Configurações para o JWT
var jwtSecretKey = []byte("SUA_CHAVE_SECRETA") // Substitua pela sua chave secreta

// Estrutura para a resposta do token
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CPF       string    `json:"cpf"`
		Active    bool      `json:"active"`
		Inicial   bool      `json:"inicial"`
		Dell      bool      `json:"dell"`
		DateCreate time.Time `json:"date_create"`
	} `json:"user"`
}

// LoginHandler lida com a autenticação de usuários
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validar cabeçalhos
		authHeader := r.Header.Get("Authorization")
		contentType := r.Header.Get("Content-Type")
		if authHeader != "Basic QVBJX05BTUVfQUNDRVNTOkFQSV9TRUNSRVRfQUNDRVNT" || contentType != "application/x-www-form-urlencoded" {
			http.Error(w, "Cabeçalhos inválidos", http.StatusUnauthorized)
			return
		}

		// Analisar os parâmetros recebidos
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Erro ao processar os parâmetros", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		grantType := r.FormValue("grant_type")

		// Validar os parâmetros
		if grantType != "password" || username == "" || password == "" {
			http.Error(w, "Parâmetros inválidos", http.StatusBadRequest)
			return
		}

		// Buscar o usuário no banco de dados
		var user struct {
			ID         string
			Name       string
			Email      string
			Password   string
			CPF        string
			Active     bool
			Inicial    bool
			Dell       bool
			DateCreate time.Time
		}

		query := `
			SELECT id, name, email, password, cpf, active, inicial, dell, date_create
			FROM core.user
			WHERE email = $1
		`
		err := db.QueryRow(query, username).Scan(
			&user.ID, &user.Name, &user.Email, &user.Password, &user.CPF,
			&user.Active, &user.Inicial, &user.Dell, &user.DateCreate,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Usuário ou senha inválidos", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Erro ao buscar usuário", http.StatusInternalServerError)
			return
		}

		// Validar a senha
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			http.Error(w, "Usuário ou senha inválidos", http.StatusUnauthorized)
			return
		}

		// Gerar o token JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": user.ID,
			"exp": time.Now().Add(time.Hour * 24).Unix(),
		})

		tokenString, err := token.SignedString(jwtSecretKey)
		if err != nil {
			http.Error(w, "Erro ao gerar token", http.StatusInternalServerError)
			return
		}

		// Montar a resposta
		response := LoginResponse{
			Token: tokenString,
		}
		response.User = struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Email     string    `json:"email"`
			CPF       string    `json:"cpf"`
			Active    bool      `json:"active"`
			Inicial   bool      `json:"inicial"`
			Dell      bool      `json:"dell"`
			DateCreate time.Time `json:"date_create"`
		}{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			CPF:       user.CPF,
			Active:    user.Active,
			Inicial:   user.Inicial,
			Dell:      user.Dell,
			DateCreate: user.DateCreate,
		}

		// Retornar a resposta
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
