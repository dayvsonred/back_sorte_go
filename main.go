package main

import (
	"log"
	"net/http"

	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/database"
	"BACK_SORTE_GO/routes"
)

func main() {
	// Carregar configuração
	config.LoadEnv()

	// Conectar ao banco de dados
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Executar migrações
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Erro ao executar migrações: %v", err)
	}

	// Continuar com a inicialização normal da aplicação
	log.Println("Migrações executadas com sucesso!")

	// Configurar as rotas
	router := routes.SetupRoutes(db)

	// Iniciar o servidor
	portServerRum := config.GetPortServerStart()
	log.Println("Servidor rodando na porta :", portServerRum, "...")
	log.Fatal(http.ListenAndServe(":"+portServerRum, router))
}
