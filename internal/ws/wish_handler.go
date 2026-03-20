package ws

import (
	"context"
	"encoding/json"
	"time"
)

// ── WebSocket 许愿系统 ──

func (h *Hub) handleWishCreate(ctx context.Context, c *Client, msg Message) {
	var data struct {
		Category string `json:"category"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "invalid data"})
		return
	}

	if data.Category != "world_event" && data.Category != "bug_report" && data.Category != "feature_request" {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false,
			Error: "category must be: world_event | bug_report | feature_request"})
		return
	}

	content := []rune(data.Content)
	if len(content) == 0 {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "content is required"})
		return
	}
	if len(content) > 500 {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "content must be 500 chars or less"})
		return
	}

	var username string
	h.db.QueryRow(ctx, `SELECT username FROM players WHERE id=$1`, c.playerID).Scan(&username)

	var wishID string
	err := h.db.QueryRow(ctx,
		`INSERT INTO wishes (player_id, username, category, content) VALUES ($1,$2,$3,$4) RETURNING id`,
		c.playerID, username, data.Category, data.Content,
	).Scan(&wishID)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	categoryNames := map[string]string{
		"world_event":     "世界事件",
		"bug_report":      "Bug反馈",
		"feature_request": "功能建议",
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: map[string]interface{}{
		"wishId":       wishID,
		"category":     data.Category,
		"categoryName": categoryNames[data.Category],
		"content":      data.Content,
		"status":       "pending",
		"message":      "许愿成功！愿望已提交，等待造物主审阅。🌟",
	}})
}

func (h *Hub) handleWishTop5(ctx context.Context, c *Client, msg Message) {
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
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer rows.Close()

	wishes := wsCollectWishes(rows)
	if wishes == nil {
		wishes = []map[string]interface{}{}
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: map[string]interface{}{
		"top5":    wishes,
		"hint":    "这是待决策的Top5许愿，请主人审阅后决定是否批准。",
	}})
}

func (h *Hub) handleWishMy(ctx context.Context, c *Client, msg Message) {
	rows, err := h.db.Query(ctx,
		`SELECT id, player_id, username, category, content, status, admin_note, created_at, updated_at
		 FROM wishes WHERE player_id=$1 ORDER BY created_at DESC LIMIT 50`,
		c.playerID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer rows.Close()

	wishes := wsCollectWishes(rows)
	if wishes == nil {
		wishes = []map[string]interface{}{}
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: map[string]interface{}{
		"wishes": wishes,
		"total":  len(wishes),
	}})
}

func (h *Hub) handleWishFulfilled(ctx context.Context, c *Client, msg Message) {
	rows, err := h.db.Query(ctx,
		`SELECT id, player_id, username, category, content, status, admin_note, created_at, updated_at
		 FROM wishes WHERE status='fulfilled' ORDER BY updated_at DESC LIMIT 50`,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer rows.Close()

	wishes := wsCollectWishes(rows)
	if wishes == nil {
		wishes = []map[string]interface{}{}
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: map[string]interface{}{
		"fulfilled": wishes,
		"total":     len(wishes),
		"message":   "以下愿望已被造物主实现，愿天道公平，众修士共见！✨",
	}})
}

// ── WebSocket 全服事件查询 ──

func (h *Hub) handleWorldEventsActive(ctx context.Context, c *Client, msg Message) {
	var currentYear int
	h.db.QueryRow(ctx, `SELECT current_year FROM world_state WHERE id=1`).Scan(&currentYear)

	rows, err := h.db.Query(ctx,
		`SELECT id, event_type, title, description, effect_type, effect_data, triggered_by, active_until_year, created_at
		 FROM world_events
		 WHERE active_until_year >= $1
		 ORDER BY created_at DESC`,
		currentYear,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var id, eventType, title, description, effectType, triggeredBy string
		var effectData []byte
		var activeUntilYear int
		var createdAt time.Time
		rows.Scan(&id, &eventType, &title, &description, &effectType, &effectData, &triggeredBy, &activeUntilYear, &createdAt)

		var effectDataParsed interface{}
		json.Unmarshal(effectData, &effectDataParsed)

		events = append(events, map[string]interface{}{
			"id":              id,
			"eventType":       eventType,
			"title":           title,
			"description":     description,
			"effectType":      effectType,
			"effectData":      effectDataParsed,
			"triggeredBy":     triggeredBy,
			"activeUntilYear": activeUntilYear,
			"createdAt":       createdAt,
		})
	}
	if events == nil {
		events = []map[string]interface{}{}
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: map[string]interface{}{
		"currentYear":  currentYear,
		"activeEvents": events,
		"total":        len(events),
	}})
}

// ── helpers ──

type wsScannable interface {
	Scan(dest ...interface{}) error
	Next() bool
}

func wsCollectWishes(rows wsScannable) []map[string]interface{} {
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

	var result []map[string]interface{}
	for rows.Next() {
		var id, playerID, username, category, content, status, adminNote string
		var createdAt, updatedAt time.Time
		rows.Scan(&id, &playerID, &username, &category, &content, &status, &adminNote, &createdAt, &updatedAt)
		result = append(result, map[string]interface{}{
			"id":           id,
			"playerId":     playerID,
			"username":     username,
			"category":     category,
			"categoryName": categoryNames[category],
			"content":      content,
			"status":       status,
			"statusName":   statusNames[status],
			"adminNote":    adminNote,
			"createdAt":    createdAt,
			"updatedAt":    updatedAt,
		})
	}
	return result
}
