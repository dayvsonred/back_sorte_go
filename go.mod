module BACK_SORTE_GO

go 1.23.0

toolchain go1.23.4

require (
	github.com/golang-jwt/jwt/v4 v4.5.1
	github.com/google/uuid v1.3.0 // Para geração de UUIDs
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9 // Driver PostgreSQL
	golang.org/x/crypto v0.32.0
)

require (
	github.com/efipay/sdk-go-apis-efi v0.0.0-20231207185217-6dca10834f8f
	golang.org/x/text v0.26.0
)

replace github.com/lib/pq => github.com/lib/pq v1.10.9
