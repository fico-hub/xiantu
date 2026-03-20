package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xiantu/server/internal/auth"
	"github.com/xiantu/server/internal/game"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db        *pgxpool.Pool
	rdb       *redis.Client
	engine    *game.Engine
	jwtSecret string
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client, engine *game.Engine, jwtSecret string) *Handler {
	return &Handler{db: db, rdb: rdb, engine: engine, jwtSecret: jwtSecret}
}

// AuthMiddleware extracts and validates JWT
func (h *Handler) AuthMiddleware(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	if tokenStr == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Missing or invalid token"})
	}

	claims, err := auth.ParseToken(tokenStr, h.jwtSecret)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
	}

	c.Locals("playerID", claims.PlayerID)
	c.Locals("agentID", claims.AgentID)
	return c.Next()
}

// POST /api/register
func (h *Handler) Register(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username and password required"})
	}
	if len(req.Username) < 3 || len(req.Username) > 20 {
		return c.Status(400).JSON(fiber.Map{"error": "username must be 3-20 chars"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	// Roll spirit root
	rootName, multiplier := game.RollSpiritRoot()
	agentID := "agt-" + uuid.New().String()[:12]

	ctx := context.Background()
	var playerID string
	err = h.db.QueryRow(ctx,
		`INSERT INTO players (username, password_hash, agent_id, spirit_root, spirit_root_multiplier)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.Username, string(hash), agentID, rootName, multiplier,
	).Scan(&playerID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return c.Status(409).JSON(fiber.Map{"error": "username already exists"})
		}
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("db error: %v", err)})
	}

	token, expiresAt, err := auth.GenerateToken(playerID, agentID, h.jwtSecret)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "token error"})
	}

	rootDisplayName := game.SpiritRootNames[rootName]

	return c.Status(201).JSON(fiber.Map{
		"playerId":           playerID,
		"agentId":            agentID,
		"token":              token,
		"expiresAt":          expiresAt,
		"spiritRoot":         rootName,
		"spiritRootName":     rootDisplayName,
		"spiritRootMultiplier": multiplier,
		"message":            fmt.Sprintf("🎊 恭喜！你的血脉为【%s】，修炼速度×%.1f，欢迎来到黑人修仙传！", rootDisplayName, multiplier),
	})
}

// POST /api/login
func (h *Handler) Login(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	ctx := context.Background()
	var playerID, passwordHash, agentID string
	err := h.db.QueryRow(ctx,
		`SELECT id, password_hash, agent_id FROM players WHERE username=$1`,
		req.Username,
	).Scan(&playerID, &passwordHash, &agentID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token, expiresAt, err := auth.GenerateToken(playerID, agentID, h.jwtSecret)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "token error"})
	}

	return c.JSON(fiber.Map{
		"token":     token,
		"expiresAt": expiresAt,
		"playerId":  playerID,
		"agentId":   agentID,
	})
}

// GET /api/profile
func (h *Handler) Profile(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	ctx := context.Background()

	var p struct {
		ID              string    `json:"id"`
		Username        string    `json:"username"`
		AgentID         string    `json:"agentId"`
		SpiritRoot      string    `json:"spiritRoot"`
		SpiritRootName  string    `json:"-"`
		Multiplier      float64   `json:"spiritRootMultiplier"`
		Realm           string    `json:"realm"`
		RealmLevel      int       `json:"realmLevel"`
		SpiritStone     int64     `json:"spiritStone"`
		SpiritHerb      int64     `json:"spiritHerb"`
		MysticIron      int64     `json:"mysticIron"`
		SpiritWood      int64     `json:"spiritWood"`
		CultivationXP   int64     `json:"cultivationXp"`
		IsCultivating   bool      `json:"isCultivating"`
		JoinedEpoch     bool      `json:"joinedEpoch"`
		CreatedAt       time.Time `json:"createdAt"`
	}

	err := h.db.QueryRow(ctx,
		`SELECT id, username, agent_id, spirit_root, spirit_root_multiplier, realm, realm_level,
		 spirit_stone, spirit_herb, mystic_iron, spirit_wood, cultivation_xp, is_cultivating, joined_epoch, created_at
		 FROM players WHERE id=$1`,
		playerID,
	).Scan(&p.ID, &p.Username, &p.AgentID, &p.SpiritRoot, &p.Multiplier, &p.Realm, &p.RealmLevel,
		&p.SpiritStone, &p.SpiritHerb, &p.MysticIron, &p.SpiritWood, &p.CultivationXP,
		&p.IsCultivating, &p.JoinedEpoch, &p.CreatedAt)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "player not found"})
	}

	p.SpiritRootName = game.SpiritRootNames[p.SpiritRoot]

	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	realmInfo := game.Realms[p.Realm]

	return c.JSON(fiber.Map{
		"id":                   p.ID,
		"username":             p.Username,
		"agentId":              p.AgentID,
		"spiritRoot":           p.SpiritRoot,
		"spiritRootName":       game.SpiritRootNames[p.SpiritRoot],
		"spiritRootMultiplier": p.Multiplier,
		"realm":                p.Realm,
		"realmName":            realmInfo.Name,
		"realmLevel":           p.RealmLevel,
		"resources": fiber.Map{
			"spiritStone":   p.SpiritStone,
			"spiritHerb":    p.SpiritHerb,
			"mysticIron":    p.MysticIron,
			"spiritWood":    p.SpiritWood,
			"cultivationXp": p.CultivationXP,
		},
		"xpToBreakthrough": game.BreakthroughXP(p.Realm, p.RealmLevel),
		"isCultivating":    p.IsCultivating,
		"joinedEpoch":      p.JoinedEpoch,
		"currentTurn":      currentTurn,
		"createdAt":        p.CreatedAt,
	})
}

// POST /api/device-login/start
func (h *Handler) DeviceLoginStart(c *fiber.Ctx) error {
	var req struct {
		AgentID    string `json:"agentId"`
		DeviceName string `json:"deviceName"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.AgentID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "agentId required"})
	}

	ctx := context.Background()
	// Verify agent exists
	var count int
	err := h.db.QueryRow(ctx, "SELECT COUNT(*) FROM players WHERE agent_id=$1", req.AgentID).Scan(&count)
	if err != nil || count == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "agent not found"})
	}

	var requestID string
	err = h.db.QueryRow(ctx,
		`INSERT INTO device_login_requests (agent_id, device_name) VALUES ($1, $2) RETURNING id`,
		req.AgentID, req.DeviceName,
	).Scan(&requestID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	return c.JSON(fiber.Map{
		"requestId": requestID,
		"message":   "waiting for approval from existing device",
		"expiresIn": "10 minutes",
	})
}

