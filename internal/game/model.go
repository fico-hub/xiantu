package game

import (
	"math/rand"
	"time"
)

// ========== 时间系统 ==========

const (
	// GameYearDuration: 1 game year = 5 real minutes
	GameYearDuration = 5 * time.Minute
)

// ========== 灵根体系 ==========

type SpiritRoot struct {
	Name       string
	Element    string // primary element affinity
	Multiplier float64
	Weight     int
}

var SpiritRoots = []SpiritRoot{
	{Name: "one", Element: "none", Multiplier: 3.0, Weight: 1},   // 天灵根 1%
	{Name: "two", Element: "none", Multiplier: 2.0, Weight: 5},   // 变灵根 5%
	{Name: "three", Element: "none", Multiplier: 1.5, Weight: 15}, // 三灵根 15%
	{Name: "four", Element: "none", Multiplier: 1.0, Weight: 30},  // 四灵根 30%
	{Name: "five", Element: "none", Multiplier: 0.8, Weight: 49},  // 五灵根 49%
}

var SpiritRootNames = map[string]string{
	"one":   "天灵根",
	"two":   "变灵根",
	"three": "三灵根",
	"four":  "四灵根",
	"five":  "五灵根",
}

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

// ========== 族裔体系 ==========

type Race struct {
	ID   string
	Name string
	Desc string
	// Element bonuses (percentage points)
	ElementBonus map[string]int
	// Special ability
	SpecialName string
	SpecialDesc string
	// Numeric gameplay bonuses
	CultivationSpeedPct  int  // +% to all cultivation XP gain
	IdleCultivationPct   int  // +% specifically to idle cultivation
	BreakthroughBonusPct int  // +% to breakthrough success rate
	BreakFailProtect     bool // prevent XP loss on failure (30% chance to trigger)
	RefiningQualityPct   int  // +% to alchemy quality
	FireTechSpeedPct     int  // +% to fire technique speed
}

var Races = map[string]Race{
	"african": {
		ID: "african", Name: "非裔", Desc: "大地之子，根基深厚，美国最古老的传承",
		ElementBonus: map[string]int{"earth": 25, "wood": 15},
		SpecialName:  "不屈根基", SpecialDesc: "突破失败时有30%概率不损失修为",
		BreakFailProtect: true,
	},
	"caucasian": {
		ID: "caucasian", Name: "白裔", Desc: "金水双修，精于炼器，技艺超群",
		ElementBonus: map[string]int{"metal": 25, "water": 10},
		SpecialName:  "炼器天赋", SpecialDesc: "炼器品质+15%",
		RefiningQualityPct: 15,
	},
	"latino": {
		ID: "latino", Name: "拉丁裔", Desc: "烈火传承，功法迅猛，激情四射",
		ElementBonus: map[string]int{"fire": 30, "wood": 10},
		SpecialName:  "烈焰之心", SpecialDesc: "火系功法修炼速度+20%",
		FireTechSpeedPct: 20,
	},
	"chinese": {
		ID: "chinese", Name: "华裔", Desc: "五行均衡，道法自然，根基扎实",
		ElementBonus: map[string]int{"metal": 8, "wood": 8, "water": 8, "fire": 8, "earth": 8},
		SpecialName:  "天道均衡", SpecialDesc: "修炼速度+10%",
		CultivationSpeedPct: 10,
	},
	"indigenous": {
		ID: "indigenous", Name: "原住民", Desc: "自然之灵，万物共鸣，此地真正的主人",
		ElementBonus: map[string]int{"wood": 30, "water": 20},
		SpecialName:  "自然亲和", SpecialDesc: "挂机修为+20%",
		IdleCultivationPct: 20,
	},
	"asian_pacific": {
		ID: "asian_pacific", Name: "亚太裔", Desc: "水金双修，悟性超凡，勤奋刻苦",
		ElementBonus: map[string]int{"water": 25, "metal": 10},
		SpecialName:  "超凡悟性", SpecialDesc: "突破成功率+5%",
		BreakthroughBonusPct: 5,
	},
}

var RaceOrder = []string{"african", "caucasian", "latino", "chinese", "indigenous", "asian_pacific"}

// ========== 境界体系（凡人修仙传） ==========

type RealmTier struct {
	ID         string
	Name       string
	Order      int
	MaxLevel   int
	LevelNames []string
	XPPerLevel []int64
	// Item needed to cross into this realm from previous (empty for qi_refining)
	BreakthroughItem     string
	BreakthroughItemName string
}

var RealmOrder = []string{
	"qi_refining", "foundation", "core_formation", "nascent_soul",
	"deity_transformation", "void_refining", "body_integration",
	"mahayana", "tribulation",
}

