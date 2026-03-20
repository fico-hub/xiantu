package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xiantu/server/internal/game"
	"github.com/xiantu/server/internal/ws"
)

func NewApp(pool *pgxpool.Pool, rdb *redis.Client, engine *game.Engine, jwtSecret string) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "仙途 Xiantu v0.1",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Static files
	app.Static("/", "./public")

	h := NewHandler(pool, rdb, engine, jwtSecret)

	// HTTP API routes
	api := app.Group("/api")
	api.Post("/register", h.Register)
	api.Post("/login", h.Login)
	api.Get("/profile", h.AuthMiddleware, h.Profile)
	api.Post("/device-login/start", h.DeviceLoginStart)
	api.Post("/device-login/poll", h.DeviceLoginPoll)
	api.Post("/device-login/approve", h.AuthMiddleware, h.DeviceLoginApprove)
	api.Get("/device-login/pending", h.AuthMiddleware, h.DeviceLoginPending)

	// WebSocket
	hub := ws.NewHub(pool, rdb, engine, jwtSecret)
	go hub.Run()

	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			c.Locals("hub", hub)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", fiberws.New(func(c *fiberws.Conn) {
		hub := c.Locals("hub").(*ws.Hub)
		hub.Handle(c)
	}))

	return app
}
