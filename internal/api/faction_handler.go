package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/xiantu/server/internal/game"
)

// ========== REST API: 门派系统 ==========

// GET /api/factions
func (h *Handler) FactionList(c *fiber.Ctx) error {
	type FactionInfo struct {
		ID          string              `json:"id"`
		Name        string              `json:"name"`
		NameEn      string              `json:"nameEn"`
		Race        string              `json:"race"`
		Territory   string              `json:"territory"`
		Alliance    string              `json:"alliance"`
		Bonuses     []game.FactionBonus `json:"bonuses"`
		Passives    []string            `json:"passives"`
		SpecialRule string              `json:"specialRule,omitempty"`
		IsVillain   bool                `json:"isVillain,omitempty"`
		Relations   map[string]string   `json:"relations"`
	}

	result := make([]FactionInfo, 0, len(game.FactionOrder))
	for _, fid := range game.FactionOrder {
		f := game.Factions[fid]
		rels := make(map[string]string)
		for _, other := range game.FactionOrder {
			if other == fid {
				continue
			}
			rel := game.GetFactionRelation(fid, other)
			if rel != game.RelationNeutral {
				rels[other] = string(rel)
			}
		}
		result = append(result, FactionInfo{
			ID: f.ID, Name: f.Name, NameEn: f.NameEn,
			Race: f.Race, Territory: f.Territory, Alliance: f.Alliance,
			Bonuses: f.Bonuses, Passives: f.Passives,
			SpecialRule: f.SpecialRule, IsVillain: f.IsVillain,
			Relations: rels,
		})
	}
	return c.JSON(fiber.Map{"factions": result, "count": len(result)})
}

