package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pureGo/config"
	"pureGo/middleware"
	"pureGo/routes"
	"pureGo/utils"
        
)

func main() {
	// 1. Initialize Database
	config.ConnectDB()
	config.ConnectRedis()
       
       config.Migrate(config.DB)

	limiter := middleware.NewIPRateLimiter(2, 5)

	go utils.StartWorkerPool(5)

	handler := routes.SetUpRoutes()
	finalHandler := middleware.RateLimitMiddleware(limiter, handler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: finalHandler,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		log.Println(" Server is running on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %s\n", err)
		}
	}()

	<-stop
	log.Println("\n Shutdown signal received. Cleaning up...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	close(utils.JobQueue)

	utils.WG.Wait()

	log.Println(" Server exited gracefully. Goodbye!")
}
