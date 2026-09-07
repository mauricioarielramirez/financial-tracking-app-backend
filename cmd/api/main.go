// Comando de arranque del servidor HTTP (Plan Técnico sección 9).
// Cablea configuración, base de datos, repositorios, servicios, handlers y
// middlewares. Es intencionalmente el único lugar que conoce todas las
// piezas concretas del sistema.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/config"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/http/handlers"
	appmiddleware "github.com/mauricioarielramirez/financial-tracking-app-backend/internal/http/middleware"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/quotes"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository/sqlite"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/account"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/quote"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("cargando configuración: %v", err)
	}

	if cfg.DBDriver != "sqlite3" {
		// El repository pattern (Plan Técnico sección 1) deja el resto del
		// sistema listo para Postgres/MySQL; sólo falta implementar el
		// subpaquete correspondiente (internal/repository/postgres, etc.)
		// y agregar el caso acá.
		log.Fatalf("DB_DRIVER=%q todavía no tiene implementación (sólo sqlite3 por ahora)", cfg.DBDriver)
	}

	db, err := sqlite.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("conectando a la base: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error cerrando la base: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := sqlite.Migrate(ctx, db, cfg.MigrationsPath); err != nil {
		cancel()
		log.Fatalf("aplicando migraciones: %v", err)
	}
	cancel()

	// Repositorios
	accountRepo := sqlite.NewAccountRepository(db)

	// Proveedores externos de cotización (RF-10)
	dolarClient := quotes.NewDolarAPIClient(cfg.DolarAPIBaseURL, cfg.DolarTipo, cfg.DolarCampo, cfg.ExternalAPITimeout)
	coinGeckoClient := quotes.NewCoinGeckoClient(cfg.CoinGeckoBaseURL, cfg.ExternalAPITimeout)
	quoteProvider := quotes.NewCompositeProvider(dolarClient, coinGeckoClient)

	// Usecases (orquestan mapper + repository/proveedores externos)
	accountUseCase := account.New(accountRepo, account.NewMapper())
	quoteUseCase := quote.New(quoteProvider)

	// Handlers
	accountHandler := handlers.NewAccountHandler(accountUseCase)
	quoteHandler := handlers.NewQuoteHandler(quoteUseCase)

	router := chi.NewRouter()
	router.Use(appmiddleware.Recover)
	router.Use(appmiddleware.Logging)
	router.Use(appmiddleware.CORS)

	router.Get("/healthz", handlers.Health)

	router.Route("/api/v1", func(api chi.Router) {
		api.Use(appmiddleware.APIKey(cfg.APIKey))

		api.Route("/accounts", accountHandler.Routes)
		api.Get("/quotes/suggested", quoteHandler.GetSuggested)

		// TODO (próxima capa): /snapshots, /income-statements, /reports/*
		// (Plan Técnico sección 6), siguiendo el mismo patrón
		// handler -> service -> repository que /accounts.
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("escuchando en :%s (DB: %s)", cfg.HTTPPort, cfg.DBDSN)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error en el servidor HTTP: %v", err)
		}
	}()

	// Apagado ordenado ante SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("apagando servidor...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("error apagando servidor: %v", err)
	}
}
