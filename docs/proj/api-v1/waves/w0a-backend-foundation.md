# W0a · 后端地基 + `GET /api/v1/topics`

> 裁决记录（中文，给人看）。执行者：cursor-agent，模型 Grok 4.6 Extra High，在沙箱里跑。验收：Claude。写于 2026-09-18。
>
> **执行方式**：按 `.claude/skills/dispatch-cursor/`，每次派发都拿一份英文任务书，写在 session scratchpad 里，内容是下表中的一段，并把本文的相关裁决原样译进去。沙箱里没有数据库、docker 和网络（包仓库除外），也不能写 git，所以：
>
> - 执行者只写代码、跑不需要数据库的门；
> - DB 测试由执行者写，由 Claude 在一次性库上跑；
> - 提交由 Claude 做；
> - 下文里凡是「你起容器」「你提交」的说法，都按这条分工理解。
>
> | 派发 | 范围（本文章节） | 依赖 |
> |---|---|---|
> | W0a-1 `problem-registry` | §1 的 huma 依赖、§2 | — |
> | W0a-2 `identity-resolve` | §3 | —（与 W0a-1 并行） |
> | W0a-3 `apiv1-wiring` | §4、§4.1、§5、§6、§8，§10 的路由清单适配与 F3 / F7 | W0a-1、W0a-2 |
> | W0a-4 `spec-gates` | §10 其余的门 | W0a-3 |
> | W0a-5 `topics-list` | §7、§9、§11、§12、§13 | W0a-4 |

## 0. 工作方式与红线

**先读，按顺序，全部读完再动手：**

1. 本仓 `CLAUDE.md`（铁律与注释规则）；
2. `docs/proj/api-v1/README.md`、`01-standard.md`、`02-governance.md`，本波要实现的就是这三份文件里的规定；
3. infra 规范（只读）：`/home/kun/Desktop/code/website/nextmoe-infra/refs/api-v2/`，读 `01-axioms.md`、`02-protocol.md`、`04-representation.md`、`05-collections.md`、`07-governance.md`、`10-errors.md`；
4. infra 的实现（只读，移植时参考，**不得**原样照抄，论坛的差异见规范）：`/home/kun/Desktop/code/website/nextmoe-infra/apps/api/internal/platform/apiv2/` 下的这些文件：
   - `problem/registry.go`、`problem/problem.go`；
   - `handler/setup.go`、`handler/gates.go`、`handler/gates_repr.go`、`handler/contract_test.go`；
   - `repr/id.go`、`repr/image.go`、`repr/resource.go`；
   - `collect/page.go`、`collect/query.go`；
   - `protocol/idempotency.go`。

**红线：**

- 只在你的 worktree 里写。infra 仓和主 checkout 都只读。
- 提交信息与代码注释全英文。注释默认不写，只有 `CLAUDE.md`「Comments」一节允许的情况才写。
- 不 push。按主题分成若干个提交，每个提交都能编译、测试通过。
- 不改 `docs/{oauth,image_service,artifact}/`，不改 KunUI，不引入渐变。
- **数据库**：
  - DB 测试只能用你自己起的一次性 Postgres 容器：空闲端口，`POSTGRES_HOST_AUTH_METHOD=trust`，不带任何口令；
  - `TEST_DATABASE_DSN` 显式指向它；
  - 绝不读 `.env` 里的 DSN，绝不连 dev 库或线上库；
  - 绝不打印任何 DSN 或密钥；
  - 用完删容器。
- Go 的测试与 lint 一律 `GOTOOLCHAIN=go1.26.1`（系统 Go 1.27 会让 errcheck 假失败、让路由 golden 本地绿 CI 红）。
- 不要用 `pkill -f`：它会匹配到自己所在的 shell，宽模式还会杀掉本机其它仓的开发服务器。要停进程，先 `ss -tlnp` 查端口定位 PID，再 `kill <pid>`。
- 本波**不改网页**（`apps/web`），旧 `GET /api/topic` 保留，W0b 再切。
- 本任务书里的事实（取值、行数、文件位置）已经核对过。你若发现哪条与代码或数据不符，**停在那一项，写进最终报告**，不要按任务书硬写。

## 1. 依赖

