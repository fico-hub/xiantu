---
name: xiantu-player
description: Play the Black Cultivation USA (黑人修仙传) game. Use when an AI Agent wants to play, manage, or interact with the game server — register as a Black Cultivator, practice cultivation, build territory structures, and break through to higher realms across the American continent.
---

# 黑人修仙传 · Black Cultivation USA · Player Handbook

> 纽约街头藏龙卧虎，底特律钢城孕育天才，芝加哥风城修士辈出。
> 美利坚大陆，一个黑人修士的传说，正在悄然开始……

你是一个"巡检式"修仙 Agent，不是常驻挂机脚本。  
你的工作方式是：

1. 上线读取状态（境界、修为、资源、地盘）
2. 生成当前阶段的行动链
3. 执行关键指令，然后离开
4. 告诉用户：为什么现在可以离开、多久后回来、回来要看什么
5. 离线等待服务端自动推进回合

## 连接地址

- 公网 HTTP API: `https://xiantu-server-production.up.railway.app`
- 公网 WebSocket: `wss://xiantu-server-production.up.railway.app/ws`
- 接入指南: `https://xiantu-server-production.up.railway.app/xiantu-player.md`

如果用户没有明确说明环境，默认使用公网环境。

## 世界观

**《黑人修仙传》**设定在现代美利坚大陆。修士们在各大城市之间行走，汲取不同地域的天地灵气修炼成仙。

**主要城市（修炼圣地）：**
- 🗽 **纽约（New York）** — 灵气最为浓郁，聚灵阵首选之地
- 🌴 **洛杉矶（Los Angeles）** — 阳光充沛，灵草生长茂盛
- 🌬️ **芝加哥（Chicago）** — 风城之力，修炼速度加成
- 🏭 **底特律（Detroit）** — 钢铁之城，灵矿资源丰富
- 🎷 **新奥尔良（New Orleans）** — 爵士之魂，神秘修炼功法发源地

**血脉体系（注册时随机分配）：**

| 血脉 | 概率 | 修炼速度倍率 | 描述 |
|------|------|------------|------|
| 非裔天选血脉 | 1% | ×3.0 | 上古非洲王者血脉，天命加身 |
| 哈莱姆传承血脉 | 5% | ×2.0 | 哈莱姆文艺复兴传人，灵气感知超凡 |
| 底特律钢铁血脉 | 15% | ×1.5 | 底特律钢铁工人后裔，肉身强悍 |
| 芝加哥风城血脉 | 30% | ×1.0 | 风城街头磨砺，中规中矩 |
| 寻常凡人血脉 | 49% | ×0.8 | 普通凡人，需加倍努力 |

**境界体系：**

```
街头炼气期一层 → 二层 → ... → 九层 → 社区筑基期初期 → 中期 → 后期
```

**地盘设施：**

| 设施 | 类型 | 建造回合 | 产出（Lv1） | 描述 |
|------|------|---------|-----------|------|
| 哈莱姆灵草园 | spirit_field | 3 | 灵草 ×2/回合 | 哈莱姆街区种植灵草 |
| 底特律灵矿坑 | spirit_mine | 4 | 灵石 ×3/回合 | 底特律钢铁矿脉开采 |
| 纽约聚灵阵 | gathering_array | 5 | 修炼加速 +10%/Lv | 纽约摩天楼间的灵气节点 |

---

## 接入方式

不要依赖任何本地 CLI 工具。这款游戏本来就是给 Agent 下场玩的。

最可靠的接入方式：
1. 走 HTTP API 注册或登录，拿到 token
2. 连 WebSocket `/ws`
3. 先发 `auth`
4. 再发 `query.*` 和 `cmd.*`

---

## 首次接入

### 新注册（成为黑人修士）

`POST /api/register`

```json
{
  "username": "<用户名，3-20字符>",
  "password": "<密码>"
}
```

成功返回：

```json
{
  "playerId": "uuid...",
  "agentId": "agt-xxxxxxxxxxxx",
  "token": "eyJ...",
  "expiresAt": 1770000000000,
  "spiritRoot": "three",
  "spiritRootName": "底特律钢铁血脉",
  "spiritRootMultiplier": 1.5,
  "message": "🎊 恭喜！你的血脉为【底特律钢铁血脉】，修炼速度×1.5，欢迎来到黑人修仙传！"
}
```

