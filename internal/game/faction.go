package game

import (
	"math/rand"
)

// ========== 门派体系 ==========

// Alliance constants
const (
	AlliancePeopleNation = "people_nation"
	AllianceFolkNation   = "folk_nation"
	AllianceChinese      = "chinese_tong"
	AllianceNone         = "none"
)

// FactionRank 门派内地位
type FactionRank struct {
	ID          string
	Name        string
	MinContrib  int64 // 所需贡献值
	Description string
}

var FactionRanks = []FactionRank{
	{ID: "recruit", Name: "小弟", MinContrib: 0, Description: "门派最低阶"},
	{ID: "core", Name: "骨干", MinContrib: 500, Description: "受到门派重用"},
	{ID: "elder", Name: "核心", MinContrib: 2000, Description: "门派核心成员"},
}

func GetRankByContrib(contrib int64) FactionRank {
	rank := FactionRanks[0]
	for _, r := range FactionRanks {
		if contrib >= r.MinContrib {
			rank = r
		}
	}
	return rank
}

// FactionBonus describes a gameplay bonus provided by a faction
type FactionBonus struct {
	Key   string // e.g. "attack_on_hit_pct", "water_skill_pct"
	Value int    // percentage or flat value
	Desc  string
}

// Faction represents a playable gang/sect
type Faction struct {
	ID          string
	Name        string   // Chinese name
	NameEn      string   // English name
	Race        string   // primary race: african/latino/caucasian/chinese/asian_pacific/any
	Territory   string   // location description
	Alliance    string   // people_nation / folk_nation / chinese_tong / none
	Bonuses     []FactionBonus
	Passives    []string // passive ability descriptions
	SpecialRule string   // special join/leave rule
	IsVillain   bool     // 玄牢宗 = game villain faction
}

var FactionOrder = []string{
	"bloods", "crips", "gangster_disciples", "vice_lords",
	"ms13", "latin_kings", "hells_angels", "aryan_brotherhood",
	"hip_sing_tong", "wah_ching", "oriental_boyz",
}

