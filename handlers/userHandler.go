package handlers

import (
	"BACK_SORTE_GO/config"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	//"BACK_SORTE_GO/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

		userID := uuid.NewString()
		now := time.Now()

		// Inserir usuário
		_, err = db.Exec(`
			INSERT INTO core.user 
				(id, name, email, password, cpf, active, inicial, dell, date_create, date_update)
			VALUES 
				($1, $2, $3, $4, $5, true, false, false, $6, NULL)
		`, userID, req.Name, req.Email, string(hashedPassword), req.CPF, now)
		if err != nil {
			http.Error(w, "Erro ao criar o usuário: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Inserir em conta_nivel
		_, err = db.Exec(`
			INSERT INTO core.conta_nivel (
				id, id_user, nivel, ativo, status, data_pagamento, tipo_pagamento, data_update
			) VALUES (
				$1, $2, 'BASICO', false, 'INATIVO', NULL, 'INATIVO', $3
			)
		`, uuid.NewString(), userID, now)
		if err != nil {
			http.Error(w, "Erro ao criar conta_nivel: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Inserir em conta_nivel_pagamento
		_, err = db.Exec(`
			INSERT INTO core.conta_nivel_pagamento (
				id, id_user, pago_data, pago, valor, status, codigo, data_create,
				referente, valido, txid, pg_status, cpf, chave, pixCopiaECola, expiracao
			) VALUES (
				$1, $2, NULL, false, 0, 'INATIVO', '111', $3, '01', true, NULL, 'INATIVO', NULL, NULL, NULL, NULL
			)
		`, uuid.NewString(), userID, now)
		if err != nil {
			http.Error(w, "Erro ao criar conta_nivel_pagamento: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, map[string]string{
			"message": "Usuário criado com sucesso",
			"id":      userID,
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

func UserBankAccountHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Obter token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse do token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Extrair id_user
		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuário inválido", http.StatusUnauthorized)
			return
		}

		// Estrutura da requisição
		var req struct {
			Banco     string `json:"banco"`
			BancoNome string `json:"banco_nome"`
			Conta     string `json:"conta"`
			Agencia   string `json:"agencia"`
			Digito    string `json:"digito"`
			CPF       string `json:"cpf"`
			Telefone  string `json:"telefone"`
			Pix       string `json:"pix"`
		}

		// Decodificar JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}

		// Validar campos obrigatórios
		if req.Banco == "" || req.BancoNome == "" || req.Conta == "" || req.Agencia == "" || req.Digito == "" || req.CPF == "" || req.Telefone == "" {
			http.Error(w, "Todos os campos são obrigatórios", http.StatusBadRequest)
			return
		}

		// Inserir no banco
		id := uuid.NewString()
		_, err = db.Exec(`
			INSERT INTO core.saque_conta (
				id, id_user, banco, banco_nome, conta, agencia, digito, cpf, telefone, pix, active, dell, date_create
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, false, NOW()
			)
		`, id, idUser, req.Banco, req.BancoNome, req.Conta, req.Agencia, req.Digito, req.CPF, req.Telefone, req.Pix)

		if err != nil {
			http.Error(w, "Erro ao salvar os dados bancários: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusCreated, map[string]string{
			"message": "Conta bancária cadastrada com sucesso",
			"id":      id,
		})
	}
}

func UserBankAccountUpdateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validar e extrair o token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuário inválido", http.StatusUnauthorized)
			return
		}

		// Estrutura da requisição
		var req struct {
			IDContaOld string `json:"id_conta_old"`
			Banco      string `json:"banco"`
			BancoNome  string `json:"banco_nome"`
			Conta      string `json:"conta"`
			Agencia    string `json:"agencia"`
			Digito     string `json:"digito"`
			CPF        string `json:"cpf"`
			Telefone   string `json:"telefone"`
			Pix        string `json:"pix"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao processar o JSON", http.StatusBadRequest)
			return
		}

		// Verificar se a conta antiga pertence ao usuário e está ativa
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM core.saque_conta 
				WHERE id = $1 AND id_user = $1 AND active = true
			)
		`, req.IDContaOld, idUser).Scan(&exists)
		if err != nil {
			http.Error(w, "Erro ao verificar conta antiga: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "Conta antiga não encontrada ou não pertence ao usuário", http.StatusForbidden)
			return
		}

		// Desativar conta antiga
		_, err = db.Exec(`
			UPDATE core.saque_conta 
			SET active = false, dell = true, date_update = NOW()
			WHERE id = $1
		`, req.IDContaOld)
		if err != nil {
			http.Error(w, "Erro ao desativar a conta antiga: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Inserir nova conta
		newID := uuid.NewString()
		_, err = db.Exec(`
			INSERT INTO core.saque_conta (
				id, id_user, banco, banco_nome, conta, agencia, digito, cpf, telefone, pix, active, dell, date_create
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, false, NOW()
			)
		`, newID, idUser, req.Banco, req.BancoNome, req.Conta, req.Agencia, req.Digito, req.CPF, req.Telefone, req.Pix)
		if err != nil {
			http.Error(w, "Erro ao criar nova conta: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Conta atualizada com sucesso",
			"new_id":  newID,
			"old_id":  req.IDContaOld,
		})
	}
}

func UserBankAccountGetHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extrair o token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Pegar id do token
		idFromToken, ok := claims["sub"].(string)
		if !ok || idFromToken == "" {
			http.Error(w, "ID do usuário inválido", http.StatusUnauthorized)
			return
		}

		// Pegar id_user da query
		idFromQuery := r.URL.Query().Get("id_user")
		if idFromQuery == "" {
			http.Error(w, "Parâmetro id_user é obrigatório", http.StatusBadRequest)
			return
		}

		// Comparar os dois IDs
		if idFromToken != idFromQuery {
			http.Error(w, "Usuário não autorizado a acessar esta conta bancária", http.StatusForbidden)
			return
		}

		// Buscar a conta ativa
		var conta struct {
			ID        string `json:"id"`
			Banco     string `json:"banco"`
			BancoNome string `json:"banco_nome"`
			Conta     string `json:"conta"`
			Agencia   string `json:"agencia"`
			Digito    string `json:"digito"`
			CPF       string `json:"cpf"`
			Telefone  string `json:"telefone"`
			Pix       string `json:"pix"`
		}

		err = db.QueryRow(`
			SELECT id, banco, banco_nome, conta, agencia, digito, cpf, telefone, COALESCE(pix, '') 
			FROM core.saque_conta 
			WHERE id_user = $1 AND active = true AND dell = false
			LIMIT 1
		`, idFromToken).Scan(
			&conta.ID, &conta.Banco, &conta.BancoNome, &conta.Conta,
			&conta.Agencia, &conta.Digito, &conta.CPF, &conta.Telefone, &conta.Pix,
		)
		if err == sql.ErrNoRows {
			http.Error(w, "Nenhuma conta ativa encontrada para este usuário", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Erro ao buscar os dados: "+err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, http.StatusOK, conta)
	}
}

func UploadUserProfileImageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validar token JWT
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token ausente", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validar e extrair claims config.GetJwtSecret()
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Erro ao processar claims do token", http.StatusUnauthorized)
			return
		}

		idFromToken, ok := claims["sub"].(string)
		if !ok || idFromToken == "" {
			http.Error(w, "ID do usuário inválido", http.StatusUnauthorized)
			return
		}

		// Parse do arquivo enviado
		err = r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			http.Error(w, "Erro ao parsear o formulário: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Erro ao ler o arquivo: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Pega extensão segura
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			http.Error(w, "Formato de imagem não suportado", http.StatusBadRequest)
			return
		}

		fileName := idFromToken + ext // Nome da imagem no S3

		// Sessão S3
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(config.GetAwsRegion()),
			Credentials: credentials.NewStaticCredentials(
				config.GetAwsAccessKey(),
				config.GetAwsSecretKey(),
				"",
			),
		})
		if err != nil {
			http.Error(w, "Erro na sessão AWS: "+err.Error(), http.StatusInternalServerError)
			return
		}

		uploader := s3manager.NewUploader(sess)
		result, err := uploader.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(config.GetAwsBucket()),
			Key:         aws.String(fileName),
			Body:        file.(multipart.File),
			ContentType: aws.String(handler.Header.Get("Content-Type")),
			//ACL:         aws.String("public-read"),
		})
		if err != nil {
			http.Error(w, "Erro ao fazer upload no S3: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Atualiza ou insere em core.user_details
		var exists bool
		err = db.QueryRow(`SELECT EXISTS (SELECT 1 FROM core.user_details WHERE id_user = $1)`, idFromToken).Scan(&exists)
		if err != nil {
			http.Error(w, "Erro ao verificar existência de user_details: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Gerar UUID da img
		imgID := uuid.NewString()

		if exists {
			_, err = db.Exec(`
				UPDATE core.user_details 
				SET img_perfil = $1, date_update = now()
				WHERE id_user = $2
			`, fileName, idFromToken)
		} else {
			_, err = db.Exec(`
				INSERT INTO core.user_details (id, id_user, img_perfil)
				VALUES ($1, $2, $3)
			`, imgID, idFromToken, fileName)
		}
		if err != nil {
			http.Error(w, "Erro ao salvar imagem no banco: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Retorno final
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]string{"url": result.Location}
		json.NewEncoder(w).Encode(resp)
	}
}

func UserProfileImageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		if userID == "" {
			http.Error(w, "ID do usuário é obrigatório", http.StatusBadRequest)
			return
		}

		var imgPerfil sql.NullString
		err := db.QueryRow(`
			SELECT img_perfil
			FROM core.user_details
			WHERE id_user = $1
		`, userID).Scan(&imgPerfil)

		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Usuário não encontrado ou sem imagem", http.StatusNotFound)
				return
			}
			http.Error(w, "Erro ao buscar imagem do perfil: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !imgPerfil.Valid || imgPerfil.String == "" {
			http.Error(w, "Imagem de perfil não cadastrada", http.StatusNotFound)
			return
		}

		// Monta a URL pública do S3 com base nos dados da configuração
		region := config.GetAwsRegion()
		bucket := config.GetAwsBucket() // ex: doacao-users-prefil-v1-2025
		if region == "" || bucket == "" {
			http.Error(w, "Configuração do bucket não encontrada", http.StatusInternalServerError)
			return
		}

		url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, imgPerfil.String)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"image_url": "%s"}`, url)))
	}
}

// UserShowHandler busca e retorna os dados básicos do usuário pelo ID
func UserShowHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Extrair o ID da URL
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "ID do usuário não fornecido", http.StatusBadRequest)
			return
		}

		// Consulta no banco
		var (
			name       string
			email      string
			dateCreate string
		)
		err := db.QueryRow(`
			SELECT name, email, date_create
			FROM core.user
			WHERE id = $1
		`, id).Scan(&name, &email, &dateCreate)

		if err == sql.ErrNoRows {
			http.Error(w, "Usuário não encontrado", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Erro ao buscar usuário: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Retornar resposta JSON
		response := map[string]interface{}{
			"name":        name,
			"email":       email,
			"date_create": dateCreate,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