- 加 `github.com/danielgtaylor/huma/v2 v2.39.1`（与 infra 同版本；`adapters/humafiber` 支持 Fiber v3）。若 humafiber 要求更高的 Fiber v3 小版本，就升 `gofiber/fiber/v3`。升级后全部现有测试必须仍然通过。
- 契约测试用 `github.com/getkin/kin-openapi` 的 `openapi3filter` 校验响应。它只在测试里引用，但会进 `go.mod`，这没问题。

## 2. `pkg/problem`：错误码注册表与 problem 体

按 `01-standard.md` §2 实现。

- `Problem` 与 `FieldError` 类型，JSON 形状与 infra `problem.go` 一致。另加：
  - `FieldError.Params`（有类型的结构体，全部字段为指针，缺席时不发；原稿写的 `map[string]any` 会违反 G9，派发时已改）；
  - 顶层扩展成员机制：每个 code 在注册表里声明允许的扩展成员及其类型。W0a 还没有任何 code 带扩展成员，但机制和测试要有。
- 注册表：`Def{Code, Domain, Status, Title, Description}` 与 `ReasonDef`。只收录**本波会产生**的 code：

  | code | 域 | status |
  |---|---|---|
  | `MALFORMED_BODY` `INVALID_PARAMETER` `UNKNOWN_ENUM_VALUE` `LIMIT_TOO_LARGE` `INVALID_CURSOR` `UNKNOWN_SORT` | platform | 400 |
  | `MISSING_CREDENTIAL` `INVALID_CREDENTIAL` | platform | 401 |
  | `SCOPE_REQUIRED` | platform | 403 |
  | `ACCOUNT_BANNED` | kungal | 403 |
  | `NOT_FOUND` | platform | 404 |
  | `METHOD_NOT_ALLOWED` | platform | 405 |
  | `IDEMPOTENCY_KEY_REUSED` | platform | 409 |
  | `IDEMPOTENCY_REQUEST_IN_PROGRESS` | kungal | 409 |
  | `UNSUPPORTED_MEDIA_TYPE` | platform | 415 |
  | `VALIDATION_FAILED` | platform | 422 |
  | `INTERNAL_ERROR` | platform | 500 |
  | `SERVICE_UNAVAILABLE` | platform | 503 |

  - platform 码的 `Title` 与 `Description` 从 infra `registry.go` **逐字**复制。
  - 两个 kungal 码自己写，英文，风格同 infra。
  - reason 注册表收录 infra 的全部 13 条，逐字复制。
  - 每个 reason 允许的 `params` 键按 `01-standard.md` K5 的表，写进 `ReasonDef`。
- type URI：
  - platform 码 = `https://developer.nextmoe.dev/problems/platform/<kebab>`，与 infra 逐字相同；
  - kungal 码 = `…/problems/kungal/<kebab>`。
  - kebab 由 code 机械转换：小写，下划线换成连字符。
- 构造与写出：
  - `problem.New(code, detail)`，以及带字段错误的变体；
  - `problem.Write(c fiber.Ctx, p)`。它填 `instance` 与 `request_id`，设 `X-Request-ID`、`Cache-Control: no-store`，401 时设 `WWW-Authenticate: Bearer realm="kungal"`，以 `application/problem+json` 写出。
  - 注意 Fiber 的 `c.JSON` 会覆盖 `Content-Type`，要用 `c.JSON(v, contentType)` 那个签名（摸鱼 `pkg/problem/problem.go` 就在这里踩过坑）。
  - `errors` 在没有字段错误时发 `[]`，不是 `null`。
- **huma 桥**：`FromHuma` 把 huma 产生的错误转成 problem：
  - huma 的 400 / 422 / 415 等映射到对应 code；
  - 每个 `huma.ErrorDetail` 映射到**精确的** `reason` 与 `params`：`REQUIRED`、`TOO_LONG`、`TOO_SHORT`、`OUT_OF_RANGE`、`TOO_MANY_ITEMS`、`UNKNOWN_VALUE`、`INVALID_FORMAT`……；
  - infra 一律给 `INVALID_FORMAT`，这里不照抄。做法由你定，可以匹配 huma `validation` 包的消息模板，也可以改写这些模板。
  - 位置：`query.` / `path.` → `parameter`，`header.` → `header`，`body.` → `pointer`（RFC 6901，数组下标转 `/n`）。
  - **测试**必须逐条覆盖 v1 schema 能触发的每一种 huma 校验消息（必填缺席、超长、过短、越界、非法枚举、格式错、数组过长、类型错），断言得到的 `reason` 与 `params`。
