package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/utils"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
func DonationHandler1(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		var donationReq DonationRequest
		if err := json.NewDecoder(r.Body).Decode(&donationReq); err != nil {
			http.Error(w, "Erro ao processar JSON", http.StatusBadRequest)
			return
		}

		if donationReq.IDUser == "" || donationReq.Name == "" || donationReq.Valor <= 0 ||
			donationReq.Texto == "" || donationReq.Area == "" || donationReq.Img == "" {
			http.Error(w, "Todos os campos são obrigatórios", http.StatusBadRequest)
			return
		}

		donationID := uuid.NewString()
		now := time.Now()

		// 1. Inserir doação
		queryDonation := `
			INSERT INTO core.doacao (id, id_user, name, valor, active, dell, date_start, date_create)
			VALUES ($1, $2, $3, $4, true, false, $5, $6)
		`
		_, err = db.Exec(queryDonation, donationID, donationReq.IDUser, donationReq.Name, donationReq.Valor, now, now)
		if err != nil {
			http.Error(w, "Erro ao salvar a doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. Inserir detalhes
		queryDetails := `
			INSERT INTO core.doacao_details (id, id_doacao, texto, img_caminho, area)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err = db.Exec(queryDetails, uuid.NewString(), donationID, donationReq.Texto, donationReq.Img, donationReq.Area)
		if err != nil {
			http.Error(w, "Erro ao salvar os detalhes da doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. Gerar nome_link único e inserir em doacao_link
		nomeLink, err := generateUniqueLinkName(db, donationReq.Name)
		if err != nil {
			http.Error(w, "Erro ao gerar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		queryLink := `
			INSERT INTO core.doacao_link (id, id_doacao, nome_link)
			VALUES ($1, $2, $3)
		`
		_, err = db.Exec(queryLink, uuid.NewString(), donationID, nomeLink)
		if err != nil {
			http.Error(w, "Erro ao salvar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Sucesso
		json.NewEncoder(w).Encode(map[string]string{
			"message":   "Doação criada com sucesso",
			"id":        donationID,
			"nome_link": nomeLink,
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

		// Paginação
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

		// Consulta com JOIN na tabela de link
		query := `
			SELECT 
				d.id, d.name, d.valor, d.date_create, d.date_start, d.active, d.dell,
				dd.texto, dd.img_caminho, dd.area,
				dl.nome_link
			FROM core.doacao d
			JOIN core.doacao_details dd ON d.id = dd.id_doacao
			LEFT JOIN core.doacao_link dl ON d.id = dl.id_doacao
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
				nomeLink                   sql.NullString
			)

			if err := rows.Scan(&id, &name, &valor, &dateCreate, &dateStart, &active, &dell, &texto, &img, &area, &nomeLink); err != nil {
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
				"nome_link":   nomeLink.String,
			})
		}

		// Contar total de doações
		var total int
		err = db.QueryRow(`SELECT COUNT(*) FROM core.doacao WHERE id_user = $1`, idUser).Scan(&total)
		if err != nil {
			http.Error(w, "Erro ao contar doações: "+err.Error(), http.StatusInternalServerError)
			return
		}

		hasNext := (offset + limit) < total

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

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func generateUniqueLinkName(db *sql.DB, base string) (string, error) {
	base = strings.ToLower(base)
	base = removeAccents(base)

	// Substituir espaços por _
	base = strings.ReplaceAll(base, " ", "_")

	// Remover caracteres não alfanuméricos nem underline
	re := regexp.MustCompile(`[^a-z0-9_]+`)
	base = re.ReplaceAllString(base, "")

	link := "@" + base
	finalLink := link

	letters := "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())

	for {
		var exists bool
		err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM core.doacao_link WHERE nome_link = $1)`, finalLink).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			break
		}
		// Acrescenta letra aleatória
		finalLink = fmt.Sprintf("%s_%c", link, letters[rand.Intn(len(letters))])
	}

	return finalLink, nil
}

func DonationByLinkHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		nomeLink := vars["nome_link"]

		if nomeLink == "" || !strings.HasPrefix(nomeLink, "@") {
			http.Error(w, "nome_link inválido", http.StatusBadRequest)
			return
		}

		// Buscar o ID da doação a partir do nome_link
		var idDoacao string
		err := db.QueryRow(`
			SELECT id_doacao FROM core.doacao_link
			WHERE nome_link = $1
		`, nomeLink).Scan(&idDoacao)
		if err == sql.ErrNoRows {
			http.Error(w, "Doação não encontrada", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Erro ao buscar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Buscar dados da doação
		var doacao struct {
			ID      string  `json:"id"`
			IDUser  string  `json:"id_user"`
			Name    string  `json:"name"`
			Valor   float64 `json:"valor"`
			Active  bool    `json:"active"`
			Dell    bool    `json:"dell"`
			Start   string  `json:"date_start"`
			Created string  `json:"date_create"`
		}
		err = db.QueryRow(`
			SELECT id, id_user, name, valor, active, dell, date_start, date_create
			FROM core.doacao
			WHERE id = $1
		`, idDoacao).Scan(
			&doacao.ID, &doacao.IDUser, &doacao.Name, &doacao.Valor, &doacao.Active,
			&doacao.Dell, &doacao.Start, &doacao.Created,
		)
		if err != nil {
			http.Error(w, "Erro ao buscar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Buscar detalhes
		var details struct {
			Texto string `json:"texto"`
			Img   string `json:"img_caminho"`
			Area  string `json:"area"`
		}
		err = db.QueryRow(`
			SELECT texto, img_caminho, area
			FROM core.doacao_details
			WHERE id_doacao = $1
		`, idDoacao).Scan(&details.Texto, &details.Img, &details.Area)
		if err != nil {
			http.Error(w, "Erro ao buscar detalhes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Montar resposta
		response := map[string]interface{}{
			"id":          doacao.ID,
			"id_user":     doacao.IDUser,
			"name":        doacao.Name,
			"valor":       doacao.Valor,
			"active":      doacao.Active,
			"dell":        doacao.Dell,
			"date_start":  doacao.Start,
			"date_create": doacao.Created,
			"texto":       details.Texto,
			"img_caminho": details.Img,
			"area":        details.Area,
			"nome_link":   nomeLink,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Estrutura do retorno completo
type DonationMessageFull struct {
	ID          string    `json:"id"`
	Valor       string    `json:"valor"`
	CPF         string    `json:"cpf"`
	Nome        string    `json:"nome"`
	Mensagem    string    `json:"mensagem"`
	Anonimo     bool      `json:"anonimo"`
	DataCriacao time.Time `json:"data_criacao"`
}

// DonationMensagesHandler retorna mensagens com paginação
func DonationMensagesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parâmetros de query
		idDoacao := r.URL.Query().Get("id")
		if idDoacao == "" {
			http.Error(w, "Parâmetro 'id' é obrigatório", http.StatusBadRequest)
			return
		}

		// Paginação
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		// Consulta com paginação
		rows, err := db.Query(`
			SELECT id, valor, cpf, nome, mensagem, anonimo, data_criacao
			FROM core.pix_qrcode
			WHERE id_doacao = $1 AND visivel = TRUE
			ORDER BY data_criacao DESC
			LIMIT $2 OFFSET $3
		`, idDoacao, limit, offset)
		if err != nil {
			http.Error(w, "Erro ao buscar mensagens: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var mensagens []DonationMessageFull
		for rows.Next() {
			var msg DonationMessageFull
			if err := rows.Scan(
				&msg.ID,
				&msg.Valor,
				&msg.CPF,
				&msg.Nome,
				&msg.Mensagem,
				&msg.Anonimo,
				&msg.DataCriacao,
			); err != nil {
				http.Error(w, "Erro ao ler resultado: "+err.Error(), http.StatusInternalServerError)
				return
			}
			mensagens = append(mensagens, msg)
		}

		// Resposta JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mensagens)
	}
}

type DonationSummary struct {
	ValorTotal    string `json:"valor_total"`
	TotalDoadores int    `json:"total_doadores"`
}

func DonationSummaryByIDHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idDoacao := vars["id"]
		if idDoacao == "" {
			http.Error(w, "Parâmetro 'id' é obrigatório", http.StatusBadRequest)
			return
		}

		var resumo DonationSummary

		query := `
			SELECT 
				COALESCE(SUM(valor), 0)::TEXT AS valor_total,
				COUNT(DISTINCT cpf) AS total_doadores
			FROM core.pix_qrcode
			WHERE id_doacao = $1 AND visivel = true
		`

		err := db.QueryRow(query, idDoacao).Scan(&resumo.ValorTotal, &resumo.TotalDoadores)
		if err != nil {
			http.Error(w, "Erro ao buscar resumo da doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resumo)
	}
}


func DonationHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Valida token
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

		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuário inválido", http.StatusUnauthorized)
			return
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Erro ao ler formulário", http.StatusBadRequest)
			return
		}

		// Ler campos
		name := r.FormValue("name")
		valorStr := r.FormValue("valor")
		texto := r.FormValue("texto")
		area := r.FormValue("area")

		if idUser == "" || name == "" || valorStr == "" || texto == "" || area == "" {
			http.Error(w, "Todos os campos são obrigatórios", http.StatusBadRequest)
			return
		}


		valor, err := strconv.ParseFloat(valorStr, 64)
		if err != nil || valor <= 0 {
			http.Error(w, "Valor inválido", http.StatusBadRequest)
			return
		}
		if name == "" || texto == "" || area == "" {
			http.Error(w, "Campos obrigatórios ausentes", http.StatusBadRequest)
			return
		}

		// Upload da imagem
		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Imagem obrigatória", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Gerar nome do arquivo
		imgFileName := fmt.Sprintf("%s_%d_%s", idUser, time.Now().Unix(), handler.Filename)

		// Upload para S3
		imgPath, err := utils.UploadToS3(file, imgFileName, config.GetawsBucketNameImgDoacao())
		if err != nil {
			http.Error(w, "Erro ao subir imagem: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Inserções no banco
		donationID := uuid.NewString()
		now := time.Now()

		_, err = db.Exec(`
			INSERT INTO core.doacao (id, id_user, name, valor, active, dell, date_start, date_create)
			VALUES ($1, $2, $3, $4, true, false, $5, $6)
		`, donationID, idUser, name, valor, now, now)
		if err != nil {
			http.Error(w, "Erro ao salvar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(`
			INSERT INTO core.doacao_details (id, id_doacao, texto, img_caminho, area)
			VALUES ($1, $2, $3, $4, $5)
		`, uuid.NewString(), donationID, texto, imgPath, area)
		if err != nil {
			http.Error(w, "Erro ao salvar detalhes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Gerar nome_link
		nomeLink, err := generateUniqueLinkName(db, name)
		if err != nil {
			http.Error(w, "Erro ao gerar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = db.Exec(`
			INSERT INTO core.doacao_link (id, id_doacao, nome_link)
			VALUES ($1, $2, $3)
		`, uuid.NewString(), donationID, nomeLink)
		if err != nil {
			http.Error(w, "Erro ao salvar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"message":   "Doação criada com sucesso",
			"id":        donationID,
			"nome_link": nomeLink,
			"img":       imgPath,
		})
	}
}