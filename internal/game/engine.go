package game

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Engine struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewEngine(db *pgxpool.Pool, rdb *redis.Client) *Engine {
	return &Engine{db: db, rdb: rdb}
}

// Run ticks every 30 seconds (one game turn)
func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("⏱️  Game engine started (30s per turn)")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.processTurn(ctx); err != nil {
				log.Printf("Turn processing error: %v", err)
			}
		}
	}
}

// GetCurrentTurn returns the current turn number from DB
func (e *Engine) GetCurrentTurn(ctx context.Context) (int64, error) {
	var turn int64
	err := e.db.QueryRow(ctx, "SELECT MAX(turn_number) FROM game_turns").Scan(&turn)
	return turn, err
}

func (e *Engine) processTurn(ctx context.Context) error {
	// 1. Increment turn
	var turnNum int64
	err := e.db.QueryRow(ctx,
		"INSERT INTO game_turns (turn_number) SELECT COALESCE(MAX(turn_number),0)+1 FROM game_turns RETURNING turn_number",
	).Scan(&turnNum)
	if err != nil {
		return fmt.Errorf("increment turn: %w", err)
	}
	log.Printf("🔄 Turn %d processing...", turnNum)

	// 2. Complete finished buildings
	if err := e.completeBuildingsForTurn(ctx, turnNum); err != nil {
		log.Printf("complete buildings error: %v", err)
	}

	// 3. Produce resources from buildings
	if err := e.produceResources(ctx, turnNum); err != nil {
		log.Printf("produce resources error: %v", err)
	}

	// 4. Advance cultivation for cultivating players
	if err := e.advanceCultivation(ctx, turnNum); err != nil {
		log.Printf("advance cultivation error: %v", err)
	}

	// 5. Publish turn event to Redis
	e.rdb.Publish(ctx, "game:turn", fmt.Sprintf("%d", turnNum))

	return nil
}

func (e *Engine) completeBuildingsForTurn(ctx context.Context, turnNum int64) error {
	_, err := e.db.Exec(ctx,
		`UPDATE buildings SET is_building=false, build_started_turn=NULL, build_finish_turn=NULL, updated_at=NOW()
		 WHERE is_building=true AND build_finish_turn <= $1`,
		turnNum,
	)
	return err
}

func (e *Engine) produceResources(ctx context.Context, turnNum int64) error {
	// Get all players with their buildings
	rows, err := e.db.Query(ctx,
		`SELECT p.id, p.spirit_stone, p.spirit_herb, p.mystic_iron, p.spirit_wood
		 FROM players p WHERE p.joined_epoch=true`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type playerRes struct {
		id           string
		spiritStone  int64
		spiritHerb   int64
		mysticIron   int64
		spiritWood   int64
	}

	var players []playerRes
	for rows.Next() {
		var p playerRes
		if err := rows.Scan(&p.id, &p.spiritStone, &p.spiritHerb, &p.mysticIron, &p.spiritWood); err != nil {
			continue
		}
		players = append(players, p)
	}
	rows.Close()

	for _, p := range players {
		bRows, err := e.db.Query(ctx,
			`SELECT type, level FROM buildings WHERE player_id=$1 AND is_building=false`,
			p.id,
		)
		if err != nil {
			continue
		}

		var stoneAdd, herbAdd int64
		for bRows.Next() {
			var btype string
			var level int
			if err := bRows.Scan(&btype, &level); err != nil {
				continue
			}
			cfg, ok := BuildingConfigs[btype]
			if !ok {
				continue
			}
			prod := cfg.Production(level)
			if v, ok := prod["spirit_stone"]; ok {
				stoneAdd += v
			}
			if v, ok := prod["spirit_herb"]; ok {
				herbAdd += v
			}
		}
		bRows.Close()

		if stoneAdd > 0 || herbAdd > 0 {
			_, _ = e.db.Exec(ctx,
				`UPDATE players SET spirit_stone=spirit_stone+$1, spirit_herb=spirit_herb+$2, updated_at=NOW() WHERE id=$3`,
				stoneAdd, herbAdd, p.id,
			)
		}
	}

	return nil
}

func (e *Engine) advanceCultivation(ctx context.Context, turnNum int64) error {
	rows, err := e.db.Query(ctx,
		`SELECT p.id, p.cultivation_xp, p.spirit_root_multiplier, p.realm, p.realm_level
		 FROM players p WHERE p.is_cultivating=true AND p.joined_epoch=true`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type cultivator struct {
		id         string
		xp         int64
		multiplier float64
		realm      string
		realmLevel int
	}

	var cultivators []cultivator
	for rows.Next() {
		var c cultivator
		if err := rows.Scan(&c.id, &c.xp, &c.multiplier, &c.realm, &c.realmLevel); err != nil {
			continue
		}
		cultivators = append(cultivators, c)
	}
	rows.Close()

	for _, c := range cultivators {
		// Calculate XP gain from buildings
		var bonusPct int64
		bRows, _ := e.db.Query(ctx,
			`SELECT type, level FROM buildings WHERE player_id=$1 AND is_building=false AND type='gathering_array'`,
			c.id,
		)
		if bRows != nil {
			for bRows.Next() {
				var btype string
				var level int
				if err := bRows.Scan(&btype, &level); err == nil {
					cfg, ok := BuildingConfigs[btype]
					if ok {
						prod := cfg.Production(level)
						if v, ok2 := prod["cultivation_bonus_pct"]; ok2 {
							bonusPct += v
						}
					}
				}
			}
			bRows.Close()
		}

		xpGain := int64(float64(BaseXPPerTurn) * c.multiplier * (1.0 + float64(bonusPct)/100.0))
		newXP := c.xp + xpGain

		_, _ = e.db.Exec(ctx,
			`UPDATE players SET cultivation_xp=$1, updated_at=NOW() WHERE id=$2`,
			newXP, c.id,
		)
	}

	return nil
}

// RollSpiritRoot randomly assigns a spirit root
func RollSpiritRoot() (string, float64) {
	total := 0
	for _, r := range SpiritRoots {
		total += r.Weight
	}

	n := rand.Intn(total)
	acc := 0
	for _, r := range SpiritRoots {
		acc += r.Weight
		if n < acc {
			return r.Name, r.Multiplier
		}
	}
	return "five", 0.8
}
