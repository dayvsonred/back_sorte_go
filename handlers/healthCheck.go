package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthCheckHandler lida com a verificação de saúde do sistema
func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Configurar o cabeçalho da resposta como JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Resposta de sucesso com status "online"
		json.NewEncoder(w).Encode(map[string]string{
			"status": "online",
		})
	}
}
