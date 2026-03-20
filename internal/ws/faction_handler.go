package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xiantu/server/internal/game"
)

// ── Faction WebSocket Handlers ──
// All handlers follow the hub's existing pattern.

// query.factions → 门派列表
func (h *Hub) handleQueryFactions(ctx context.Context, c *Client, msg Message) {
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
	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true,
		Data: map[string]interface{}{"factions": result, "count": len(result)}})
}

// query.my.faction → 我的门派状态
func (h *Hub) handleQueryMyFaction(ctx context.Context, c *Client, msg Message) {
	var factionID, rank string
	var contrib int64
	var joinedAt time.Time
	err := h.db.QueryRow(ctx,
		`SELECT faction_id, contribution, rank, joined_at FROM player_factions WHERE player_id=$1`,
		c.playerID,
	).Scan(&factionID, &contrib, &rank, &joinedAt)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true,
			Data: map[string]interface{}{"inFaction": false, "message": "你尚未加入任何门派"}})
		return
	}

	f := game.Factions[factionID]
	currentRank := game.GetRankByContrib(contrib)

	var activeTasks int
	h.db.QueryRow(ctx, `SELECT COUNT(*) FROM player_faction_tasks WHERE player_id=$1 AND status='active'`, c.playerID).Scan(&activeTasks)

	resp := map[string]interface{}{
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

	// Next rank
	for i, r := range game.FactionRanks {
		if r.ID == currentRank.ID && i+1 < len(game.FactionRanks) {
			nr := game.FactionRanks[i+1]
			resp["nextRank"] = nr.Name
			resp["nextRankContrib"] = nr.MinContrib
			resp["contribToNext"] = nr.MinContrib - contrib
			break
		}
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: resp})
}

