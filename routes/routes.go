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

	// Muda password
	router.HandleFunc("/users/passwordChange", handlers.UserPasswordChangeHandler(db)).Methods("POST")

	// registra conta bancaria de recebimento 
	router.HandleFunc("/users/bankAccount", handlers.UserBankAccountHandler(db)).Methods("POST")

	// alterar conta bancaria de recebimento 
	router.HandleFunc("/users/bankAccount", handlers.UserBankAccountUpdateHandler(db)).Methods("PATCH")

	// Busca conta bancaria de recebimento 
	router.HandleFunc("/users/bankAccount", handlers.UserBankAccountGetHandler(db)).Methods("GET")

	// Rota para fazer login com usuario email e senha retorna tokemn
	//router.Handle("/login", middleware.CorsMiddleware(handlers.LoginHandler(db))).Methods("POST")
	router.HandleFunc("/login", handlers.LoginHandler(db)).Methods("POST")

	// Rota para criar donation valida tokemn
	router.HandleFunc("/donation", handlers.DonationHandler(db)).Methods("POST")

	// Listar doações por usuário com paginação
	router.HandleFunc("/donation/list", handlers.DonationListByIDUserHandler(db)).Methods("GET")

	//deleta doações
	router.HandleFunc("/donation/{id}", handlers.DonationDellHandler(db)).Methods("DELETE")

	//Buscar doação por nome link
	router.HandleFunc("/donation/link/{nome_link}", handlers.DonationByLinkHandler(db)).Methods("GET")

	// Rota testa token e gerado pelo certificado e valido
	//router.HandleFunc("/testToken", handlers.TestTokenHandler()).Methods("GET")

	//rota para crair pix teste 
	router.HandleFunc("/pix/create", handlers.CreatePixTokenHandler()).Methods("POST")

	router.HandleFunc("/pix/status/{txid}", handlers.PixChargeStatusHandler()).Methods("GET")

	return router
}
