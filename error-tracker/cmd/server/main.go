package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"error-tracker/internal/handler"
	"error-tracker/internal/middleware"
	"error-tracker/internal/store"
)

func main() {
	dbPath := "./errors.db"
	if v := os.Getenv("DB_PATH"); v != "" {
		dbPath = v
	}
	db, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer db.Close()

	h := handler.NewReportHandler(db)

	r := gin.Default()
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	{
		api.POST("/report", h.Report)
		api.GET("/events", h.GetEvents)
		api.GET("/stats", h.GetStats)
		api.GET("/trend", h.GetTrend)
	}

	// serve SDK and example static files
	r.Static("/sdk", "./web/sdk")
	r.Static("/example", "./example")
	r.Static("/dashboard", "./web/dashboard")

	log.Println("Server starting on :8080")
	srv := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// give in-flight requests 5s to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}

	// flush remaining buffered events
	h.Close()
	log.Println("Buffer flushed, bye.")
}
