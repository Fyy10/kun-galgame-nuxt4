# App 直连论坛 API（2026-09-17）

> 本仓自有工程笔记（**非** infra 镜像）。kungal-apps 工单 02「App 直连论坛 API 的四项前置」的论坛侧交付与契约。
> 取代 [app-aggregation-api.md](./app-aggregation-api.md) 的方向。

## 裁决（2026-09-17，App 侧拍板）

1. **复用 `/api/*`，信封 `{code, message, data}` 不变。** 不另起 `/app/v1` 或 problem+json：App 消费的就是论坛现有面，同一数据两种信封并存没有收益。真出现负载分叉时，归宿是蓝图预留的 mobile BFF，不是平行 API 面。
2. **所有需要登录的路由统一接受 Bearer；staff 能力在 Bearer 通道一律拒绝**（生态「staff 面永不对外」）。
3. **NSFW 偏好走请求头**，不让 App 伪造 web 的 `KUNGalgameSettings` cookie。
4. **版本闸单端点，一张表覆盖全平台**；`min_version` 不按平台拆。
5. 网页补 `/app` 下载页与 `/app/oauth/callback` 兜底页。

## 1. 鉴权：`Authorization: Bearer`

App 用 AppAuth + PKCE 直接从 OP 换出 access token，然后 `Authorization: Bearer <token>` 直打 `https://www.kungal.com/api/*`（Cloudflare → kungal-api，Nitro 不在这条链上）。

### 校验（`internal/user/oauth/verify.go`）

- 本地用 OP 的 JWKS 验签：只收 ES256 / RS256，必须带 `kid`，头部 `typ` 必须是 `at+jwt`（挡掉 id_token）。
- `iss` 必须等于 OP 的**公网**源 `https://account.nextmoe.com`，`exp` 必填，容忍 30s 时钟偏差。
- `client_id` 必须在 `KUN_BEARER_CLIENT_IDS` 白名单里。**不看 `aud`**：OP 把 `aud` 填成站点域名（`www.kungal.com`），同站点的任何 client 都会带它。
- 用户 id 取自定义声明 `id`（跨库同一个整数）；`sub` 是 UUID。
- JWKS 按 kid 缓存；遇到未知 kid 才重拉，且最多 30s 一次。

**实测（2026-09-17）**：抽样 40 个线上会话里的 access token，34 个是 `ES256` + `typ: at+jwt`，kid `u1x9P7ub…` 与线上 JWKS 一致。另外 6 个 `HS256` 是 OP 切换非对称签名前签发、之后没再刷新的老会话，App 拿到的 token 不会是这种。本地 dev OP 走完整 PKCE 流程签出的 token 形状相同（`iss=http://127.0.0.1:9277`，`aud=["www.kungal.com"]`，`site_id=2`）。

### 语义（`internal/middleware/bearer.go`）

| 情况 | 响应 |
|---|---|
| 带 `Bearer`（含空 token） | 只走 Bearer 路径，不看 cookie |
| token 无效/过期/client 不在白名单/通道未开 | `401 {"code":205}`，**可选登录的读路由也一样**，不降级为匿名（否则 App 发现不了过期） |
| JWKS 拉不到（OP 故障） | `500 {"code":233}`，不是 401，App 不要因此登出 |
| 首次见到该用户的初始化失败 | `500 {"code":233}`，下次请求会重试 |
| 访问任何管理/权限闸 | `403 {"code":233,"message":"管理操作请在网页端进行"}` |

