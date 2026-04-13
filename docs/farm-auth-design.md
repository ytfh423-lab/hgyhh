# 农场防脚本鉴权系统设计

> 已落地到 Go/Gin 项目。核心代码见 `middleware/farm_nonce.go`，前端见 `web/src/helpers/api.js`。

## 实际实现方案（Go + 内存存储）

本项目使用 **cookie-based session**（gin-contrib/sessions），因此防刷系统采用以下架构：

| 组件 | 存储位置 | 说明 |
|------|---------|------|
| 根密钥 K | 内存（从 SESSION_SECRET 派生） | SHA256("farm_nonce_guard_v1:" + SessionSecret) |
| 用户计数器 | 内存 map[userId]counter | 每用户 1 条，~2MB / 10 万用户 |
| Nonce 去重 | 内存 map[nonce]expireTime | 仅存 60 秒，30 秒清理一次 |
| Farm Token | HttpOnly Cookie `_ft` | 格式: HMAC前16字符.counter |

### 中间件执行链

```
UserAuth → FarmSessionOnly → FarmActionRateLimit → FarmDailyActionCap → FarmNonceGuard → CheckFarmBetaAccess → Handler
```

### 防线说明

| 层 | 中间件 | 防什么 |
|----|--------|--------|
| 1 | FarmSessionOnly | 禁止 access token 直调 |
| 2 | FarmActionRateLimit | 10 次 POST / 60 秒 / 用户 |
| 3 | FarmDailyActionCap | 500 次 POST / 天 / 用户 |
| 4 | FarmNonceGuard | Nonce 防重放 + 计数器防盗 cookie + HMAC 防伪造 |

---

## 参考设计文档（Node.js + MySQL 版本）

以下为原始设计文档，供参考。实际项目已用 Go + 内存方案替代数据库存储。

## 一、数据库表结构

### 1. sys_config — 根密钥存储

```sql
CREATE TABLE sys_config (
  `key`   VARCHAR(64)  PRIMARY KEY COMMENT '配置项名称',
  `value` TEXT         NOT NULL    COMMENT 'AES加密后的值(Base64)',
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- 插入根密钥(256位AES密钥, 用另一个ENV密钥加密存储)
-- INSERT INTO sys_config(`key`,`value`) VALUES('ROOT_KEY_K','<AES加密后的Base64>');
```

- 仅存1条根密钥记录，< 200 字节

### 2. user_session — 用户会话(每人仅1条)

```sql
CREATE TABLE user_session (
  user_id      INT          PRIMARY KEY COMMENT '用户ID, 一人一条',
  session_id   CHAR(32)     NOT NULL    COMMENT '加密会话ID(Hex), 存Cookie',
  sk           CHAR(64)     NOT NULL    COMMENT '当前签名密钥(HMAC派生, Hex)',
  counter      BIGINT       NOT NULL DEFAULT 0 COMMENT '隐形计数器, 每次请求+1',
  expire_at    DATETIME     NOT NULL    COMMENT '7天后过期',
  KEY idx_expire (expire_at)            COMMENT '用于定时清理过期会话'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- 单条 ≈ 4+32+64+8+8 = **116字节**
- 10万用户 ≈ 11MB，永不膨胀

### 3. request_nonce — Nonce防重放(仅存60秒)

```sql
CREATE TABLE request_nonce (
  nonce      CHAR(32)  PRIMARY KEY COMMENT '一次性随机数(Hex)',
  expire_at  DATETIME  NOT NULL    COMMENT '写入时间+60秒, 用于强制清理',
  KEY idx_expire (expire_at)        COMMENT '定时DELETE清理依据'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- 单条 = 32+8 = **40字节**
- 峰值: 60秒窗口内最大并发量。100万请求/天 ≈ 12次/秒 × 60秒 = **720条** ≈ 28KB

---

## 二、完整业务流程

### 阶段1: 用户登录

```
客户端: POST /login {username, password}
服务端:
  1. 验证用户名密码
  2. 生成 session_id = randomHex(32)
  3. 初始 counter = 0
  4. 初始 sk = HMAC-SHA256(K, session_id + "0")  // K=根密钥
  5. UPSERT user_session (user_id, session_id, sk, counter, expire_at=now+7d)
  6. 设置3个Cookie(均 HttpOnly+Secure+SameSite=Strict):
     - _sid = AES-GCM加密(session_id, K)  → 密文, 前端不可读
     - _sk  = sk                           → 前端不可读
     - _ct  = AES-GCM加密(counter, K)      → 密文, 前端不可读
```

**⚠️ counter 和 sk 的明文绝不出现在网络传输中** — counter 加密后才放Cookie, sk 通过 HttpOnly Cookie 传输且前端JS无法读取。

### 阶段2: 客户端发起请求

```
浏览器自动携带Cookie(_sid, _sk, _ct), 前端JS:
  1. 生成 nonce = crypto.randomUUID().replace(/-/g,'')  // 32位Hex
  2. 签名 sign = HMAC-SHA256(_sk, _sid + nonce + _ct)
     → 但_sk/_sid/_ct都是HttpOnly, JS读不到!
```

**关键设计变更**: 因为 HttpOnly Cookie 前端JS无法读取，签名必须在服务端完成。客户端只需:
- 生成 nonce 放在请求头 `X-Nonce`
- 浏览器自动带上3个HttpOnly Cookie

服务端侧完成签名验证(见阶段3)。

### 阶段3: 服务端校验

```
服务端收到请求:
  1. 从Cookie解密得到 session_id, sk, counter
  2. 从请求头取 nonce
  3. 查DB确认 session_id 对应的会话存在且未过期
  4. 比对Cookie中的counter与DB中的counter是否一致(防篡改)
  5. 比对Cookie中的sk与DB中的sk是否一致
  6. INSERT nonce → 若唯一索引冲突 → 重放攻击, 拒绝
  7. 全部通过 → 放行请求
```

### 阶段4: 服务端更新凭证

```
请求通过后:
  1. new_counter = counter + 1
  2. new_sk = HMAC-SHA256(K, nonce + new_counter的字符串)
  3. UPDATE user_session SET sk=new_sk, counter=new_counter
  4. 响应中更新Cookie:
     - _sk  = new_sk
     - _ct  = AES-GCM加密(new_counter, K)
```

→ 每次请求后 sk 和 counter 都变，旧参数立即失效。

### 阶段5: 数据库自动清理

```
定时任务(每30秒):
  1. DELETE FROM request_nonce WHERE expire_at < NOW()   -- 物理删除60秒前的Nonce
  2. DELETE FROM user_session WHERE expire_at < NOW()     -- 物理删除7天前的会话
```

---

## 三、安全验证

| 攻击方式 | 防御机制 |
|---------|---------|
| 抓包重放 | Nonce唯一索引, 同一nonce第二次INSERT必失败 |
| 偷Cookie复用 | counter+1后旧counter失效, sk也随之变化 |
| 伪造签名 | 没有根密钥K无法解密counter, 无法生成正确的sk |
| 批量刷请求 | HttpOnly+SameSite=Strict, 脚本无法获取Cookie |
| 数据库爆盘 | Nonce仅存60秒自动清理, Session一人一条 |

## 四、存储容量计算(日请求100万次)

| 表 | 峰值条数 | 单条大小 | 峰值占用 |
|---|---------|---------|---------|
| request_nonce | ~720条(60秒窗口) | 40B | **28KB** |
| user_session | 用户数(如10万) | 116B | **11MB** |

Nonce清理后: 60秒前的数据全部物理DELETE, 表内始终只有最近60秒的数据。
**100万请求/天, 数据库额外占用 < 12MB**, 永不爆盘。