// GET /api/factions/:id
func (h *Handler) FactionDetail(c *fiber.Ctx) error {
	fid := c.Params("id")
	f, ok := game.Factions[fid]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "faction not found"})
	}

	rels := make(map[string]string)
	for _, other := range game.FactionOrder {
		if other == fid {
			continue
		}
		rel := game.GetFactionRelation(fid, other)
		if rel != game.RelationNeutral {
			rels[other] = string(rel)
		}
	}

	// Member count
	ctx := context.Background()
	var memberCount int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM player_factions WHERE faction_id=$1`, fid).Scan(&memberCount)

	return c.JSON(fiber.Map{
		"faction":     f,
		"relations":   rels,
		"memberCount": memberCount,
	})
}

// POST /api/factions/:id/join
func (h *Handler) FactionJoin(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	fid := c.Params("id")
	ctx := context.Background()

	if _, ok := game.Factions[fid]; !ok {
		return c.Status(404).JSON(fiber.Map{"error": "faction not found"})
	}

	// Check if already in a faction
	var existingFaction string
	err := h.db.QueryRow(ctx, `SELECT faction_id FROM player_factions WHERE player_id=$1`, playerID).Scan(&existingFaction)
	if err == nil {
		return c.Status(409).JSON(fiber.Map{
			"error":           "已在门派中",
			"currentFaction":  existingFaction,
			"currentFactionName": game.Factions[existingFaction].Name,
		})
	}

	// Fetch player race
	var race string
	if err := h.db.QueryRow(ctx, `SELECT race FROM players WHERE id=$1`, playerID).Scan(&race); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	// Check blood oath for MS-13
	var hasBloodOath bool
	if fid == "ms13" {
		var count int
		h.db.QueryRow(ctx, `SELECT COUNT(*) FROM player_quest_flags WHERE player_id=$1 AND flag='blood_oath_ms13'`, playerID).Scan(&count)
		hasBloodOath = count > 0
	}

	allowed, needsSpecial, reason := game.CanJoinFaction(fid, race, hasBloodOath)
	if !allowed {
		return c.Status(403).JSON(fiber.Map{"error": reason})
	}
	if needsSpecial {
		return c.Status(403).JSON(fiber.Map{
			"error":       reason,
			"requiresTask": "blood_oath_ms13",
		})
	}

	// Join the faction
	_, err = h.db.Exec(ctx,
		`INSERT INTO player_factions (id, player_id, faction_id, joined_at, contribution, rank)
		 VALUES ($1, $2, $3, NOW(), 0, 'recruit')`,
		uuid.New().String(), playerID, fid,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("db error: %v", err)})
	}

	f := game.Factions[fid]
	msg := "恭喜加入【" + f.Name + "】（" + f.NameEn + "）！"
	if reason != "" {
		msg += " 注：" + reason
	}

	// Check if we should assign a task immediately
	var taskResp interface{}
	if game.ShouldAssignFactionTask() {
		var realm string
		h.db.QueryRow(ctx, `SELECT realm FROM players WHERE id=$1`, playerID).Scan(&realm)
		taskResp = h.assignFactionTask(ctx, playerID, fid, realm, 0)
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"faction":     f.Name,
		"factionId":   fid,
		"message":     msg,
		"bonuses":     f.Bonuses,
		"passives":    f.Passives,
		"initialTask": taskResp,
	})
}

// POST /api/factions/:id/leave
func (h *Handler) FactionLeave(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	fid := c.Params("id")
	ctx := context.Background()

	// Verify player is in this faction
	var memberFaction string
	err := h.db.QueryRow(ctx, `SELECT faction_id FROM player_factions WHERE player_id=$1`, playerID).Scan(&memberFaction)
	if err != nil || memberFaction != fid {
		return c.Status(400).JSON(fiber.Map{"error": "你不在该门派中"})
	}

	canLeave, reason := game.CanLeaveFaction(fid)
	if !canLeave {
		return c.Status(403).JSON(fiber.Map{"error": reason})
	}

	// Cancel any active tasks
	_, _ = h.db.Exec(ctx, `UPDATE player_faction_tasks SET status='cancelled' WHERE player_id=$1 AND status='active'`, playerID)

	// Remove from faction
	_, err = h.db.Exec(ctx, `DELETE FROM player_factions WHERE player_id=$1`, playerID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}

	f := game.Factions[fid]
	return c.JSON(fiber.Map{
		"success": true,
		"message": "你已离开【" + f.Name + "】，门派佛系，缘聚缘散。",
	})
}

// GET /api/factions/my/tasks
func (h *Handler) FactionMyTasks(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	ctx := context.Background()

	rows, err := h.db.Query(ctx,
		`SELECT id, faction_id, task_type, title, description, task_data,
		        reward_data, status, assigned_at, expires_at
		 FROM player_faction_tasks
		 WHERE player_id=$1 AND status IN ('active','completed')
		 ORDER BY assigned_at DESC LIMIT 20`,
		playerID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	type TaskRow struct {
		ID         string      `json:"id"`
		FactionID  string      `json:"factionId"`
		FactionName string     `json:"factionName"`
		TaskType   string      `json:"taskType"`
		Title      string      `json:"title"`
		Desc       string      `json:"description"`
		TaskData   interface{} `json:"taskData"`
		RewardData interface{} `json:"rewardData"`
		Status     string      `json:"status"`
		AssignedAt time.Time   `json:"assignedAt"`
		ExpiresAt  *time.Time  `json:"expiresAt"`
	}

	var tasks []TaskRow
	for rows.Next() {
		var t TaskRow
		var taskDataRaw, rewardDataRaw []byte
		var expiresAt *time.Time
		if err := rows.Scan(&t.ID, &t.FactionID, &t.TaskType, &t.Title, &t.Desc,
			&taskDataRaw, &rewardDataRaw, &t.Status, &t.AssignedAt, &expiresAt); err != nil {
			continue
		}
		t.ExpiresAt = expiresAt
		json.Unmarshal(taskDataRaw, &t.TaskData)
		json.Unmarshal(rewardDataRaw, &t.RewardData)
		if f, ok := game.Factions[t.FactionID]; ok {
			t.FactionName = f.Name
		}
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []TaskRow{}
	}
	return c.JSON(fiber.Map{"tasks": tasks, "count": len(tasks)})
}

// POST /api/factions/tasks/:id/complete
func (h *Handler) FactionTaskComplete(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	taskID := c.Params("id")
	ctx := context.Background()

	return h.completeFactionTask(ctx, c, playerID, taskID)
}

// GET /api/factions/my/rank
func (h *Handler) FactionMyRank(c *fiber.Ctx) error {
	playerID := c.Locals("playerID").(string)
	ctx := context.Background()

	var factionID, rank string
	var contrib int64
	var joinedAt time.Time
	err := h.db.QueryRow(ctx,
		`SELECT faction_id, contribution, rank, joined_at FROM player_factions WHERE player_id=$1`,
		playerID,
	).Scan(&factionID, &contrib, &rank, &joinedAt)
	if err != nil {
		return c.JSON(fiber.Map{
			"inFaction": false,
			"message":   "你尚未加入任何门派",
		})
	}

	f := game.Factions[factionID]
	currentRank := game.GetRankByContrib(contrib)

	// Find next rank
	var nextRank *game.FactionRank
	for i, r := range game.FactionRanks {
		if r.ID == currentRank.ID && i+1 < len(game.FactionRanks) {
			nr := game.FactionRanks[i+1]
			nextRank = &nr
			break
		}
	}

	// Count active tasks
	var activeTasks int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM player_faction_tasks WHERE player_id=$1 AND status='active'`, playerID).Scan(&activeTasks)

	resp := fiber.Map{
		"inFaction":    true,
		"factionId":    factionID,
		"factionName":  f.Name,
		"factionNameEn": f.NameEn,
		"territory":    f.Territory,
		"alliance":     f.Alliance,
		"rank":         rank,
		"rankName":     currentRank.Name,
		"contribution": contrib,
		"joinedAt":     joinedAt,
		"activeTasks":  activeTasks,
		"bonuses":      f.Bonuses,
		"passives":     f.Passives,
	}
	if nextRank != nil {
		resp["nextRank"] = nextRank.Name
		resp["nextRankContrib"] = nextRank.MinContrib
		resp["contribToNext"] = nextRank.MinContrib - contrib
	}
	return c.JSON(resp)
}