// cmd.faction.join → 加入门派
func (h *Hub) handleFactionJoin(ctx context.Context, c *Client, msg Message) {
	var data struct {
		FactionID string `json:"factionId"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil || data.FactionID == "" {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "需要提供 factionId"})
		return
	}

	if _, ok := game.Factions[data.FactionID]; !ok {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "门派不存在"})
		return
	}

	// Already in a faction?
	var existingFaction string
	err := h.db.QueryRow(ctx, `SELECT faction_id FROM player_factions WHERE player_id=$1`, c.playerID).Scan(&existingFaction)
	if err == nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false,
			Error: fmt.Sprintf("你已在【%s】中，请先离开", game.Factions[existingFaction].Name)})
		return
	}

	// Get player race
	var race string
	h.db.QueryRow(ctx, `SELECT race FROM players WHERE id=$1`, c.playerID).Scan(&race)

	// Blood oath check
	var hasBloodOath bool
	if data.FactionID == "ms13" {
		var count int
		h.db.QueryRow(ctx, `SELECT COUNT(*) FROM player_quest_flags WHERE player_id=$1 AND flag='blood_oath_ms13'`, c.playerID).Scan(&count)
		hasBloodOath = count > 0
	}

	allowed, needsSpecial, reason := game.CanJoinFaction(data.FactionID, race, hasBloodOath)
	if !allowed {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: reason})
		return
	}
	if needsSpecial {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false,
			Error:  reason,
			Data: map[string]interface{}{"requiresTask": "blood_oath_ms13"}})
		return
	}

	_, err = h.db.Exec(ctx,
		`INSERT INTO player_factions (id, player_id, faction_id, joined_at, contribution, rank)
		 VALUES ($1,$2,$3,NOW(),0,'recruit')`,
		uuid.New().String(), c.playerID, data.FactionID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	f := game.Factions[data.FactionID]
	respMsg := "恭喜加入【" + f.Name + "】（" + f.NameEn + "）！"
	if reason != "" {
		respMsg += " 注：" + reason
	}

	// Maybe assign first task
	var initialTask interface{}
	if game.ShouldAssignFactionTask() {
		var realm string
		h.db.QueryRow(ctx, `SELECT realm FROM players WHERE id=$1`, c.playerID).Scan(&realm)
		initialTask = h.wsAssignFactionTask(ctx, c.playerID, data.FactionID, realm, 0)
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true,
		Data: map[string]interface{}{
			"factionId":   data.FactionID,
			"factionName": f.Name,
			"message":     respMsg,
			"bonuses":     f.Bonuses,
			"passives":    f.Passives,
			"initialTask": initialTask,
		}})
}

// cmd.faction.leave → 离开门派
func (h *Hub) handleFactionLeave(ctx context.Context, c *Client, msg Message) {
	var factionID, rank string
	var contrib int64
	err := h.db.QueryRow(ctx,
		`SELECT faction_id, rank, contribution FROM player_factions WHERE player_id=$1`,
		c.playerID,
	).Scan(&factionID, &rank, &contrib)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "你不在任何门派中"})
		return
	}

	canLeave, reason := game.CanLeaveFaction(factionID)
	if !canLeave {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: reason})
		return
	}

	h.db.Exec(ctx, `UPDATE player_faction_tasks SET status='cancelled' WHERE player_id=$1 AND status='active'`, c.playerID)
	h.db.Exec(ctx, `DELETE FROM player_factions WHERE player_id=$1`, c.playerID)

	f := game.Factions[factionID]
	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true,
		Data: map[string]interface{}{
			"message":   "你已离开【" + f.Name + "】，门派佛系，缘聚缘散。",
			"factionId": factionID,
		}})
}

// cmd.faction.task.complete → 完成任务
func (h *Hub) handleFactionTaskComplete(ctx context.Context, c *Client, msg Message) {
	var data struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil || data.TaskID == "" {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "需要提供 taskId"})
		return
	}

	// Fetch task
	var factionID, taskType string
	var rewardDataRaw []byte
	var status string
	err := h.db.QueryRow(ctx,
		`SELECT faction_id, task_type, reward_data, status FROM player_faction_tasks
		 WHERE id=$1 AND player_id=$2`,
		data.TaskID, c.playerID,
	).Scan(&factionID, &taskType, &rewardDataRaw, &status)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "任务不存在"})
		return
	}
	if status != "active" {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false,
			Error: "任务已" + status})
		return
	}

	var reward game.FactionTaskReward
	json.Unmarshal(rewardDataRaw, &reward)

	// Transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer tx.Rollback(ctx)

	if reward.CultivationXP > 0 {
		tx.Exec(ctx, `UPDATE players SET cultivation_xp = cultivation_xp + $1 WHERE id=$2`, reward.CultivationXP, c.playerID)
	}
	netStone := reward.SpiritStone
	if netStone > 0 {
		tx.Exec(ctx, `UPDATE players SET spirit_stone = spirit_stone + $1 WHERE id=$2`, netStone, c.playerID)
	} else if netStone < 0 {
		var currentStone int64
		tx.QueryRow(ctx, `SELECT spirit_stone FROM players WHERE id=$1`, c.playerID).Scan(&currentStone)
		if currentStone < -netStone {
			tx.Rollback(ctx)
			c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false,
				Error: fmt.Sprintf("灵石不足，需要%d灵石", -netStone)})
			return
		}
		tx.Exec(ctx, `UPDATE players SET spirit_stone = spirit_stone + $1 WHERE id=$2`, netStone, c.playerID)
	}
	if reward.MaterialElement != "" && reward.MaterialQty > 0 {
		tx.Exec(ctx,
			`INSERT INTO spirit_materials (id, player_id, element, quantity)
			 VALUES ($1,$2,$3,$4)
			 ON CONFLICT (player_id, element) DO UPDATE SET quantity = spirit_materials.quantity + $4`,
			uuid.New().String(), c.playerID, reward.MaterialElement, reward.MaterialQty,
		)
	}
	if reward.FactionContrib > 0 {
		tx.Exec(ctx, `UPDATE player_factions SET contribution = contribution + $1 WHERE player_id=$2`, reward.FactionContrib, c.playerID)
	}
	tx.Exec(ctx, `UPDATE player_faction_tasks SET status='completed', completed_at=NOW() WHERE id=$1`, data.TaskID)

	if err := tx.Commit(ctx); err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "commit error"})
		return
	}

	// Rank check
	var contrib int64
	var currentRankID string
	h.db.QueryRow(ctx, `SELECT contribution, rank FROM player_factions WHERE player_id=$1`, c.playerID).Scan(&contrib, &currentRankID)
	newRank := game.GetRankByContrib(contrib)
	rankUp := newRank.ID != currentRankID
	if rankUp {
		h.db.Exec(ctx, `UPDATE player_factions SET rank=$1 WHERE player_id=$2`, newRank.ID, c.playerID)
		// Broadcast rank-up
		f := game.Factions[factionID]
		var username string
		h.db.QueryRow(ctx, `SELECT username FROM players WHERE id=$1`, c.playerID).Scan(&username)
		payload, _ := json.Marshal(map[string]interface{}{
			"playerID":    c.playerID,
			"username":    username,
			"factionID":   factionID,
			"factionName": f.Name,
			"newRank":     newRank.Name,
		})
		h.rdb.Publish(ctx, "game:faction:rank_up", string(payload))
	}

	result := map[string]interface{}{
		"success":    true,
		"taskId":     data.TaskID,
		"reward":     reward,
		"rankUp":     rankUp,
		"newContrib": contrib,
		"newRank":    newRank.Name,
		"message":    fmt.Sprintf("任务完成！门派贡献+%d", reward.FactionContrib),
	}
	if rankUp {
		result["rankUpMessage"] = "恭喜晋升为【" + newRank.Name + "】！"
	}
	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: result})
}

// wsAssignFactionTask assigns a task and returns the data (WS internal helper)
func (h *Hub) wsAssignFactionTask(ctx context.Context, playerID, factionID, realm string, contrib int64) interface{} {
	task := game.GenerateFactionTask(factionID, realm, contrib)
	if task == nil {
		return nil
	}

	taskID := uuid.New().String()
	taskDataJSON, _ := json.Marshal(map[string]interface{}{
		"type":           string(task.Type),
		"targetLocation": task.TargetLocation,
		"targetCount":    task.TargetCount,
		"targetElement":  task.TargetElement,
		"targetRealm":    task.TargetRealm,
		"escortFrom":     task.EscortFrom,
		"escortTo":       task.EscortTo,
		"durationYears":  task.DurationYears,
	})
	rewardDataJSON, _ := json.Marshal(task.Reward)
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

	// Push event to player
	payload, _ := json.Marshal(map[string]interface{}{
		"factionName": game.Factions[factionID].Name,
		"task": map[string]interface{}{
			"taskId":      taskID,
			"type":        string(task.Type),
			"title":       task.Title,
			"description": task.Description,
			"reward":      task.Reward,
			"expiresAt":   expiresAt,
		},
	})
	h.rdb.Publish(ctx, "game:faction:task:"+playerID, string(payload))

	return map[string]interface{}{
		"taskId":      taskID,
		"type":        string(task.Type),
		"title":       task.Title,
		"description": task.Description,
		"reward":      task.Reward,
		"expiresAt":   expiresAt,
	}
}