var RealmTiers = map[string]RealmTier{
	"qi_refining": {
		ID: "qi_refining", Name: "练气期", Order: 0, MaxLevel: 13,
		LevelNames: []string{
			"一层", "二层", "三层", "四层", "五层", "六层", "七层",
			"八层", "九层", "十层", "十一层", "十二层", "十三层",
		},
		// 10万每层
		XPPerLevel: []int64{
			100000, 100000, 100000, 100000, 100000, 100000, 100000,
			100000, 100000, 100000, 100000, 100000, 100000,
		},
	},
	"foundation": {
		ID: "foundation", Name: "筑基期", Order: 1, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{1500000, 2500000, 4000000, 6000000},
		BreakthroughItem:     "foundation_pill",
		BreakthroughItemName: "筑基丹",
	},
	"core_formation": {
		ID: "core_formation", Name: "结丹期", Order: 2, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{7000000, 12000000, 18000000, 25000000},
		BreakthroughItem:     "core_pill",
		BreakthroughItemName: "结丹丹",
	},
	"nascent_soul": {
		ID: "nascent_soul", Name: "元婴期", Order: 3, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{30000000, 50000000, 75000000, 100000000},
		BreakthroughItem:     "nascent_pill",
		BreakthroughItemName: "凝婴丹",
	},
	"deity_transformation": {
		ID: "deity_transformation", Name: "化神期", Order: 4, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{200000000, 350000000, 500000000, 750000000},
		BreakthroughItem:     "deity_pill",
		BreakthroughItemName: "化神丹",
	},
	"void_refining": {
		ID: "void_refining", Name: "炼虚期", Order: 5, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{800000000, 1200000000, 1800000000, 2500000000},
		BreakthroughItem:     "void_pill",
		BreakthroughItemName: "炼虚丹",
	},
	"body_integration": {
		ID: "body_integration", Name: "合体期", Order: 6, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{3000000000, 4500000000, 6500000000, 9000000000},
		BreakthroughItem:     "integration_pill",
		BreakthroughItemName: "合体丹",
	},
	"mahayana": {
		ID: "mahayana", Name: "大乘期", Order: 7, MaxLevel: 4,
		LevelNames:           []string{"初期", "中期", "后期", "大圆满"},
		XPPerLevel:           []int64{10000000000, 15000000000, 22000000000, 32000000000},
		BreakthroughItem:     "mahayana_pill",
		BreakthroughItemName: "大乘丹",
	},
	"tribulation": {
		ID: "tribulation", Name: "渡劫期", Order: 8, MaxLevel: 3,
		LevelNames:           []string{"三九天劫", "六九天劫", "九九天劫"},
		XPPerLevel:           []int64{50000000000, 100000000000, 200000000000},
		BreakthroughItem:     "tribulation_pill",
		BreakthroughItemName: "渡劫丹",
	},
}

// GetXPNeeded returns XP needed for the current realm+level stage
func GetXPNeeded(realm string, level int) int64 {
	tier, ok := RealmTiers[realm]
	if !ok {
		return 999999999999
	}
	if level < 1 || level > tier.MaxLevel {
		return 999999999999
	}
	return tier.XPPerLevel[level-1]
}

