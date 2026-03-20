package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ========== 许愿系统 ==========

// POST /api/wishes
func (h *Handler) WishCreate(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	ctx := context.Background()

	var req struct {
		Category string `json:"category"`
		Content  string `json:"content"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Category != "world_event" && req.Category != "bug_report" && req.Category != "feature_request" {
		return c.Status(400).JSON(fiber.Map{
			"error": "category must be one of: world_event, bug_report, feature_request",
		})
	}

	content := []rune(req.Content)
	if len(content) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "content is required"})
	}
	if len(content) > 500 {
		return c.Status(400).JSON(fiber.Map{"error": "content must be 500 characters or less"})
	}

	var username string
	h.db.QueryRow(ctx, `SELECT username FROM players WHERE id=$1`, playerID).Scan(&username)

	var wishID string
	err := h.db.QueryRow(ctx,
		`INSERT INTO wishes (player_id, username, category, content) VALUES ($1,$2,$3,$4) RETURNING id`,
		playerID, username, req.Category, req.Content,
	).Scan(&wishID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	categoryNames := map[string]string{
		"world_event":     "世界事件",
		"bug_report":      "Bug反馈",
		"feature_request": "功能建议",
	}

	return c.Status(201).JSON(fiber.Map{
		"wishId":       wishID,
		"category":     req.Category,
		"categoryName": categoryNames[req.Category],
		"content":      req.Content,
		"status":       "pending",
		"message":      "许愿成功！愿望已提交，等待造物主审阅。🌟",
	})
}

// GET /api/wishes/top5 — 王妈专用，按category+时间排序取pending中最有代表性的5条
func (h *Handler) WishTop5(c *fiber.Ctx) error {
	ctx := context.Background()

	// 策略：优先 world_event，然后 bug_report，最后 feature_request；同类按时间最早
	rows, err := h.db.Query(ctx,
		`SELECT id, player_id, username, category, content, status, admin_note, created_at, updated_at
		 FROM wishes
		 WHERE status='pending'
		 ORDER BY
		   CASE category
		     WHEN 'world_event'     THEN 1
		     WHEN 'bug_report'      THEN 2
		     WHEN 'feature_request' THEN 3
		     ELSE 4
		   END,
		   created_at ASC
		 LIMIT 5`,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	wishes := collectWishes(rows)
	if wishes == nil {
		wishes = []fiber.Map{}
	}

	// Also return total pending count by category
	type catCount struct {
		Category string
		Count    int
	}
	var catCounts []fiber.Map
	catRows, _ := h.db.Query(ctx,
		`SELECT category, COUNT(*) FROM wishes WHERE status='pending' GROUP BY category ORDER BY category`,
	)
	if catRows != nil {
		defer catRows.Close()
		for catRows.Next() {
			var cat string
			var cnt int
			catRows.Scan(&cat, &cnt)
			catCounts = append(catCounts, fiber.Map{"category": cat, "count": cnt})
		}
	}
	if catCounts == nil {
		catCounts = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"top5":          wishes,
		"pendingByCategory": catCounts,
		"hint":          "这是待决策的Top5许愿，请主人审阅后决定是否批准。",
	})
}

// GET /api/wishes/my
func (h *Handler) WishMy(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	ctx := context.Background()

	rows, err := h.db.Query(ctx,
		`SELECT id, player_id, username, category, content, status, admin_note, created_at, updated_at
		 FROM wishes WHERE player_id=$1 ORDER BY created_at DESC LIMIT 50`,
		playerID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	wishes := collectWishes(rows)
	if wishes == nil {
		wishes = []fiber.Map{}
	}

	return c.JSON(fiber.Map{"wishes": wishes, "total": len(wishes)})
}

// GET /api/wishes/fulfilled — 全服公示
func (h *Handler) WishFulfilled(c *fiber.Ctx) error {
	ctx := context.Background()

	rows, err := h.db.Query(ctx,
		`SELECT id, player_id, username, category, content, status, admin_note, created_at, updated_at
		 FROM wishes WHERE status='fulfilled' ORDER BY updated_at DESC LIMIT 50`,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	wishes := collectWishes(rows)
	if wishes == nil {
		wishes = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"fulfilled": wishes,
		"total":     len(wishes),
		"message":   "以下愿望已被造物主实现，愿天道公平，众修士共见！✨",
	})
}

// POST /api/wishes/:id/approve — 管理员批准
func (h *Handler) WishApprove(c *fiber.Ctx) error {
	wishID := c.Params("id")
	ctx := context.Background()

	var req struct {
		AdminNote string `json:"adminNote"`
	}
	c.BodyParser(&req)

	tag, err := h.db.Exec(ctx,
		`UPDATE wishes SET status='approved', admin_note=$1, updated_at=NOW() WHERE id=$2 AND status='pending'`,
		req.AdminNote, wishID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	if tag.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "wish not found or not pending"})
	}

	return c.JSON(fiber.Map{"wishId": wishID, "status": "approved", "message": "愿望已批准 ✅"})
}

// POST /api/wishes/:id/reject — 管理员拒绝（可附带回复）
func (h *Handler) WishReject(c *fiber.Ctx) error {
	wishID := c.Params("id")
	ctx := context.Background()

	var req struct {
		AdminNote string `json:"adminNote"`
	}
	c.BodyParser(&req)

	tag, err := h.db.Exec(ctx,
		`UPDATE wishes SET status='rejected', admin_note=$1, updated_at=NOW() WHERE id=$2 AND status='pending'`,
		req.AdminNote, wishID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	if tag.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "wish not found or not pending"})
	}

	return c.JSON(fiber.Map{"wishId": wishID, "status": "rejected", "message": "愿望已拒绝 ❌"})
}

// POST /api/wishes/:id/fulfill — 标记已实现，全服广播
func (h *Handler) WishFulfill(c *fiber.Ctx) error {
	wishID := c.Params("id")
	ctx := context.Background()

	var req struct {
		AdminNote string `json:"adminNote"`
	}
	c.BodyParser(&req)

	var username, category, content string
	err := h.db.QueryRow(ctx,
		`UPDATE wishes SET status='fulfilled', admin_note=$1, updated_at=NOW()
		 WHERE id=$2 AND status IN ('pending','approved')
		 RETURNING username, category, content`,
		req.AdminNote, wishID,
	).Scan(&username, &category, &content)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "wish not found or already handled"})
	}

	// Publish Redis event for WebSocket broadcast
	payload := fiber.Map{
		"wishId":   wishID,
		"username": username,
		"category": category,
		"content":  content,
		"note":     req.AdminNote,
	}
	if data, err2 := json.Marshal(payload); err2 == nil {
		h.rdb.Publish(ctx, "game:wish:fulfilled", string(data))
	}

	return c.JSON(fiber.Map{
		"wishId":  wishID,
		"status":  "fulfilled",
		"message": "愿望已实现！全服广播中... 🎉",
	})
}

// ========== helpers ==========

type wishRow struct {
	ID        string
	PlayerID  string
	Username  string
	Category  string
	Content   string
	Status    string
	AdminNote string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type scannable interface {
	Scan(dest ...interface{}) error
	Next() bool
}

func collectWishes(rows scannable) []fiber.Map {
	categoryNames := map[string]string{
		"world_event":     "世界事件",
		"bug_report":      "Bug反馈",
		"feature_request": "功能建议",
	}
	statusNames := map[string]string{
		"pending":   "待审阅",
		"approved":  "已批准",
		"rejected":  "已拒绝",
		"fulfilled": "已实现",
	}

	var result []fiber.Map
	for rows.Next() {
		var w wishRow
		rows.Scan(&w.ID, &w.PlayerID, &w.Username, &w.Category, &w.Content,
			&w.Status, &w.AdminNote, &w.CreatedAt, &w.UpdatedAt)
		result = append(result, fiber.Map{
			"id":           w.ID,
			"playerId":     w.PlayerID,
			"username":     w.Username,
			"category":     w.Category,
			"categoryName": categoryNames[w.Category],
			"content":      w.Content,
			"status":       w.Status,
			"statusName":   statusNames[w.Status],
			"adminNote":    w.AdminNote,
			"createdAt":    w.CreatedAt,
			"updatedAt":    w.UpdatedAt,
		})
	}
	return result
}
