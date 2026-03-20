package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/xiantu/server/internal/auth"
	"github.com/xiantu/server/internal/game"
)

// Message is the standard WS message format
type Message struct {
	Seq  int             `json:"seq"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Response is the standard WS response format
type Response struct {
	Seq    int         `json:"seq"`
	Type   string      `json:"type"`
	Ok     bool        `json:"ok"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type Client struct {
	conn     *fiberws.Conn
	playerID string
	agentID  string
	mu       sync.Mutex
	send     chan Response
}

func (c *Client) write(r Response) {
	select {
	case c.send <- r:
	default:
		// Drop if buffer full
	}
}

type Hub struct {
	db        *pgxpool.Pool
	rdb       *redis.Client
	engine    *game.Engine
	jwtSecret string

	mu      sync.RWMutex
	clients map[string]*Client // playerID -> client
}

func NewHub(db *pgxpool.Pool, rdb *redis.Client, engine *game.Engine, jwtSecret string) *Hub {
	return &Hub{
		db:        db,
		rdb:       rdb,
		engine:    engine,
		jwtSecret: jwtSecret,
		clients:   make(map[string]*Client),
	}
}

func (h *Hub) Run() {
	ctx := context.Background()
	sub := h.rdb.Subscribe(ctx, "game:turn")
	ch := sub.Channel()

	for msg := range ch {
		_ = msg
		// Broadcast turn event to all connected clients
		h.mu.RLock()
		for _, c := range h.clients {
			c.write(Response{
				Seq:  0,
				Type: "event.turn",
				Ok:   true,
				Data: map[string]interface{}{"message": "turn advanced"},
			})
		}
		h.mu.RUnlock()
	}
}

func (h *Hub) Handle(c *fiberws.Conn) {
	client := &Client{
		conn: c,
		send: make(chan Response, 64),
	}

	// Start writer goroutine
	go func() {
		for r := range client.send {
			data, _ := json.Marshal(r)
			client.mu.Lock()
			_ = c.WriteMessage(1, data)
			client.mu.Unlock()
		}
	}()

	// Read loop
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			client.write(Response{Seq: 0, Type: "error", Ok: false, Error: "invalid JSON"})
			continue
		}

		h.dispatch(client, msg)
	}

	// Cleanup
	if client.playerID != "" {
		h.mu.Lock()
		delete(h.clients, client.playerID)
		h.mu.Unlock()
	}
	close(client.send)
}

func (h *Hub) dispatch(c *Client, msg Message) {
	ctx := context.Background()

	switch msg.Type {
	case "auth":
		h.handleAuth(ctx, c, msg)

	case "query.my.status":
		h.requireAuth(c, msg, func() { h.handleQueryStatus(ctx, c, msg) })

	case "query.my.cave":
		h.requireAuth(c, msg, func() { h.handleQueryCave(ctx, c, msg) })

	case "query.ranking":
		h.requireAuth(c, msg, func() { h.handleQueryRanking(ctx, c, msg) })

	case "cmd.world.join":
		h.requireAuth(c, msg, func() { h.handleWorldJoin(ctx, c, msg) })

	case "cmd.cave.build":
		h.requireAuth(c, msg, func() { h.handleCaveBuild(ctx, c, msg) })

	case "cmd.cave.upgrade":
		h.requireAuth(c, msg, func() { h.handleCaveUpgrade(ctx, c, msg) })

	case "cmd.cultivate.start":
		h.requireAuth(c, msg, func() { h.handleCultivateStart(ctx, c, msg) })

	case "cmd.cultivate.break":
		h.requireAuth(c, msg, func() { h.handleCultivateBreak(ctx, c, msg) })

	case "cmd.plan.patrol":
		h.requireAuth(c, msg, func() { h.handlePlanPatrol(ctx, c, msg) })

	default:
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: fmt.Sprintf("unknown message type: %s", msg.Type)})
	}
}

func (h *Hub) requireAuth(c *Client, msg Message, fn func()) {
	if c.playerID == "" {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "not authenticated"})
		return
	}
	fn()
}