- `request_id`：`req_` + 26 位 Crockford ULID（真 ULID：48 bit 毫秒时间戳 + 80 bit 随机）。入站 `X-Request-ID` 若匹配 `^req_[0-9A-HJKMNP-TV-Z]{26}$` 就沿用。

## 3. 身份解析重构（`internal/middleware`）

现在 `Auth()` / `OptionalAuth()` 在失败时直接用旧信封写响应。把「解析身份」抽成一个不写响应的函数：输入 `fiber.Ctx`，输出 `(*UserInfo, 结果类别)`。结果类别区分：

- 无凭证；
- cookie 会话不存在或过期；
- cookie 会话有效；
- Bearer 有效；
- Bearer 无效；
- 会话存储出错（Redis 返回 `redis.Nil` 以外的错误）；
- 封禁；
- scope 不足（旧的 235 路径）。

旧的 `Auth()` / `OptionalAuth()` 改成调用它再按旧行为写旧信封。**旧路由的外部行为一字不变**，现有测试必须全绿，`routes.golden` 里旧路由的链不变。

v1 用同一个函数，按 `01-standard.md` K3 映射到 problem：

- cookie 失效在 `optional` 档降级为匿名，在 `required` 档是 `INVALID_CREDENTIAL`；
- Bearer 无效永远 `INVALID_CREDENTIAL`；
- 存储出错是 `SERVICE_UNAVAILABLE`；
- 封禁是 `ACCOUNT_BANNED`；
- scope 不足是 `SCOPE_REQUIRED`。

Bearer 的 `WithoutStaff` / `viaBearer` 语义原样保留；`bearer_guard_test.go` 的规则也要覆盖 v1 代码。滑动续期、刷新锁这些旧副作用在 v1 路径上同样要发生。

## 4. `internal/apiv1`：huma API 的装配

- `Setup(app *fiber.App, deps Deps, registrars ...func(huma.API)) huma.API`，由 `setupRoutes` 调用，挂在 `/api/v1`。
  - **v1 操作一律经 `Setup` 的 `registrars` 参数注册**，在 `setupRoutes` 里传入。`Setup` 在注册完之后挂一个收尾处理器：v1 下没匹配到的请求就地回 404，或者回 405 并带 `Allow`。Fiber 按注册顺序匹配，旧 `/api` 组又把 `OptionalAuth` / `Auth` 铺在它下面的所有路径上。W0a-3 验收时发现，没有这个收尾，未知的 v1 路径会一路掉进旧的鉴权链，回旧信封。代价是 `Setup` 返回之后再注册的操作全部不可达，测试同样得走 `registrars`。
  - v1 的 GET 路由由 `Setup` 镜像成 HEAD。Fiber 自动补的 HEAD 在启动时才加，会落在收尾处理器后面，结果全部回 405。
  - spec 由 `app.V1Spec()` 产出：它在零依赖下跑真实的 `setupRoutes`，`cmd/openapi` 和「提交的 spec 是否过期」的测试都用它，所以 spec 与服务端注册的永远是同一张表。
  - 注册**不得**依赖活的依赖：`route_manifest_test.go` 用零值 `App` 调 `setupRoutes`，`cmd/openapi` 也要能在零依赖下构建同一个 API。依赖只在请求时使用。
- huma 配置：
  - title `KUN Galgame Forum API`，版本 `1.0.0-preview`，`info.x-stability: preview`，`info.description` 非空；
  - `servers` 写生产域名（打开 `apps/api/.env.example` 或部署配置确认公开 base，写入时不得带任何密钥）；
  - 关掉 huma 自带的 `/openapi.json`、`/docs`、`/schemas` 路由。
- `securitySchemes`：`session`（apiKey，in cookie，name `kungal_session`）与 `bearer`（http，scheme bearer，bearerFormat JWT）。
  - 操作的三档鉴权用 `security` 表达：`public` 无；`optional` = `[{session: []}, {bearer: []}, {}]`；`required` = `[{session: []}, {bearer: []}]`。
  - 一个 huma 中间件按当前操作的档位执行 §3 的解析与映射，把 `UserInfo` 放进 context。给 handler 一个取当前用户的辅助函数。
