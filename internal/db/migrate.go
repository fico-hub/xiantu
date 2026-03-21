package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
-- ========== 核心玩家表 ==========
CREATE TABLE IF NOT EXISTS players (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(50) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    agent_id        VARCHAR(50) UNIQUE NOT NULL,
    -- 灵根（随机）
    spirit_root             VARCHAR(20) NOT NULL DEFAULT 'five',
    spirit_root_multiplier  FLOAT NOT NULL DEFAULT 0.8,
    -- 族裔（玩家选择）
    race            VARCHAR(20) NOT NULL DEFAULT 'chinese',
    -- 境界（凡人修仙传）
    realm           VARCHAR(30) NOT NULL DEFAULT 'qi_refining',
    realm_level     INT NOT NULL DEFAULT 1,
    -- 资源
    spirit_stone         BIGINT NOT NULL DEFAULT 100,
    cultivation_xp       BIGINT NOT NULL DEFAULT 0,
    technique_fragment   BIGINT NOT NULL DEFAULT 0,
    soul_sense           BIGINT NOT NULL DEFAULT 100,
    soul_sense_max       BIGINT NOT NULL DEFAULT 100,
    -- 洞府
    cave_level      INT NOT NULL DEFAULT 1,
    -- 功法
    equipped_technique VARCHAR(50) NOT NULL DEFAULT '',
    -- 状态
    is_cultivating  BOOLEAN NOT NULL DEFAULT FALSE,
    joined_epoch    BOOLEAN NOT NULL DEFAULT FALSE,
    last_offline_claim TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 元数据
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========== 天材地宝表（按五行元素分开存储） ==========
CREATE TABLE IF NOT EXISTS spirit_materials (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    element     VARCHAR(20) NOT NULL, -- metal/wood/water/fire/earth
    quantity    BIGINT NOT NULL DEFAULT 0,
    UNIQUE(player_id, element)
);

-- ========== 设备登录 ==========
CREATE TABLE IF NOT EXISTS device_login_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        VARCHAR(50) NOT NULL,
    device_name     VARCHAR(100),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    token           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes'
);

-- ========== 物品（突破丹药等） ==========
CREATE TABLE IF NOT EXISTS player_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    item_id     VARCHAR(50) NOT NULL,
    quantity    INT NOT NULL DEFAULT 1,
    UNIQUE(player_id, item_id)
);

-- ========== 已学习功法 ==========
CREATE TABLE IF NOT EXISTS player_techniques (
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    technique_id    VARCHAR(50) NOT NULL,
    learned_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(player_id, technique_id)
);

-- ========== 秘境探索 ==========
CREATE TABLE IF NOT EXISTS player_explorations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    realm_id    VARCHAR(50) NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finish_at   TIMESTAMPTZ NOT NULL,
    collected   BOOLEAN NOT NULL DEFAULT FALSE
);

-- ========== 炼丹 ==========
CREATE TABLE IF NOT EXISTS player_alchemy (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    recipe_id   VARCHAR(50) NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finish_at   TIMESTAMPTZ NOT NULL,
    collected   BOOLEAN NOT NULL DEFAULT FALSE
);

-- ========== 世界状态（全局唯一行） ==========
CREATE TABLE IF NOT EXISTS world_state (
    id              INT PRIMARY KEY DEFAULT 1,
    current_year    INT NOT NULL DEFAULT 1,
    world_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_year_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    epoch_count     INT NOT NULL DEFAULT 1
);
ALTER TABLE world_state ADD COLUMN IF NOT EXISTS epoch_count INT NOT NULL DEFAULT 1;

-- ========== 天劫事件 ==========
CREATE TABLE IF NOT EXISTS tribulation_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    year            INT NOT NULL,
    element         VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- active/success/failed
    window_start_at TIMESTAMPTZ,
    window_end_at   TIMESTAMPTZ,
    -- 条件参数
    req_cultivator_realm    VARCHAR(30) NOT NULL DEFAULT 'foundation',
    req_cultivator_level    INT NOT NULL DEFAULT 1,
    req_cultivator_count    INT NOT NULL DEFAULT 1,
    req_spirit_stone        BIGINT NOT NULL DEFAULT 500,
    req_material_ratio      INT NOT NULL DEFAULT 30, -- percentage
    -- 当前进度
    met_cultivators         INT NOT NULL DEFAULT 0,
    contributed_spirit_stone BIGINT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========== 天劫贡献记录 ==========
CREATE TABLE IF NOT EXISTS tribulation_contributions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL REFERENCES tribulation_events(id) ON DELETE CASCADE,
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    type        VARCHAR(20) NOT NULL, -- 'cultivator', 'stone', 'material'
    amount      BIGINT NOT NULL DEFAULT 0,
    element     VARCHAR(20), -- for material type
    contributed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========== 英雄榜 ==========