// handleAuth authenticates the WebSocket connection
func (h *Hub) handleAuth(ctx context.Context, c *Client, msg Message) {
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.write(Response{Seq: msg.Seq, Type: "auth", Ok: false, Error: "invalid auth data"})
		return
	}

	claims, err := auth.ParseToken(data.Token, h.jwtSecret)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: "auth", Ok: false, Error: "invalid token"})
		return
	}

	// Check if player needs to join world
	var joinedEpoch bool
	err = h.db.QueryRow(ctx, "SELECT joined_epoch FROM players WHERE id=$1", claims.PlayerID).Scan(&joinedEpoch)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: "auth", Ok: false, Error: "player not found"})
		return
	}

	c.playerID = claims.PlayerID
	c.agentID = claims.AgentID

	h.mu.Lock()
	h.clients[claims.PlayerID] = c
	h.mu.Unlock()

	c.write(Response{
		Seq:  msg.Seq,
		Type: "auth_ok",
		Ok:   true,
		Data: map[string]interface{}{
			"playerID":   claims.PlayerID,
			"agentID":    claims.AgentID,
			"needsJoin":  !joinedEpoch,
			"message":    "auth ok 🌙",
		},
	})
}

// handleQueryStatus returns player status
func (h *Hub) handleQueryStatus(ctx context.Context, c *Client, msg Message) {
	var p struct {
		Username      string  `json:"username"`
		SpiritRoot    string  `json:"spiritRoot"`
		Multiplier    float64 `json:"multiplier"`
		Realm         string  `json:"realm"`
		RealmLevel    int     `json:"realmLevel"`
		SpiritStone   int64   `json:"spiritStone"`
		SpiritHerb    int64   `json:"spiritHerb"`
		MysticIron    int64   `json:"mysticIron"`
		SpiritWood    int64   `json:"spiritWood"`
		CultivationXP int64   `json:"cultivationXp"`
		IsCultivating bool    `json:"isCultivating"`
		JoinedEpoch   bool    `json:"joinedEpoch"`
	}

	err := h.db.QueryRow(ctx,
		`SELECT username, spirit_root, spirit_root_multiplier, realm, realm_level,
		 spirit_stone, spirit_herb, mystic_iron, spirit_wood, cultivation_xp, is_cultivating, joined_epoch
		 FROM players WHERE id=$1`,
		c.playerID,
	).Scan(&p.Username, &p.SpiritRoot, &p.Multiplier, &p.Realm, &p.RealmLevel,
		&p.SpiritStone, &p.SpiritHerb, &p.MysticIron, &p.SpiritWood, &p.CultivationXP,
		&p.IsCultivating, &p.JoinedEpoch)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	realmInfo := game.Realms[p.Realm]
	xpNeeded := game.BreakthroughXP(p.Realm, p.RealmLevel)

	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"username":             p.Username,
			"spiritRoot":           p.SpiritRoot,
			"spiritRootName":       game.SpiritRootNames[p.SpiritRoot],
			"spiritRootMultiplier": p.Multiplier,
			"realm":                p.Realm,
			"realmName":            realmInfo.Name,
			"realmLevel":           p.RealmLevel,
			"resources": map[string]interface{}{
				"spiritStone":   p.SpiritStone,
				"spiritHerb":    p.SpiritHerb,
				"mysticIron":    p.MysticIron,
				"spiritWood":    p.SpiritWood,
				"cultivationXp": p.CultivationXP,
			},
			"xpToBreakthrough": xpNeeded,
			"xpProgress":       fmt.Sprintf("%d/%d", p.CultivationXP, xpNeeded),
			"isCultivating":    p.IsCultivating,
			"joinedEpoch":      p.JoinedEpoch,
			"currentTurn":      currentTurn,
		},
	})
}

// handleQueryCave returns cave buildings
func (h *Hub) handleQueryCave(ctx context.Context, c *Client, msg Message) {
	rows, err := h.db.Query(ctx,
		`SELECT id, type, level, is_building, build_started_turn, build_finish_turn, created_at
		 FROM buildings WHERE player_id=$1 ORDER BY created_at`,
		c.playerID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer rows.Close()

	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	var buildings []map[string]interface{}
	for rows.Next() {
		var id, btype string
		var level int
		var isBuilding bool
		var startTurn, finishTurn *int64
		var createdAt time.Time

		if err := rows.Scan(&id, &btype, &level, &isBuilding, &startTurn, &finishTurn, &createdAt); err != nil {
			continue
		}

		cfg, _ := game.BuildingConfigs[btype]
		b := map[string]interface{}{
			"id":         id,
			"type":       btype,
			"name":       cfg.Name,
			"level":      level,
			"isBuilding": isBuilding,
			"createdAt":  createdAt,
		}
		if isBuilding && finishTurn != nil {
			b["buildFinishTurn"] = *finishTurn
			b["turnsRemaining"] = *finishTurn - currentTurn
		}
		if !isBuilding {
			b["production"] = cfg.Production(level)
		}
		buildings = append(buildings, b)
	}

	if buildings == nil {
		buildings = []map[string]interface{}{}
	}

	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"buildings":   buildings,
			"currentTurn": currentTurn,
		},
	})
}

