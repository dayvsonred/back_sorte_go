package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv carrega as variáveis de ambiente do arquivo .env
func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente padrão.")
	}
}

// GetDatabaseURL retorna a URL de conexão com o banco de dados
func GetDatabaseURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL não definida nas variáveis de ambiente.")
	}
	return dbURL
}


// GetDatabaseURL retorna a URL de conexão com o banco de dados
func GetPortServerStart() string {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		log.Fatal("SERVER_PORT não definida nas variáveis de ambiente.")
	}
	return port
}