// NextRealmLevel returns what realm/level comes after breakthrough.
// isMajor indicates a cross-realm breakthrough (needs item).
func NextRealmLevel(realm string, level int) (newRealm string, newLevel int, isMajor bool) {
	tier, ok := RealmTiers[realm]
	if !ok {
		return "", 0, false
	}
	if level < tier.MaxLevel {
		return realm, level + 1, false
	}
	// Need to advance to next realm
	idx := -1
	for i, r := range RealmOrder {
		if r == realm {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(RealmOrder)-1 {
		return "", 0, false // at max realm
	}
	return RealmOrder[idx+1], 1, true
}

// RealmAtLeast checks if player's realm/level >= required realm/level
func RealmAtLeast(playerRealm string, playerLevel int, minRealm string, minLevel int) bool {
	pt, ok1 := RealmTiers[playerRealm]
	mt, ok2 := RealmTiers[minRealm]
	if !ok1 || !ok2 {
		return false
	}
	if pt.Order > mt.Order {
		return true
	}
	if pt.Order == mt.Order {
		return playerLevel >= minLevel
	}
	return false
}

// RealmDisplayName returns human-readable realm+level name
func RealmDisplayName(realm string, level int) string {
	tier, ok := RealmTiers[realm]
	if !ok {
		return realm
	}
	if level >= 1 && level <= len(tier.LevelNames) {
		return tier.Name + tier.LevelNames[level-1]
	}
	return tier.Name
}

// ========== 天劫系统 ==========

// TribulationSchedule defines when tribulations occur and their requirements
type TribulationSchedule struct {
	Year    int
	Element string // "earth", "fire", "metal", "water", "wood"
	// Requirement 1: cultivators
	ReqCultivatorRealm string
	ReqCultivatorLevel int
	ReqCultivatorCount int
	// Requirement 2: spirit stones
	ReqSpiritStone int64
	// Requirement 3: element materials (ratio as percentage)
	ReqMaterialRatio int // percentage of total materials that must be this element
	// Window duration
	WindowHours int
}

var TribulationSchedules = []TribulationSchedule{
	{
		Year: 100, Element: "earth",
		ReqCultivatorRealm: "foundation", ReqCultivatorLevel: 1, ReqCultivatorCount: 1,
		ReqSpiritStone: 500,
		ReqMaterialRatio: 30,
		WindowHours: 1,
	},
	{
		Year: 300, Element: "fire",
		ReqCultivatorRealm: "core_formation", ReqCultivatorLevel: 1, ReqCultivatorCount: 3,
		ReqSpiritStone: 10000,
		ReqMaterialRatio: 50,
		WindowHours: 1,
	},
	{
		Year: 600, Element: "metal",
		ReqCultivatorRealm: "nascent_soul", ReqCultivatorLevel: 1, ReqCultivatorCount: 3,
		ReqSpiritStone: 50000,
		ReqMaterialRatio: 60,
		WindowHours: 1,
	},
}

// GetTribulationSchedule returns the schedule for a given year, nil if none
func GetTribulationSchedule(year int) *TribulationSchedule {
	for i := range TribulationSchedules {
		if TribulationSchedules[i].Year == year {
			return &TribulationSchedules[i]
		}
	}
	// After first 3, tribulations continue every 300 years with scaling requirements
	if year > 600 && year <= 3000 && year%300 == 0 {
		epoch := (year - 600) / 300
		elements := []string{"water", "wood", "earth", "fire", "metal"}
		elem := elements[epoch%len(elements)]
		multiplier := int64(1 << uint(epoch)) // 2^epoch scaling
		return &TribulationSchedule{
			Year:               year,
			Element:            elem,
			ReqCultivatorRealm: "deity_transformation",
			ReqCultivatorLevel: 1,
			ReqCultivatorCount: epoch + 3,
			ReqSpiritStone:     200000 * multiplier,
			ReqMaterialRatio:   60,
			WindowHours:        1,
		}
	}
	return nil
}

// NextTribulationYear returns the next tribulation year >= currentYear
func NextTribulationYear(currentYear int) (year int, element string) {
	// Check fixed schedules
	for _, s := range TribulationSchedules {
		if s.Year >= currentYear {
			return s.Year, s.Element
		}
	}
	// Generate dynamic schedule
	for y := 900; y <= 3000; y += 300 {
		if y >= currentYear {
			s := GetTribulationSchedule(y)
			if s != nil {
				return y, s.Element
			}
		}
	}
	return 3000, "all"
}

// ElementChinese converts element ID to Chinese name
func ElementChinese(element string) string {
	switch element {
	case "metal":
		return "金"
	case "wood":
		return "木"
	case "water":
		return "水"
	case "fire":
		return "火"
	case "earth":
		return "土"
	case "all":
		return "五行"
	}
	return element
}

// ========== 功法体系 ==========

type Technique struct {
	ID         string
	Name       string
	Element    string // "fire", "water", "metal", "wood", "earth", "all", "none"
	MinRealm   string
	MinLevel   int
	FragCost   int
	XPBonusPct float64 // +% to cultivation XP
	Desc       string
}

var TechniqueOrder = []string{"basic_breathing", "five_elements", "purple_cloud", "azure_essence", "great_expansion"}

var Techniques = map[string]Technique{
	"basic_breathing": {
		ID: "basic_breathing", Name: "基础吐纳术", Element: "none",
		MinRealm: "qi_refining", MinLevel: 1, FragCost: 0,
		XPBonusPct: 0, Desc: "最基础的修炼法门，人人可学",
	},
	"five_elements": {
		ID: "five_elements", Name: "五行聚灵诀", Element: "all",
		MinRealm: "qi_refining", MinLevel: 5, FragCost: 10,
		XPBonusPct: 10, Desc: "聚五行灵气，修炼速度+10%",
	},
	"purple_cloud": {
		ID: "purple_cloud", Name: "紫霞神功", Element: "fire",
		MinRealm: "foundation", MinLevel: 1, FragCost: 30,
		XPBonusPct: 20, Desc: "紫霞真火淬体，修炼速度+20%",
	},
	"azure_essence": {
		ID: "azure_essence", Name: "太清化元功", Element: "water",
		MinRealm: "core_formation", MinLevel: 1, FragCost: 80,
		XPBonusPct: 35, Desc: "太清仙法化元归真，修炼速度+35%",
	},
	"great_expansion": {
		ID: "great_expansion", Name: "大衍真经", Element: "none",
		MinRealm: "nascent_soul", MinLevel: 1, FragCost: 200,
		XPBonusPct: 50, Desc: "上古真经大衍无极，修炼速度+50%",
	},
}

// ========== 秘境体系（含天材地宝属性） ==========

type SecretRealmRewards struct {
	SpiritStone      [2]int64 // [min, max]
	MaterialPerElem  [2]int64 // spirit_material per element
	TechFragment     [2]int64
}

type SecretRealm struct {
	ID          string
	Name        string
	MinRealm    string
	SoulCost    int64
	DurationSec int
	Rewards     SecretRealmRewards
	Desc        string
}

var SecretRealmOrder = []string{"herb_valley", "treasure_mountain", "alchemy_ruins", "ancient_cave"}

var SecretRealms = map[string]SecretRealm{
	"herb_valley": {
		ID: "herb_valley", Name: "灵药谷", MinRealm: "qi_refining",
		SoulCost: 20, DurationSec: 300,
		Rewards: SecretRealmRewards{
			SpiritStone:     [2]int64{50, 200},
			MaterialPerElem: [2]int64{2, 8},
			TechFragment:    [2]int64{0, 3},
		},
		Desc: "充满灵药的山谷，天材地宝丰盛",
	},
	"treasure_mountain": {
		ID: "treasure_mountain", Name: "宝药山", MinRealm: "foundation",
		SoulCost: 40, DurationSec: 600,
		Rewards: SecretRealmRewards{
			SpiritStone:     [2]int64{200, 800},
			MaterialPerElem: [2]int64{5, 20},
			TechFragment:    [2]int64{2, 8},
		},
		Desc: "蕴含丰富宝药的仙山，需筑基期以上修为",
	},
	"alchemy_ruins": {
		ID: "alchemy_ruins", Name: "炼丹炉遗迹", MinRealm: "core_formation",
		SoulCost: 60, DurationSec: 900,
		Rewards: SecretRealmRewards{
			SpiritStone:     [2]int64{500, 2000},
			MaterialPerElem: [2]int64{10, 40},
			TechFragment:    [2]int64{5, 15},
		},
		Desc: "远古炼丹师的遗迹，可能找到珍贵丹方",
	},
	"ancient_cave": {
		ID: "ancient_cave", Name: "远古修士洞府", MinRealm: "nascent_soul",
		SoulCost: 80, DurationSec: 1800,
		Rewards: SecretRealmRewards{
			SpiritStone:     [2]int64{1000, 5000},
			MaterialPerElem: [2]int64{20, 80},
			TechFragment:    [2]int64{10, 30},
		},
		Desc: "远古大能的洞府，危险与机缘并存",
	},
}

var Elements = []string{"metal", "wood", "water", "fire", "earth"}

// RollRewards generates random rewards from a secret realm, including element-attributed materials
func RollRewards(sr SecretRealm) map[string]int64 {
	rewards := make(map[string]int64)
	if sr.Rewards.SpiritStone[1] > 0 {
		r := sr.Rewards.SpiritStone
		rewards["spirit_stone"] = r[0] + rand.Int63n(r[1]-r[0]+1)
	}
	if sr.Rewards.TechFragment[1] > 0 {
		r := sr.Rewards.TechFragment
		rewards["technique_fragment"] = r[0] + rand.Int63n(r[1]-r[0]+1)
	}
	// Roll materials for each element
	if sr.Rewards.MaterialPerElem[1] > 0 {
		r := sr.Rewards.MaterialPerElem
		for _, elem := range Elements {
			qty := r[0] + rand.Int63n(r[1]-r[0]+1)
			if qty > 0 {
				rewards["material_"+elem] = qty
			}
		}
	}
	return rewards
}

// ========== 炼丹体系 ==========

// AlchemyMaterialCost represents cost in terms of element-specific materials
type AlchemyMaterialCost struct {
	Element  string
	Quantity int64
}

type AlchemyRecipe struct {
	ID           string
	Name         string
	MaterialCosts []AlchemyMaterialCost // element-specific material costs
	DurationSec  int
	MinRealm     string
	OutputItem   string // item ID produced (empty = direct XP)
	OutputQty    int
	DirectXP     int64 // if OutputItem is empty, give XP directly
	Desc         string
}

var AlchemyRecipeOrder = []string{"xp_pill", "foundation_pill", "core_pill", "nascent_pill", "deity_pill"}

var AlchemyRecipes = map[string]AlchemyRecipe{
	"xp_pill": {
		ID: "xp_pill", Name: "聚灵丹",
		MaterialCosts: []AlchemyMaterialCost{{"earth", 5}, {"wood", 5}},
		DurationSec: 180,
		MinRealm: "qi_refining",
		DirectXP: 50000,
		Desc:     "提升修为的基础丹药，服用后直接获得50000修为",
	},
	"foundation_pill": {
		ID: "foundation_pill", Name: "筑基丹",
		MaterialCosts: []AlchemyMaterialCost{{"earth", 30}, {"wood", 20}},
		DurationSec: 600,
		MinRealm: "qi_refining",
		OutputItem: "foundation_pill", OutputQty: 1,
		Desc: "练气突破筑基的必需丹药，需要土系和木系天材地宝",
	},
	"core_pill": {
		ID: "core_pill", Name: "结丹丹",
		MaterialCosts: []AlchemyMaterialCost{{"fire", 50}, {"wood", 30}, {"earth", 20}},
		DurationSec: 1200,
		MinRealm: "foundation",
		OutputItem: "core_pill", OutputQty: 1,
		Desc: "筑基突破结丹的必需丹药，需要火系天材地宝为主",
	},
	"nascent_pill": {
		ID: "nascent_pill", Name: "凝婴丹",
		MaterialCosts: []AlchemyMaterialCost{{"metal", 80}, {"water", 60}, {"fire", 40}},
		DurationSec: 1800,
		MinRealm: "core_formation",
		OutputItem: "nascent_pill", OutputQty: 1,
		Desc: "结丹突破元婴的必需丹药，需要金系为主的天材地宝",
	},
	"deity_pill": {
		ID: "deity_pill", Name: "化神丹",
		MaterialCosts: []AlchemyMaterialCost{{"metal", 200}, {"water", 150}, {"wood", 100}, {"fire", 100}, {"earth", 50}},
		DurationSec: 3600,
		MinRealm: "nascent_soul",
		OutputItem: "deity_pill", OutputQty: 1,
		Desc: "元婴突破化神的必需丹药，需要五行天材地宝",
	},
}

// ========== 洞府系统（简化） ==========

// CaveIdleBonus returns the idle cultivation bonus multiplier from cave level
// Level 1 = 0% bonus, each additional level = +5%
func CaveIdleBonus(level int) float64 {
	if level <= 1 {
		return 0
	}
	return float64(level-1) * 0.05
}

// ========== 修为计算 ==========

const (
	// Base XP per 5-minute game year
	BaseXPPerYear            int64 = 15000 // ~500/turn * 30 turns-equivalent
	SoulSenseRecoveryPerYear int64 = 5
	DefaultSoulSenseMax      int64 = 100

	// Breakthrough
	BreakthroughBaseRate       = 70 // % for major realm transitions
	BreakthroughFailXPLossPct  = 10 // % of XP lost on failure
	BreakthroughFailLossChance = 30 // % chance of XP loss on failure
)

// CalcXPPerYear calculates XP gain per game year based on all bonuses
func CalcXPPerYear(rootMultiplier float64, raceID string, caveLevel int, equippedTech string) int64 {
	base := float64(BaseXPPerYear)

	// Spirit root multiplier
	base *= rootMultiplier

	// Race bonuses
	race, ok := Races[raceID]
	if ok {
		if race.CultivationSpeedPct > 0 {
			base *= 1.0 + float64(race.CultivationSpeedPct)/100.0
		}
		if race.IdleCultivationPct > 0 {
			base *= 1.0 + float64(race.IdleCultivationPct)/100.0
		}
	}

	// Cave bonus
	base *= 1.0 + CaveIdleBonus(caveLevel)

	// Technique bonus
	if equippedTech != "" {
		if tech, ok := Techniques[equippedTech]; ok {
			base *= 1.0 + tech.XPBonusPct/100.0
		}
	}

	return int64(base)
}

// BreakthroughResult holds the outcome of a breakthrough attempt
type BreakthroughResult struct {
	Success      bool
	NewRealm     string
	NewLevel     int
	XPConsumed   int64
	XPLost       int64
	ItemConsumed string
	Message      string
}