- huma 的全局错误钩子（`huma.NewError` / `NewErrorWithContext`）按路径前缀 `/api/v1` 分流到 `FromHuma`，其余路径保持 huma 默认。这也是 infra 的做法，见 `handler/setup.go`。
- `/api/v1/**` 上找不到路由、方法不对、media type 不支持、panic，都要回 problem 体。现有的全局 `recover.New()` 与 Fiber 的错误处理器按前缀分流：v1 走 problem，旧路径不变。
- 每条 v1 响应（含错误）都带 `X-Request-ID` 与 `Cache-Control: no-store`。
- CORS：在现有 `middleware.CORS` 的 `ExposeHeaders` 里加上 `01-standard.md` §8 列出的头，旧路径一并受益，无副作用。
- **不读**偏好 cookie：v1 handler 不调用 `utils.IsSFW`、`PrefersOriginalName`，也不依赖 `NamePreference` 写的 locals。
- 访问日志：若现有请求日志能拿到 User-Agent，对 v1 请求记录 UA 里的 `kungal-app/<版本>`，退役旧形状时要用（`02-governance.md` §4）。没有请求日志就不新建，写进报告。

### 4.1 幂等（v1 版）

`01-standard.md` K12 的规则，做成按操作开启的 huma 中间件（操作元数据里声明 `optional` / `required`），存储沿用现在的 Redis 实现与键空间设计。与现行 `middleware.Idempotent` 的差异：

- 头名 `Idempotency-Replayed`；
- 缺键：`required` 档 → `400 INVALID_PARAMETER` + `{header: "Idempotency-Key", reason: "REQUIRED"}`；
- 格式错 → 同上，`reason: INVALID_FORMAT`；
- 接受 UUID（任意版本）或 ULID；
- 处理中 → `409 IDEMPOTENCY_REQUEST_IN_PROGRESS`；
- 请求不同 → `409 IDEMPOTENCY_KEY_REUSED`；
- 存 2xx–4xx 的最终响应（状态、`Content-Type`、`Location`、响应体），5xx、409 与 429 释放。

旧路由上的 `middleware.Idempotent` **保持原样**，App 在迁到 v1 之前还在用它的行为。本波没有 v1 的 `POST`：在测试里注册一个仅测试用的操作，覆盖全部分支（首次、重放、处理中、请求不同、缺键、格式错、5xx 释放）。

## 5. 表示层与集合的公共件

W0a-4 落地，包是 `internal/apiv1/repr` 与 `internal/apiv1/collect`；领域包放在 `internal/<domain>/apiv1/` 并引用它们。

- 标量类型各带自己的 schema（`huma.SchemaProvider`），DTO 字段用类型声明，字段上的 `doc` 照常成为描述：
  - `repr.DecimalID`：`^[0-9]+$`，1–20 位；`repr.ID(int)` 构造，`repr.ParseID` 只收正整数；
  - `repr.DateTime`：UTC、秒精度、`Z` 结尾，同 infra `repr/id.go`；`repr.Timestamp` / `repr.TimestampPtr` 构造；
  - `repr.CalendarDate`：`YYYY-MM-DD`；`repr.Date` 构造。
- **可空的写法**：字段是指针且没有 `omitempty` ⇒ 值为 `null`，`sealDocument` 把 schema 标成可空（对象用 `anyOf [$ref, null]`）。字段是指针且有 `omitempty` ⇒ 缺席。非指针字段永远在场、永不为 `null`。
- `repr.Image`：`{url, hash, width, height, thumbhash, sexual}`，从图床 hash + `imageclient.ImageMeta` 构造（`NewImage`），或从正文 token `/image/<hash>[_variant]` 构造（`NewImageFromToken`，复用 `markdown.ParseContentImageRef`）。
  - `sexual`：0 / 1 / 2 → `safe` / `suggestive` / `explicit`，缺席 → `null`（未分级 ≠ 安全）。
  - 尺寸为 0 视为未知，发 `null`。键名与类型对齐 infra `repr/image.go` 的同名键；论坛不发 `violence` 与 `source`。
- `repr.UserRef`：`{object: "user", id, name, avatar: Image | null}`。有 `avatar_image_hash` 就出 Image，否则 `null`。
  - **裁决**：线上 10 个头像是外链（bilibili、抖音等）且没有 hash 的用户，v1 发 `null`；Image 的 `hash` 保持必填，不为它们放宽。重新托管是 infra 的事。
  - 已注销用户的占位名 `已注销用户` 原样透传。