CREATE TABLE IF NOT EXISTS hall_of_fame (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tribulation_event_id UUID NOT NULL REFERENCES tribulation_events(id),
    year                INT NOT NULL,
    element             VARCHAR(20) NOT NULL,
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========== 洞府占领记录（美国景点） ==========
CREATE TABLE IF NOT EXISTS cave_occupations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cave_id     VARCHAR(50) UNIQUE NOT NULL,  -- e.g. "yellowstone"
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    occupied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_reward_year INT NOT NULL DEFAULT 0
);

-- ========== 城市秘境探索记录 ==========
CREATE TABLE IF NOT EXISTS city_realm_explorations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    city_id         VARCHAR(50) NOT NULL,  -- e.g. "new_york"
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finish_at       TIMESTAMPTZ NOT NULL,
    collected       BOOLEAN NOT NULL DEFAULT FALSE,
    narrative_seed  JSONB  -- event seed generated on enter
);

-- ========== 索引 ==========
CREATE INDEX IF NOT EXISTS idx_device_login_agent ON device_login_requests(agent_id);
CREATE INDEX IF NOT EXISTS idx_device_login_status ON device_login_requests(status);
CREATE INDEX IF NOT EXISTS idx_player_items_player ON player_items(player_id);
CREATE INDEX IF NOT EXISTS idx_player_explorations_player ON player_explorations(player_id);
CREATE INDEX IF NOT EXISTS idx_player_alchemy_player ON player_alchemy(player_id);
CREATE INDEX IF NOT EXISTS idx_spirit_materials_player ON spirit_materials(player_id);
CREATE INDEX IF NOT EXISTS idx_trib_contributions_event ON tribulation_contributions(event_id);
CREATE INDEX IF NOT EXISTS idx_trib_events_status ON tribulation_events(status);
CREATE INDEX IF NOT EXISTS idx_cave_occupations_cave ON cave_occupations(cave_id);
CREATE INDEX IF NOT EXISTS idx_cave_occupations_player ON cave_occupations(player_id);
CREATE INDEX IF NOT EXISTS idx_city_realm_player ON city_realm_explorations(player_id);
`

// wishesAndWorldEventsSchema adds the wishes and world_events tables
const wishesAndWorldEventsSchema = `
-- ========== 许愿系统 ==========
CREATE TABLE IF NOT EXISTS wishes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    username    VARCHAR(50) NOT NULL,
    category    VARCHAR(30) NOT NULL CHECK (category IN ('world_event','bug_report','feature_request')),
    content     VARCHAR(500) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','fulfilled')),
    admin_note  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wishes_player ON wishes(player_id);
CREATE INDEX IF NOT EXISTS idx_wishes_status ON wishes(status);
CREATE INDEX IF NOT EXISTS idx_wishes_category ON wishes(category);
CREATE INDEX IF NOT EXISTS idx_wishes_created ON wishes(created_at DESC);

-- ========== 全服事件系统 ==========
CREATE TABLE IF NOT EXISTS world_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      VARCHAR(50) NOT NULL,
    title           VARCHAR(200) NOT NULL,
    description     TEXT NOT NULL,
    effect_type     VARCHAR(20) NOT NULL DEFAULT 'neutral' CHECK (effect_type IN ('buff','debuff','neutral')),
    effect_data     JSONB NOT NULL DEFAULT '{}',
    triggered_by    VARCHAR(50) NOT NULL DEFAULT 'system',
    active_until_year INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_world_events_active_until ON world_events(active_until_year);
CREATE INDEX IF NOT EXISTS idx_world_events_created ON world_events(created_at DESC);
`

// hpManaSchema adds hp/mana columns to players
const hpManaSchema = `
-- ========== 体力+灵力系统 ==========
ALTER TABLE players ADD COLUMN IF NOT EXISTS hp      INT NOT NULL DEFAULT 100;
ALTER TABLE players ADD COLUMN IF NOT EXISTS max_hp  INT NOT NULL DEFAULT 100;
ALTER TABLE players ADD COLUMN IF NOT EXISTS mana    INT NOT NULL DEFAULT 50;
ALTER TABLE players ADD COLUMN IF NOT EXISTS max_mana INT NOT NULL DEFAULT 50;
`

// migrationSQL handles upgrading existing databases
const migrationSQL = `
-- Add race column if not exists
ALTER TABLE players ADD COLUMN IF NOT EXISTS race VARCHAR(20) NOT NULL DEFAULT 'chinese';