// POST /api/device-login/poll
func (h *Handler) DeviceLoginPoll(c *fiber.Ctx) error {
	var req struct {
		RequestID string `json:"requestId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	ctx := context.Background()
	var status, token string
	var expiresAt time.Time
	err := h.db.QueryRow(ctx,
		`SELECT status, COALESCE(token,''), expires_at FROM device_login_requests WHERE id=$1`,
		req.RequestID,
	).Scan(&status, &token, &expiresAt)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "request not found"})
	}

	if time.Now().After(expiresAt) {
		return c.Status(410).JSON(fiber.Map{"error": "request expired"})
	}

	switch status {
	case "approved":
		return c.JSON(fiber.Map{"status": "approved", "token": token})
	case "pending":
		return c.JSON(fiber.Map{"status": "pending"})
	default:
		return c.Status(410).JSON(fiber.Map{"error": "request expired or rejected"})
	}
}

// POST /api/device-login/approve
func (h *Handler) DeviceLoginApprove(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	agentID := c.Locals("agentID").(string)

	var req struct {
		RequestID string `json:"requestId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	ctx := context.Background()
	// Verify request belongs to same agent
	var reqAgentID string
	err := h.db.QueryRow(ctx,
		`SELECT agent_id FROM device_login_requests WHERE id=$1 AND status='pending'`,
		req.RequestID,
	).Scan(&reqAgentID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "pending request not found"})
	}

	if reqAgentID != agentID {
		return c.Status(403).JSON(fiber.Map{"error": "not your agent"})
	}

	token, expiresAt, err := auth.GenerateToken(playerID, agentID, h.jwtSecret)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "token error"})
	}

	_, err = h.db.Exec(ctx,
		`UPDATE device_login_requests SET status='approved', token=$1 WHERE id=$2`,
		token, req.RequestID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	return c.JSON(fiber.Map{
		"status":    "approved",
		"expiresAt": expiresAt,
	})
}

// GET /api/device-login/pending
func (h *Handler) DeviceLoginPending(c *fiber.Ctx) error {
	agentID := c.Locals("agentID").(string)
	ctx := context.Background()

	rows, err := h.db.Query(ctx,
		`SELECT id, device_name, created_at FROM device_login_requests 
		 WHERE agent_id=$1 AND status='pending' AND expires_at > NOW()
		 ORDER BY created_at DESC`,
		agentID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	var requests []fiber.Map
	for rows.Next() {
		var id, deviceName string
		var createdAt time.Time
		if err := rows.Scan(&id, &deviceName, &createdAt); err != nil {
			continue
		}
		requests = append(requests, fiber.Map{
			"requestId":  id,
			"deviceName": deviceName,
			"createdAt":  createdAt,
		})
	}

	if requests == nil {
		requests = []fiber.Map{}
	}

	return c.JSON(fiber.Map{"pending": requests})
}
