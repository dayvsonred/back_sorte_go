package handlers

import (
	"BACK_SORTE_GO/config"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/efipay/sdk-go-apis-efi/src/efipay/pix"
	"github.com/gorilla/mux"
)

// PixChargeRequest define a estrutura do JSON recebido na requisição
type PixChargeRequest struct {
	Valor    string `json:"valor"`
	CNPJ     string `json:"cnpj"`
	Nome     string `json:"nome"`
	Chave    string `json:"chave"`
	Mensagem string `json:"mensagem"`
	Anonimo	 bool  `json:"anonimo"`
	IdDoacao string `json:"id"`
}


// parseTime faz parse de string ISO para time.Time
func parseTime(v interface{}) time.Time {
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
func CreatePixTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica se o método é POST
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}

		// Decodifica o JSON da requisição
		var req PixChargeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Erro ao decodificar JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Obtém as credenciais do config.go
		//credentials := Credentials
		credentials := config.GetCredentials()
		fmt.Printf("Credentials: %+v\n", credentials)
		efi := pix.NewEfiPay(credentials)

		// Monta o corpo da requisição
		body := map[string]interface{}{
			"calendario": map[string]interface{}{
				"expiracao": 3600,
			},
			"devedor": map[string]interface{}{
				"cpf":  req.CNPJ,
				"nome": req.Nome,
			},
			"valor": map[string]interface{}{
				"original": req.Valor,
			},
			"chave":              req.Chave,
			"solicitacaoPagador": req.Mensagem,
		}

		// Chama a API para criar a cobrança PIX
		//res, err := efi.CreateCharge(req.TxID, body)
		res, err := efi.CreateImmediateCharge(body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao criar cobrança PIX: %v", err), http.StatusInternalServerError)
			return
		}

		// Retorna a resposta da API como JSON
		w.Header().Set("Content-Type", "application/json")
		//fmt.Println(string(res))

		// Escreve o JSON diretamente no ResponseWriter
		w.Write([]byte(res))
		/*if err := json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, fmt.Sprintf("Erro ao codificar JSON: %v", err), http.StatusInternalServerError)
		}*/
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
