// Server startup command (Technical Plan section 9).
// Wires configuration, database, repositories, services, handlers, and
// middlewares. It's intentionally the only place that knows about all the
// concrete pieces of the system.
//
//	@title			Financial Tracking App - Backend API
//	@version		1.0
//	@description	Backend API for personal finance tracking (accounts, monthly snapshots, exchange rates).
//
// Note: @BasePath isn't declared because /healthz lives outside /api/v1; each
// @Router below includes the full prefix it needs.
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
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/mauricioarielramirez/financial-tracking-app-backend/docs"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/config"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/http/handlers"
	appmiddleware "github.com/mauricioarielramirez/financial-tracking-app-backend/internal/http/middleware"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/quotes"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository/sqlite"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/account"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/quote"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/snapshot"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading configuration: %v", err)
	}

	if cfg.DBDriver != "sqlite3" {
		// The repository pattern (Technical Plan section 1) leaves the rest
		// of the system ready for Postgres/MySQL; the only thing missing is
		// implementing the corresponding subpackage
		// (internal/repository/postgres, etc.) and adding the case here.
		log.Fatalf("DB_DRIVER=%q is not implemented yet (only sqlite3 for now)", cfg.DBDriver)
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
		log.Fatalf("applying migrations: %v", err)
	}
	cancel()

	// Repositories
	accountRepo := sqlite.NewAccountRepository(db)
	snapshotRepo := sqlite.NewSnapshotRepository(db)

	// External exchange rate providers (RF-10)
	dolarClient := quotes.NewDolarAPIClient(cfg.DolarAPIBaseURL, cfg.DolarTipo, cfg.DolarCampo, cfg.ExternalAPITimeout)
	coinGeckoClient := quotes.NewCoinGeckoClient(cfg.CoinGeckoBaseURL, cfg.ExternalAPITimeout)
	quoteProvider := quotes.NewCompositeProvider(dolarClient, coinGeckoClient)

	// Usecases (orchestrate mapper + repository/external providers)
	accountUseCase := account.New(accountRepo, account.NewMapper())
	quoteUseCase := quote.New(quoteProvider)
	snapshotUseCase := snapshot.New(snapshotRepo, accountRepo, snapshot.NewMapper())

	// Handlers
	accountHandler := handlers.NewAccountHandler(accountUseCase)
	quoteHandler := handlers.NewQuoteHandler(quoteUseCase)
	snapshotHandler := handlers.NewSnapshotHandler(snapshotUseCase)

	router := chi.NewRouter()
	router.Use(appmiddleware.Recover)
	router.Use(appmiddleware.Logging)
	router.Use(appmiddleware.CORS)

	router.Get("/healthz", handlers.Health)
	router.Get("/swagger/*", httpSwagger.WrapHandler)

	router.Route("/api/v1", func(api chi.Router) {
		api.Use(appmiddleware.APIKey(cfg.APIKey))

		api.Route("/accounts", accountHandler.Routes)
		api.Get("/quotes/suggested", quoteHandler.GetSuggested)
		api.Route("/snapshots", snapshotHandler.Routes)

		// TODO (next layer): /income-statements, /reports/* (Technical Plan
		// section 6), following the same handler -> usecase -> repository
		// pattern as /accounts and /snapshots.
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

	// Graceful shutdown on SIGINT/SIGTERM.
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
