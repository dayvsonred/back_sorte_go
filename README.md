# back_sorte_go
back end golan donation


go mod tidy

go mod init myapi

go run main.go

GOPATH

%USERPROFILE%\go


fmt.Println(string(jsonData))



https://pix-h.api.efipay.com.br




C:\Users\niore\Documents\projeto sorteio doacao\converte_p12_to_pem\conversor-p12-efi\homologacao-678045-donation_2.p12


Q2xpZW50X0lkXzhkOWM1ZThlOGI0ZjE3MmMyYWJhY2FiNGNhNGFlNWE4ZTRkZmIzYT hDbGllbnRfU2VjcmV0X2ZiMTJmMmQwOTczZTk0NDQ3MmZmNjllNDdlZjliY2RmMjM0OTc1M2I=
Q2xpZW50X0lkXzhkOWM1ZThlOGI0ZjE3MmMyYWJhY2FiNGNhNGFlNWE4ZTRkZmIzYT g6Q2xpZW50X1NlY3JldF9mYjEyZjJkMDk3M2U5NDQ0NzJmZjY5ZTQ3ZWY5YmNkZjIzNDk3NTNi


Client_Id_8d9c5e8e8b4f172c2abacab4ca4ae5a8e4dfb3a8
Client_Id_8d9c5e8e8b4f172c2abacab4ca4ae5a8e4dfb3a8

Client_Secret_fb12f2d0973e944472ff69e47ef9bcdf2349753b
Client_Secret_fb12f2d0973e944472ff69e47ef9bcdf2349753b

arquivo .pem
openssl pkcs12 -in homologacao-678045-donation_2.p12 -out certificado.pem -clcerts -nokeys

#certificado
openssl pkcs12 -in homologacao-678045-donation_2.p12 -out newfile.crt.pem -clcerts -nokeys -password pass:"" 

#chave privada
openssl pkcs12 -in homologacao-678045-donation_2.p12 -out newfile.key.pem -nocerts -nodes -password pass:"" 

package configs

var Credentials = map[string]interface{} {
	"client_id": "Client_Id_8d9c5e8e8b4f172c2abacab4ca4ae5a8e4dfb3a8",
    "client_secret": "Client_Secret_fb12f2d0973e944472ff69e47ef9bcdf2349753b",
    "sandbox": false,
    "timeout": 20,
    "CA" : "certs/newfile.crt.pem", //caminho da chave publica da gerencianet
    "Key" : "certs/newfile.key.pem", //caminho da chave privada da sua conta Gerencianet
}