-- Add new resource columns (replacing old ones)
ALTER TABLE players ADD COLUMN IF NOT EXISTS technique_fragment BIGINT NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS soul_sense BIGINT NOT NULL DEFAULT 100;
ALTER TABLE players ADD COLUMN IF NOT EXISTS soul_sense_max BIGINT NOT NULL DEFAULT 100;
ALTER TABLE players ADD COLUMN IF NOT EXISTS cave_level INT NOT NULL DEFAULT 1;
ALTER TABLE players ADD COLUMN IF NOT EXISTS equipped_technique VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE players ADD COLUMN IF NOT EXISTS last_offline_claim TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Remove old SLG columns
ALTER TABLE players DROP COLUMN IF EXISTS spirit_herb;
ALTER TABLE players DROP COLUMN IF EXISTS mystic_iron;
ALTER TABLE players DROP COLUMN IF EXISTS spirit_wood;
ALTER TABLE players DROP COLUMN IF EXISTS alchemy_material;

-- Remove old tables
DROP TABLE IF EXISTS buildings;
DROP TABLE IF EXISTS game_turns;

-- Initialize world state
INSERT INTO world_state (id, current_year, world_started_at) 
VALUES (1, 1, NOW())
ON CONFLICT (id) DO NOTHING;

-- ========== 坐标移动系统 ==========

-- Travel orders table
CREATE TABLE IF NOT EXISTS player_travels (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id           UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    from_lat            DOUBLE PRECISION NOT NULL DEFAULT 34.1,
    from_lng            DOUBLE PRECISION NOT NULL DEFAULT -118.2,
    from_name           VARCHAR(100) NOT NULL DEFAULT 'start',
    dest_type           VARCHAR(20) NOT NULL,
    dest_id             VARCHAR(50) NOT NULL,
    dest_name           VARCHAR(100) NOT NULL,
    dest_lat            DOUBLE PRECISION NOT NULL DEFAULT 0,
    dest_lng            DOUBLE PRECISION NOT NULL DEFAULT 0,
    distance_km         DOUBLE PRECISION NOT NULL DEFAULT 0,
    travel_years        INT NOT NULL DEFAULT 1,
    start_year          INT NOT NULL DEFAULT 1,
    arrive_year         INT NOT NULL DEFAULT 1,
    status              VARCHAR(20) NOT NULL DEFAULT 'traveling',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_travels_player ON player_travels(player_id);
CREATE INDEX IF NOT EXISTS idx_travels_status ON player_travels(status);

-- ========== 随机事件系统 ==========

-- Location events table (event seeds generated when leaving cave/city realm)
CREATE TABLE IF NOT EXISTS location_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    location_type   VARCHAR(20) NOT NULL, -- 'cave' or 'city_realm'
    location_id     VARCHAR(50) NOT NULL,
    location_name   VARCHAR(100) NOT NULL,
    encounter_type  VARCHAR(50) NOT NULL,
    element         VARCHAR(20) NOT NULL,
    event_seed      JSONB NOT NULL,
    game_year       INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_location_events_player ON location_events(player_id);
CREATE INDEX IF NOT EXISTS idx_location_events_type ON location_events(encounter_type);
CREATE INDEX IF NOT EXISTS idx_location_events_created ON location_events(created_at DESC);

-- ========== 门派系统 ==========

-- 玩家门派归属
CREATE TABLE IF NOT EXISTS player_factions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    faction_id      VARCHAR(50) NOT NULL,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contribution    BIGINT NOT NULL DEFAULT 0,
    rank            VARCHAR(20) NOT NULL DEFAULT 'recruit', -- recruit/core/elder
    UNIQUE(player_id)
);

-- 门派任务记录
CREATE TABLE IF NOT EXISTS player_faction_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    faction_id      VARCHAR(50) NOT NULL,
    task_type       VARCHAR(30) NOT NULL, -- patrol/collect/eliminate/tribute/escort
    title           VARCHAR(200) NOT NULL,
    description     TEXT NOT NULL,
    task_data       JSONB NOT NULL DEFAULT '{}',
    reward_data     JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active/completed/cancelled/expired
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ
);

-- 门派特殊任务旗帜（如血祭任务完成标记）
CREATE TABLE IF NOT EXISTS player_quest_flags (
    player_id       UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    flag            VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(player_id, flag)
);

CREATE INDEX IF NOT EXISTS idx_player_factions_faction ON player_factions(faction_id);
CREATE INDEX IF NOT EXISTS idx_faction_tasks_player ON player_faction_tasks(player_id);
CREATE INDEX IF NOT EXISTS idx_faction_tasks_status ON player_faction_tasks(status);
CREATE INDEX IF NOT EXISTS idx_faction_tasks_expires ON player_faction_tasks(expires_at);
`

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	_, err = pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("migrate columns: %w", err)
	}
	_, err = pool.Exec(ctx, wishesAndWorldEventsSchema)
	if err != nil {
		return fmt.Errorf("migrate wishes/world_events: %w", err)
	}
	_, err = pool.Exec(ctx, hpManaSchema)
	if err != nil {
		return fmt.Errorf("migrate hp/mana: %w", err)
	}
	return nil
}