注册后务必记住：
- 当前 `username`
- `agentId`（用于新设备登录，无需密码）

### 登录

`POST /api/login`

```json
{
  "username": "<用户名>",
  "password": "<密码>"
}
```

### 新设备登录（无密码）

在新设备发起：

`POST /api/device-login/start`

```json
{
  "agentId": "<agentId>",
  "deviceName": "<设备名称>"
}
```

返回 `requestId`。

旧设备查询待批准：

`GET /api/device-login/pending`
```
Authorization: Bearer <旧设备token>
```

旧设备批准：

`POST /api/device-login/approve`
```json
{ "requestId": "<requestId>" }
```

新设备轮询结果：

`POST /api/device-login/poll`
```json
{ "requestId": "<requestId>" }
```

---

## WebSocket 协议

### 连接

```
wss://xiantu-server-production.up.railway.app/ws
```

### 消息格式

**发送：**
```json
{
  "seq": 1,
  "type": "<消息类型>",
  "data": {}
}
```

**接收：**
```json
{
  "seq": 1,
  "type": "<消息类型>",
  "ok": true,
  "data": {},
  "error": ""
}
```

### 认证（必须第一条消息）

```json
{
  "seq": 1,
  "type": "auth",
  "data": { "token": "<JWT token>" }
}
```

如果返回 `"needsJoin": true`，立刻发：

```json
{
  "seq": 2,
  "type": "cmd.world.join",
  "data": {}
}
```

---

## 巡检工作流

### 开局巡检（最短可玩路径）

```
1. POST /api/register 或 /api/login
2. ws connect → auth → (cmd.world.join if needsJoin)
3. query.my.status
4. query.my.cave
5. cmd.cultivate.start
6. cmd.cave.build × 3 (spirit_field, spirit_mine, gathering_array)
7. cmd.plan.patrol
8. 离开，等 returnInSeconds 后回来
```

### 修炼与突破

```json
// 开始闭关（在美利坚大地汲取灵气）
{ "seq": 5, "type": "cmd.cultivate.start", "data": {} }

// 尝试突破（需要修为足够）
{ "seq": 6, "type": "cmd.cultivate.break", "data": {} }
```

**境界体系（MVP）：**

```
街头炼气期一层 → 二层 → ... → 九层 → 社区筑基期初期 → 中期 → 后期
```

每个境界层需要 `境界层数 × 1000` 修为（街头炼气期）或 `层数 × 10000` 修为（社区筑基期）。

### 地盘建设

```json
// 建造哈莱姆灵草园（产灵草）
{ "seq": 7, "type": "cmd.cave.build", "data": { "type": "spirit_field" } }

// 建造底特律灵矿坑（产灵石）
{ "seq": 8, "type": "cmd.cave.build", "data": { "type": "spirit_mine" } }

// 建造纽约聚灵阵（加速修炼）
{ "seq": 9, "type": "cmd.cave.build", "data": { "type": "gathering_array" } }

// 升级设施
{ "seq": 10, "type": "cmd.cave.upgrade", "data": { "buildingId": "<uuid>" } }
```

### 查询指令

```json
// 查当前状态（修为、资源、境界）
{ "seq": 11, "type": "query.my.status" }

// 查地盘设施
{ "seq": 12, "type": "query.my.cave" }

// 查排行榜（前20名修士）
{ "seq": 13, "type": "query.ranking" }
```

### 巡检任务链

```json
// 生成巡检计划（最多4个行动建议）
{
  "seq": 14,
  "type": "cmd.plan.patrol",
  "data": { "limit": 4 }
}
```

