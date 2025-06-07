package routes

import (
	"database/sql"
	"BACK_SORTE_GO/handlers"
	"github.com/gorilla/mux"
)

func SetupRoutes(db *sql.DB) *mux.Router {
	router := mux.NewRouter()
	
	// Health Check
	router.HandleFunc("/health", handlers.HealthCheckHandler()).Methods("GET")

	// Rotas de usuário
	router.HandleFunc("/users", handlers.CreateUserHandler(db)).Methods("POST")

	// Rota para fazer login com usuario email e senha retorna tokemn
	//router.Handle("/login", middleware.CorsMiddleware(handlers.LoginHandler(db))).Methods("POST")
	router.HandleFunc("/login", handlers.LoginHandler(db)).Methods("POST")

	// Rota para criar donation valida tokemn
	router.HandleFunc("/donation", handlers.DonationHandler(db)).Methods("POST")

	// Listar doações por usuário com paginação
	router.HandleFunc("/donation/list", handlers.DonationListByIDUserHandler(db)).Methods("GET")

	//deleta doações
	router.HandleFunc("/donation/{id}", handlers.DonationDellHandler(db)).Methods("DELETE")

	// Rota testa token e gerado pelo certificado e valido
	//router.HandleFunc("/testToken", handlers.TestTokenHandler()).Methods("GET")

	//rota para crair pix teste 
	router.HandleFunc("/pix/create", handlers.TestPixTokenHandler()).Methods("POST")

	return router
}
