package handlers

import (
	"BACK_SORTE_GO/config"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/efipay/sdk-go-apis-efi/src/efipay/pix"
)

// PixChargeRequest define a estrutura do JSON recebido na requisição
type PixChargeRequest struct {
	TxID   string  `json:"txid"`
	Valor  string  `json:"valor"`
	CNPJ   string  `json:"cnpj"`
	Nome   string  `json:"nome"`
	Chave  string  `json:"chave"`
	Mensagem string `json:"mensagem"`
}

// TestPixTokenHandler cria uma cobrança PIX ao receber uma requisição HTTP
func TestPixTokenHandler() http.HandlerFunc {
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
		
		// GetCredentials retorna as credenciais carregadas das variáveis de ambiente
		// var Credentials = map[string]interface{}{
		//	"client_id":     config.Getclient_id(),
		//	"client_secret": os.Getenv("CLIENT_SECRET"),
		//	"sandbox":       strconv.ParseBool(os.Getenv("SANDBOX")), // Converte string para bool
		//	"timeout":       os.Getenv("TIMEOUT"),
		//	"CA":            os.Getenv("CA_PEM"),
		//	"Key":           os.Getenv("KEY_PEM"),
		// }

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
				"cnpj": req.CNPJ,
				"nome": req.Nome,
			},
			"valor": map[string]interface{}{
				"original": req.Valor,
			},
			"chave":              req.Chave,
			"solicitacaoPagador": req.Mensagem,
			"infoAdicionais": []map[string]interface{}{
				{
					"nome":  "Campo 1",
					"valor": "Informação Adicional1 do PSP-Recebedor",
				},
			},
		}

		// Chama a API para criar a cobrança PIX
		res, err := efi.CreateCharge(req.TxID, body)
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