- **没有 staff 能力**：Bearer 用户的 `roles` 会剥掉 `moderator` / `admin` / `ren`（`creator` 等保留）。`UserInfo.Can` / `CanModerate` / `CanAdminister` 对 Bearer 恒为 false，**连个人权限覆盖也不生效**；`/api/perm/mine` 恒返回空列表。`internal/middleware/bearer_guard_test.go` 禁止任何代码绕过这些方法、直接在 `u.Roles` 上做能力检查。
- **首次见到用户**（每用户 24h 一次）：补做网页 OAuth 回调里做的两件事：`kungal_user_state` 行（没有这行，发帖会失败）和社区 trust boost。boost 用的是**未剥离**的 roles，因为它每个用户只声明一次、由 SETNX 守着，从 App 先声明一个剥离后的值，会把版主之后从网页登录时的 boost 锁掉。
- **封禁**：access token 有效期 15 分钟，没有吊销列表，所以封禁最多滞后 15 分钟，和网页会话的刷新周期相同。封禁用户发的内容仍在渲染层隐藏。
- **下游转发**：论坛把 App 的 token 原样当 Bearer 转给 catalog `/v2` 用户面和 OAuth `/auth/me`。catalog 不校验 `aud` / `client_id` 白名单，但会读取该 client 的 `catalog_site` 和 scope。社区接口和图床（`pkg/imageclient`）都走论坛 client 的 Basic 认证，不经过用户 token，不受影响。

### 对 infra 注册 kungal-app 的要求（工单 01）

- `site_id = 2`（www.kungal.com）：否则 `site_roles` 为空。
- `catalog_site = kungal`：否则编辑提案 / 认领返回 `SITE_NOT_BOUND`。
- `owner_user_id` 为空：否则会被当成第三方 client（编辑受限，catalog 管理面拒绝）。
- scope 需包含 `catalog:edit`、`folder:read`、`folder:write`。
- 固定的 client_id（如 `kungal-app`）后台生成不了，只能手工插入或写进 seed。
- playtime 按 `(user, work, client_id)` 分行存，App 上报的是独立一行，读取时取各行最大值。

**实际注册（2026-09-17，线上 `oauth_clients` 核对过）**：两个 public client，上述四条都满足，scope 均为 `openid profile email catalog:read catalog:edit folder:read folder:write`，RT 90 天。

| client_id | 平台 | redirect_uris |
|---|---|---|
| `kungal-app` | Android / iOS | `com.kungal.app://oauth2redirect`、`https://www.kungal.com/app/oauth/callback`、`com.kungal.app://logout` |
| `kungal-app-desktop` | Windows / Linux | `http://127.0.0.1/oauth/callback`（loopback，端口宽松匹配） |

**两个都要进白名单**：漏掉哪个，那个平台的所有用户都会 401。

### 放行范围

凡是挂了 `Auth` / `OptionalAuth` 的路由都接受 Bearer（完整清单见 `internal/app/testdata/routes.golden`）。App 首批消费：

| 端点 | 方式 |
|---|---|
| `GET /api/topic`（`page` `limit` `sort_field` `sort_order` `category`） | 匿名可读，带 Bearer 时附带登录态 |
| `GET /api/topic/:tid`、`GET /api/topic/:tid/reply` | 同上 |
| `GET /api/galgame`、`GET /api/galgame/:gid` | 同上（Go api 自己调 catalog 拼数据，不经 Nitro） |
| `GET /api/user/:id` | 匿名 |
| `POST /api/topic`、`POST /api/topic/:tid/reply` | 必须 Bearer，支持幂等键 |

```sh
# 匿名
curl -s 'https://www.kungal.com/api/topic?page=1&limit=10'
# Bearer 发回复
curl -s -X POST 'https://www.kungal.com/api/topic/4230/reply' \
  -H "Authorization: Bearer $AT" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H 'Content-Type: application/json' \
  -d '{"topic_id":4230,"content":"…"}'
```

## 2. 幂等键：`Idempotency-Key`

挂在 `POST /api/topic`、`POST /api/topic/:tid/reply`（`internal/middleware/idempotency.go`）。

- 头可选，不带就照旧处理（网页不带）。值必须是 UUID，否则 `400`。
- Redis 键为 `kungal:idem:{uid}:{scope}:{uuid}`：按用户和端点隔离，别人用同一个键不会拿到你的结果。请求指纹 = sha256(方法 + 路径 + 请求体)。
- 第一次请求：先写一个 2 分钟 TTL 的「处理中」标记（进程中途挂掉也不会锁死键 24 小时）。
  - 返回 2xx：把状态码和响应体保存 **24 小时**。
  - 返回非 2xx：删除键，因为什么都没创建，可以用同一个键重试。