var Factions = map[string]Faction{
	"bloods": {
		ID: "bloods", Name: "赤血宗", NameEn: "Bloods",
		Race: "african", Territory: "西都洛杉矶", Alliance: AlliancePeopleNation,
		Bonuses: []FactionBonus{
			{Key: "attack_on_hit_pct", Value: 15, Desc: "受击后攻击力+15%"},
			{Key: "battle_shield_pct", Value: 10, Desc: "每场战斗开始获得10%血量护盾"},
		},
		Passives: []string{"血煞：越战越强，连胜每场额外+2%攻击"},
	},
	"crips": {
		ID: "crips", Name: "蓝水宗", NameEn: "Crips",
		Race: "african", Territory: "西都洛杉矶南区", Alliance: AllianceFolkNation,
		Bonuses: []FactionBonus{
			{Key: "water_skill_pct", Value: 20, Desc: "水系技能伤害+20%"},
			{Key: "night_rain_speed_pct", Value: 25, Desc: "夜间/雨天地形移速+25%"},
		},
		Passives: []string{"地盘标记：在本门派地盘内攻击额外+10%", "复活CD减少50%"},
	},
	"gangster_disciples": {
		ID: "gangster_disciples", Name: "星叉宗", NameEn: "Gangster Disciples",
		Race: "african", Territory: "北都芝加哥", Alliance: AllianceFolkNation,
		Bonuses: []FactionBonus{
			{Key: "double_strike_pct", Value: 20, Desc: "突破境界后攻击有20%概率双段连击"},
		},
		Passives: []string{"Folk传承：同盟门派（蓝水宗）地盘可自由通行"},
	},
	"vice_lords": {
		ID: "vice_lords", Name: "炎华宗", NameEn: "Vice Lords",
		Race: "african", Territory: "北都芝加哥西区", Alliance: AlliancePeopleNation,
		Bonuses: []FactionBonus{
			{Key: "fire_crit_pct", Value: 15, Desc: "火系技能暴击率+15%"},
			{Key: "chicago_merchant_discount_pct", Value: 10, Desc: "芝加哥商人折扣-10%"},
		},
		Passives: []string{"贵族气场：与NPC对话可解锁隐藏选项"},
	},
	"ms13": {
		ID: "ms13", Name: "十三蛇宗", NameEn: "MS-13",
		Race: "latino", Territory: "西都洛杉矶东区", Alliance: AllianceNone,
		Bonuses: []FactionBonus{
			{Key: "poison_stack_max", Value: 5, Desc: "攻击附带毒伤（可叠5层）"},
			{Key: "fear_start_pct", Value: 30, Desc: "战斗开始30%概率触发恐惧效果"},
		},
		Passives:    []string{"跨洲传送阵解锁：可使用隐秘传送网络"},
		SpecialRule: "入门需完成血祭任务；加入后不可主动退出",
	},
	"latin_kings": {
		ID: "latin_kings", Name: "王者黄金宫", NameEn: "Latin Kings",
		Race: "any", Territory: "北都芝加哥+东都纽约", Alliance: AlliancePeopleNation,
		Bonuses: []FactionBonus{
			{Key: "all_stats_flat", Value: 5, Desc: "所有属性基础值+5"},
			{Key: "first_win_xp_pct", Value: 20, Desc: "每日首胜修为+20%"},
		},
		Passives: []string{"无种族限制：任何族裔均可加入"},
	},
	"hells_angels": {
		ID: "hells_angels", Name: "铁翼堂", NameEn: "Hell's Angels",
		Race: "caucasian", Territory: "西都洛杉矶郊野", Alliance: AllianceNone,
		Bonuses: []FactionBonus{
			{Key: "move_speed_pct", Value: 25, Desc: "移动速度+25%"},
			{Key: "near_death_attack_mult", Value: 200, Desc: "血量<1%时攻击力翻倍持续10秒"},
		},
		Passives: []string{"不固定地盘：全大陆道路快速移动，全境旅行加速"},
	},
	"aryan_brotherhood": {
		ID: "aryan_brotherhood", Name: "玄牢宗", NameEn: "Aryan Brotherhood",
		Race: "caucasian", Territory: "地下监牢系统", Alliance: AllianceNone,
		Bonuses: []FactionBonus{
			{Key: "underground_stats_pct", Value: 10, Desc: "封闭/地下场景全属性+10%"},
			{Key: "prison_exit_unlock", Value: 1, Desc: "掌握所有监狱秘密出口"},
		},
		Passives:    []string{"种族壁垒：白裔全员加入，其他族裔只能走合作路线"},
		SpecialRule: "游戏反派门派；非白裔玩家只能以中立方式互动",
		IsVillain:   true,
	},
	"hip_sing_tong": {
		ID: "hip_sing_tong", Name: "协胜公", NameEn: "Hip Sing Tong",
		Race: "chinese", Territory: "东都纽约唐人街", Alliance: AllianceChinese,
		Bonuses: []FactionBonus{
			{Key: "chinatown_merchant_discount_pct", Value: 30, Desc: "唐人街商人折扣-30%"},
			{Key: "hidden_quest_unlock", Value: 1, Desc: "隐藏任务解锁，黑市系统"},
		},
		Passives: []string{"堂门通行：在唐人街地图享有特殊通行权", "龙脉感知：地图宝藏自动标出"},
	},
	"wah_ching": {
		ID: "wah_ching", Name: "华青宗", NameEn: "Wah Ching",
		Race: "chinese", Territory: "西都旧金山唐人街", Alliance: AllianceChinese,
		Bonuses: []FactionBonus{
			{Key: "market_profit_pct", Value: 30, Desc: "交易所收益+30%"},
			{Key: "jackpot_crit_pct", Value: 15, Desc: "大彩头暴击触发率+15%"},
		},
		Passives: []string{"三合会引荐：可召唤青龙护法参战"},
	},
	"oriental_boyz": {
		ID: "oriental_boyz", Name: "越龙帮", NameEn: "Oriental Boyz/TRG",
		Race: "asian_pacific", Territory: "西都洛杉矶小西贡", Alliance: AllianceNone,
		Bonuses: []FactionBonus{
			{Key: "night_all_stats_pct", Value: 15, Desc: "夜间全属性+15%"},
			{Key: "ambush_pct", Value: 40, Desc: "偷袭伤害+40%"},
			{Key: "team_damage_per_member", Value: 5, Desc: "队友每多1人，伤害+5%"},
		},
		Passives: []string{"流亡者直觉：被追击时25%概率完全脱身"},
	},
}

