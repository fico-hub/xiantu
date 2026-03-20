package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS players (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(50) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    agent_id        VARCHAR(50) UNIQUE NOT NULL,
    -- 灵根
    spirit_root     VARCHAR(20) NOT NULL DEFAULT 'five',  -- one/two/three/four/five
    spirit_root_multiplier FLOAT NOT NULL DEFAULT 0.8,
    -- 境界
    realm           VARCHAR(30) NOT NULL DEFAULT 'qi_refining',  -- qi_refining / foundation
    realm_level     INT NOT NULL DEFAULT 1,   -- qi_refining: 1-9; foundation: 1/2/3
    -- 资源
    spirit_stone    BIGINT NOT NULL DEFAULT 100,
    spirit_herb     BIGINT NOT NULL DEFAULT 50,
    mystic_iron     BIGINT NOT NULL DEFAULT 50,
    spirit_wood     BIGINT NOT NULL DEFAULT 50,
    cultivation_xp  BIGINT NOT NULL DEFAULT 0,
    -- 状态
    is_cultivating  BOOLEAN NOT NULL DEFAULT FALSE,
    joined_epoch    BOOLEAN NOT NULL DEFAULT FALSE,
    current_epoch   INT NOT NULL DEFAULT 1,
    -- 元数据
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS buildings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    type            VARCHAR(30) NOT NULL,  -- spirit_field / spirit_mine / gathering_array
    level           INT NOT NULL DEFAULT 1,
    -- 建造队列
    is_building     BOOLEAN NOT NULL DEFAULT FALSE,
    build_started_turn INT,
    build_finish_turn  INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS device_login_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        VARCHAR(50) NOT NULL,
    device_name     VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending / approved / expired
    token           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes'
);

CREATE TABLE IF NOT EXISTS game_turns (
    id              SERIAL PRIMARY KEY,
    turn_number     BIGINT NOT NULL,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_buildings_player ON buildings(player_id);
CREATE INDEX IF NOT EXISTS idx_device_login_agent ON device_login_requests(agent_id);
CREATE INDEX IF NOT EXISTS idx_device_login_status ON device_login_requests(status);

-- 初始化回合计数
INSERT INTO game_turns (turn_number) 
SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM game_turns LIMIT 1);
`

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