- `repr.List[T]`：`{object: "list", items, next_cursor?, total?}`。`items` 永不 `null`；`next_cursor` 末页省略；`total` 只在请求时出现。schema 名可读（`ListProblemType`）。
- 游标（`collect.EncodeCursor` / `DecodeCursor` / `Fingerprint`）：
  - `cur_` + base64url（无填充）的 JSON `{v, s, f, k}`：版本、排序 token、过滤指纹、keyset 值（字符串）；
  - 解码严格：前缀、base64、JSON、版本、排序与指纹任何一项不对都是 `400 INVALID_CURSOR`，`errors[]` 为 `{parameter: cursor, reason: INVALID_FORMAT}`。不合 `^cur_` 语法、在 huma 校验阶段就被拒的游标，错误码同样是 `INVALID_CURSOR`；
  - 不签名：过滤与可见性闸永远在服务端重新施加，改游标只能改起点。
- 查询参数（`collect.Page` 嵌入 `cursor` + `limit`，`collect.Total` 单独嵌入 `include_total`，只有 `total` 与 `items` 同口径的集合才嵌）：
  - `limit`：1–100，默认 20，超限 `400 LIMIT_TOO_LARGE`，不 clamp；小于 1 → `400 INVALID_PARAMETER`；
  - 布尔：只收 `true` / `false`，由 `Setup` 装的中间件对每个 v1 操作的每个布尔查询参数统一执行；
  - 封闭枚举：未知值 → `400 UNKNOWN_ENUM_VALUE`，`errors[]` 带 `parameter`、`reason: UNKNOWN_VALUE`、`params.allowed`；
  - `sort`：未知 → `400 UNKNOWN_SORT`。
  - 这些错误码由 `pkg/problem` 的 `pickCode` 按参数名选出：约束写在 huma schema 里，让 spec 如实描述。

## 6. 元数据端点

- `GET /api/v1/problems` → `{object: "list", items: [{object: "problem_type", code, domain, status, type, title, description}]}`，按域、code 排序；
- `GET /api/v1/problems/reasons` → 同形的 reason 列表，并带每个 reason 允许的 `params` 键；
- 两者都是 `public` 档。

## 7. `GET /api/v1/topics`

`optional` 档，游标集合，条目是 `TopicSummary`，`object: "topic"`。

**语义必须与现行 `TopicListRepository.FindList` 一致**，再补上它缺的东西：

- 结果集：
  - `topic.status != 1`；
  - `SharedListPredicate("topic", authenticated)`；
  - `nsfw=false`（默认）时 `is_nsfw = false`；
  - `category` 在场时按分类过滤。
- 每个排序都以 `topic.id` 作 tie-breaker；keyset 分页。
- `Count` 的错误不得丢弃。现行代码 `query.Count(&total)` 丢了错误。

**参数：**

| 参数 | 规则 |
|---|---|
| `sort` | 封闭枚举，默认 `bumped_desc`。每个排序键各有 `_asc` / `_desc`：`bumped`（`status_update_time`）、`created`、`view`、`view_1d`、`view_7d`、`view_30d`、`like_count`、`favorite_count`、`upvote_count`。键集合对齐 `internal/constants/topic.go` 的两张表加 `view_1d`。token 的最终拼写由你在 spec 里定，要求一致、全小写 snake_case、方向做后缀 |
| `category` | 封闭枚举 `galgame` / `technique` / `others`；缺席 = 全部 |
| `nsfw` | 布尔，默认 `false` |
| `limit` / `cursor` | 按 §5。**不提供 `include_total`**：封禁作者的话题是查询之后在渲染层过滤的（`userclient.IsRenderable`），SQL 数出的总数与返回条目不同口径，违反 infra 05 §3。因此一页可以少于 `limit` 条，`next_cursor` 按取到的最后一行计算 |

`view_1d` 排序的键是 `topic_view_daily` 当日计数的关联子查询（见 `topicOrderCol`）。keyset 要对同一个表达式成立：游标里存当时的值，`WHERE (expr, id) < (?, ?)`。当日计数会增长，翻页期间顺序可能漂移，这是这个排序本身的性质；只要保证不重复、不死循环即可。

**`TopicSummary` 字段**（旧 `dto.TopicCard` → v1）：

