package handlers

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/utils"
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

		// Consulta com JOINs
		query := `
			SELECT 
				d.id, d.name, d.valor, d.date_create, d.date_start, d.active, d.dell, d.closed,
				dd.texto, dd.img_caminho, dd.area,
				dl.nome_link,

				dp.valor_disponivel, dp.valor_tranferido, dp.data_tranferido,
				dp.solicitado, dp.data_solicitado, dp.status,
				dp.img, dp.pdf, dp.banco, dp.conta, dp.agencia, dp.digito, dp.pix, dp.data_update

			FROM core.doacao d
			JOIN core.doacao_details dd ON d.id = dd.id_doacao
			LEFT JOIN core.doacao_link dl ON d.id = dl.id_doacao
			LEFT JOIN core.doacao_pagamentos dp ON d.id = dp.id_doacao
			WHERE d.id_user = $1 AND d.dell = false
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
				active, dell, closed       bool
				nomeLink                   sql.NullString

				valorDisponivel, valorTransferido sql.NullFloat64
				dataTransferido, dataSolicitado   sql.NullTime
				solicitado                        sql.NullBool
				status, imgComp, pdfComp          sql.NullString
				banco, conta, agencia, digito     sql.NullString
				pix, dataUpdate                   sql.NullString
			)

			err := rows.Scan(
				&id, &name, &valor, &dateCreate, &dateStart, &active, &dell, &closed,
				&texto, &img, &area,
				&nomeLink,

				&valorDisponivel, &valorTransferido, &dataTransferido,
				&solicitado, &dataSolicitado, &status,
				&imgComp, &pdfComp, &banco, &conta, &agencia, &digito, &pix, &dataUpdate,
			)
			if err != nil {
				http.Error(w, "Erro ao processar dados: "+err.Error(), http.StatusInternalServerError)
				return
			}

			donation := map[string]interface{}{
				"id":          id,
				"name":        name,
				"valor":       valor,
				"date_create": dateCreate,
				"date_start":  dateStart,
				"active":      active,
				"dell":        dell,
				"closed":      closed,
				"texto":       texto,
				"img":         img,
				"area":        area,
				"nome_link":   nomeLink.String,
			}

			// Adicionar pagamentos se existirem
			if valorDisponivel.Valid || valorTransferido.Valid {
				donation["pagamento"] = map[string]interface{}{
					"valor_disponivel": valorDisponivel.Float64,
					"valor_tranferido": valorTransferido.Float64,
					"data_tranferido":  dataTransferido.Time,
					"solicitado":       solicitado.Bool,
					"data_solicitado":  dataSolicitado.Time,
					"status":           status.String,
					"img":              imgComp.String,
					"pdf":              pdfComp.String,
					"banco":            banco.String,
					"conta":            conta.String,
					"agencia":          agencia.String,
					"digito":           digito.String,
					"pix":              pix.String,
					"data_update":      dataUpdate.String,
				}
			}

			donations = append(donations, donation)
		}

		// Contar total
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

		// Atualizar o campo dell para true e date_update com a data atual
		_, err = db.Exec(`UPDATE core.doacao SET dell = true, date_update = NOW() WHERE id = $1`, donationID)
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
			Closed  bool    `json:"closed"`
			Start   string  `json:"date_start"`
			Created string  `json:"date_create"`
		}
		err = db.QueryRow(`
			SELECT id, id_user, name, valor, active, dell, closed, date_start, date_create
			FROM core.doacao
			WHERE id = $1
		`, idDoacao).Scan(
			&doacao.ID, &doacao.IDUser, &doacao.Name, &doacao.Valor,
			&doacao.Active, &doacao.Dell, &doacao.Closed, &doacao.Start, &doacao.Created,
		)
		if err != nil {
			http.Error(w, "Erro ao buscar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Validação se doação está fechada
		if doacao.Closed {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Doação fechada. Acesso não autorizado", http.StatusUnauthorized)
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

			// Comparar ID do token com ID do dono da doação
			idFromToken, ok := claims["sub"].(string)
			if !ok || idFromToken == "" || idFromToken != doacao.IDUser {
				http.Error(w, "Você não tem permissão para acessar esta doação fechada", http.StatusForbidden)
				return
			}
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
			"closed":      doacao.Closed,
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

		idUser, ok := claims["sub"].(string)
		if !ok || idUser == "" {
			http.Error(w, "ID do usuário inválido", http.StatusUnauthorized)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Erro ao ler formulário", http.StatusBadRequest)
			return
		}

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

		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Imagem obrigatória", http.StatusBadRequest)
			return
		}
		defer file.Close()

		imgFileName := fmt.Sprintf("%s_%d_%s", idUser, time.Now().Unix(), handler.Filename)
		imgPath, err := utils.UploadToS3(file, imgFileName, config.GetawsBucketNameImgDoacao())
		if err != nil {
			http.Error(w, "Erro ao subir imagem: "+err.Error(), http.StatusInternalServerError)
			return
		}

		donationID := uuid.NewString()
		now := time.Now()

		// Transação
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Erro ao iniciar transação: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Inserir doação
		_, err = tx.Exec(`
			INSERT INTO core.doacao (id, id_user, name, valor, active, dell, closed, date_start, date_create)
			VALUES ($1, $2, $3, $4, true, false, false, $5, $6)
		`, donationID, idUser, name, valor, now, now)
		if err != nil {
			http.Error(w, "Erro ao salvar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(`
			INSERT INTO core.doacao_details (id, id_doacao, texto, img_caminho, area)
			VALUES ($1, $2, $3, $4, $5)
		`, uuid.NewString(), donationID, texto, imgPath, area)
		if err != nil {
			http.Error(w, "Erro ao salvar detalhes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		nomeLink, err := generateUniqueLinkName(db, name)
		if err != nil {
			http.Error(w, "Erro ao gerar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec(`
			INSERT INTO core.doacao_link (id, id_doacao, nome_link)
			VALUES ($1, $2, $3)
		`, uuid.NewString(), donationID, nomeLink)
		if err != nil {
			http.Error(w, "Erro ao salvar nome_link: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Inserir em doacao_pagamentos
		_, err = tx.Exec(`
			INSERT INTO core.doacao_pagamentos (id, id_doacao, valor_disponivel, valor_tranferido, solicitado, status, data_update)
			VALUES ($1, $2, 0, 0, NULL, 'START', $3)
		`, uuid.New(), donationID, now)
		if err != nil {
			http.Error(w, "Erro ao salvar dados iniciais de pagamento: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Commit
		if err := tx.Commit(); err != nil {
			http.Error(w, "Erro ao finalizar transação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Sucesso
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message":   "Doação criada com sucesso",
			"id":        donationID,
			"nome_link": nomeLink,
			"img":       imgPath,
		})
	}
}

func DonationClosedHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica se foi enviado token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token não fornecido", http.StatusUnauthorized)
			return
		}

		// Extrai o token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey1, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// Pega o ID do usuário do token
		idUserToken, ok := claims["sub"].(string)
		if !ok || idUserToken == "" {
			http.Error(w, "ID do usuário inválido no token", http.StatusUnauthorized)
			return
		}

		// Pega o ID da doação na URL
		vars := mux.Vars(r)
		donationID := vars["id"]
		if donationID == "" {
			http.Error(w, "ID da doação é obrigatório", http.StatusBadRequest)
			return
		}

		// Verifica se a doação pertence ao usuário do token
		var idUserFromDB string
		err = db.QueryRow("SELECT id_user FROM core.doacao WHERE id = $1", donationID).Scan(&idUserFromDB)
		if err == sql.ErrNoRows {
			http.Error(w, "Doação não encontrada", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Erro ao verificar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if idUserFromDB != idUserToken {
			http.Error(w, "Você não tem permissão para encerrar esta doação", http.StatusForbidden)
			return
		}

		// Atualiza a doação como encerrada
		_, err = db.Exec(`
			UPDATE core.doacao 
			SET active = false, closed = true, date_update = NOW()
			WHERE id = $1 AND dell = false
		`, donationID)
		if err != nil {
			http.Error(w, "Erro ao encerrar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Sucesso
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Doação encerrada com sucesso",
		})
	}
}

func DonationRescueHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica token JWT
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

		// Pega ID da doação
		vars := mux.Vars(r)
		idDoacao := vars["id"]
		if idDoacao == "" {
			http.Error(w, "ID da doação é obrigatório", http.StatusBadRequest)
			return
		}

		// Verifica se a doação pertence ao usuário
		var donoDoacao string
		err = db.QueryRow("SELECT id_user FROM core.doacao WHERE id = $1", idDoacao).Scan(&donoDoacao)
		if err == sql.ErrNoRows {
			http.Error(w, "Doação não encontrada", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Erro ao verificar doação: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if donoDoacao != idUser {
			http.Error(w, "Você não tem permissão para resgatar essa doação", http.StatusForbidden)
			return
		}

		// Soma dos valores da doação com status CONCLUIDA, buscar=false, finalizado=true e visivel=true
		var totalValor float64
		err = db.QueryRow(`
			SELECT COALESCE(SUM(pq.valor), 0)
			FROM core.pix_qrcode pq
			JOIN core.pix_qrcode_status pqs ON pqs.id_pix_qrcode = pq.id
			WHERE pq.id_doacao = $1
			AND pqs.status = 'CONCLUIDA'
			AND pqs.buscar = false
			AND pqs.finalizado = true
		`, idDoacao).Scan(&totalValor)
		if err != nil {
			http.Error(w, "Erro ao calcular total recebido: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if totalValor <= 0 {
			http.Error(w, "Nenhum valor disponível para resgate", http.StatusBadRequest)
			return
		}

		// Aplica 10% de taxa
		valorDisponivel := totalValor * 0.90
		dataSolicitado := time.Now()

		// Atualiza doacao_pagamentos
		_, err = db.Exec(`
			UPDATE core.doacao_pagamentos
			SET valor_disponivel = $1,
				data_solicitado = $2,
				status = 'PROCESS',
				solicitado = true,
				data_update = NOW()
			WHERE id_doacao = $3
		`, valorDisponivel, dataSolicitado, idDoacao)
		if err != nil {
			http.Error(w, "Erro ao atualizar pagamento: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Retorno
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":          "Resgate processado com sucesso",
			"valor_disponivel": valorDisponivel,
			"resgate_total":    totalValor,
		})
	}
}

type DonationVisualizationRequest struct {
	IDDoacao       string `json:"id_doacao"`
	IDUser         string `json:"id_user"`
	Visuaization   bool   `json:"visuaization"`
	Idioma         string `json:"idioma"`
	Tema           string `json:"tema"`
	Form           string `json:"form"`
	Google         string `json:"google"`
	GoogleMaps     string `json:"google_maps"`
	GoogleAds      string `json:"google_ads"`
	MetaPixel      string `json:"meta_pixel"`
	CookiesStripe  string `json:"Cookies_Stripe"`
	CookiesPayPal  string `json:"Cookies_PayPal"`
	VisitorInfo    string `json:"visitor_info1_live"`
	DonationLike   bool   `json:"donation_like"`
	Love           bool   `json:"love"`
	Shared         bool   `json:"shared"`
	AcesseDonation bool   `json:"acesse_donation"`
	CreatePix      bool   `json:"create_pix"`
	CreateCartao   bool   `json:"create_cartao"`
	CreatePayPal   bool   `json:"create_paypal"`
	CreateGoogle   bool   `json:"create_google"`
	CreatePag1     bool   `json:"create_pag1"`
	CreatePag2     bool   `json:"create_pag2"`
	CreatePag3     bool   `json:"create_pag3"`
}

func DonationVisualization(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DonationVisualizationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON", http.StatusBadRequest)
			return
		}

		// Verifica se existe uma visualização para essa doação
		var idVisualization uuid.UUID
		err := db.QueryRow(`SELECT id FROM core.visualization WHERE id_doacao = $1`, req.IDDoacao).Scan(&idVisualization)
		if err == sql.ErrNoRows {
			idVisualization = uuid.New()
			_, err = db.Exec(`INSERT INTO core.visualization (id, id_doacao, date_create, date_update)
				VALUES ($1, $2, now(), now())`, idVisualization, req.IDDoacao)
			if err != nil {
				http.Error(w, "Erro ao criar visualização", http.StatusInternalServerError)
				return
			}
		} else if err != nil {
			http.Error(w, "Erro ao buscar visualização", http.StatusInternalServerError)
			return
		}

		// Incrementa visualizações e interações se aplicável
		if req.Visuaization || req.DonationLike || req.Love || req.Shared || req.AcesseDonation ||
			req.CreatePix || req.CreateCartao || req.CreatePayPal || req.CreateGoogle || req.CreatePag1 || req.CreatePag2 || req.CreatePag3 {

			fields := []string{}
			if req.Visuaization {
				fields = append(fields, "visualization = visualization + 1")
			}
			if req.DonationLike {
				fields = append(fields, "like = like + 1")
			}
			if req.Love {
				fields = append(fields, "love = love + 1")
			}
			if req.Shared {
				fields = append(fields, "shared = shared + 1")
			}
			if req.AcesseDonation {
				fields = append(fields, "acesse_donation = acesse_donation + 1")
			}
			if req.CreatePix {
				fields = append(fields, "create_pix = create_pix + 1")
			}
			if req.CreateCartao {
				fields = append(fields, "create_cartao = create_cartao + 1")
			}
			if req.CreatePayPal {
				fields = append(fields, "create_paypal = create_paypal + 1")
			}
			if req.CreateGoogle {
				fields = append(fields, "create_google = create_google + 1")
			}
			if req.CreatePag1 {
				fields = append(fields, "create_pag1 = create_pag1 + 1")
			}
			if req.CreatePag2 {
				fields = append(fields, "create_pag2 = create_pag2 + 1")
			}
			if req.CreatePag3 {
				fields = append(fields, "create_pag3 = create_pag3 + 1")
			}

			updateQuery := `UPDATE core.visualization SET ` +
				strings.Join(fields, ", ") + `, date_update = now() WHERE id = $1`
			_, err = db.Exec(updateQuery, idVisualization)
			if err != nil {
				http.Error(w, "Erro ao atualizar contadores", http.StatusInternalServerError)
				return
			}
		}

		// Inserir em visualization_dth
		_, err = db.Exec(`
			INSERT INTO core.visualization_dth (
				id, id_visualization, ip, id_user, idioma, tema, form, google, google_maps, google_ads,
				meta_pixel, Cookies_Stripe, Cookies_PayPal, visitor_info1_live, donation_like, love, shared,
				acesse_donation, create_pix, create_cartao, create_paypal, create_google, create_pag1,
				create_pag2, create_pag3, date_create
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17,
				$18, $19, $20, $21, $22, $23,
				$24, $25, $26
			)
		`,
			uuid.New(),
			idVisualization,
			r.RemoteAddr,
			req.IDUser,
			req.Idioma,
			req.Tema,
			req.Form,
			req.Google,
			req.GoogleMaps,
			req.GoogleAds,
			req.MetaPixel,
			req.CookiesStripe,
			req.CookiesPayPal,
			req.VisitorInfo,
			req.DonationLike,
			req.Love,
			req.Shared,
			req.AcesseDonation,
			req.CreatePix,
			req.CreateCartao,
			req.CreatePayPal,
			req.CreateGoogle,
			req.CreatePag1,
			req.CreatePag2,
			req.CreatePag3,
			time.Now(),
		)
		if err != nil {
			http.Error(w, "Erro ao registrar detalhe de visualização", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Visualização registrada com sucesso",
		})
	}
}
