package routes

import (
	"database/sql"
	"net/http"
	//"net/http"

	"BACK_SORTE_GO/handlers"
	"BACK_SORTE_GO/middleware"

	"github.com/gorilla/mux"
)

func SetupRoutes(db *sql.DB) *mux.Router {
	router := mux.NewRouter()
	
	// Health Check
	router.HandleFunc("/health", handlers.HealthCheckHandler()).Methods("GET")

	// Rotas de usuário
	router.HandleFunc("/users", handlers.CreateUserHandler(db)).Methods("POST")

	// Rota para fazer login com usuario email e senha retorna tokemn
	router.Handle("/login", middleware.CorsMiddleware(handlers.LoginHandler(db))).Methods("POST")

	router.Handle("/login", middleware.CorsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	}))).Methods("OPTIONS")

	// Rota para criar donation valida tokemn
	router.HandleFunc("/donation", handlers.DonationHandler(db)).Methods("POST")

	// Rota testa token e gerado pelo certificado e valido
	//router.HandleFunc("/testToken", handlers.TestTokenHandler()).Methods("GET")

	//rota para crair pix teste 
	router.HandleFunc("/pix/create", handlers.TestPixTokenHandler()).Methods("POST")

	return router
}