// ========== Internal helpers ==========

// assignFactionTask creates a new faction task record in DB and returns it
func (h *Handler) assignFactionTask(ctx context.Context, playerID, factionID, realm string, contrib int64) interface{} {
	task := game.GenerateFactionTask(factionID, realm, contrib)
	if task == nil {
		return nil
	}

	taskID := uuid.New().String()
	taskDataJSON, _ := json.Marshal(map[string]interface{}{
		"type":            string(task.Type),
		"targetLocation":  task.TargetLocation,
		"targetCount":     task.TargetCount,
		"targetElement":   task.TargetElement,
		"targetRealm":     task.TargetRealm,
		"escortFrom":      task.EscortFrom,
		"escortTo":        task.EscortTo,
		"durationYears":   task.DurationYears,
	})
	rewardDataJSON, _ := json.Marshal(task.Reward)

	// Tasks expire in 30 game years (150 real minutes)
	expiresAt := time.Now().Add(150 * time.Minute)

	_, err := h.db.Exec(ctx,
		`INSERT INTO player_faction_tasks
		 (id, player_id, faction_id, task_type, title, description, task_data, reward_data, status, assigned_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active',NOW(),$9)`,
		taskID, playerID, factionID,
		string(task.Type), task.Title, task.Description,
		taskDataJSON, rewardDataJSON, expiresAt,
	)
	if err != nil {
		return nil
	}

	return map[string]interface{}{
		"taskId":      taskID,
		"type":        string(task.Type),
		"title":       task.Title,
		"description": task.Description,
		"reward":      task.Reward,
		"expiresAt":   expiresAt,
	}
}

