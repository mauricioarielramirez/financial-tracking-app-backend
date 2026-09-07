module github.com/mauricioarielramirez/financial-tracking-app-backend

go 1.24.7

require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/mattn/go-sqlite3 v1.14.50
	github.com/shopspring/decimal v1.4.0
)

require go.uber.org/mock v0.6.0

replace go.uber.org/mock => github.com/uber-go/mock v0.6.0
