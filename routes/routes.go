package routes

import (
	"database/sql"
	//"net/http"

	"github.com/gorilla/mux"
	"BACK_SORTE_GO/handlers"
)

func SetupRoutes(db *sql.DB) *mux.Router {
	router := mux.NewRouter()
	
	// Health Check
	router.HandleFunc("/health", handlers.HealthCheckHandler()).Methods("GET")

	// Rotas de usuário
	router.HandleFunc("/users", handlers.CreateUserHandler(db)).Methods("POST")

	return router
}
