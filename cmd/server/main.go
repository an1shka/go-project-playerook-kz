package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/an1shka/go-project-playerook-kz/config"
	"github.com/an1shka/go-project-playerook-kz/database"
	"github.com/an1shka/go-project-playerook-kz/handlers"
	"github.com/an1shka/go-project-playerook-kz/routes"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	postgres, err := database.ConnectPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer postgres.Close()
	if err := database.SeedDemoMarketplace(ctx, postgres); err != nil {
		log.Fatal(err)
	}
	mongoDB, err := database.ConnectMongo(ctx, cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := mongoDB.Client().Disconnect(context.Background()); err != nil {
			log.Printf("disconnect MongoDB: %v", err)
		}
	}()
	if err := database.SeedNews(ctx, mongoDB); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	routes.Register(router, handlers.New(postgres, mongoDB, cfg.JWTSecret))
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("Playerook KZ listening on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
