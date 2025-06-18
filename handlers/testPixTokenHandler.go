package handlers

import (
	"BACK_SORTE_GO/config"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/efipay/sdk-go-apis-efi/src/efipay/pix"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// PixChargeRequest define a estrutura do JSON recebido na requisição
type PixChargeRequest struct {
	Valor    string `json:"valor"`
	CPF     string `json:"cpf"`
	Nome     string `json:"nome"`
	Chave    string `json:"chave"`
	Mensagem string `json:"mensagem"`
	Anonimo	 bool  `json:"anonimo"`
	IdDoacao string `json:"id"`
}

// parseTime faz parse de string ISO para time.Time
func parseTimeISO(v interface{}) time.Time {
	if v == nil {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, v.(string))
	if err != nil {
		return time.Now()
	}
	return t
}

// TestPixTokenHandler cria uma cobrança PIX ao receber uma requisição HTTP
func CreatePixTokenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}

		var req PixChargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		efi := pix.NewEfiPay(config.GetCredentials())

		body := map[string]interface{}{
			"calendario": map[string]interface{}{"expiracao": 3600},
			"devedor": map[string]interface{}{
				"cpf":  req.CPF,
				"nome": req.Nome,
			},
			"valor":              map[string]interface{}{"original": req.Valor},
			"chave":              req.Chave,
			"solicitacaoPagador": "pagamento de doação",
		}

		// Chamada da API
		resStr, err := efi.CreateImmediateCharge(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao criar cobrança PIX: %v", err), http.StatusInternalServerError)
			return
		}

		// Converte resposta em map
		var resMap map[string]interface{}
		if err := json.Unmarshal([]byte(resStr), &resMap); err != nil {
			http.Error(w, "Erro ao decodificar resposta do PIX: "+err.Error(), http.StatusInternalServerError)
			return
		}

		txid, ok := resMap["txid"].(string)
		if !ok || txid == "" {
			http.Error(w, "Resposta inválida da API (txid ausente)", http.StatusInternalServerError)
			return
		}

		// Inicia transação
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Erro ao iniciar transação: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		idPixQRCode := uuid.New()

		// Insert pix_qrcode
		_, err = tx.Exec(`
			INSERT INTO core.pix_qrcode 
			(id, id_doacao, valor, cpf, nome, mensagem, anonimo, visivel, data_criacao)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		`,
			idPixQRCode,
			req.IdDoacao,
			req.Valor,
			req.CPF,
			req.Nome,
			req.Mensagem,
			req.Anonimo,
			false,
		)
		if err != nil {
			http.Error(w, "Erro ao salvar pix_qrcode: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Salvando status para id_pix_qrcode: %v\n", idPixQRCode)

		// Insert pix_qrcode_status
		_, err = tx.Exec(`
			INSERT INTO core.pix_qrcode_status (
				id, id_pix_qrcode, data_criacao, expiracao, tipo_pagamento,
				loc_id, loc_tipo_cob, loc_criacao, location, pix_copia_e_cola,
				chave, id_pix, status, buscar, finalizado, data_pago
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16
			)
		`,
			uuid.NewString(),
			idPixQRCode,
			parseTimeISO(resMap["calendario"].(map[string]interface{})["criacao"]),
			int(resMap["calendario"].(map[string]interface{})["expiracao"].(float64)),
			"v1",
			int(resMap["loc"].(map[string]interface{})["id"].(float64)),
			resMap["loc"].(map[string]interface{})["tipoCob"],
			parseTimeISO(resMap["loc"].(map[string]interface{})["criacao"]),
			resMap["loc"].(map[string]interface{})["location"],
			resMap["loc"].(map[string]interface{})["location"], // ou outro campo
			req.Chave,
			txid,
			resMap["status"],
			true,
			false,
			nil,
		)
		if err != nil {
			http.Error(w, "Erro ao salvar pix_qrcode_status: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Commit transação
		if err := tx.Commit(); err != nil {
			http.Error(w, "Erro ao commitar transação: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Iniciar verificação de status em background (não bloqueia)
		go func(txid string) {
			err := IniciarMonitoramentoStatusPagamento(db, txid)
			if err != nil {
				fmt.Println("Erro ao iniciar monitoramento do pagamento:", err)
			}
		}(txid)

		// Retorno da API
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resStr))
	}
}

func PixChargeStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		txid := vars["txid"]
		if txid == "" {
			http.Error(w, "txid é obrigatório", http.StatusBadRequest)
			return
		}

		// Obtém as credenciais do config.go
		//credentials := Credentials
		credentials := config.GetCredentials()
		fmt.Printf("Credentials: %+v\n", credentials)
		efi := pix.NewEfiPay(credentials)

		//efi := pix.NewEfiPay(config.GetCredentials())

		// Consulta o status da cobrança PIX
		res, err := efi.DetailCharge(txid)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao consultar status do PIX: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(res)) // Retorna a resposta original da API
	}
}