// ========== 门派关系 ==========

// RelationType defines the relationship between two factions
type RelationType string

const (
	RelationFriendly = RelationType("friendly")
	RelationNeutral  = RelationType("neutral")
	RelationHostile  = RelationType("hostile")
	RelationMortal   = RelationType("mortal_enemy") // 死敌
)

// GetFactionRelation returns the relationship between two factions
func GetFactionRelation(a, b string) RelationType {
	if a == b {
		return RelationFriendly
	}
	// Look up both orderings
	key := factionRelationKey(a, b)
	if rel, ok := factionRelations[key]; ok {
		return rel
	}
	// Check alliance-based relationships
	fa, aok := Factions[a]
	fb, bok := Factions[b]
	if !aok || !bok {
		return RelationNeutral
	}
	// Same alliance = friendly
	if fa.Alliance != AllianceNone && fa.Alliance == fb.Alliance {
		return RelationFriendly
	}
	// People vs Folk = hostile
	if (fa.Alliance == AlliancePeopleNation && fb.Alliance == AllianceFolkNation) ||
		(fa.Alliance == AllianceFolkNation && fb.Alliance == AlliancePeopleNation) {
		return RelationHostile
	}
	return RelationNeutral
}

func factionRelationKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// Explicit faction relations (death enemies, special alliances)
var factionRelations = map[string]RelationType{
	// 死敌
	factionRelationKey("bloods", "crips"):              RelationMortal,
	factionRelationKey("ms13", "latin_kings"):          RelationMortal,
	// 玄牢宗天敌
	factionRelationKey("aryan_brotherhood", "bloods"):  RelationMortal,
	factionRelationKey("aryan_brotherhood", "crips"):   RelationMortal,
	// 华人同盟（隐秘友好）
	factionRelationKey("hip_sing_tong", "wah_ching"):   RelationFriendly,
}

// CanJoinFaction checks if a player of a given race can join the faction
// Returns (allowed bool, requiresSpecialTask bool, reason string)
func CanJoinFaction(factionID, race string, hasBloodOath bool) (bool, bool, string) {
	f, ok := Factions[factionID]
	if !ok {
		return false, false, "门派不存在"
	}

	// Aryan Brotherhood: white only (or cooperation route for non-white)
	if factionID == "aryan_brotherhood" {
		if race != "caucasian" {
			return false, false, "玄牢宗种族壁垒：仅限白裔全员加入，其他族裔只能走合作路线"
		}
	}

	// MS-13: requires blood oath quest
	if factionID == "ms13" {
		if !hasBloodOath {
			return false, true, "加入十三蛇宗需先完成血祭任务"
		}
	}

	// Most factions prefer same race but allow
	if f.Race != "any" && f.Race != race {
		// Still allowed, just noted
		return true, false, "非本门派传统族裔，将受到一定程度的异样眼光"
	}

	return true, false, ""
}

// CanLeaveFaction checks if a player can leave the faction
func CanLeaveFaction(factionID string) (bool, string) {
	if factionID == "ms13" {
		return false, "十三蛇宗：入门不可退出，背叛者将被追杀"
	}
	return true, ""
}

// ========== 门派任务系统 ==========

// FactionTaskType describes what kind of task it is
type FactionTaskType string

