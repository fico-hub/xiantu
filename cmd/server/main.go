package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiantu/server/internal/api"
	"github.com/xiantu/server/internal/db"
	"github.com/xiantu/server/internal/game"
)

func main() {
	// Initialize DB
	pgURL := os.Getenv("DATABASE_URL")
	if pgURL == "" {
		pgURL = "postgresql://postgres:postgres@localhost:5432/xiantu?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "xiantu-dev-secret-change-in-production"
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := db.NewPool(ctx, pgURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	// Run migrations
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Connect to Redis
	rdb, err := db.NewRedis(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	// Start game engine (turn ticker)
	engine := game.NewEngine(pool, rdb)
	engineCtx, engineCancel := context.WithCancel(ctx)
	defer engineCancel()
	go engine.Run(engineCtx)

	// Start HTTP/WS server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := api.NewApp(pool, rdb, engine, jwtSecret)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 仙途服务启动，端口 :%s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")
	engineCancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = shutCtx

	if err := app.Shutdown(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	log.Println("Goodbye 🌙")
}
