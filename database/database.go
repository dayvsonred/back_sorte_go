package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // Driver PostgreSQL
	"BACK_SORTE_GO/config"
)

// Connect cria uma conexão com o banco de dados PostgreSQL
func Connect() (*sql.DB, error) {
	dbURL := config.GetDatabaseURL()

	// Abre a conexão com o banco
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("não foi possível conectar ao banco de dados: %v", err)
	}

	// Testa a conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao testar a conexão com o banco: %v", err)
	}

	return db, nil
}