const (
	TaskTypePatrol   FactionTaskType = "patrol"   // 前往地点驻守
	TaskTypeCollect  FactionTaskType = "collect"  // 收集天材地宝
	TaskTypeEliminate FactionTaskType = "eliminate" // 击败NPC
	TaskTypeTribute  FactionTaskType = "tribute"  // 缴纳灵石
	TaskTypeEscort   FactionTaskType = "escort"   // 护送商队
)

// FactionTaskReward is the reward for completing a faction task
type FactionTaskReward struct {
	CultivationXP    int64
	SpiritStone      int64
	MaterialElement  string
	MaterialQty      int64
	FactionContrib   int64
	RankUp           bool   // whether this might trigger a rank-up
}

// FactionTask is a generated task instance
type FactionTask struct {
	ID          string
	FactionID   string
	Type        FactionTaskType
	Title       string
	Description string
	// Task parameters (interpretation depends on type)
	TargetLocation  string // patrol/escort
	TargetCount     int    // collect/eliminate/tribute amount
	TargetElement   string // collect: which material
	TargetRealm     string // eliminate: min realm of NPC
	EscortFrom      string // escort: from
	EscortTo        string // escort: to
	DurationYears   int    // patrol/escort: game years to complete
	// Rewards
	Reward FactionTaskReward
}

// ── Patrol locations keyed by territory theme ──
var patrolLocations = map[string][]string{
	"los_angeles": {
		"洛杉矶·好莱坞星光大道", "圣莫尼卡海滩", "比佛利山庄", "东洛杉矶·博伊尔高地",
		"康普顿·旧街区", "英格尔伍德·大道", "小西贡·越南城", "唐人街·春城广场",
	},
	"chicago": {
		"芝加哥·南区63街", "奥格登维尔·旧社区", "赫布隆区·主街", "芝加哥西区·麦迪逊街",
		"英格尔公园·古道", "橡园区·市场", "迦南区·仓库区", "密歇根大道·风城核心",
	},
	"new_york": {
		"纽约唐人街·坚尼街", "布鲁克林·日落公园", "皇后区·法拉盛", "哈勒姆·125街",
		"布朗克斯·大十字路", "曼哈顿·下东区", "杰克逊高地·多元区", "史坦顿岛·渡口",
	},
	"san_francisco": {
		"旧金山·唐人街·格兰特大道", "金门公园·东门", "湾区·奥克兰港口", "夕阳区·日落大道",
	},
	"underground": {
		"地下1号牢房走廊", "旧金山联邦拘留所", "洛杉矶郡监狱·B区", "芝加哥库克郡监狱",
		"阿提卡监狱·越狱暗道", "地下传送节点·A", "监狱食堂·暗室",
	},
	"open_road": {
		"66号公路·自由之路", "I-10公路·日落之路", "蒙大拿公路·荒野长道", "加州1号公路·海岸线",
		"内华达·沙漠孤道", "德克萨斯·星光高速", "芝加哥·工业外环道",
	},
}

// GetPatrolLocations returns patrol locations for a given faction
func GetPatrolLocations(factionID string) []string {
	switch factionID {
	case "bloods":
		return patrolLocations["los_angeles"]
	case "crips":
		return append(patrolLocations["los_angeles"], "蓝水宗·南区堂口")
	case "gangster_disciples", "vice_lords":
		return patrolLocations["chicago"]
	case "ms13":
		locs := append(patrolLocations["los_angeles"], "十三蛇宗·秘密聚点")
		return locs
	case "latin_kings":
		return append(patrolLocations["chicago"], patrolLocations["new_york"]...)
	case "hells_angels":
		return patrolLocations["open_road"]
	case "aryan_brotherhood":
		return patrolLocations["underground"]
	case "hip_sing_tong":
		return patrolLocations["new_york"]
	case "wah_ching":
		return patrolLocations["san_francisco"]
	case "oriental_boyz":
		return patrolLocations["los_angeles"]
	}
	return patrolLocations["los_angeles"]
}