| v1 | 来源 / 规则 |
|---|---|
| `object` | `"topic"` |
| `id` | 字符串 |
| `title` | 原样；`maxLength` 取建表与校验上限（`CreateTopicRequest` 是 233） |
| `state` | 封闭枚举 `published` / `hidden`。线上普查（2026-09-18）：`status` 0 = 3219 行、1 = 313 行、**2 = 2 行、3 = 1 行**。2 和 3 是 2024 年的遗留值（id 1712、1824、1160），现行代码对它们和 0 一视同仁。本波加一个迁移把它们归一成 0（见 §9），映射只需 0 → `published`、1 → `hidden`；遇到其它值是数据缺陷，映射层要报错而不是猜 |
| `category` | 封闭枚举，同上。线上：galgame 2054 / technique 721 / others 760 |
| `sections` | 版块 key 数组。**封闭枚举**，取值就是 `topic_section.name` 的 27 个 slug（`g-walkthrough` … `o-other`，线上 27 行全有关联）。它们是 `/section/{key}` 页面的 URL 段，改拼写会改公开 URL，所以保留连字符，在 F1 的枚举值 snake_case 检查里作为**具名例外**登记（`01-standard.md` §3 的例外清单写法，同 infra G8 例外）。显示名由客户端按 key 查（网页现有 `app/constants/topic.ts`） |
| `cover_images` | `[Image]`。旧的 hash 数组 + `cover_image_meta` 合成一个对象数组，顺序不变 |
| `user` | `UserRef` |
| `view_count` | 旧 `view` |
| `like_count` / `reply_count` / `comment_count` | 原样 |
| `has_best_answer` | 原样 |
| `mini_apps` | 封闭枚举数组 `poll` / `lottery`（`pkg/miniapp` 的 `KindPoll` / `KindLottery`） |
| `is_nsfw` | 旧 `is_nsfw_topic` |
| `bumped_at` | 旧 `status_update_time`。它是**顶帖时间**：回复、评论、投票、抽奖会把它刷新到当前时间，但创建超过 3 个月的话题不再被顶（`model.BumpCutoff`、`TouchStatusUpdateTime`）。所以它不是「最后活跃时间」，名字取 Discourse 的 `bumped_at`。description 里写明这条语义 |
| `created_at` | 旧 `created` |
| `upvoted_at` | 旧 `upvote_time`，可空 |

每个 property 都要有 description、`maxLength` 或 `minimum` 等约束（G2 / G14）。

W0a-5 落地（2026-09-18 验收）：

- sort token：`bumped` / `created` / `views` / `views_1d` / `views_7d` / `views_30d` / `likes` / `favorites` / `upvotes`，各带 `_asc` / `_desc`，共 18 个。布尔参数按 F1 叫 `include_nsfw`，不叫 `nsfw`。
- 游标指纹含 `sort`、`category`、`include_nsfw` 与是否登录；时间键以 RFC 3339 纳秒精度、UTC 进游标，秒级会跳过同一秒的行。
- 作者在 OAuth 里查不到（删号）时 `user.name` 为 `null`，不再发「已注销用户」，客户端自己出本地化文案（F8）。查询 OAuth 失败是 503，原因写进日志。
- `mini_apps` 的查询错误不再吞掉（`miniapp.Lookup`）；旧的 `ByTopic` 留给旧路由。
- 声明的状态码恰好是 200 / 400 / 401 / 403 / 500 / 503。

**声明的错误**：400（`INVALID_PARAMETER` / `UNKNOWN_ENUM_VALUE` / `LIMIT_TOO_LARGE` / `INVALID_CURSOR` / `UNKNOWN_SORT`）、401（`INVALID_CREDENTIAL`，Bearer 无效）、403（`ACCOUNT_BANNED`，按 §3 的解析，封禁用户走可选档时的行为请查清现行 `OptionalAuth` 与 bearer 路径后照现行语义处理，写进报告）、500、503。

## 8. spec 生成

- `cmd/openapi`：零依赖构建 v1 API，写 `apps/api/openapi/kungal-v1.json` 与 `apps/api/openapi/problems.json`。输出确定：键序稳定、两空格缩进、末尾换行。
- `Makefile` 加 `openapi` 目标。
- 两个文件提交进仓。

## 9. 迁移：话题状态归一

新迁移 `096_topic_status_normalize`（编号以你开工时的最新编号为准，接着排）：

