package handlers

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings" 
)

// TokenResponse representa a resposta da API ao solicitar um token
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// LoadPEMCert carrega o certificado e a chave privada do formato .pem
func LoadPEMCert(certPath, keyPath, caCertPath string) (*tls.Certificate, *x509.CertPool, error) {
	// Verifica se os arquivos existem
	for _, path := range []string{certPath, keyPath, caCertPath} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("arquivo não encontrado: %s", path)
		}
	}

	// Carregar o certificado e a chave privada
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao carregar certificado/key PEM: %v", err)
	}

	// Carregar a CA (opcional, depende da API)
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler CA cert: %v", err)
	}

	// Criar um pool de CA
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, nil, fmt.Errorf("falha ao adicionar CA ao pool")
	}

	return &cert, certPool, nil
}

// GetEfiBankToken faz a requisição para obter um token de acesso
func GetEfiBankToken() (*TokenResponse, error) {
	// Configurações do certificado e credenciais
	certPath := "certs/certificado.pem"  // Caminho do certificado
	keyPath := "certs/newfile.key.pem"       // Caminho da chave privada
	caCertPath := "certs/newfile.crt.pem"         // Caminho do certificado da CA (opcional)
	clientID := "Client_Id_8d9c5e8e8b4f172c2abacab4ca4ae5a8e4dfb3a8"
	clientSecret := "Client_Secret_fb12f2d0973e944472ff69e47ef9bcdf2349753b"

	// Verificar arquivos de certificado
	cert, certPool, err := LoadPEMCert(certPath, keyPath, caCertPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar certificados: %v", err)
	}

	// Criar credencial Base64 (ClientID:ClientSecret)
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	// Configurar transporte TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		RootCAs:      certPool,
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}

	// Criar corpo da requisição
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Criar requisição HTTP
	req, err := http.NewRequest("POST", "https://pix-h.api.efipay.com.br/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Definir headers
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+auth)

	// Enviar requisição
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar requisição: %v", err)
	}
	defer resp.Body.Close()

	// Ler resposta
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler a resposta: %v", err)
	}

	// Processar JSON de resposta
	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("erro ao processar JSON: %v", err)
	}

	return &tokenResp, nil
}

// TestTokenHandler é um endpoint que chama GetEfiBankToken e retorna o token
func TestTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenResp := "{}"
		fmt.Println("resss:", tokenResp)
		 
		// Retornar o token como JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}
}
