package application

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"businessapp/backend/internal/config"
	"businessapp/backend/internal/httpapi"
	"businessapp/backend/internal/store"
)

func Run() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := store.New(ctx, cfg.MongoURI, cfg.Database)
	if err != nil {
		log.Fatalf("connect to MongoDB: %v", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = db.Close(closeCtx)
	}()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.New(db, cfg.JWTSecret, cfg.AppOrigin).Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("API listening on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}