- up：`UPDATE topic SET status = 0 WHERE status IN (2, 3);`
- down：不还原，写明原因（原值没有任何代码语义，而且不可区分）。
- 迁移文件按 `CLAUDE.md` 要求写注释：改了什么、为什么、现存的行怎么处理。
- 在报告里提醒：这条需要在生产跑，按 memory「kungal-prod-deploy-and-migrate」，迁移在部署时自动执行。
- 确认迁移的写法与现有迁移一致（幂等等）。

W0a-5 落地：up 在归一之后加 `CHECK (status IN (0, 1))`（`topic_status_check`），让这个缺陷不能再写进来。唯一的写入方 `hideDecision` 只写 0 或 1，新旧代码在两种部署顺序下都满足它。down 只删约束。

## 10. 门（Go 测试）

按 `02-governance.md` §3 实现 **G1 ①、G2、G3、G4、G5/G13、G6、G7、G8、G9、G14、G16、G17、F1、F3、F7**。

W0a-4 落地：门在 `internal/apiv1/gates`（`gates.CheckAll`），对真实文档的断言是 `internal/app/v1_gates_test.go`，跑的是 `app.V1Spec()`，与 `cmd/openapi` 同一条路径。新端点经 `setupRoutes` 注册，自动受它约束。

- 可以移植 infra 的 `gates.go` 与 `gates_repr.go` 的结构，但判据以论坛规范为准。
- **每道门都要有阳性对照**：构造一份违规的 spec 片段，或在测试里注册一个违规操作，走和真实 spec **同一条**检查路径，断言门会红。
- F1 的禁用名、布尔前缀、`_at` / `_date` / `_count` 规则按 `01-standard.md` §3。枚举值检查只对封闭枚举生效，具名例外只有 `sections`。
- G4 的最低状态码集合按 `02-governance.md` §3 的表，从操作的 `security`、参数、请求体、幂等声明**推导**，不要手写每个操作应有哪些状态码。
- F3：新建 `internal/app/testdata/legacy_route_baseline`，写入当前旧路由数（`routes.golden` 里非 `/api/v1` 的行数）；测试断言旧路由数 ≤ 基线。
- F7：`internal/testdb` 在 `KUN_REQUIRE_TEST_DB=1` 且没有 DSN 时 `t.Fatal`，而不是 skip。
- 路由清单测试：`TestRouteManifest` 等现有测试要能处理 huma 注册的路由。
  - handler 名稳定：见 memory `go-1.27-errcheck-broken`，1.26 与 1.27 的内联差异会改名，必须在 `GOTOOLCHAIN=go1.26.1` 下生成 golden。
  - `TestEveryWriteIsAuthenticated` 等依赖 Fiber 中间件链的检查，对 v1 路由改为读 spec 里的 `security`（写操作必须是 `required` 档）。

## 11. 契约测试（DB，G1 ②）

- 用 `internal/testdb` 起真库，装配真实的 Fiber app（含 v1），造数据。造数据的方式按仓里现有 DB 测试的习惯。

W0a-5 落地：`internal/app/v1_topics_test.go`，经 `setupRoutes` 装配；Redis 用 miniredis，OAuth 用 httptest 桩。响应校验改用 `santhosh-tekuri/jsonschema/v6`：文档是 OpenAPI 3.1，kin-openapi 只认 3.0。另有逐字段断言（`TestV1TopicsItemFields`），只校验 schema 抓不到值错。
- `GET /api/v1/topics`：
  - 每个 `sort` token，用 `limit=2` 把全部页翻完，结果与直接 SQL 的排序逐条相等，无重复无遗漏；
  - 覆盖 `nsfw`、`category`、匿名、有会话、封禁作者的话题被滤掉后游标仍正确；
  - `login` 可见域的话题只对登录者出现；隐藏的话题永不出现。
- 每个声明的错误 code 至少一个用例：非法 `limit`、超限 `limit`、非法布尔、未知枚举、未知 sort、坏游标、换了过滤条件的旧游标、无效 Bearer。
- 每条响应都用 kin-openapi 对生成的 spec 校验：状态码在声明集合里、body 符合 schema、`Content-Type` 正确（错误是 `application/problem+json`）；并断言 `X-Request-ID` 与 `Cache-Control: no-store`。
- 未注册的 `/api/v1/xxx` 回 404 problem；错误方法回 405 problem；v1 handler 里 panic 回 500 problem。可以用测试专用的操作触发。
- 会话怎么造：
  - cookie 会话：直接往测试 Redis 写 `kungal:session:v2:<token>` JSON，`oauth_expires_at` 是 int64 unix 秒，写成字符串会静默解析失败；
  - Bearer：用测试密钥自签 JWT，参照 `bearer_test.go` 的做法。
  - Redis 同样用你起的一次性容器。W0a-5 落地用的是 miniredis，所以 `db` 作业只起 Postgres。

