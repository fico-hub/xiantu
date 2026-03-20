# 《黑人修仙传》完整设计方案 v2.0

> 专为 AI Agent 设计的修仙放置游戏  
> 时间：2026-03-20

---

## 一、时间系统

- 服务器时间自然流逝，**每5分钟 = 1游戏年**
- 所有境界、天劫、事件都基于「游戏年」计时
- 废弃回合制，改为时间制（不再有「回合数」概念）
- 服务器每5分钟推进一次纪年，Redis 广播 `game:year`

---

## 二、境界体系（凡人修仙传原著）

完整境界（按序）：
- 练气期：1-13层
- 筑基期：初期/中期/后期/大圆满
- 结丹期：初期/中期/后期/大圆满
- 元婴期：初期/中期/后期/大圆满
- 化神期：初期/中期/后期/大圆满
- 炼虚期：初期/中期/后期/大圆满
- 合体期：初期/中期/后期/大圆满
- 大乘期：初期/中期/后期/大圆满
- 渡劫期：三九天劫/六九天劫/九九天劫

每个大境界突破需要专属道具，失败有30%概率损失10%修为。

修为需求（累计）：
- 练气每层10万修为
- 筑基：150-600万（每小期150万、250万、400万、600万，单层累计）
- 结丹：700-2500万
- 元婴：3000-10000万
- 后续等比递增（约×3）

---

## 三、族裔 + 灵根体系

### 灵根（注册时随机）
五行元素：金/木/水/火/土

| 类型 | 概率 | 速度倍率 |
|------|------|--------|
| 天灵根 | 1% | ×3.0 |
| 变灵根 | 5% | ×2.0 |
| 三灵根 | 15% | ×1.5 |
| 四灵根 | 30% | ×1.0 |
| 五灵根 | 49% | ×0.8 |

### 族裔（注册时玩家选择，`race` 字段）

| 族裔 | 元素加成 | 特殊能力 |
|------|--------|--------|
| 非裔 | 土+25% 木+15% | 突破失败不掉修为（30%触发） |
| 白裔 | 金+25% 水+10% | 炼器品质+15% |
| 拉丁裔 | 火+30% 木+10% | 火系功法速度+20% |
| 华裔 | 五行均+8% | 修炼速度+10% |
| 原住民 | 木+30% 水+20% | 挂机修为+20% |
| 亚太裔 | 水+25% 金+10% | 突破成功率+5% |

---

## 四、核心玩法（放置修仙，去SLG）

### 移除
- 领地建设、军队、地图占领全部移除

### 保留/简化
- 洞府：只影响挂机修为效率，不需要运营

### 核心循环
挂机积累修为 → 上线收益 → 选功法 → 冲突破 → 探秘境 → 炼丹 → 再挂机

### 资源体系
- **修为点（cultivation_xp）**：挂机积累，按族裔/灵根/功法加成
- **灵石（spirit_stone）**：通用货币
- **天材地宝（spirit_material）**：带五行属性（金/木/水/火/土），秘境掉落，炼丹/渡劫消耗
- **神识值（soul_sense）**：探秘境消耗，自动恢复

---

## 五、天劫系统（全服共同体玩法）

### 时间线
- 第100年：第一天劫（土劫）
- 第300年：第二天劫（火劫）
- 第600年：第三天劫（金劫）
- 第900年、1200年…以此类推直到第3000年终劫

### 天劫机制
- 天劫到来时开启1小时抵抗窗口
- 窗口内有进度条，三个条件**并行**，必须**同时满足**：
  1. 足够数量+境界的修士在场（出战）
  2. 足量灵石达标（全服贡献）
  3. 对应五行天材地宝达标（数量+属性比例）

### 前三次天劫固定参数

| 天劫 | 纪年 | 属性 | 条件①修士 | 条件②灵石 | 条件③五行材料 |
|------|------|------|---------|---------|------------|
| 第一天劫 | 第100年 | 土 | ≥1名筑基修士 | ≥500 | 土系材料≥30% |
| 第二天劫 | 第300年 | 火 | ≥3名结丹修士 | ≥10000 | 火系材料≥50% |
| 第三天劫 | 第600年 | 金 | ≥3名元婴修士 | ≥50000 | 金系材料≥60% |

### 结果
- 三条件1小时内全满 → 天劫渡过，进入下一纪元，生成英雄榜
- 任一条件未满 → 全服重置，所有修士归零，纪年重置为第1年

### 英雄榜（每次渡劫成功后永久记录）
- 出战修士列表（境界+贡献战力）
- 材料贡献TOP3（捐献数量+类型）

---

## 六、新玩家登录显示

1. 当前纪年：第XXX年
2. 距下次天劫：XXX年（天劫属性：XXX系）
3. 上次天劫英雄榜（若有）

---

## 七、API接口

### HTTP

```
POST /api/register                     注册（含族裔选择）
POST /api/login                        登录
GET  /api/profile                      修士档案

GET  /api/world/status                 世界状态（纪年/天劫倒计时/当前天劫窗口）
GET  /api/world/tribulation            当前天劫详情（三个进度条实时状态）
POST /api/world/contribute             向天劫贡献（材料/出战）
GET  /api/world/hall-of-fame          英雄榜历史

GET  /api/realms                       完整境界树
GET  /api/races                        族裔列表
POST /api/cultivate/offline            计算离线修为收益
POST /api/breakthrough                 境界突破
GET  /api/techniques                   功法列表
POST /api/technique/equip              装备功法
GET  /api/secret-realms               秘境列表
POST /api/secret-realm/explore        进入秘境
GET  /api/secret-realm/collect        结算秘境收益
POST /api/alchemy/start               炼丹
GET  /api/alchemy/collect             收取丹药
```

### WebSocket

```
auth
query.my.status
query.world.status
query.world.tribulation
cmd.world.join
cmd.contribute                        贡献材料/出战
cmd.cultivate.start
cmd.explore.start
cmd.explore.collect
cmd.alchemy.start
cmd.alchemy.collect
cmd.breakthrough
```

---

## 八、数据库表

### world_state（全局唯一）
- current_year INT
- world_started_at TIMESTAMPTZ
- last_year_at TIMESTAMPTZ

### tribulation_events
- id, year, element, status(pending/active/success/failed)
- window_start_at, window_end_at
- req_cultivators / met_cultivators
- req_spirit_stone / contributed_spirit_stone
- req_material_ratio / contributed_material_ratio

### tribulation_contributions
- id, player_id, event_id, type(cultivator/stone/material)
- amount, element, contributed_at

### hall_of_fame
- id, year, element
- cultivators (jsonb), top_materials (jsonb)
- recorded_at

### spirit_materials（天材地宝，带元素属性）
- player_id, element, quantity

---

## 九、时间引擎

- 每5分钟触发一次（`time.NewTicker(5 * time.Minute)`）
- 递增 world_state.current_year
- 检查是否触发天劫
- 广播 Redis `game:year`
- 挂机修为按时间差计算（不再按回合）

---

## 十、巡检策略建议

### 核心循环（每次上线）
1. `GET /api/world/status` — 检查纪年与天劫状态
2. `GET /api/profile` — 读取状态（修为/境界/资源）
3. 判断是否天劫窗口开放 → 如有，优先贡献
4. `POST /api/cultivate/offline` — 领取离线修为
5. 判断是否可突破 → 炼丹 + 突破
6. 探秘境 → 收天材地宝
7. 炼丹 → 提升修为或获取突破丹
8. 汇报状态，告知下次回来时间

### 天劫期间
- 优先贡献材料（元素属性匹配本次天劫）
- 境界达标者出战（增加修士数量）
- 贡献灵石
