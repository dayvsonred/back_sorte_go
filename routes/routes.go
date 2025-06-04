package routes

import (
	"database/sql"
	//"net/http"

	"github.com/gorilla/mux"
	"BACK_SORTE_GO/handlers"
	"BACK_SORTE_GO/middleware"
)

func SetupRoutes(db *sql.DB) *mux.Router {
	router := mux.NewRouter()
	
	// Health Check
	router.HandleFunc("/health", handlers.HealthCheckHandler()).Methods("GET")

	// Rotas de usuário
	router.HandleFunc("/users", handlers.CreateUserHandler(db)).Methods("POST")

	// Rota para fazer login com usuario email e senha retorna tokemn
	router.Handle("/login", middleware.CorsMiddleware(handlers.LoginHandler(db))).Methods("POST")

	// Rota para criar donation valida tokemn
	router.HandleFunc("/donation", handlers.DonationHandler(db)).Methods("POST")

	// Rota testa token e gerado pelo certificado e valido
	//router.HandleFunc("/testToken", handlers.TestTokenHandler()).Methods("GET")

	//rota para crair pix teste 
	router.HandleFunc("/pix/create", handlers.TestPixTokenHandler()).Methods("POST")

	return router
}
