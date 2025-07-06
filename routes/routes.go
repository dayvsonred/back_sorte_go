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

	//atualiza img do perfil do usuario
	router.HandleFunc("/users/uploadProfileImage", handlers.UploadUserProfileImageHandler(db)).Methods("POST")
	
	//get img do perfil do usuario
	router.HandleFunc("/users/ProfileImage/{id}", handlers.UserProfileImageHandler(db)).Methods("GET")

	// dados ususario criador da doação para exibir na doação
	router.HandleFunc("/users/show/{id}", handlers.UserShowHandler(db)).Methods("GET")

	//Muda nome usuario
	router.HandleFunc("/users/nameChange", handlers.UserNameChangeHandler(db)).Methods("POST")

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

	//buscar as mensagens visíveis da doação
	router.HandleFunc("/donation/mensagem", handlers.DonationMensagesHandler(db)).Methods("GET")

	//encerra envento de doação 
	router.HandleFunc("/donation/closed/{id}", handlers.DonationClosedHandler(db)).Methods("GET")

	//prepara os valores para serem enviado para o criado da doação e bloquei visualização da doação
	router.HandleFunc("/donation/rescue/{id}", handlers.DonationRescueHandler(db)).Methods("GET")

	// registra lod de atividades na doação 
	router.HandleFunc("/donation/visualization", handlers.DonationVisualization(db)).Methods("POST")

	// Rota testa token e gerado pelo certificado e valido
	//router.HandleFunc("/testToken", handlers.TestTokenHandler()).Methods("GET")

	//rota para crair pix teste 
	router.HandleFunc("/pix/create", handlers.CreatePixTokenHandler(db)).Methods("POST")

	router.HandleFunc("/pix/status/{txid}", handlers.PixChargeStatusHandler()).Methods("GET")
	
	router.HandleFunc("/pix/monitora/{txid}", handlers.MonitorarStatusPagamentoHandler(db)).Methods("POST")

	// valor total da doação e total de doadores 
	router.HandleFunc("/pix/total/{id}", handlers.DonationSummaryByIDHandler(db)).Methods("GET")

	// inicializar busca de todo os pagamento com status em andamento não finalizado ainda com prazo de venciamnete ativos pendeentes de verificação 
	router.HandleFunc("/pix/monitora/all", handlers.MonitorarStatusAllPagamentosHandler(db)).Methods("GET")

	return router
}