//executa monitoramente de pagamento apos chamada da função por param
func IniciarMonitoramentoStatusPagamento(db *sql.DB, txid string) error {
	checkInterval := []time.Duration{30 * time.Second, 1 * time.Minute}
	attempts := []int{10, 21}

	for phase := 0; phase < 2; phase++ {
		for i := 0; i < attempts[phase]; i++ {
			fmt.Printf("Verificando status para txid: %s (tentativa %d/%d)\n", txid, i+1, attempts[phase])

			status, err := consultarStatusPix(txid)
			if err != nil {
				fmt.Println("Erro ao consultar status PIX:", err)
				return err
			}

			if status == "CONCLUIDA" {
				atualizarStatusPagamento(db, txid)
				return nil
			}

			time.Sleep(checkInterval[phase])
		}
	}

	fmt.Println("Verificações encerradas sem pagamento concluído para:", txid)
	marcarPagamentoVencido(db, txid)
	return nil
}

// executa monitoramente de pagamento apos chamada por POST
func MonitorarStatusPagamentoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		txid := vars["txid"]
		if txid == "" {
			http.Error(w, "txid é obrigatório", http.StatusBadRequest)
			return
		}

		go func() {
			checkInterval := []time.Duration{30 * time.Second, 1 * time.Minute}
			attempts := []int{10, 21}

			for phase := 0; phase < 2; phase++ {
				for i := 0; i < attempts[phase]; i++ {
					fmt.Printf("Verificando status para txid: %s (tentativa %d/%d)\n", txid, i+1, attempts[phase])

					status, err := consultarStatusPix(txid)
					if err != nil {
						fmt.Println("Erro ao consultar status PIX:", err)
						break
					}

					if status == "CONCLUIDA" {
						// Atualizar banco de dados
						atualizarStatusPagamento(db, txid)
						return
					}

					time.Sleep(checkInterval[phase])
				}
			}
			fmt.Println("Verificações encerradas sem pagamento concluído para:", txid)
			marcarPagamentoVencido(db, txid) // ✅ Adicionando chamada para marcar como VENCIDO
		}()

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Monitoramento iniciado"))
	}
}

func consultarStatusPix(txid string) (string, error) {
	//fmt.Println("Verifica status pix para id:", txid)
	efi := pix.NewEfiPay(config.GetCredentials())
	res, err := efi.DetailCharge(txid)
	if err != nil {
		return "", err
	}

	var resMap map[string]interface{}
	if err := json.Unmarshal([]byte(res), &resMap); err != nil {
		return "", err
	}

	status, ok := resMap["status"].(string)
	if !ok {
		return "", fmt.Errorf("status não encontrado na resposta")
	}

	return status, nil
}

func atualizarStatusPagamento(db *sql.DB, txid string) {
	now := time.Now()
	fmt.Println("Atializa pagamento confirmado pix para id:", txid)
	// Atualiza core.pix_qrcode_status
	_, err := db.Exec(`
		UPDATE core.pix_qrcode_status
		SET status = 'CONCLUIDA', buscar = false, finalizado = true, data_pago = $1
		WHERE id_pix = $2
	`, now, txid)
	if err != nil {
		fmt.Println("Erro ao atualizar pix_qrcode_status:", err)
		return
	}

	// Atualiza core.pix_qrcode (assumindo que temos o id_pix_qrcode vinculado ao txid)
	_, err = db.Exec(`
		UPDATE core.pix_qrcode
		SET visivel = true
		WHERE id = (
			SELECT id_pix_qrcode
			FROM core.pix_qrcode_status
			WHERE id_pix = $1
			LIMIT 1
		)
	`, txid)
	if err != nil {
		fmt.Println("Erro ao atualizar pix_qrcode:", err)
	}
}

func marcarPagamentoVencido(db *sql.DB, txid string) {
	fmt.Println("Atializa Vencido pix para id:", txid)
	_, err := db.Exec(`
		UPDATE core.pix_qrcode_status
		SET status = 'VENCIDO', buscar = false
		WHERE id_pix = $1
	`, txid)
	if err != nil {
		fmt.Println("Erro ao marcar cobrança como vencida:", err)
	}
}