- 重复请求：

| 情况 | 响应 |
|---|---|
| 同键、同指纹、已完成 | 原样返回首次的状态码和响应体，加 `Idempotent-Replayed: true` |
| 同键、首次请求仍在处理 | `409 {"code":237}`，App 稍等后用同一个键重试 |
| 同键、不同指纹（换了内容或路径） | `422 {"code":238}`，属于客户端 bug |

## 3. NSFW 偏好：`X-Kungal-Nsfw`

- `X-Kungal-Nsfw: true|false|1|0`（`strconv.ParseBool` 能解析的值），**优先于** `KUNGalgameSettings` cookie。
- 不带或解析失败时回落到 cookie；两者都没有时默认 SFW。
- 「优先显示原名」偏好目前仍只认 cookie，App 需要时再加一个同类请求头。

## 4. 版本闸：`GET /api/app/version`

公开接口，返回现有信封：

```json
{ "code": 0, "message": "成功", "data": {
  "min_version": "0.1.0", "latest_version": "0.1.0", "notes": "",
  "downloads": { "android": "…", "ios": "…", "windows": "…", "linux": "…" } } }
```

- 数据来自环境变量 `KUN_APP_MIN_VERSION` / `KUN_APP_LATEST_VERSION` / `KUN_APP_RELEASE_NOTES` / `KUN_APP_DOWNLOAD_{ANDROID,IOS,WINDOWS,LINUX}`，改了要重启 kungal-api。
- 版本号必须是 `MAJOR.MINOR.PATCH`；格式不对或 min 高于 latest，服务**启动失败**。否则 App 会把所有用户强更到一个不存在的版本。
- 某个平台没配下载地址时，回落到 `https://www.kungal.com/app`，所以 App 总能拿到一个可以打开的链接。下载页会把这种平台显示为「即将推出」。

## 5. App Links 与网页

- `apps/web/public/.well-known/assetlinks.json`：`com.kungal.app`，**指纹是全 0 占位**，等 App 侧提供直发签名钥的 SHA-256 后替换。本地验证返回 200 + `application/json`。
- 裸域 `kungal.com` 会 302 到 www，而 Android 验证不跟随跳转，所以 App 的 intent-filter **只能声明 `www.kungal.com`**。
- `/app`：下载页，数据来自版本闸接口。
- `/app/oauth/callback`：App Links 验证失败或没装 App 时的落地页。页面提示「请在 App 中完成登录」，不消费授权码，挂载后把 `code` / `state` 从地址栏清掉，并带 `noindex` 和 `referrer: no-referrer`；已加入 sitemap 排除列表。

## 6. 上线

- **不需要数据库迁移**（幂等键存在 Redis 里）。
- `docker-compose.prod.yml` 已写入 `KUN_OIDC_ISSUER`、`KUN_OIDC_JWKS_URL`（内网 `http://oauth:9277/oauth/jwks`，已从生产网络实测可达）和 `KUN_APP_*`。
- `KUN_BEARER_CLIENT_IDS=kungal-app,kungal-app-desktop` 直接写在 compose 里（client_id 是公开标识，和 `OAUTH_CLIENT_ID` 同样处理），不走 Dokploy 面板。以后 infra 新增一方 client 时改这一行；要紧急关掉 Bearer 通道，就把它改成空值后重新部署。

## 7. 没做 / 已知缺口

- **ETag / 304**：论坛 API 目前没有（线上响应头里没有 `etag`，Cloudflare `DYNAMIC`）。
- **429**：来自 Cloudflare，**不带信封**，App 要按 HTTP 状态码处理。
- **catalog `/v2/me` 的按 IP 限额**：上游把用户 token 也按客户端 IP 分桶，而论坛只有一个出口 IP。App 用户的请求经论坛转发，会加重这个已知问题；论坛侧无法修复。
- **论坛面 OpenAPI 化（路 A）**：尚未开始；App 先用手写薄客户端。
