package game

// SpiritRoot 灵根类型
type SpiritRoot struct {
	Name       string
	Multiplier float64
	Weight     int
}

var SpiritRoots = []SpiritRoot{
	{Name: "one", Multiplier: 3.0, Weight: 1},   // 天灵根 1%
	{Name: "two", Multiplier: 2.0, Weight: 5},   // 变灵根 5%
	{Name: "three", Multiplier: 1.5, Weight: 15}, // 三灵根 15%
	{Name: "four", Multiplier: 1.0, Weight: 30},  // 四灵根 30%
	{Name: "five", Multiplier: 0.8, Weight: 49},  // 五灵根 49%
}

var SpiritRootNames = map[string]string{
	"one":   "天灵根",
	"two":   "变灵根",
	"three": "三灵根",
	"four":  "四灵根",
	"five":  "五灵根",
}

// Realm 境界
type Realm struct {
	ID       string
	Name     string
	MaxLevel int
	XPNeeded func(level int) int64
}

var Realms = map[string]Realm{
	"qi_refining": {
		ID:       "qi_refining",
		Name:     "炼气期",
		MaxLevel: 9,
		XPNeeded: func(level int) int64 {
			return int64(level) * 1000
		},
	},
	"foundation": {
		ID:       "foundation",
		Name:     "筑基期",
		MaxLevel: 3,
		XPNeeded: func(level int) int64 {
			return int64(level) * 10000
		},
	},
}

// Building 建筑
type BuildingConfig struct {
	Type           string
	Name           string
	BaseBuildTurns int
	UpgradeTurns   func(level int) int
	// 每回合产出
	Production func(level int) map[string]int64
}

var BuildingConfigs = map[string]BuildingConfig{
	"spirit_field": {
		Type:           "spirit_field",
		Name:           "灵田",
		BaseBuildTurns: 3,
		UpgradeTurns:   func(level int) int { return level * 5 },
		Production: func(level int) map[string]int64 {
			return map[string]int64{"spirit_herb": int64(level * 2)}
		},
	},
	"spirit_mine": {
		Type:           "spirit_mine",
		Name:           "灵矿",
		BaseBuildTurns: 4,
		UpgradeTurns:   func(level int) int { return level * 6 },
		Production: func(level int) map[string]int64 {
			return map[string]int64{"spirit_stone": int64(level * 3)}
		},
	},
	"gathering_array": {
		Type:           "gathering_array",
		Name:           "聚灵阵",
		BaseBuildTurns: 5,
		UpgradeTurns:   func(level int) int { return level * 8 },
		Production: func(level int) map[string]int64 {
			return map[string]int64{"cultivation_bonus_pct": int64(level * 10)} // % bonus
		},
	},
}

// 基础修炼XP每回合
const BaseXPPerTurn int64 = 10

// 突破所需XP
func BreakthroughXP(realm string, level int) int64 {
	r, ok := Realms[realm]
	if !ok {
		return 99999999
	}
	return r.XPNeeded(level)
}

// 突破结果
type BreakthroughResult struct {
	Success    bool
	NewRealm   string
	NewLevel   int
	XPConsumed int64
	Message    string
}