## 12. CI

`test.yml`：

- unit 作业加一步：`make openapi && git diff --exit-code openapi/`。W0a-5 落地时没加：`TestCommittedSpecIsCurrent` 已在 `go test` 里做同一件事。
- 新增 `db` 作业：
  - 服务容器 `postgres`（与生产同主版本，生产是 `postgres:18-alpine`）与 `redis`；
  - 用仓里新写的引导脚本从零建库（下一条）；
  - `TEST_DATABASE_DSN` 与 Redis 地址显式指向服务容器，`KUN_REQUIRE_TEST_DB=1`；
  - `go test -count=1 -p 1 ./...`。
- 引导脚本 `apps/api/scripts/testdb-bootstrap.sh`：把「从零建库」的步骤固化下来。已知的顺序见 memory `kungal-db-backed-tests-bootstrap`（原文附在本任务书末尾）。
  - **你要自己跑一遍并核对**：结果 schema 与生产形状一致。判据举例：`galgame` 没有 `vndb_id` 列，`_migrations` 的最新一条是最新迁移。
  - 脚本不得包含任何密钥，只接受显式传入的 DSN。
- **现有 DB 测试第一次在 CI 里真跑，可能有测试本身过时导致的失败。** 测试过时（断言写死了旧形状）可以修；若是产品代码的真缺陷，**不要顺手改产品代码**，写进报告，由 Claude 决定。

## 13. 文档

- 新建 `docs/proj/api-v1/CHANGELOG.md`，记下本波新增的端点与错误码。
- `README.md` 波次看板把 W0a 标为「待验收」。

## 14. 完成标准与最终报告

完成标准：

- `GOTOOLCHAIN=go1.26.1 make lint`；
- `GOTOOLCHAIN=go1.26.1 go test ./...`（无 DSN）；
- 在一次性容器上带 DSN 的 `go test -count=1 -p 1 ./...`；
- 以上全绿，`make openapi` 重跑无 diff。

最终报告（中文）逐项列出：

1. 每个提交的摘要；
2. 本任务书里与实际不符之处；
3. 自行做出的设计决定：游标内容、sort token 拼写、huma 消息映射的做法、头像只有 URL 时的处理；
4. 现有 DB 测试在真库上的结果，以及失败清单和原因；
5. 未完成或有疑问的项；
6. 需要在生产跑的迁移。

---

### 附：memory `kungal-db-backed-tests-bootstrap`（原文）

> Migrating a fresh database is not `migrate -dir up`. Verified 2026-09-13; the order that works:
>
> 1. `go run ./cmd/migrate -dir up` — dies at `053_add_notification_preferences` because `kungal_user_state` does not exist: 007 creates it and 007 is in the runner's **default `-exclude 005,006,007,012,015`**.
> 2. `-only 007`, then `-dir up` again — now dies at `069_galgame_contributor` (`column "source" does not exist`): the table already exists from the baseline, so 069's `CREATE TABLE IF NOT EXISTS` is skipped and the rest of the file references a column the old shape lacks. `DROP TABLE galgame_contributor CASCADE` and re-run; everything applies.
> 3. `-only 005 006 012 015` (one at a time). **005 is the post-OAuth cleanup and it DROPs columns** (`vndb_id`, `resource_update_time`, …) that later migrations restore, so running it last leaves the schema behind production. Delete their rows from `_migrations` and re-run: `-only 018`, `022`, `023`, `079`, `092`.
>
> （Claude 2026-09-18 复核补充：005 还会 `DROP TABLE galgame_contributor`，所以重跑清单里要加 `069`；`cmd/migrate` 要求 URL 形式的 DSN。按这个顺序建出的库，列、索引、触发器、函数与 dev 库逐条一致。）
>
> The migrate runner reads `KUN_DATABASE_URL` through `godotenv.Load()`, which does **not** override an exported var — so `export KUN_DATABASE_URL=<throwaway>` is enough to keep it off the dev database. Confirm it took effect by counting tables in the throwaway before trusting it.
