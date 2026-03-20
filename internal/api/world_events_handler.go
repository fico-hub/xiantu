package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ========== 全服事件系统 ==========

// GET /api/world/events/active — 当前生效的全服事件（全服可见）
func (h *Handler) WorldEventsActive(c *fiber.Ctx) error {
	ctx := context.Background()

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
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	events := collectWorldEvents(rows)
	if events == nil {
		events = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"currentYear":   currentYear,
		"activeEvents":  events,
		"total":         len(events),
	})
}

// POST /api/world/events — 触发全服事件（管理员/王妈专用）
func (h *Handler) WorldEventCreate(c *fiber.Ctx) error {
	ctx := context.Background()

	var req struct {
		EventType      string                 `json:"eventType"`
		Title          string                 `json:"title"`
		Description    string                 `json:"description"`
		EffectType     string                 `json:"effectType"`
		EffectData     map[string]interface{} `json:"effectData"`
		TriggeredBy    string                 `json:"triggeredBy"`
		DurationYears  int                    `json:"durationYears"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Title == "" {
		return c.Status(400).JSON(fiber.Map{"error": "title is required"})
	}
	if req.Description == "" {
		return c.Status(400).JSON(fiber.Map{"error": "description is required"})
	}
	if req.EffectType == "" {
		req.EffectType = "neutral"
	}
	if req.EffectType != "buff" && req.EffectType != "debuff" && req.EffectType != "neutral" {
		return c.Status(400).JSON(fiber.Map{"error": "effectType must be buff, debuff, or neutral"})
	}
	if req.TriggeredBy == "" {
		req.TriggeredBy = "wang_ma"
	}
	if req.DurationYears <= 0 {
		req.DurationYears = 10
	}
	if req.EventType == "" {
		req.EventType = "custom"
	}
	if req.EffectData == nil {
		req.EffectData = map[string]interface{}{}
	}

	var currentYear int
	h.db.QueryRow(ctx, `SELECT current_year FROM world_state WHERE id=1`).Scan(&currentYear)
	activeUntilYear := currentYear + req.DurationYears

	effectDataJSON, _ := json.Marshal(req.EffectData)

	var eventID string
	err := h.db.QueryRow(ctx,
		`INSERT INTO world_events (event_type, title, description, effect_type, effect_data, triggered_by, active_until_year)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		req.EventType, req.Title, req.Description, req.EffectType,
		effectDataJSON, req.TriggeredBy, activeUntilYear,
	).Scan(&eventID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	// Broadcast to all online players via Redis
	broadcastPayload := fiber.Map{
		"eventId":       eventID,
		"eventType":     req.EventType,
		"title":         req.Title,
		"description":   req.Description,
		"effectType":    req.EffectType,
		"effectData":    req.EffectData,
		"triggeredBy":   req.TriggeredBy,
		"currentYear":   currentYear,
		"activeUntilYear": activeUntilYear,
		"durationYears": req.DurationYears,
	}
	if data, err2 := json.Marshal(broadcastPayload); err2 == nil {
		h.rdb.Publish(ctx, "game:world_event", string(data))
	}

	return c.Status(201).JSON(fiber.Map{
		"eventId":         eventID,
		"title":           req.Title,
		"activeUntilYear": activeUntilYear,
		"durationYears":   req.DurationYears,
		"message":         "全服事件已触发，正在广播！⚡",
	})
}

// GET /api/world/events/history — 历史事件（全服可见）
func (h *Handler) WorldEventsHistory(c *fiber.Ctx) error {
	ctx := context.Background()

	rows, err := h.db.Query(ctx,
		`SELECT id, event_type, title, description, effect_type, effect_data, triggered_by, active_until_year, created_at
		 FROM world_events
		 ORDER BY created_at DESC
		 LIMIT 50`,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	events := collectWorldEvents(rows)
	if events == nil {
		events = []fiber.Map{}
	}

	return c.JSON(fiber.Map{"events": events, "total": len(events)})
}

// ========== helpers ==========

type worldEventRow struct {
	ID             string
	EventType      string
	Title          string
	Description    string
	EffectType     string
	EffectData     []byte
	TriggeredBy    string
	ActiveUntilYear int
	CreatedAt      time.Time
}

type worldEventScannable interface {
	Scan(dest ...interface{}) error
	Next() bool
}

func collectWorldEvents(rows worldEventScannable) []fiber.Map {
	effectTypeNames := map[string]string{
		"buff":    "增益",
		"debuff":  "减益",
		"neutral": "中性",
	}

	var result []fiber.Map
	for rows.Next() {
		var e worldEventRow
		rows.Scan(&e.ID, &e.EventType, &e.Title, &e.Description,
			&e.EffectType, &e.EffectData, &e.TriggeredBy, &e.ActiveUntilYear, &e.CreatedAt)

		var effectData interface{}
		json.Unmarshal(e.EffectData, &effectData)

		result = append(result, fiber.Map{
			"id":              e.ID,
			"eventType":       e.EventType,
			"title":           e.Title,
			"description":     e.Description,
			"effectType":      e.EffectType,
			"effectTypeName":  effectTypeNames[e.EffectType],
			"effectData":      effectData,
			"triggeredBy":     e.TriggeredBy,
			"activeUntilYear": e.ActiveUntilYear,
			"createdAt":       e.CreatedAt,
		})
	}
	return result
}