// handleQueryRanking returns top players by cultivation XP
func (h *Hub) handleQueryRanking(ctx context.Context, c *Client, msg Message) {
	rows, err := h.db.Query(ctx,
		`SELECT username, realm, realm_level, cultivation_xp, spirit_root
		 FROM players WHERE joined_epoch=true
		 ORDER BY cultivation_xp DESC LIMIT 20`,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}
	defer rows.Close()

	var ranking []map[string]interface{}
	rank := 1
	for rows.Next() {
		var username, realm, spiritRoot string
		var realmLevel int
		var xp int64
		if err := rows.Scan(&username, &realm, &realmLevel, &xp, &spiritRoot); err != nil {
			continue
		}
		realmInfo := game.Realms[realm]
		ranking = append(ranking, map[string]interface{}{
			"rank":           rank,
			"username":       username,
			"realm":          realm,
			"realmName":      realmInfo.Name,
			"realmLevel":     realmLevel,
			"cultivationXp":  xp,
			"spiritRoot":     spiritRoot,
			"spiritRootName": game.SpiritRootNames[spiritRoot],
		})
		rank++
	}

	if ranking == nil {
		ranking = []map[string]interface{}{}
	}

	c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: true, Data: map[string]interface{}{"ranking": ranking}})
}

// handleWorldJoin joins the current epoch
func (h *Hub) handleWorldJoin(ctx context.Context, c *Client, msg Message) {
	_, err := h.db.Exec(ctx,
		`UPDATE players SET joined_epoch=true, updated_at=NOW() WHERE id=$1`,
		c.playerID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"message":     "🌟 已加入修真界，纪元一正式开始！",
			"currentTurn": currentTurn,
		},
	})
}

// handleCaveBuild builds a new building
func (h *Hub) handleCaveBuild(ctx context.Context, c *Client, msg Message) {
	var data struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "invalid data"})
		return
	}

	cfg, ok := game.BuildingConfigs[data.Type]
	if !ok {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: fmt.Sprintf("unknown building type: %s. valid: spirit_field, spirit_mine, gathering_array", data.Type)})
		return
	}

	// Check if player joined epoch
	var joinedEpoch bool
	h.db.QueryRow(ctx, "SELECT joined_epoch FROM players WHERE id=$1", c.playerID).Scan(&joinedEpoch)
	if !joinedEpoch {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "must join world first (cmd.world.join)"})
		return
	}

	// Check if already has one building of this type
	var existingCount int
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM buildings WHERE player_id=$1 AND type=$2", c.playerID, data.Type).Scan(&existingCount)
	if existingCount > 0 {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: fmt.Sprintf("already have a %s, use cmd.cave.upgrade to level it up", cfg.Name)})
		return
	}

	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	finishTurn := currentTurn + int64(cfg.BaseBuildTurns)

	var buildingID string
	err := h.db.QueryRow(ctx,
		`INSERT INTO buildings (player_id, type, level, is_building, build_started_turn, build_finish_turn)
		 VALUES ($1, $2, 1, true, $3, $4) RETURNING id`,
		c.playerID, data.Type, currentTurn, finishTurn,
	).Scan(&buildingID)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	log.Printf("Player %s building %s (ID: %s), finishes turn %d", c.playerID, cfg.Name, buildingID, finishTurn)

	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"buildingId":   buildingID,
			"type":         data.Type,
			"name":         cfg.Name,
			"buildTurns":   cfg.BaseBuildTurns,
			"finishTurn":   finishTurn,
			"currentTurn":  currentTurn,
			"message":      fmt.Sprintf("🏗️ 开始建造【%s】，%d回合后完成", cfg.Name, cfg.BaseBuildTurns),
		},
	})
}