// completeFactionTask handles task completion logic
func (h *Handler) completeFactionTask(ctx context.Context, c *fiber.Ctx, playerID, taskID string) error {
	// Fetch task
	var factionID, taskType string
	var rewardDataRaw []byte
	var status string
	err := h.db.QueryRow(ctx,
		`SELECT faction_id, task_type, reward_data, status FROM player_faction_tasks
		 WHERE id=$1 AND player_id=$2`,
		taskID, playerID,
	).Scan(&factionID, &taskType, &rewardDataRaw, &status)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "任务不存在"})
	}
	if status != "active" {
		return c.Status(400).JSON(fiber.Map{"error": "任务已" + status})
	}

	var reward game.FactionTaskReward
	json.Unmarshal(rewardDataRaw, &reward)

	// Apply rewards in a transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "db error"})
	}
	defer tx.Rollback(ctx)

	// XP
	if reward.CultivationXP > 0 {
		_, err = tx.Exec(ctx, `UPDATE players SET cultivation_xp = cultivation_xp + $1 WHERE id=$2`, reward.CultivationXP, playerID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "reward xp error"})
		}
	}
	// Spirit stone
	netStone := reward.SpiritStone
	if netStone > 0 {
		tx.Exec(ctx, `UPDATE players SET spirit_stone = spirit_stone + $1 WHERE id=$2`, netStone, playerID)
	} else if netStone < 0 {
		// tribute: deduct stones
		var currentStone int64
		tx.QueryRow(ctx, `SELECT spirit_stone FROM players WHERE id=$1`, playerID).Scan(&currentStone)
		if currentStone < -netStone {
			tx.Rollback(ctx)
			return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("灵石不足，完成此任务需贡献%d灵石，施主还需积累", -netStone)})
		}
		tx.Exec(ctx, `UPDATE players SET spirit_stone = spirit_stone + $1 WHERE id=$2`, netStone, playerID)
	}
	// Material reward
	if reward.MaterialElement != "" && reward.MaterialQty > 0 {
		_, err = tx.Exec(ctx,
			`INSERT INTO spirit_materials (id, player_id, element, quantity)
			 VALUES ($1,$2,$3,$4)
			 ON CONFLICT (player_id, element) DO UPDATE SET quantity = spirit_materials.quantity + $4`,
			uuid.New().String(), playerID, reward.MaterialElement, reward.MaterialQty,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "reward material error"})
		}
	}

	// Faction contribution
	if reward.FactionContrib > 0 {
		tx.Exec(ctx, `UPDATE player_factions SET contribution = contribution + $1 WHERE player_id=$2`, reward.FactionContrib, playerID)
	}

	// Mark task done
	tx.Exec(ctx, `UPDATE player_faction_tasks SET status='completed', completed_at=NOW() WHERE id=$1`, taskID)

	if err := tx.Commit(ctx); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "commit error"})
	}

	// Check rank-up
	var contrib int64
	var currentRankID string
	h.db.QueryRow(ctx, `SELECT contribution, rank FROM player_factions WHERE player_id=$1`, playerID).Scan(&contrib, &currentRankID)
	newRank := game.GetRankByContrib(contrib)
	rankUp := newRank.ID != currentRankID
	if rankUp {
		h.db.Exec(ctx, `UPDATE player_factions SET rank=$1 WHERE player_id=$2`, newRank.ID, playerID)
	}

	result := fiber.Map{
		"success":       true,
		"taskId":        taskID,
		"factionId":     factionID,
		"reward":        reward,
		"rankUp":        rankUp,
		"newContrib":    contrib,
		"newRank":       newRank.Name,
		"message":       "任务完成！门派贡献+" + fmt.Sprintf("%d", reward.FactionContrib),
	}
	if rankUp {
		result["rankUpMessage"] = "恭喜晋升为【" + newRank.Name + "】！"
	}

	// Publish rank-up event to Redis if needed
	if rankUp {
		f := game.Factions[factionID]
		var username string
		h.db.QueryRow(ctx, `SELECT username FROM players WHERE id=$1`, playerID).Scan(&username)
		payload, _ := json.Marshal(map[string]interface{}{
			"playerID":   playerID,
			"username":   username,
			"factionID":  factionID,
			"factionName": f.Name,
			"newRank":    newRank.Name,
		})
		h.rdb.Publish(ctx, "game:faction:rank_up", string(payload))
	}

	return c.JSON(result)
}

// CheckAndAssignFactionTasks is called during player check-in (offline cultivation)
// It rolls the 30% chance and assigns a task if the player is in a faction
func (h *Handler) CheckAndAssignFactionTasks(ctx context.Context, playerID string) interface{} {
	var factionID, realm string
	var contrib int64
	err := h.db.QueryRow(ctx,
		`SELECT pf.faction_id, p.realm, pf.contribution
		 FROM player_factions pf
		 JOIN players p ON p.id = pf.player_id
		 WHERE pf.player_id=$1`,
		playerID,
	).Scan(&factionID, &realm, &contrib)
	if err != nil {
		return nil // not in a faction
	}

	// Check if already has an active task
	var activeCount int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM player_faction_tasks WHERE player_id=$1 AND status='active'`, playerID).Scan(&activeCount)
	if activeCount >= 3 {
		return nil // too many active tasks
	}

	if !game.ShouldAssignFactionTask() {
		return nil
	}

	task := h.assignFactionTask(ctx, playerID, factionID, realm, contrib)
	if task == nil {
		return nil
	}

	// Publish to player's personal channel
	payload, _ := json.Marshal(map[string]interface{}{
		"factionName": game.Factions[factionID].Name,
		"task":        task,
	})
	h.rdb.Publish(ctx, "game:faction:task:"+playerID, string(payload))

	return task
}