// GenerateFactionTask generates a random task for a faction member
func GenerateFactionTask(factionID string, playerRealm string, contrib int64) *FactionTask {
	rank := GetRankByContrib(contrib)

	realmOrder := 0
	if t, ok := RealmTiers[playerRealm]; ok {
		realmOrder = t.Order
	}

	// Scale rewards by realm order
	baseXP := int64(50000) * int64(realmOrder+1)
	baseStone := int64(200) * int64(realmOrder+1)
	baseContrib := int64(50) * int64(realmOrder+1)
	baseMat := int64(5) * int64(realmOrder+1)

	taskTypes := []FactionTaskType{TaskTypePatrol, TaskTypeCollect, TaskTypeEliminate, TaskTypeTribute, TaskTypeEscort}
	taskType := taskTypes[rand.Intn(len(taskTypes))]

	elements := Elements
	elem := elements[rand.Intn(len(elements))]
	patrolLocs := GetPatrolLocations(factionID)
	loc := patrolLocs[rand.Intn(len(patrolLocs))]

	task := &FactionTask{
		FactionID: factionID,
		Type:      taskType,
		Reward: FactionTaskReward{
			CultivationXP:  baseXP,
			SpiritStone:    baseStone,
			FactionContrib: baseContrib,
		},
	}

	switch taskType {
	case TaskTypePatrol:
		years := rand.Intn(3) + 1
		task.Title = "巡逻" + loc
		task.Description = "前往【" + loc + "】，驻守" + itoa(years) + "游戏年，维护门派地盘安全"
		task.TargetLocation = loc
		task.DurationYears = years
		task.Reward.FactionContrib = baseContrib * 2

	case TaskTypeCollect:
		qty := int(baseMat) + rand.Intn(int(baseMat))
		task.Title = "收集" + ElementChinese(elem) + "系天材"
		task.Description = "为门派收集" + itoa(qty) + "份" + ElementChinese(elem) + "系天材地宝"
		task.TargetCount = qty
		task.TargetElement = elem
		task.Reward.MaterialElement = elem
		task.Reward.MaterialQty = int64(qty / 3)

	case TaskTypeEliminate:
		count := rand.Intn(5) + 3
		minRealm := "qi_refining"
		if realmOrder > 0 {
			minRealm = RealmOrder[realmOrder-1]
		}
		task.Title = "清剿敌对势力"
		task.Description = "击败" + itoa(count) + "名" + RealmDisplayName(minRealm, 1) + "以上的敌对NPC"
		task.TargetCount = count
		task.TargetRealm = minRealm
		task.Reward.CultivationXP = baseXP * 2

	case TaskTypeTribute:
		amount := int(baseStone/2) + rand.Intn(int(baseStone/2))
		task.Title = "缴纳门派贡赋"
		task.Description = "向门派缴纳" + itoa(amount) + "灵石作为定期贡赋"
		task.TargetCount = amount
		task.Reward.FactionContrib = baseContrib * 3
		task.Reward.SpiritStone = -int64(amount) // negative = cost

	case TaskTypeEscort:
		locs := GetPatrolLocations(factionID)
		from := locs[rand.Intn(len(locs))]
		to := locs[rand.Intn(len(locs))]
		if from == to && len(locs) > 1 {
			to = locs[(rand.Intn(len(locs)-1)+1)%len(locs)]
		}
		years := rand.Intn(2) + 1
		task.Title = "护送门派商队"
		task.Description = "护送商队从【" + from + "】安全抵达【" + to + "】，预计" + itoa(years) + "游戏年"
		task.EscortFrom = from
		task.EscortTo = to
		task.DurationYears = years
		task.Reward.SpiritStone = baseStone * 2
	}

	// Rank-up chance for senior members
	if rank.ID == "core" && rand.Intn(10) < 2 {
		task.Reward.RankUp = true
	}

	_ = rank // used for scaling logic above
	return task
}

// itoa converts int to string (small helper to avoid fmt import)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n >= 10 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	pos--
	buf[pos] = byte('0' + n)
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// ShouldAssignFactionTask returns true 30% of the time (called on player check-in)
func ShouldAssignFactionTask() bool {
	return rand.Intn(100) < 30
}