// handleCaveUpgrade upgrades a building
func (h *Hub) handleCaveUpgrade(ctx context.Context, c *Client, msg Message) {
	var data struct {
		BuildingID string `json:"buildingId"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "invalid data"})
		return
	}

	var btype string
	var level int
	var isBuilding bool
	err := h.db.QueryRow(ctx,
		`SELECT type, level, is_building FROM buildings WHERE id=$1 AND player_id=$2`,
		data.BuildingID, c.playerID,
	).Scan(&btype, &level, &isBuilding)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "building not found"})
		return
	}
	if isBuilding {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "building is already under construction"})
		return
	}

	cfg := game.BuildingConfigs[btype]
	upgradeTurns := cfg.UpgradeTurns(level)
	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	finishTurn := currentTurn + int64(upgradeTurns)

	_, err = h.db.Exec(ctx,
		`UPDATE buildings SET is_building=true, build_started_turn=$1, build_finish_turn=$2, level=level+1, updated_at=NOW() WHERE id=$3`,
		currentTurn, finishTurn, data.BuildingID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"buildingId":  data.BuildingID,
			"type":        btype,
			"name":        cfg.Name,
			"newLevel":    level + 1,
			"upgradeTurns": upgradeTurns,
			"finishTurn":  finishTurn,
			"currentTurn": currentTurn,
			"message":     fmt.Sprintf("⬆️ 开始升级【%s】至 Lv%d，%d回合后完成", cfg.Name, level+1, upgradeTurns),
		},
	})
}

// handleCultivateStart starts cultivation
func (h *Hub) handleCultivateStart(ctx context.Context, c *Client, msg Message) {
	var joinedEpoch bool
	h.db.QueryRow(ctx, "SELECT joined_epoch FROM players WHERE id=$1", c.playerID).Scan(&joinedEpoch)
	if !joinedEpoch {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "must join world first (cmd.world.join)"})
		return
	}

	_, err := h.db.Exec(ctx,
		`UPDATE players SET is_cultivating=true, updated_at=NOW() WHERE id=$1`,
		c.playerID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"message": "🧘 已进入闭关状态，每回合自动获得修为",
		},
	})
}

// handleCultivateBreak attempts breakthrough
func (h *Hub) handleCultivateBreak(ctx context.Context, c *Client, msg Message) {
	var xp int64
	var realm string
	var realmLevel int
	err := h.db.QueryRow(ctx,
		`SELECT cultivation_xp, realm, realm_level FROM players WHERE id=$1`,
		c.playerID,
	).Scan(&xp, &realm, &realmLevel)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	needed := game.BreakthroughXP(realm, realmLevel)
	if xp < needed {
		c.write(Response{
			Seq:  msg.Seq,
			Type: msg.Type,
			Ok:   false,
			Error: fmt.Sprintf("修为不足，需要 %d，当前 %d，还差 %d", needed, xp, needed-xp),
		})
		return
	}

	// Determine new realm/level
	var newRealm string
	var newLevel int
	realmInfo := game.Realms[realm]

	if realmLevel < realmInfo.MaxLevel {
		newRealm = realm
		newLevel = realmLevel + 1
	} else {
		// Break to next realm
		switch realm {
		case "qi_refining":
			newRealm = "foundation"
			newLevel = 1
		default:
			c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "already at highest realm (MVP)"})
			return
		}
	}

	// Consume XP and advance
	newXP := xp - needed
	_, err = h.db.Exec(ctx,
		`UPDATE players SET realm=$1, realm_level=$2, cultivation_xp=$3, is_cultivating=false, updated_at=NOW() WHERE id=$4`,
		newRealm, newLevel, newXP, c.playerID,
	)
	if err != nil {
		c.write(Response{Seq: msg.Seq, Type: msg.Type, Ok: false, Error: "db error"})
		return
	}

	newRealmInfo := game.Realms[newRealm]
	message := fmt.Sprintf("🎉 突破成功！进阶至【%s 第%d层】！剩余修为：%d", newRealmInfo.Name, newLevel, newXP)
	if newRealm != realm {
		message = fmt.Sprintf("🔥 大突破！踏入【%s】，天地灵气涌动！剩余修为：%d", newRealmInfo.Name, newXP)
	}

	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"success":      true,
			"prevRealm":    realm,
			"prevLevel":    realmLevel,
			"newRealm":     newRealm,
			"newRealmName": newRealmInfo.Name,
			"newLevel":     newLevel,
			"xpConsumed":   needed,
			"xpRemaining":  newXP,
			"message":      message,
		},
	})
}

// handlePlanPatrol generates a patrol task chain
func (h *Hub) handlePlanPatrol(ctx context.Context, c *Client, msg Message) {
	var data struct {
		Limit int `json:"limit"`
	}
	json.Unmarshal(msg.Data, &data)
	if data.Limit <= 0 {
		data.Limit = 4
	}

	// Get current state
	var p struct {
		Realm         string
		RealmLevel    int
		SpiritStone   int64
		SpiritHerb    int64
		CultivationXP int64
		IsCultivating bool
		JoinedEpoch   bool
	}
	h.db.QueryRow(ctx,
		`SELECT realm, realm_level, spirit_stone, spirit_herb, cultivation_xp, is_cultivating, joined_epoch
		 FROM players WHERE id=$1`, c.playerID,
	).Scan(&p.Realm, &p.RealmLevel, &p.SpiritStone, &p.SpiritHerb, &p.CultivationXP,
		&p.IsCultivating, &p.JoinedEpoch)

	// Count buildings
	var buildingCounts struct {
		spiritField    int
		spiritMine     int
		gatheringArray int
	}
	rows, _ := h.db.Query(ctx, "SELECT type, COUNT(*) FROM buildings WHERE player_id=$1 GROUP BY type", c.playerID)
	if rows != nil {
		for rows.Next() {
			var btype string
			var count int
			rows.Scan(&btype, &count)
			switch btype {
			case "spirit_field":
				buildingCounts.spiritField = count
			case "spirit_mine":
				buildingCounts.spiritMine = count
			case "gathering_array":
				buildingCounts.gatheringArray = count
			}
		}
		rows.Close()
	}

	currentTurn, _ := h.engine.GetCurrentTurn(ctx)
	xpNeeded := game.BreakthroughXP(p.Realm, p.RealmLevel)

	var actions []map[string]interface{}
	var reason string

	if !p.JoinedEpoch {
		actions = append(actions, map[string]interface{}{
			"action":  "cmd.world.join",
			"reason":  "尚未加入纪元，先加入修真界",
			"urgent":  true,
		})
	}

	if !p.IsCultivating {
		actions = append(actions, map[string]interface{}{
			"action": "cmd.cultivate.start",
			"reason": "未开始闭关，立即开始修炼",
		})
	}

	if buildingCounts.spiritField == 0 {
		actions = append(actions, map[string]interface{}{
			"action": "cmd.cave.build",
			"data":   map[string]string{"type": "spirit_field"},
			"reason": "尚无灵田，建造以产灵草",
			"turns":  3,
		})
	}

	if buildingCounts.spiritMine == 0 {
		actions = append(actions, map[string]interface{}{
			"action": "cmd.cave.build",
			"data":   map[string]string{"type": "spirit_mine"},
			"reason": "尚无灵矿，建造以产灵石",
			"turns":  4,
		})
	}

	if buildingCounts.gatheringArray == 0 {
		actions = append(actions, map[string]interface{}{
			"action": "cmd.cave.build",
			"data":   map[string]string{"type": "gathering_array"},
			"reason": "尚无聚灵阵，建造以加速修炼",
			"turns":  5,
		})
	}

	if p.CultivationXP >= xpNeeded {
		actions = append(actions, map[string]interface{}{
			"action": "cmd.cultivate.break",
			"reason": fmt.Sprintf("修为已足（%d/%d），可尝试突破！", p.CultivationXP, xpNeeded),
			"urgent": true,
		})
	}

	if len(actions) > data.Limit {
		actions = actions[:data.Limit]
	}

	if len(actions) == 0 {
		reason = "当前状态良好，继续修炼即可"
	} else {
		reason = "按优先级排列，先处理紧急项"
	}

	realmInfo := game.Realms[p.Realm]
	returnIn := 10 // turns
	c.write(Response{
		Seq:  msg.Seq,
		Type: msg.Type,
		Ok:   true,
		Data: map[string]interface{}{
			"currentTurn":      currentTurn,
			"realm":            p.Realm,
			"realmName":        realmInfo.Name,
			"realmLevel":       p.RealmLevel,
			"xpProgress":      fmt.Sprintf("%d/%d", p.CultivationXP, xpNeeded),
			"actions":          actions,
			"reason":           reason,
			"leaveReason":      "已排任务链，服务端将自动推进",
			"returnInTurns":    returnIn,
			"returnInSeconds":  returnIn * 30,
			"expectedOutcome":  "建筑完工、修为提升、资源产出",
			"wakeTriggers":     []string{"任务链全部完成", "修为足够突破", "遭遇异常"},
		},
	})
}
