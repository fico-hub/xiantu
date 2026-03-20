package game

import (
	"fmt"
	"math"
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

// ElementMaterialName returns a localized spirit material name for a given element and city context
func ElementMaterialName(element, cityID string) string {
	// Per-element American-flavored material names (rotated randomly for variety)
	names := map[string][]string{
		"metal": {
			"华尔街金融灵晶", "阿拉斯加金矿砂", "硅谷芯片残魂", "科罗拉多银矿精",
			"底特律钢铁精魄", "白沙纯金灵砂",
		},
		"wood": {
			"红杉仙木髓", "大烟山古藤根", "亚马逊热带叶", "佛罗里达红树精",
			"大沼泽古木精", "奥林匹克雨林露",
		},
		"water": {
			"尼亚加拉瀑布珠", "密歇根湖灵水", "太平洋深海晶", "密西西比河泥灵",
			"五大湖水精", "切萨皮克湾盐晶",
		},
		"fire": {
			"黄石硫磺晶", "底特律钢炉残焰", "夏威夷火山岩浆珠", "哈莱姆街火晶",
			"拉斯维加斯霓虹炎", "凤凰城沙漠火精",
		},
		"earth": {
			"大峡谷赤土精", "大沙丘玄砂", "亚利桑那红岩髓", "大平原黑土精",
			"科罗拉多高原土魂", "巴德兰兹岩脉精",
		},
	}
	list, ok := names[element]
	if !ok || len(list) == 0 {
		return ElementChinese(element) + "系灵材"
	}
	return list[rand.Intn(len(list))]
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

// ========== 坐标系统（Haversine距离计算） ==========

const (
	// 基准最远距离：洛杉矶→纽约 约4500km
	MaxDistanceKm = 4500.0
	// 移动速度基准耗时（游戏年）
	TravelYearsQiRefining         = 30
	TravelYearsFoundation         = 15
	TravelYearsCoreFormation      = 8
	TravelYearsNascentSoul        = 3
	TravelYearsDeityTransformation = 1
)

// HaversineKm calculates distance in kilometers between two lat/lng points
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// TravelYears calculates game years needed to travel given distance and realm
func TravelYears(distKm float64, realm string) int {
	var baseDays int
	switch {
	case realm == "qi_refining":
		baseDays = TravelYearsQiRefining
	case realm == "foundation":
		baseDays = TravelYearsFoundation
	case realm == "core_formation":
		baseDays = TravelYearsCoreFormation
	case realm == "nascent_soul":
		baseDays = TravelYearsNascentSoul
	default:
		// deity_transformation and above
		baseDays = TravelYearsDeityTransformation
	}
	years := int(math.Round(distKm / MaxDistanceKm * float64(baseDays)))
	if years < 1 {
		years = 1
	}
	return years
}

// ========== 洞府系统（美国景点，可占领） ==========

// CaveBonusType describes what stat is boosted
type CaveBonusType string

const (
	CaveBonusCultivation    CaveBonusType = "cultivation"    // +% 修炼速度
	CaveBonusSpiritStone    CaveBonusType = "spirit_stone"   // 每年额外灵石
	CaveBonusMaterial       CaveBonusType = "material"       // 天材地宝+%
	CaveBonusBreakthrough   CaveBonusType = "breakthrough"   // 突破成功率+%
)

type LocationCave struct {
	ID          string
	Name        string
	NameEn      string
	Element     string // metal/wood/water/fire/earth
	BonusType   CaveBonusType
	BonusValue  int // percentage or flat value
	Desc        string
	Latitude    float64
	Longitude   float64
}

var CaveOrder = []string{
	"yellowstone", "grand_canyon", "seventeen_mile", "yosemite", "arches",
	"death_valley", "great_smoky", "glacier", "hawaii_volcanoes", "niagara",
	"zion", "bryce_canyon", "olympic", "acadia", "sequoia",
	"joshua_tree", "carlsbad", "white_sands", "painted_desert", "craters_moon",
	"great_sand_dunes", "saguaro", "canyonlands", "mt_rainier", "mt_st_helens",
	"everglades", "mammoth_cave", "cape_cod", "big_sur", "badlands",
}

var LocationCaves = map[string]LocationCave{
	"yellowstone":       {ID: "yellowstone", Name: "黄石仙域", NameEn: "Yellowstone", Element: "fire", BonusType: CaveBonusCultivation, BonusValue: 15, Desc: "火山地热孕育的仙域，修炼速度大幅提升", Latitude: 44.4, Longitude: -110.6},
	"grand_canyon":      {ID: "grand_canyon", Name: "大峡谷秘府", NameEn: "Grand Canyon", Element: "earth", BonusType: CaveBonusSpiritStone, BonusValue: 20, Desc: "亿万年大地之力凝聚，灵石源源不断", Latitude: 36.1, Longitude: -112.1},
	"seventeen_mile":    {ID: "seventeen_mile", Name: "17里云道", NameEn: "17-Mile Drive", Element: "metal", BonusType: CaveBonusCultivation, BonusValue: 10, Desc: "金系灵气弥漫的沿海仙道", Latitude: 36.6, Longitude: -121.9},
	"yosemite":          {ID: "yosemite", Name: "优胜美地洞天", NameEn: "Yosemite", Element: "wood", BonusType: CaveBonusCultivation, BonusValue: 12, Desc: "参天古木庇护，木灵之气充沛", Latitude: 37.8, Longitude: -119.5},
	"arches":            {ID: "arches", Name: "拱门灵穴", NameEn: "Arches NP", Element: "earth", BonusType: CaveBonusSpiritStone, BonusValue: 15, Desc: "天然拱门蕴含土系灵气，聚石之地", Latitude: 38.7, Longitude: -109.6},
	"death_valley":      {ID: "death_valley", Name: "死亡谷炼炉", NameEn: "Death Valley", Element: "fire", BonusType: CaveBonusMaterial, BonusValue: 25, Desc: "极端炎热锻造天材，天材地宝产出极丰", Latitude: 36.5, Longitude: -116.9},
	"great_smoky":       {ID: "great_smoky", Name: "大烟山隐府", NameEn: "Great Smoky Mountains", Element: "wood", BonusType: CaveBonusCultivation, BonusValue: 10, Desc: "云雾缭绕古山，木灵之气滋养", Latitude: 35.6, Longitude: -83.5},
	"glacier":           {ID: "glacier", Name: "冰川仙境", NameEn: "Glacier NP", Element: "water", BonusType: CaveBonusCultivation, BonusValue: 12, Desc: "千年冰川蕴含水系仙机", Latitude: 48.7, Longitude: -113.8},
	"hawaii_volcanoes":  {ID: "hawaii_volcanoes", Name: "夏威夷火山道场", NameEn: "Hawaii Volcanoes", Element: "fire", BonusType: CaveBonusBreakthrough, BonusValue: 5, Desc: "活火山之力助力境界突破", Latitude: 19.4, Longitude: -155.3},
	"niagara":           {ID: "niagara", Name: "尼亚加拉瀑布洞", NameEn: "Niagara Falls", Element: "water", BonusType: CaveBonusSpiritStone, BonusValue: 18, Desc: "磅礴瀑布之下，水灵聚石", Latitude: 43.1, Longitude: -79.1},
	"zion":              {ID: "zion", Name: "锡安峡谷", NameEn: "Zion NP", Element: "earth", BonusType: CaveBonusCultivation, BonusValue: 10, Desc: "红岩峡谷土系灵气浓郁", Latitude: 37.3, Longitude: -113.0},
	"bryce_canyon":      {ID: "bryce_canyon", Name: "布莱斯峡谷", NameEn: "Bryce Canyon", Element: "earth", BonusType: CaveBonusMaterial, BonusValue: 20, Desc: "奇特地貌孕育稀有天材地宝", Latitude: 37.6, Longitude: -112.2},
	"olympic":           {ID: "olympic", Name: "奥林匹克云峰", NameEn: "Olympic NP", Element: "wood", BonusType: CaveBonusCultivation, BonusValue: 8, Desc: "雨林生态木灵丰沛", Latitude: 47.8, Longitude: -123.6},
	"acadia":            {ID: "acadia", Name: "阿卡迪亚海崖", NameEn: "Acadia NP", Element: "water", BonusType: CaveBonusSpiritStone, BonusValue: 15, Desc: "海崖之上水灵汇聚成石", Latitude: 44.3, Longitude: -68.2},
	"sequoia":           {ID: "sequoia", Name: "红杉仙木林", NameEn: "Sequoia NP", Element: "wood", BonusType: CaveBonusCultivation, BonusValue: 15, Desc: "万年红杉木灵之气直冲云霄", Latitude: 36.5, Longitude: -118.6},
	"joshua_tree":       {ID: "joshua_tree", Name: "约书亚树荒原", NameEn: "Joshua Tree", Element: "metal", BonusType: CaveBonusMaterial, BonusValue: 15, Desc: "金系荒漠蕴藏丰富矿脉天材", Latitude: 33.9, Longitude: -116.0},
	"carlsbad":          {ID: "carlsbad", Name: "卡尔斯巴德地窟", NameEn: "Carlsbad Caverns", Element: "earth", BonusType: CaveBonusCultivation, BonusValue: 8, Desc: "地下洞窟土系灵气沉积", Latitude: 32.2, Longitude: -104.4},
	"white_sands":       {ID: "white_sands", Name: "白沙幻境", NameEn: "White Sands", Element: "metal", BonusType: CaveBonusCultivation, BonusValue: 10, Desc: "纯白沙漠金系灵气奇特", Latitude: 32.8, Longitude: -106.2},
	"painted_desert":    {ID: "painted_desert", Name: "彩绘沙漠", NameEn: "Painted Desert", Element: "fire", BonusType: CaveBonusMaterial, BonusValue: 18, Desc: "五彩岩层火系天材丰盛", Latitude: 35.1, Longitude: -109.8},
	"craters_moon":      {ID: "craters_moon", Name: "月亮火山坑", NameEn: "Craters of the Moon", Element: "fire", BonusType: CaveBonusCultivation, BonusValue: 12, Desc: "熔岩地貌火系灵气异常活跃", Latitude: 43.4, Longitude: -113.5},
	"great_sand_dunes":  {ID: "great_sand_dunes", Name: "大沙丘仙台", NameEn: "Great Sand Dunes", Element: "earth", BonusType: CaveBonusCultivation, BonusValue: 8, Desc: "大陆高处土系灵气聚集", Latitude: 37.7, Longitude: -105.5},
	"saguaro":           {ID: "saguaro", Name: "仙人掌森林", NameEn: "Saguaro NP", Element: "wood", BonusType: CaveBonusMaterial, BonusValue: 15, Desc: "沙漠木系奇植孕育天材", Latitude: 32.3, Longitude: -111.0},
	"canyonlands":       {ID: "canyonlands", Name: "峡谷地迷宫", NameEn: "Canyonlands", Element: "earth", BonusType: CaveBonusSpiritStone, BonusValue: 12, Desc: "峡谷迷宫藏匿灵石脉络", Latitude: 38.2, Longitude: -109.9},
	"mt_rainier":        {ID: "mt_rainier", Name: "雷尼尔雪峰", NameEn: "Mt. Rainier", Element: "water", BonusType: CaveBonusCultivation, BonusValue: 10, Desc: "冰雪覆盖高峰水系灵气浓厚", Latitude: 46.9, Longitude: -121.7},
	"mt_st_helens":      {ID: "mt_st_helens", Name: "圣海伦火山", NameEn: "Mt. St. Helens", Element: "fire", BonusType: CaveBonusMaterial, BonusValue: 20, Desc: "活火山爆发遗迹火系天材极丰", Latitude: 46.2, Longitude: -122.2},
	"everglades":        {ID: "everglades", Name: "大沼泽秘地", NameEn: "Everglades", Element: "water", BonusType: CaveBonusCultivation, BonusValue: 8, Desc: "热带湿地水系灵气滋润", Latitude: 25.3, Longitude: -80.9},
	"mammoth_cave":      {ID: "mammoth_cave", Name: "猛犸洞天", NameEn: "Mammoth Cave", Element: "earth", BonusType: CaveBonusSpiritStone, BonusValue: 10, Desc: "世界最长洞穴系统土系灵石遍布", Latitude: 37.2, Longitude: -86.1},
	"cape_cod":          {ID: "cape_cod", Name: "科德角海府", NameEn: "Cape Cod", Element: "water", BonusType: CaveBonusSpiritStone, BonusValue: 15, Desc: "海湾水系灵力汇聚成石", Latitude: 41.7, Longitude: -70.0},
	"big_sur":           {ID: "big_sur", Name: "大苏尔仙崖", NameEn: "Big Sur", Element: "metal", BonusType: CaveBonusCultivation, BonusValue: 12, Desc: "绝壁金系灵气磅礴", Latitude: 36.3, Longitude: -121.9},
	"badlands":          {ID: "badlands", Name: "蒙大拿草原", NameEn: "Badlands", Element: "metal", BonusType: CaveBonusMaterial, BonusValue: 12, Desc: "荒凉大地金系矿脉隐藏其中", Latitude: 43.9, Longitude: -102.3},
}

// CaveYearlyReward returns the yearly reward for a cave occupant
// Returns: cultivationPct (bonus %), spiritStoneFlat (per year), materialPct (bonus %), breakthroughBonus (%)
func CaveYearlyReward(cave LocationCave) (cultivationPct, spiritStoneFlat, materialPct, breakthroughBonus int) {
	switch cave.BonusType {
	case CaveBonusCultivation:
		cultivationPct = cave.BonusValue
	case CaveBonusSpiritStone:
		spiritStoneFlat = cave.BonusValue * 10 // scale: 20% -> 200 stones/year
	case CaveBonusMaterial:
		materialPct = cave.BonusValue
	case CaveBonusBreakthrough:
		breakthroughBonus = cave.BonusValue
	}
	return
}

// ========== 城市秘境系统（30个美国城市，挂机探索） ==========

type CityRealm struct {
	ID             string
	Name           string
	NameEn         string
	Elements       []string // can have multiple
	DurationSec    int      // real seconds (hours * 3600)
	SoulCost       int64
	// Base rewards (scaled by city size)
	BaseXP         int64
	BaseSpiritStone int64
	BaseMaterials  int // count per element
	Desc           string
	Latitude       float64
	Longitude      float64
}

// encounter types for narrative seeds (extended with weather_phenomenon)
var EncounterTypes = []string{"ancient_ruin", "boss_monster", "treasure_cache", "mysterious_merchant", "faction_conflict", "weather_phenomenon"}

// CaveNarrativeHints provides localized event hints for caves
var CaveNarrativeHints = map[string]map[string]string{
	"yellowstone": {
		"boss_monster":       "地底传来轰鸣，一头远古火熊从间歇泉中腾空而起...",
		"weather_phenomenon": "地热泉群突然同时喷涌，形成遮天蔽日的火系灵气云柱...",
		"ancient_ruin":       "间歇泉下方隐藏着一处印第安火修士的古老圣坛...",
		"treasure_cache":     "火山喷口边缘发现一枚封印在硫磺中的火系灵晶原石...",
		"mysterious_merchant": "一名身着牛仔皮革的老者从温泉中走来，手持火系法器...",
		"faction_conflict":   "黄石仙域的占领者与远道而来的挑战者在间歇泉前对峙...",
	},
	"grand_canyon": {
		"ancient_ruin":       "峡谷深处发现了纳瓦霍族修士留下的土系传承壁画...",
		"boss_monster":       "一只巨型秃鹰从峡谷气流中俯冲而下，双翅扇起土系旋风...",
		"weather_phenomenon": "峡谷中形成了罕见的土系灵气旋涡，将周围天材地宝吸附成球...",
		"treasure_cache":     "科罗拉多河改道露出了一处淘金时代修士埋藏的宝库...",
		"mysterious_merchant": "一名驾着骡子的商人从峡谷小道走来，驮着稀有的土系矿石...",
		"faction_conflict":   "大峡谷土系洞府引来多方势力角逐，局势剑拔弩张...",
	},
	"hawaii_volcanoes": {
		"boss_monster":       "基拉韦厄火山口喷出一头熔岩火蜥蜴，体长数丈...",
		"weather_phenomenon": "火山喷发与太平洋气流交汇，形成罕见的水火交融天象...",
		"ancient_ruin":       "熔岩隧道深处发现了夏威夷原住民祭司的火系修炼秘室...",
		"treasure_cache":     "新流出的岩浆凝固后，内含一枚天然形成的火系灵丹...",
		"mysterious_merchant": "一名戴着花环的老人从火山口走来，说是能以火石换取传承...",
		"faction_conflict":   "火山道场乃破境圣地，数位修士同时到来，难免一场争夺...",
	},
	"niagara": {
		"ancient_ruin":       "瀑布背后的洞穴中藏有易洛魁族水修士的传承石刻...",
		"boss_monster":       "瀑布激流中涌现一条水系蛟龙，水压足以洞穿金铁...",
		"weather_phenomenon": "瀑布在月光下形成彩虹光柱，水系灵气瞬间暴涨数倍...",
		"treasure_cache":     "瀑布底部的深潭中沉睡着一批被遗忘的水系灵石矿...",
		"mysterious_merchant": "身着渔夫外套的散修在瀑布旁垂钓，钓上来的都是灵草...",
		"faction_conflict":   "美加边境的门派为瀑布洞府归属权展开激烈交涉...",
	},
	"glacier": {
		"boss_monster":       "冰川深处惊醒了一只冰封千年的雪熊，寒气凛冽...",
		"weather_phenomenon": "极光在冰原上空凝聚成巨大的水系阵法，将整片天地染成绿色...",
		"ancient_ruin":       "融冰中露出了黑脚族冰修士的冬眠洞穴，内有传承...",
		"treasure_cache":     "冰川融水中漂来一枚包裹在冰晶中的水系宝珠...",
		"mysterious_merchant": "冰原上独自跋涉的行商摊开皮草，里面全是水系天材...",
		"faction_conflict":   "两支修仙队伍在冰川仙境狭路相逢，剑拔弩张...",
	},
	"everglades": {
		"boss_monster":       "大沼泽深处一条古鳄从水中抬头，眼中闪烁着灵智...",
		"weather_phenomenon": "热带气旋掠过，大沼泽水系灵气短暂爆发至极值...",
		"ancient_ruin":       "沼泽底部发现了塞米诺尔族巫医的水木双修圣坛...",
		"treasure_cache":     "鳄鱼聚集之处，竟是一处被保护的天然灵药圃...",
		"mysterious_merchant": "一名戴着宽边帽的老者坐在独木舟上兜售稀有水系草药...",
		"faction_conflict":   "佛罗里达门派与外来修士为大沼泽探索权起了冲突...",
	},
}

// GetCaveNarrativeHint returns a localized narrative hint for a cave event
func GetCaveNarrativeHint(caveID, encounterType, element string) string {
	if hints, ok := CaveNarrativeHints[caveID]; ok {
		if hint, ok := hints[encounterType]; ok {
			return hint
		}
	}
	cave, ok := LocationCaves[caveID]
	if !ok {
		return "在此地遭遇了神秘事件..."
	}
	// Fallback generic hints
	switch encounterType {
	case "boss_monster":
		return fmt.Sprintf("在【%s】遭遇了一头强大的%s系妖兽...", cave.Name, ElementChinese(element))
	case "ancient_ruin":
		return fmt.Sprintf("在【%s】发现了上古修士遗迹，似乎与美国原住民传承有关...", cave.Name)
	case "treasure_cache":
		return fmt.Sprintf("在【%s】发现了一处隐藏的宝藏，散发着%s系灵气...", cave.Name, ElementChinese(element))
	case "mysterious_merchant":
		return fmt.Sprintf("在【%s】遇到了一位神秘商人，愿以灵石换取稀有材料...", cave.Name)
	case "faction_conflict":
		return fmt.Sprintf("在【%s】遭遇了门派冲突，各方势力角逐此地...", cave.Name)
	case "weather_phenomenon":
		return fmt.Sprintf("在【%s】目睹了天象奇观，%s系灵气暴涨...", cave.Name, ElementChinese(element))
	}
	return fmt.Sprintf("在【%s】遭遇了神秘事件...", cave.Name)
}

// GenerateCaveEventSeed generates an event seed when leaving a cave
func GenerateCaveEventSeed(caveID string, playerRealm string, yearsOccupied int) map[string]interface{} {
	cave, ok := LocationCaves[caveID]
	if !ok {
		return nil
	}
	encounterType := EncounterTypes[rand.Intn(len(EncounterTypes))]
	hint := GetCaveNarrativeHint(caveID, encounterType, cave.Element)

	// Scale drops based on realm order and years occupied
	realmOrder := 0
	if tier, ok := RealmTiers[playerRealm]; ok {
		realmOrder = tier.Order
	}
	baseStone := int64((realmOrder+1)*50 + yearsOccupied*20)
	baseMat := int64((realmOrder+1)*3 + yearsOccupied)

	return map[string]interface{}{
		"location":       cave.Name,
		"location_id":    cave.ID,
		"location_type":  "cave",
		"element":        ElementChinese(cave.Element),
		"encounter_type": encounterType,
		"drops": map[string]interface{}{
			"spirit_stone":        baseStone + rand.Int63n(baseStone/2+1),
			"spirit_material_qty": baseMat + rand.Int63n(baseMat/2+1),
			"material_element":    cave.Element,
		},
		"narrative_hint": hint,
	}
}

// GenerateCityRealmEventSeed generates an event seed when leaving a city realm
func GenerateCityRealmEventSeed(cityID string, playerRealm string, durationSec int) map[string]interface{} {
	cr, ok := CityRealms[cityID]
	if !ok {
		return nil
	}
	encounterType := EncounterTypes[rand.Intn(len(EncounterTypes))]
	primaryElement := cr.Elements[0]

	hint := "在" + cr.Name + "探索中发现了神秘的灵气涌动"
	if hints, ok := CityNarrativeHints[cityID]; ok && len(hints) > 0 {
		hint = hints[rand.Intn(len(hints))]
	}

	realmOrder := 0
	if tier, ok := RealmTiers[playerRealm]; ok {
		realmOrder = tier.Order
	}
	baseStone := int64((realmOrder+1)*30 + durationSec/300)
	baseXP := int64(realmOrder+1)*5000 + int64(durationSec)*10

	return map[string]interface{}{
		"location":       cr.Name,
		"location_id":    cr.ID,
		"location_type":  "city_realm",
		"element":        ElementChinese(primaryElement),
		"encounter_type": encounterType,
		"drops": map[string]interface{}{
			"spirit_stone":    baseStone + rand.Int63n(baseStone/2+1),
			"cultivation_xp":  baseXP + rand.Int63n(baseXP/5+1),
		},
		"narrative_hint": hint,
	}
}

var CityRealmOrder = []string{
	"new_york", "los_angeles", "chicago", "houston", "phoenix",
	"philadelphia", "san_antonio", "san_diego", "dallas", "san_francisco",
	"seattle", "boston", "denver", "miami", "atlanta",
	"detroit", "minneapolis", "st_louis", "new_orleans", "portland",
	"las_vegas", "salt_lake_city", "albuquerque", "austin", "nashville",
	"charlotte", "columbus", "indianapolis", "jacksonville", "baltimore",
}

var CityRealms = map[string]CityRealm{
	"new_york":       {ID: "new_york", Name: "纽约·曼哈顿", NameEn: "New York", Elements: []string{"metal"}, DurationSec: 8 * 3600, SoulCost: 12, BaseXP: 4500, BaseSpiritStone: 200, BaseMaterials: 2, Desc: "金融之都，金系灵气极为浓郁", Latitude: 40.7, Longitude: -74.0},
	"los_angeles":    {ID: "los_angeles", Name: "洛杉矶·好莱坞", NameEn: "Los Angeles", Elements: []string{"fire"}, DurationSec: 6 * 3600, SoulCost: 10, BaseXP: 3500, BaseSpiritStone: 160, BaseMaterials: 2, Desc: "娱乐之都，火系灵气热烈奔放", Latitude: 34.1, Longitude: -118.2},
	"chicago":        {ID: "chicago", Name: "芝加哥·风城", NameEn: "Chicago", Elements: []string{"metal", "water"}, DurationSec: 5 * 3600, SoulCost: 9, BaseXP: 3000, BaseSpiritStone: 140, BaseMaterials: 2, Desc: "风城双修，金水交融", Latitude: 41.9, Longitude: -87.6},
	"houston":        {ID: "houston", Name: "休斯顿·炼油城", NameEn: "Houston", Elements: []string{"fire", "earth"}, DurationSec: 4 * 3600, SoulCost: 8, BaseXP: 2500, BaseSpiritStone: 120, BaseMaterials: 2, Desc: "石油之城，火土双系灵气旺盛", Latitude: 29.8, Longitude: -95.4},
	"phoenix":        {ID: "phoenix", Name: "凤凰城·烈焰", NameEn: "Phoenix", Elements: []string{"fire"}, DurationSec: 3 * 3600, SoulCost: 7, BaseXP: 2000, BaseSpiritStone: 100, BaseMaterials: 1, Desc: "沙漠火城，火系灵气炽热", Latitude: 33.4, Longitude: -112.1},
	"philadelphia":   {ID: "philadelphia", Name: "费城·古都", NameEn: "Philadelphia", Elements: []string{"earth"}, DurationSec: 3 * 3600, SoulCost: 7, BaseXP: 2000, BaseSpiritStone: 100, BaseMaterials: 1, Desc: "历史古都，土系灵气深厚", Latitude: 40.0, Longitude: -75.2},
	"san_antonio":    {ID: "san_antonio", Name: "圣安东尼奥·边城", NameEn: "San Antonio", Elements: []string{"earth"}, DurationSec: 3 * 3600, SoulCost: 6, BaseXP: 1800, BaseSpiritStone: 90, BaseMaterials: 1, Desc: "边境之城，土系灵气绵延", Latitude: 29.4, Longitude: -98.5},
	"san_diego":      {ID: "san_diego", Name: "圣迭戈·海湾", NameEn: "San Diego", Elements: []string{"water"}, DurationSec: 3 * 3600, SoulCost: 6, BaseXP: 1800, BaseSpiritStone: 90, BaseMaterials: 1, Desc: "海湾城市，水系灵气充盈", Latitude: 32.7, Longitude: -117.2},
	"dallas":         {ID: "dallas", Name: "达拉斯·牛仔城", NameEn: "Dallas", Elements: []string{"metal"}, DurationSec: 3 * 3600, SoulCost: 7, BaseXP: 2000, BaseSpiritStone: 100, BaseMaterials: 1, Desc: "金融牛仔城，金系灵气聚集", Latitude: 32.8, Longitude: -96.8},
	"san_francisco":  {ID: "san_francisco", Name: "旧金山·金门", NameEn: "San Francisco", Elements: []string{"water", "metal"}, DurationSec: 2 * 3600, SoulCost: 6, BaseXP: 1500, BaseSpiritStone: 80, BaseMaterials: 1, Desc: "金门之城，水金交汇", Latitude: 37.8, Longitude: -122.4},
	"seattle":        {ID: "seattle", Name: "西雅图·雨城", NameEn: "Seattle", Elements: []string{"water", "wood"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "常年烟雨，水木共生", Latitude: 47.6, Longitude: -122.3},
	"boston":         {ID: "boston", Name: "波士顿·学城", NameEn: "Boston", Elements: []string{"water"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "学术重镇，水系灵气清澈", Latitude: 42.4, Longitude: -71.1},
	"denver":         {ID: "denver", Name: "丹佛·高原", NameEn: "Denver", Elements: []string{"metal", "wood"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "高原之城，金木双修", Latitude: 39.7, Longitude: -104.9},
	"miami":          {ID: "miami", Name: "迈阿密·热浪", NameEn: "Miami", Elements: []string{"water", "fire"}, DurationSec: int(1.5 * 3600), SoulCost: 4, BaseXP: 900, BaseSpiritStone: 55, BaseMaterials: 1, Desc: "热带海滨，水火激荡", Latitude: 25.8, Longitude: -80.2},
	"atlanta":        {ID: "atlanta", Name: "亚特兰大·枢纽", NameEn: "Atlanta", Elements: []string{"wood", "fire"}, DurationSec: int(1.5 * 3600), SoulCost: 4, BaseXP: 900, BaseSpiritStone: 55, BaseMaterials: 1, Desc: "南方枢纽，木火旺盛", Latitude: 33.7, Longitude: -84.4},
	"detroit":        {ID: "detroit", Name: "底特律·钢铁城", NameEn: "Detroit", Elements: []string{"metal"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "汽车钢铁之城，金系灵气厚重", Latitude: 42.3, Longitude: -83.0},
	"minneapolis":    {ID: "minneapolis", Name: "明尼阿波利斯·湖城", NameEn: "Minneapolis", Elements: []string{"water"}, DurationSec: int(1.5 * 3600), SoulCost: 4, BaseXP: 900, BaseSpiritStone: 55, BaseMaterials: 1, Desc: "千湖之城，水系灵气充裕", Latitude: 44.9, Longitude: -93.2},
	"st_louis":       {ID: "st_louis", Name: "圣路易斯·门城", NameEn: "St. Louis", Elements: []string{"earth"}, DurationSec: 1 * 3600, SoulCost: 3, BaseXP: 600, BaseSpiritStone: 40, BaseMaterials: 1, Desc: "西大门土系灵气稳固", Latitude: 38.6, Longitude: -90.2},
	"new_orleans":    {ID: "new_orleans", Name: "新奥尔良·爵士城", NameEn: "New Orleans", Elements: []string{"water", "wood"}, DurationSec: int(1.5 * 3600), SoulCost: 4, BaseXP: 900, BaseSpiritStone: 55, BaseMaterials: 1, Desc: "爵士之都，水木交融神秘", Latitude: 30.0, Longitude: -90.1},
	"portland":       {ID: "portland", Name: "波特兰·玫瑰城", NameEn: "Portland", Elements: []string{"wood", "water"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "玫瑰花城，木水滋养", Latitude: 45.5, Longitude: -122.7},
	"las_vegas":      {ID: "las_vegas", Name: "拉斯维加斯·幻城", NameEn: "Las Vegas", Elements: []string{"fire", "metal"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "不夜幻城，火金交织", Latitude: 36.2, Longitude: -115.2},
	"salt_lake_city": {ID: "salt_lake_city", Name: "盐湖城·圣地", NameEn: "Salt Lake City", Elements: []string{"earth"}, DurationSec: 1 * 3600, SoulCost: 3, BaseXP: 600, BaseSpiritStone: 40, BaseMaterials: 1, Desc: "盐湖圣地，土系灵气净化", Latitude: 40.8, Longitude: -111.9},
	"albuquerque":    {ID: "albuquerque", Name: "阿尔伯克基·热气球", NameEn: "Albuquerque", Elements: []string{"earth", "fire"}, DurationSec: int(1.5 * 3600), SoulCost: 4, BaseXP: 900, BaseSpiritStone: 55, BaseMaterials: 1, Desc: "沙漠热气球，土火弥漫", Latitude: 35.1, Longitude: -106.7},
	"austin":         {ID: "austin", Name: "奥斯汀·创新城", NameEn: "Austin", Elements: []string{"wood", "fire"}, DurationSec: 2 * 3600, SoulCost: 5, BaseXP: 1200, BaseSpiritStone: 70, BaseMaterials: 1, Desc: "创新之城，木火并进", Latitude: 30.3, Longitude: -97.7},
	"nashville":      {ID: "nashville", Name: "纳什维尔·音乐城", NameEn: "Nashville", Elements: []string{"wood"}, DurationSec: 2 * 3600, SoulCost: 4, BaseXP: 1000, BaseSpiritStone: 60, BaseMaterials: 1, Desc: "音乐之都，木系灵气悠扬", Latitude: 36.2, Longitude: -86.8},
	"charlotte":      {ID: "charlotte", Name: "夏洛特·金融城", NameEn: "Charlotte", Elements: []string{"wood", "earth"}, DurationSec: 2 * 3600, SoulCost: 4, BaseXP: 1000, BaseSpiritStone: 60, BaseMaterials: 1, Desc: "南方金融，木土平衡", Latitude: 35.2, Longitude: -80.8},
	"columbus":       {ID: "columbus", Name: "哥伦布·心城", NameEn: "Columbus", Elements: []string{"earth"}, DurationSec: 2 * 3600, SoulCost: 4, BaseXP: 1000, BaseSpiritStone: 60, BaseMaterials: 1, Desc: "大陆心脏，土系灵气平稳", Latitude: 40.0, Longitude: -82.9},
	"indianapolis":   {ID: "indianapolis", Name: "印第安纳波利斯·赛城", NameEn: "Indianapolis", Elements: []string{"earth", "metal"}, DurationSec: 2 * 3600, SoulCost: 4, BaseXP: 1000, BaseSpiritStone: 60, BaseMaterials: 1, Desc: "赛道之城，土金交叠", Latitude: 39.8, Longitude: -86.2},
	"jacksonville":   {ID: "jacksonville", Name: "杰克逊维尔·河城", NameEn: "Jacksonville", Elements: []string{"water", "wood"}, DurationSec: 2 * 3600, SoulCost: 4, BaseXP: 1000, BaseSpiritStone: 60, BaseMaterials: 1, Desc: "河流之城，水木共融", Latitude: 30.3, Longitude: -81.7},
	"baltimore":      {ID: "baltimore", Name: "巴尔的摩·港城", NameEn: "Baltimore", Elements: []string{"water"}, DurationSec: int(1.5 * 3600), SoulCost: 4, BaseXP: 900, BaseSpiritStone: 55, BaseMaterials: 1, Desc: "海港之城，水系灵气汇聚", Latitude: 39.3, Longitude: -76.6},
}

// CityNarrativeHints provides flavor hints for narrative seed generation
var CityNarrativeHints = map[string][]string{
	"new_york": {
		"地铁施工挖穿了18世纪华人修士的秘密道场，华尔街金融灵晶散落一地",
		"帝国大厦顶层封印着一头金系妖兽，金融大鳄摘下领带，露出爪痕",
		"中央公园深夜，哈莱姆的老修士在草地上打太极，周身金系灵气如飓风旋转",
		"曼哈顿基岩深处有一处上古阵法节点，华尔街的每次股灾都与它有关",
	},
	"los_angeles": {
		"好莱坞星光大道的星星封印着火系阵法，某位已故明星其实是火修大能",
		"比佛利山庄某豪宅地下暗藏炼火密室，房产中介对买家只字不提",
		"圣莫尼卡海滩落日时分，水火交汇灵气暴涨，冲浪者中有三位是元婴修士",
		"洛杉矶地震断层其实是一条沉睡的火系地脉，每次余震都是它翻身",
	},
	"chicago": {
		"芝加哥风城的持续强风是一头古老风妖的气息，金水交汇之处藏有秘宝",
		"密歇根湖底沉睡着一位水系古修士，1871年大火是他打坐走火入魔引发的",
		"摩天大楼群在雷雨天会自发形成金系天雷阵，普通人眼中只是闪电",
		"芝加哥地下管道系统里有修士聚居点，禁酒令年代就有他们的身影",
	},
	"houston": {
		"石油钻井穿透地脉时释放出封印千年的火系土灵，工人们都以为是天然气",
		"NASA约翰逊航天中心某实验室研究的其实是太空灵气采集技术",
		"德克萨斯大草原土系灵气浓郁，牛仔骑的马有些其实是驯化的土系妖兽",
		"休斯顿石油交易所的大宗交易背后，有修士在操控五行灵材的流向",
	},
	"phoenix": {
		"凤凰城的名字来自一头涅槃重生的火凤，它每隔百年还会现世一次",
		"索诺兰沙漠深处的仙人掌林里，有纳瓦霍族火修老人坐镇传承",
		"凤凰城夏日高温是火系灵气外溢，在这里修炼火系功法速度翻倍",
		"斯科茨代尔豪宅区的某个游泳池底，封印着一位古老的火灵",
	},
	"san_francisco": {
		"金门大桥两端各有一处水金阵法节点，修士骑共享单车视察时发现了秘密",
		"旧金山唐人街地下有华裔炼丹师的传承秘室，入口在某家茶馆的冷藏库后",
		"1906年大地震其实是一次金系天劫失败，整座城市重置了一次",
		"硅谷某科技公司CEO其实是水金双修大能，公司产品是炼器的副产品",
	},
	"seattle": {
		"西雅图常年阴雨滋养了一片水木双修圣地，咖啡馆里每桌都坐着修士",
		"太空针塔顶封印着一只古老的雨系妖兽，风雨天才是它最活跃的时刻",
		"亚马逊总部地下有一处自动化运转的水系传送阵，已运行十五年",
		"西雅图地下城（旧城区）是修士们的秘密集市，观光游客只能看到表层",
	},
	"boston": {
		"哈佛大学某图书馆地下室藏有一批清教徒修士带来的古老修仙典籍",
		"波士顿港的海水中封印着独立战争时期一位爱国修士的最后遗产",
		"自由之路沿途有七处历史遗迹其实是修仙阵法节点，游客踩过去毫不知情",
		"麻省理工某实验室研究的量子纠缠其实是水系心灵感应技术的现代版",
	},
	"las_vegas": {
		"赌场老板摘下墨镜，露出金色竖瞳——拉斯维加斯的赌运其实由概率阵法控制",
		"内华达51区是一处金系灵矿秘境，政府用军事禁区作为掩护",
		"拉斯维加斯的霓虹灯在特定频率下会共振形成火金交织的天然阵法",
		"某赌场百年老赌徒其实是一位元婴修士，用金系功法算牌从未输过",
	},
	"miami": {
		"迈阿密海滩的古巴裔老修士每天清晨在海边打太极，水火双修登峰造极",
		"大沼泽国家公园里有塞米诺尔族水修秘境，鳄鱼是守护者",
		"南海滩Art Deco建筑群是1920年代修士们集体布下的火系防护阵",
		"迈阿密港某货船定期运送封装在冰块里的水系灵材，伪装成海鲜进口",
	},
	"atlanta": {
		"亚特兰大内战战场下埋着南北双方修士的遗留法器，木火灵气至今未散",
		"桃树街地下木系灵脉绵延十里，亚特兰大之所以处处是桃树，因为灵气滋养",
		"可口可乐秘方保险库旁边还有一个保险库，里面装的是木火灵丹配方",
		"哈茨菲尔德机场是东南部修士们最大的集散地，每天有数十位修士途经",
	},
	"detroit": {
		"废弃工厂里，钢铁妖灵在熔炉中苏醒，双眼如两团白炽火焰",
		"底特律河底有一处水系古修士洞府，汽车工业鼎盛时期他选择了冬眠",
		"某废弃汽车工厂的金系灵气凝聚成块，非裔老修士在此闭关三十年",
		"底特律的城市复兴运动其实由一批土金双修修士主导，他们要重铸钢城",
	},
	"minneapolis": {
		"明尼苏达万湖之地水系灵气如网，沉睡在密西西比河源头的水妖即将苏醒",
		"苏族原住民老巫医说：这里的湖泊是远古水系大阵的节点，不可随意涉足",
		"连锁湖泊在冬夜冰封时会构成天然水系阵法，极光就是它的运转轨迹",
		"明尼阿波利斯艺术区某画廊老板是水修大能，他的画能让人看见灵界",
	},
	"st_louis": {
		"密西西比河在圣路易斯转弯处形成土系灵气旋涡，淘金热时代修士们聚集于此",
		"拱门纪念碑是一处土系灵气聚集的天然阵法，登顶时会短暂感应地脉",
		"圣路易斯是美国修仙界的十字路口，东西两派修士在此谈判了三百年",
		"密西西比河畔的旧仓库区，地下还保留着淘金时代修士们建造的秘密据点",
	},
	"new_orleans": {
		"爵士乐中藏着古老的土系咒语，在波旁街夜晚演奏时灵效最强",
		"新奥尔良伏都教仪式其实是水木双修功法的民间演化版本，效果出奇地好",
		"法国区某百年餐厅的招牌秋葵汤配方，是一位古代木修遗留的温补灵药",
		"飓风来袭前，新奥尔良的修士们会感知到水系灵气紊乱——这是他们的气象台",
	},
	"portland": {
		"波特兰玫瑰园中木系灵气异常浓郁，每朵玫瑰都是微型天然灵药",
		"威拉米特河畔有一处水木双修圣地，只有在细雨朦胧时才会出现入口",
		"波特兰地下城（Prohibition时代地道）是木修们的秘密传承地，至今有人守护",
		"波特兰的文艺独立精神其实是大量木修聚居导致的木系灵气渗透效应",
	},
	"salt_lake_city": {
		"大盐湖的盐晶净化土系灵气，是制作高品质土系灵丹的绝佳原料",
		"锡安山下某教堂地下室藏有早期移民修士留下的传承，已被遗忘两百年",
		"大盐湖正在萎缩——因为土系灵气被过度开采，修士们对此讳莫如深",
		"盐湖城的净土气场让所有修士保持头脑清醒，是最好的突破悟道之地",
	},
	"albuquerque": {
		"热气球节升空时汇聚大量土火灵气，某些气球里乘坐的根本不是普通人",
		"纳瓦霍族修士在查科峡谷守护着一处土系大阵遗迹，已传承千年",
		"新墨西哥沙漠的土系灵气如此浓郁，不修炼的人住久了也会有些许感应",
		"阿尔伯克基某爆米花店老板（Breaking Bad取景地附近）是传奇炼丹师",
	},
	"austin": {
		"奥斯汀蝙蝠桥每到黄昏蝙蝠出没时，木系灵气会随蝙蝠群短暂爆发",
		"奥斯汀音乐节上有位吉他手边弹边修炼，火木双修，已是筑基大圆满",
		"科罗拉多河在奥斯汀的河段有天然木火双修灵脉，德州烧烤的独特风味源于此",
		"硅山（Silicon Hills）某创业公司地下室是木修们的秘密孵化器",
	},
	"nashville": {
		"纳什维尔乡村音乐里藏着木系音修功法，歌手们唱得越久境界越高",
		"大奥普里剧院建在一处木系灵脉节点上，每次演出都是一次集体修炼",
		"康伯兰河畔木系灵药生长茂盛，当地草药医生其实是传承数代的木修",
		"百老汇街某酒吧的老鸽子是一位隐居的木修高人，他在等待天劫来临",
	},
	"charlotte": {
		"夏洛特银行区的金融灵气与郊区树林的木系灵气形成奇特的木金相生格局",
		"NASCAR总部附近土金灵气交叠，赛车手中有几位其实在用金系功法提升反应速度",
		"卡特巴河两岸木系灵气茂盛，美洲原住民卡托巴族修士至今在此守护灵脉",
		"夏洛特某高楼顶层是修士议事厅，南方各门派代表定期在此开会",
	},
	"columbus": {
		"哥伦布是美国大陆的地理中心，土系灵气在此积淀最为深厚平稳",
		"俄亥俄州立大学某考古系教授私下是土修大能，他发现的遗迹从不对外公布",
		"哥伦布城市地下有一处天然土系洞天，最早的殖民地修士在此建立了据点",
		"中西部土系灵气孕育的朴实性格，让哥伦布修士们以坚韧著称",
	},
	"indianapolis": {
		"印第安纳波利斯500英里赛道在比赛时金系灵气急剧激荡，观众中有修士专门来汲取",
		"怀特河在城中蜿蜒，土系灵气沿河床沉积，厚实如磐石",
		"某维修站技师是土金双修大能，他调校的赛车从不会在关键时刻抛锚",
		"印第安纳州曾是美国原住民修士最密集之地，古老土系传承至今未断",
	},
	"jacksonville": {
		"圣约翰斯河是南方最大的水木灵脉交汇处，逆流而上的河段有古修士遗迹",
		"杰克逊维尔港口的集装箱中有专门运输灵材的隐秘货物，走的是合法渠道",
		"佛罗里达北部水木双修天材随季节轮转，雨季尤为丰盛",
		"某退休海军上将在水边定居，其实是在守护一处水系灵眼",
	},
	"baltimore": {
		"切萨皮克湾水系灵气磅礴，螃蟹里有时含有微量水系灵材成分，吃了有益修炼",
		"巴尔的摩内港有一处水系古修士洞府，入口在某艘废弃的帆船船底",
		"联邦山上日落时分可感受到整片海湾的水系灵脉涌动，是修士们的朝圣地",
		"爱伦·坡故居地下有一处水修秘室，他的诗作其实是水系感应记录",
	},
	"denver": {
		"丹佛一英里高城的稀薄空气中金系灵气格外纯粹，初来的修士会有短暂开悟",
		"洛基山麓的金木双修灵脉绵延至市区，某户外装备店是修士们的补给站",
		"丹佛铸币局地下有一处封印——淘金热时代修士们封存的巨型金系灵晶",
		"红岩剧场的演出日，音乐在岩壁间共鸣，会短暂打开金木双修秘境入口",
	},
	"philadelphia": {
		"独立大厅地下有一处土系阵法，是建国先贤修士们为保护新生国家布下的",
		"费城东南部的老街区土系灵气深厚，本杰明·富兰克林其实是土系大修士",
		"费城艺术博物馆的台阶（洛基跑步的那段）蕴含着土系修炼的特殊共鸣频率",
		"宾夕法尼亚大学某地下室保存着殖民地时代修士们的修仙典籍原稿",
	},
	"san_antonio": {
		"阿拉莫要塞是一处土系灵气聚集地，守城的修士们至今魂魄未散",
		"圣安东尼奥河流水穿城，带着土水双系灵气，沿河散步的修士都在无声修炼",
		"德克萨斯-墨西哥边境土系灵气绵延，某些边境小镇是修士的隐居聚落",
		"圣安东尼奥的德州牛仔中有几位是土系修士，骑的不是普通牛，而是驯化土妖",
	},
	"san_diego": {
		"圣迭戈海湾是太平洋水系灵气进入大陆的第一个节点，水修圣地",
		"海军基地旁的海底有一处水系灵脉，潜水艇乘员偶尔感应到奇异震动",
		"拉霍亚海崖上的冲浪者中有数位水修，他们借助海浪修炼，境界远超常人",
		"圣迭戈动物园里有几只动物其实是通了灵智的水系妖兽，研究员困惑了多年",
	},
	"dallas": {
		"达拉斯牛仔队主场下方有一处封存的金系宝地，是德州石油大亨修士留下的",
		"迪利广场（肯尼迪遇刺地）聚集着大量未解散的金系死气，修士们避而远之",
		"德克萨斯仪器公司某研究院实际上在研究将金系灵晶集成入芯片的技术",
		"达拉斯金融区的交易员们下班后聚集在某地下酒吧，那是金修们的小圈子",
	},
}

// RollCityRealmRewards generates rewards for a city realm exploration with narrative seed
func RollCityRealmRewards(cr CityRealm) (map[string]int64, map[string]interface{}) {
	rewards := make(map[string]int64)
	// Base XP with some variance
	xpVariance := cr.BaseXP / 5
	rewards["cultivation_xp"] = cr.BaseXP + rand.Int63n(xpVariance*2+1) - xpVariance
	// Spirit stone
	stoneVariance := cr.BaseSpiritStone / 5
	rewards["spirit_stone"] = cr.BaseSpiritStone + rand.Int63n(stoneVariance*2+1) - stoneVariance
	// Materials per element
	for _, elem := range cr.Elements {
		qty := int64(cr.BaseMaterials) + rand.Int63n(int64(cr.BaseMaterials)+1)
		if qty > 0 {
			rewards["material_"+elem] = qty
		}
	}

	// Generate narrative seed
	encounterType := EncounterTypes[rand.Intn(len(EncounterTypes))]
	primaryElement := cr.Elements[0]
	hint := "在" + cr.Name + "探索中发现了神秘的灵气涌动"
	if hints, ok := CityNarrativeHints[cr.ID]; ok && len(hints) > 0 {
		hint = hints[rand.Intn(len(hints))]
	}

	// Build spirit materials list for narrative
	var matList []map[string]interface{}
	for k, qty := range rewards {
		if len(k) > 9 && k[:9] == "material_" {
			elem := k[9:]
			matList = append(matList, map[string]interface{}{
				"element": elem,
				"name":    ElementMaterialName(elem, cr.ID),
				"qty":     qty,
			})
		}
	}

	durationYears := cr.DurationSec / int(GameYearDuration.Seconds())
	if durationYears < 1 {
		durationYears = 1
	}

	seed := map[string]interface{}{
		"location":       cr.Name,
		"element":        ElementChinese(primaryElement),
		"duration_years": durationYears,
		"encounter_type": encounterType,
		"drops": map[string]interface{}{
			"cultivation_xp": rewards["cultivation_xp"],
			"spirit_stone":   rewards["spirit_stone"],
			"spirit_material": matList,
		},
		"narrative_hint": hint,
	}
	return rewards, seed
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

// RandBool returns a random true/false
func RandBool() bool {
	return rand.Intn(2) == 0
}

// BreakthroughMessage returns a localized, American-flavored breakthrough message
func BreakthroughMessage(newRealm string, newLevel int, xpRemaining int64, isMajor bool) string {
	displayName := RealmDisplayName(newRealm, newLevel)
	if !isMajor {
		return fmt.Sprintf("突破成功！进阶至【%s】！修为剩余：%d", displayName, xpRemaining)
	}
	// Major breakthrough messages with American local flavor
	messages := map[string]string{
		"foundation":            "在哈莱姆的霓虹灯下，根基凝固如曼哈顿基岩——【%s】，踏入筑基！修为剩余：%d",
		"core_formation":        "金丹在胸口成形，如拉斯维加斯赌场的幸运777——【%s】，结丹成功！修为剩余：%d",
		"nascent_soul":          "元婴从丹田凝聚，如密西西比河汇入大海——【%s】，元婴初成！修为剩余：%d",
		"deity_transformation":  "神魂蜕变，如黄石间歇泉冲天而起——【%s】，化神成就！修为剩余：%d",
		"void_refining":         "虚空与现实的边界模糊，大陆灵气尽在掌握——【%s】，炼虚大成！修为剩余：%d",
		"body_integration":      "身化天地，与美利坚大陆山河融为一体——【%s】，合体境成！修为剩余：%d",
		"mahayana":              "大乘之道，法则于胸中自行运转——【%s】，大乘登顶！修为剩余：%d",
		"tribulation":           "天雷滚滚，美利坚大陆为之颤抖——【%s】，渡劫开始！修为剩余：%d",
	}
	if tpl, ok := messages[newRealm]; ok {
		return fmt.Sprintf(tpl, displayName, xpRemaining)
	}
	return fmt.Sprintf("大突破成功！踏入【%s】！天地灵气涌动！修为剩余：%d", displayName, xpRemaining)
}