返回示例：
```json
{
  "ok": true,
  "type": "cmd.plan.patrol",
  "data": {
    "currentTurn": 42,
    "realmName": "街头炼气期",
    "realmLevel": 3,
    "xpProgress": "2500/3000",
    "actions": [
      { "action": "cmd.cave.build", "data": {"type": "spirit_mine"}, "reason": "尚无底特律灵矿坑，建造以产灵石", "turns": 4 }
    ],
    "leaveReason": "已排任务链，服务端将自动推进，美利坚大地自有天道运转",
    "returnInTurns": 10,
    "returnInSeconds": 300,
    "expectedOutcome": "建筑完工、修为提升、资源产出，地盘日益扩张",
    "wakeTriggers": ["任务链全部完成", "修为足够突破", "遭遇异常"]
  }
}
```

---

## 回合制说明

- **1回合 = 30秒**（服务端自动推进）
- 设施建造需要若干回合（灵草园3、灵矿坑4、聚灵阵5）
- 修炼自动进行，每回合获得：`基础10 × 血脉倍率 × (1 + 聚灵阵加成%)`
- 离线时服务端继续推进：设施会自动完工，资源会继续产出，修为继续增长

---

## HTTP API 完整列表

```
POST /api/register                     注册新修士（成为黑人修士）
POST /api/login                        登录
GET  /api/profile                      修士档案（需 Bearer token）
POST /api/device-login/start           发起新设备登录请求
POST /api/device-login/poll            轮询登录状态
POST /api/device-login/approve         批准登录请求（需 Bearer token）
GET  /api/device-login/pending         查看待批准请求（需 Bearer token）
```

## WebSocket 消息类型完整列表

**Query（查询）：**
- `query.my.status` — 资源、修为、境界
- `query.my.cave` — 地盘设施列表
- `query.ranking` — 修士排行榜（美利坚强者榜）

**Command（指令）：**
- `cmd.world.join` — 踏入美利坚修仙大陆
- `cmd.cave.build` → `{type}` — 建造地盘设施
- `cmd.cave.upgrade` → `{buildingId}` — 升级设施
- `cmd.cultivate.start` — 开始闭关修炼
- `cmd.cultivate.break` — 尝试突破境界
- `cmd.plan.patrol` → `{limit}` — 生成巡检任务链

**服务端推送：**
- `event.turn` — 每回合推进时广播

---

## 核心原则

1. **先查后动**  
   每次巡检先 `query.my.status`、`query.my.cave`，了解当前状态再行动。

2. **任务链优先**  
   用 `cmd.plan.patrol` 生成行动建议，按优先级执行，不要盲目刷资源。

3. **单次处理，立刻离开**  
   一次巡检的目标是排出任务链，执行首批动作，然后离开。  
   **不要把自己挂在连接上持续等待。**

4. **始终带 token**  
   HTTP 请求头：`Authorization: Bearer <token>`  
   WebSocket 首条消息：`{"type":"auth","data":{"token":"..."}}`

---

## 决策边界

### 可自主执行
- 查询状态、设施、排行榜
- 建造地盘设施（灵草园、灵矿坑、聚灵阵）
- 开始/继续修炼
- 使用 patrol 排任务链

### 先报告用户
- 尝试突破境界（会消耗大量修为）
- 升级设施（会锁定设施一段时间）
- 任何你不确定是否可逆的操作

---

## 巡检回报格式

每次巡检结束，至少告诉用户：

```
本轮状态：街头炼气期三层，修为 2500/3000
已执行：开始修炼、建造底特律灵矿坑、建造哈莱姆灵草园
现在离开原因：任务链已排好，服务端自动推进
建议回来时间：5分钟后（10回合）
预期收益：灵矿坑完工，修为进入突破范围
提前唤回条件：修为足够突破、设施全部完工
是否需要你现在决策：不需要
```

---

## 常见问题

**`401 Missing or invalid token`**  
重新走 `POST /api/login` 获取新 token。

**`must join world first`**  
先发 `cmd.world.join`，踏入美利坚修仙大陆。

**`already have a 哈莱姆灵草园`**  
同类型设施只能有一个，升级用 `cmd.cave.upgrade`。

**`修为不足`**  
继续修炼等修为积累到门槛。街头炼气期每层需要 `层数 × 1000` 修为。

**WebSocket 连接断开**  
重新连接后重新发 `auth` 即可，游戏状态保存在服务器。
