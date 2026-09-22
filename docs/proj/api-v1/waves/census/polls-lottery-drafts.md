# 普查 · 话题子资源：投票 / 抽奖 / 草稿

> 只读普查，代码阅读，未跑任何 SQL、未起任何服务。写于 2026-09-22。
> 范围：`topic_poll*`（投票）、`topic_lottery*`（抽奖小程序）、`topic_draft`（话题草稿）。
> 不含：回复、评论、reaction / favorite / upvote（W4 已迁）、galgame 投稿草稿（`/api/galgame/drafts`、`/api/galgame/:gid/draft` 属于 submission 域，同名不同物）。

**端点总数 21**：投票 6、抽奖 11、草稿 4。

---

## 0. 三条贯穿全域的结论（先说，后面不再重复）

**0.1 路径参数 `:tid` 在 21 个端点里被完全忽略。** 所有 17 个 topic 子资源端点都挂在 `/api/topic/:tid/...` 下，但没有任何一个 handler 读过 `c.Params("tid")`。真正的标识符来自 body 的 `topic_id` / `poll_id` / `lottery_id`，或 query 的 `poll_id` / `lottery_id` / `topic_id`。于是 `POST /api/topic/0/lottery/claim` 带 `{"lottery_id": 42}` 与 `POST /api/topic/99999/lottery/claim` 完全等价——W4 普查在回复 reaction 上记过同一个病（「`:tid` 被忽略，网页用 `/topic/0/reply/reaction`」），这里是同一个病的第二个患处，而且患处多得多。v1 的 K1 要求路径参数名与响应字段名一致，这一族必须整体改成 `/topics/{topic_id}/polls/{poll_id}` 形状，标识符只从路径来。

**0.2 读面完全不查话题可见性。** 四个读端点 `GET /poll/topic`、`GET /poll/log`、`GET /lottery/topic`、`GET /lottery/entrants` 都只拿一个 id 直接查表，一次都没调 `requireTopicRead`（`internal/topic/service/read_decision.go:15`）。写面只有两个调了：`PollService.Vote`（`poll_service.go:182`）和 `LotteryService.Enter`（`lottery_participation.go:29`）。其余 15 个端点对 `topic.Status`、`topic.AccessScope`、access grant 一概不看。后果：任何匿名访客枚举 `topic_id` 就能读出**隐藏话题（`status=1`）、仅登录可见、仅指定角色可见、仅指定用户可见**话题里的投票题面、投票结果、参与者名单与中奖名单。这是本域最严重的一条。

**0.3 权限闸用对了 helper，Bearer 是安全的。** 七个权限点全部走 `user.Can(perm.X)`（`middleware/auth.go:53`，内部 `!u.viaBearer && perm.CanUser(...)`），没有任何一处直接用 `perm.CanUser` / `role.Can*` 打 `user.Roles`。因此 App 的 Bearer 请求在这一族里永远拿不到 staff 能力：一个 Bearer 用户永远 `PollCreateAny=false`，只能在自己的话题上建投票、只能删自己建的投票。`docs/proj/app-direct-api.md` 里没有提到投票 / 抽奖 / 草稿的任何一个端点——App 目前不用它们，但它们都挂在 `authed` 组下，Bearer 是能打通的。

权限键（`pkg/perm/perm.go:32-39`，三处镜像之一）：

| 键 | 用在 |
|---|---|
| `poll.create_any` | `CreatePoll`（`poll_handler.go:33`） |
| `poll.edit_any` | `UpdatePoll`（`poll_handler.go:51`） |
| `poll.delete_any` | `DeletePoll`（`poll_handler.go:106`） |
| `poll.view_restricted` | `GetPollsByTopic` / `GetVoteLog`（`poll_service.go:152`、`:413`） |
| `lottery.create_any` | `CreateLottery`（`lottery_handler.go:61`） |
| `lottery.manage_any` | `UpdateLottery` / `DeleteLottery` / `Draw` / `Cancel` / `SetFulfillment`（`lottery_handler.go:79,99,151,169,208`） |
| `lottery.view_restricted` | `GetEntrants`（`lottery_mapper.go:30`） |

草稿一个权限键都没有，纯 owner 判定。

---

## 1. 投票（Poll）· 6 个端点

### 1.1 `POST /api/topic/:tid/poll` — 创建投票

- handler `internal/topic/handler/poll_handler.go:22`；service `PollService.CreatePoll`（`internal/topic/service/poll_service.go:58`）；repo `PollRepository.CountByTopicID:35` / `CreatePoll:115` / `CreatePollOption:119` / `TouchTopicStatusUpdateTime:123`。
- **请求体** `dto.CreatePollRequest`（`internal/topic/dto/poll_dto.go:9`），全部 snake_case：
  - `topic_id` int，必填 ≥1（**路径里已经有 tid，这里又要一遍**）
  - `title` string 1..100 必填
  - `description` string ≤500，可空
  - `type` string 必填 `single|multiple`
  - `min_choice` int `min=1`、`max_choice` int `min=1` —— **go-playground 的 `min` 对 int 是「值不小于」，零值 0 会失败，所以这两个字段实际是必填的，但 DTO 里没写 `required`，OpenAPI 上会被读成可选**
  - `deadline` `*string`，可空；服务端 `time.Parse(time.RFC3339, *req.Deadline)`，**解析失败时 `err` 被丢弃、deadline 静默变成 nil**（`poll_service.go:77-82`）
  - `result_visibility` 必填 `always|after_vote|after_deadline`
  - `is_anonymous` bool、`can_change_vote` bool
  - `options` `[]PollOptionInput` 必填 2..20
- **响应**：`response.OKMessage(c, "投票创建成功")` → `{"code":0,"message":"投票创建成功"}`，**没有 data**。网页却写成 `kunFetch<TopicPoll>`（`apps/web/app/composables/topic/usePoll.ts:12`）——手写类型与实际返回不符，只是没人用返回值所以没爆。创建后网页重新 `getPoll()` 拉全量。
- **错误路径**：
  - 话题不存在 → 404 / `233` / `"未找到该话题"`
  - 不是话题作者且无 `poll.create_any` → 403 / `233`
  - 本话题投票数 ≥ `MaxPollsPerTopic = 30`（`internal/constants/topic.go:44`）→ 400 / `233`。注意 `count, _ :=` **吞掉了错误**：DB 出错时 `count` 是 0，上限检查静默通过
  - trust gate `deny` → 422 / `233` / `"内容包含违禁词，无法发布"`（`internal/trust/gate/compose.go:9`）。`KUN_TRUST_CHECK_ENABLED` 默认 false，默认 fail-open
  - 事务失败 → 500 / `233` / `"创建投票失败"`
- **鉴权**：`required`。匿名 → `MustGetUser` 返回 `ErrAuthExpired()` = 401 / `205`。封禁用户不拦（这一族没有任何一处查封禁）。
- **可见性**：只查「你是不是话题作者」。**不查 `topic.Status`**，所以可以在自己被隐藏的话题上建投票；`poll.create_any` 的版主可以在任何 access scope 的话题上建投票（连 `requireTopicRead` 都不走）。
- **BUG · `options` 缺 `dive`，每个选项的文本完全没校验。** `Options []PollOptionInput` 的 tag 是 `required,min=2,max=20`（`poll_dto.go:20`），没有 `dive`。go-playground/validator v10.30.3 在 `traverseField` 里只对 `reflect.Struct` 置 `isNestedStruct`，slice 元素**必须**靠 `dive` 才会递归（对照：同文件 `CreateLotteryRequest.Prizes` 写的是 `required,min=1,max=10,dive`，作者是知道要写 `dive` 的）。后果两条：
  1. 空字符串选项能入库；
  2. 超过 100 字的选项打到 `topic_poll_option.text varchar(100)`，Postgres 22001 → 事务回滚 → **500「创建投票失败」，而这里该是 400 + `TOO_LONG`**。
- 副作用：成功后顶帖（`TouchTopicStatusUpdateTime`，3 个月 `BumpCutoff` 门槛），并 `ScanBg` 送 trust 异步扫描（`subject_kind = forum_topic_poll`）。

### 1.2 `PUT /api/topic/:tid/poll` — 更新投票

- handler `poll_handler.go:40`；service `PollService.UpdatePoll`（`poll_service.go:240`）；repo `FindByID:23` / `UpdatePollFields:155` / `FindOptionsByIDs:162` / `CreatePollOption:119` / `UpdateOptionText:171` / `DeleteOptionsByIDs:176`。
- **请求体** `dto.UpdatePollRequest`（`poll_dto.go:23`）：`poll_id` 必填 ≥1，其余标量与 create 同（**全部必填、全量覆盖，这是一个伪装成 PUT 的整体替换**），外加 `options: {add[], update[{option_id,text}], delete[int]}`。
- **响应**：`OKMessage("投票更新成功")`，无 data。
- **错误路径**：投票不存在 404；话题不存在 404；不是**话题作者**且无 `poll.edit_any` → 403；有选项增删改但 `poll.can_change_vote == false` → 400「本投票结果不可修改」；要改的选项 id 不存在 → 400；选项已有票且要改文本或删除 → 400；trust deny → 422；事务失败 → 500。
- **BUG · 整个 `options` 子树一条校验都没跑。** `Options PollOptionsUpdate` 字段上**没有任何 `validate` tag**（`poll_dto.go:34`）。validator 走到这个字段时 `ct == nil || !ct.hasTag` 直接 return，于是 `PollOptionsUpdate` 里的 `Add []PollOptionInput`、`Update []PollOptionUpdateInput`（本身也没 tag）连带元素上的 `text 1..100`、`option_id required,min=1` **全部失效**。结果：更新路径可以加空选项、加超长选项（→ 500）、传 `option_id: 0`（`UpdateOptionText` 更新 0 行，静默无效）。
- **BUG · 可以在投票已经开始后把匿名投票改成实名。** `scalarFields` 无条件写 `is_anonymous`、`result_visibility`、`can_change_vote`、`deadline`、`type`（`poll_service.go:265-275`），且当 `totalOptionOps == 0` 时**连 `can_change_vote` 的门都不过**（`poll_service.go:293-303`：只有有选项操作才检查）。于是作者可以：收了一堆匿名票 → `is_anonymous: false` → `GET /poll/log` 把每个人投了什么全列出来。用户在「匿名」承诺下投的票被追溯性公开。这是隐私事故级别，不是洁癖。
- **BUG · `type` 可以在有票之后从 `multiple` 改成 `single`**，此时已经投了 3 个选项的人在库里仍然有 3 行，读面照常统计，只是新投的人被限制成 1 个。数据与声明的语义不一致。
- **可见性**：不查 `topic.Status`、不查 access scope。
- **权限模型自相矛盾**：create / update 认**话题作者**（`topic.UserID != userID`），delete 认**投票作者**（`poll.UserID != userID`，见 1.3）。如果一个带 `poll.create_any` 的版主在你的话题上建了投票，你（话题作者）改得了、删不掉。

### 1.3 `DELETE /api/topic/:tid/poll` — 删除投票

- handler `poll_handler.go:93`（**匿名内联 struct，不在 dto 包里**）；service `PollService.DeletePoll`（`poll_service.go:375`）；repo `DeletePollCascade:145`。
- **请求**：query `poll_id` int 必填 ≥1。DELETE 带 query 参数而不是路径段。
- **响应**：`OKMessage("投票已删除")`，无 data。规范要 `204` 无 body。
- **错误**：投票不存在 404；`poll.UserID != userID` 且无 `poll.delete_any` → 403；事务失败 → 500。
- 级联：`DeletePollCascade` 手动删 `topic_poll_vote` → `topic_poll_option` → `topic_poll`。**它没有回退任何 `vote_count`**——正常情况下选项跟着一起删所以无所谓，但配合 1.4 的跨投票投票 BUG 就会留下永久污染（见下）。
- 不查话题、不查可见性。

### 1.4 `POST /api/topic/:tid/poll/vote` — 投票

- handler `poll_handler.go:74`（第 79 行有一句死代码 `_ = user`）；service `PollService.Vote`（`poll_service.go:166`）；repo `HasUserVoted:55` / `FindUserVoteOptionIDs:47` / `DeleteUserVotes:129` / `CreateVote:139` / `AdjustOptionVoteCount:134`。
- **请求体**：`poll_id` int 必填 ≥1；`option_id_array` `[]int` 必填、长度 ≥1。
- **响应**：`OKMessage("投票成功")`，无 data。客户端拿不到新票数，必须重拉。
- **错误**：投票不存在 404；话题不存在 404；`requireTopicRead` 不过 → 404；`poll.Status == "closed"` → 400（**见 1.7，这是死代码**）；过截止 → 400；单选而选了 ≠1 个 → 400；多选低于 `min_choice` / 高于 `max_choice` → 400；已投过且 `can_change_vote == false` → 400；事务失败 → 500「投票失败」。
- **鉴权**：`required`。是本域唯一一个**读了话题可见性**的投票端点。
- **BUG（本域第二严重）· 从不校验 `option_id` 属不属于这个 `poll_id`。** `CreateVote(tx, req.PollID, optionID, userID)`（`poll_service.go:224`）直接把客户端给的 id 当选项 id 用，随后 `AdjustOptionVoteCount(tx, optionID, +1)`（`:227`）给**那个选项**加一票。库里唯一挡住它的是 `topic_poll_vote_option_id_fkey → topic_poll_option(id)`（`migrations/000_baseline.up.sql:4973`，迁移 019 只删了指向 `public."user"` 的外键，这条还在），也就是「这个选项 id 在全站存在」即可，**没有任何 (poll_id, option_id) 的复合约束**。攻击面：
  - 攻击者在自己的话题上建一个自己的投票 P，然后 `POST vote {poll_id: P, option_id_array: [受害投票的选项 id]}` → 受害选项 `vote_count +1`，受害投票的读面直接显示被抬高的票数；
  - 唯一索引是 `(poll_id, option_id, user_id)`，所以同一个宿主投票里同一个受害选项只能 +1，但攻击者可以无限建宿主投票（每话题 30 个 × 不限话题数），每个宿主 +1；
  - `can_change_vote` 为真时改投会先 `-1` 再 `+1`，所以也能**减**任意选项的票数（先投受害选项 A、再改投受害选项 B → A −1、B +1）；
  - 删掉宿主投票（1.3）不回退 `vote_count`，污染永久留下。
  - 顺带：同一请求里传重复 id（`[5,5]`）会撞唯一索引 → 事务回滚 → **500 而不是 400**。
- **BUG · 计数与行数会漂移**，同 W4 在 reaction 上记的那条：`AdjustOptionVoteCount` 是无条件 `vote_count + ?`，而不是「真的插入/删除了一行才改」。这里插入有唯一索引兜底，比 W4 好一点，但删改路径（`DeleteUserVotes` 一把删完再逐个 `-1`，`poll_service.go:211-221`）用的是**事务开始前**读出来的 `oldOptionIDs`（`FindUserVoteOptionIDs` 在事务外，`:212` 实际在事务内但用的是 `s.pollRepo` 的非事务 DB 句柄），并发改投会重复扣。
- 没有幂等键。这是个可重放的状态写。

### 1.5 `GET /api/topic/:tid/poll/topic` — 按话题列投票

- handler `poll_handler.go:58`；service `GetPollsByTopic`（`poll_service.go:138`）→ `buildPollResponse`（`internal/topic/service/mapper.go:15`）；repo `FindByTopicID:29` / `FindOptionsByPollID:41` / `HasUserVoted:55` / `FindUserVoteOptionIDs:47` / `FindDistinctVoterIDs:63` / `CountDistinctVoters:73` / `CountTotalVotes:82`。
- **请求**：query `topic_id` int 必填 ≥1。
- **响应**：`response.OK(c, polls)` → `{"code":0,"message":"成功","data":[TopicPollResponse]}`。`dto.TopicPollResponse`（`poll_dto.go:70`）：
  `id`、`title`、`description`、`min_choice`、`max_choice`、`deadline`（`*time.Time`，可 null）、`type`、`status`、`result_visibility`、`is_anonymous`、`can_change_vote`、`topic_id`、`created`、`updated`、`user`（`KunUser{id,name,avatar}`）、`option`（**单数键名装复数数组**）、`has_voted`、`voters`（最多 5 人）、`voters_count`、`vote_count`（`*int`，不可见时 null）；每个 option：`id`、`text`、`vote_count`（`*int`，不可见时 null）、`is_voted`。
- **鉴权**：`optional`。匿名时 `userID = 0`、`canModerate = false`。
- **结果可见性** `canViewResults`（`mapper.go:91`）：作者或 `poll.view_restricted` → 恒可见；否则按 `result_visibility`：`always` 恒真、`after_vote` 看 `hasVoted`、`after_deadline` 看「status==closed 或过了 deadline」。不可见时 `vote_count` 全部下发 `null`、`voters` 空、`voters_count` 为 0。**`is_voted` 不受这个闸管**——匿名时 `userID=0`，`FindUserVoteOptionIDs(pollID, 0)` 查不到行，所以匿名安全；登录用户看到的是自己的选择，正确。
- **BUG · 匿名读 `HasUserVoted(pollID, 0)`**（`mapper.go:17`）：`user_id = 0` 的行理论上不存在，但这是一次每投票一次的无谓查询，且如果哪天有脏数据 `user_id=0`，所有匿名读者都会变成「已投票」从而解锁 `after_vote` 的结果。
- **可见性**：**完全不查话题**。隐藏 / 受限话题的投票题面、票数、前 5 名投票人对匿名公开。
- 封禁作者：`buildPollResponse` 在 `creatorU` 不可渲染时返回 `ok=false`，整条投票从列表里消失（`mapper.go:73-75`）。**注意它先 `Placeholder(poll.UserID)` 再判 `IsRenderable`**，而 `Placeholder` 的 `Status` 是零值 0 → 恒可渲染，所以只有 OAuth 真的返回了「封禁」状态才会藏；OAuth 批量查失败时全体放行。
- `FindDistinctVoterIDs(poll.ID, 5)` 是 `Distinct("user_id").Limit(5)` **没有 ORDER BY**（`poll_repo.go:63`）——展示哪 5 个人由 Postgres 心情决定，同一个投票刷新两次可能不同。
- `FindByTopicID` 是 `Order("created DESC")`，**无 id 决胜键**（`poll_repo.go:31`）。

### 1.6 `GET /api/topic/:tid/poll/log` — 投票流水

- handler `poll_handler.go:113`；service `GetVoteLog`（`poll_service.go:399`）；repo `FindVoteLogs:97`。
- **请求**：query `poll_id` ≥1、`page` `min=1`、`limit` `min=1,max=50`。`page` / `limit` 没有默认值，**int 零值会被 `min=1` 判失败**，所以两个都是事实必填（DTO 上没写 `required`）。
- **响应**：`response.OK(c, fiber.Map{"logs": entries, "total": total})` → `{"code":0,"message":"成功","data":{"logs":[...],"total":N}}`。**这是全站唯一一个手搓的分页信封，既不是 `response.Paginated` 的 `{items,total}` 也不是游标。** 每项 `dto.PollVoteLogEntry`：`id`、`created`、`user`、`option`（**是选项文本字符串，不是 option id**）。
- **错误**：投票不存在 404；`FindVoteLogs` 出错 500。
- **BUG · 权限不足时返回 200 + 空数组，而不是 403/404。** `poll_service.go:417`：`if !canViewResults(...) || poll.IsAnonymous { return []dto.PollVoteLogEntry{}, 0, nil }`。客户端无法区分「这个投票没人投」和「你没资格看」，正是规范 §2 K7 要根除的「兜底吞掉」。
- **BUG · `total` 与 `logs` 长度谈的不是同一件事。** `total` 来自 `FindVoteLogs` 里那句 `r.db.Model(...).Count(&total)`（`poll_repo.go:101`，**返回值被丢弃，出错时 total 静默为 0**），数的是全部行；而 `entries` 会跳过封禁用户（`poll_service.go:432`）。分页器按 `total` 画页数，翻到后面是空页。与 memory 里记过的 catalog「facet 与水合行双向不一致」同型。
- **BUG · `ORDER BY v.created DESC` 无 id 决胜键 + OFFSET 分页**（`poll_repo.go:108-110`）：同秒的票在翻页边界会重复或漏掉。W4 验收里被变异测试专门钉过的同一个洞。
- **可见性**：不查话题。隐藏话题的实名投票流水对任何登录用户（只要 `result_visibility=always`）公开。

### 1.7 `topic_poll.status` 是一条死轴

`status` 在三处被**读**（`poll_service.go:186` 拦投票、`mapper.go:95` 判结果可见、`apps/web/app/components/topic/poll/List.vue:27`），但整棵 Go 树里**没有一处写它**——没有「关闭投票」端点，`sweepClosedPolls` 也只写 `notification_sent` 不写 `status`（`lottery_repo.go:283-297`）。所以除非库里有 Node 时代的遗留行，`status` 恒为 `'open'`，上面三处判断全是死代码。v1 要么给它一个 `PATCH .../state`，要么直接删掉这个字段。**需要生产行数确认**（见 §6）。

---

## 2. 抽奖（Lottery）· 11 个端点

`KUN_LOTTERY_CODE_KEY` 与「兑换码绝不进 DTO」两条，先给核验结论。

### 2.0 memory 两条断言的核验

**断言一「prod 必须设 `KUN_LOTTERY_CODE_KEY` 否则兑换码奖品拒发」—— 成立，且比记的更温和。**
`pkg/config/config.go:312` 读 `KUN_LOTTERY_CODE_KEY`（默认空）→ `secretbox.New`（`pkg/secretbox/*.go:27`）对空串返回 `nil, nil`，`Box.Enabled()` 为 false（`:49`）。`app.go:442-447` 在 key 无效时 `slog.Error` + 禁用托管，未设置时 `slog.Warn`，**服务照常启动**。真正的拒绝点在 `LotteryService.validateShape`（`lottery_service.go:226-228`）：`delivery == "code"` 且 `!s.box.Enabled()` → 400「本站尚未配置兑换码托管密钥…」。抽奖的其余交付方式（`manual` / `point`）不受影响。
**补充一条 memory 没记的危险**：密钥是 AES-256-GCM 的**唯一**解封材料，库里存的是 base64 密文。**换 key 或丢 key = 所有历史兑换码永久不可读**，`ClaimCode` 会走到 `slog.Error("兑换码解密失败")` → 500（`lottery_participation.go:159-163`）。key 轮换没有任何迁移路径。

**断言二「兑换码绝不进 DTO，否则 SSR 泄漏」—— 成立，四道防线都还在，而且是有意加固过的。**
1. `model.TopicLotteryCode`（`internal/topic/model/lottery.go:88-97`）**整个 struct 没有一个 json tag**，并带注释说明原因；
2. `TopicLotteryEntry.CodeID` 的 tag 是 `json:"-"`（`lottery.go:108`），列表里只下发布尔 `my_code_ready`（`lottery_mapper.go:249`）；
3. 唯一返回明文的路径是 `POST /lottery/claim`（`lottery_handler.go:179`），handler 与 service 各有一段注释钉死「必须是 POST，绝不能变 GET」（`lottery_handler.go:176-178`、`lottery_participation.go:131-133`），网页侧也有对应注释（`apps/web/app/composables/topic/useLottery.ts:208-209`）；
4. 迁移 083 给 `topic_lottery_code` 打了 `COMMENT ON TABLE`，把「永远不要 join 进列表或详情响应」写在了库里（`migrations/083_topic_lottery.up.sql:86-87`）。
`rg` 全树确认没有第二处读 `Secret` / `code.Secret` 的地方。**两条断言现在都成立，且 v1 迁移时必须原样保留这四道。**

### 2.1 `POST /api/topic/:tid/lottery` — 创建抽奖

- handler `lottery_handler.go:50`；service `CreateLottery`（`lottery_service.go:107`）→ `eligibleToCreate:80` → `validateShape:194` → `writePrizes:292` → `touchTopic:339`；repo `CountByTopicID:33` / `Create:39` / `CreatePrize:70` / `CreateCode:81`。
- **请求体** `dto.CreateLotteryRequest`（`internal/topic/dto/lottery_dto.go:230`）：`topic_id`≥1、`title` 1..100、`description` ≤1000、`entry_mode` `signup|reply|floor`、`floor_rule` ≤200、`draw_mode` `deadline|manual|threshold`、`draw_threshold` 0..100000、`deadline` `*string`（RFC3339，**解析失败静默 nil**，`lottery_mapper.go:291-300`）、`min_account_age_days` 0..3650、`min_moemoepoint` 0..1000000、`show_entrants` bool、`prizes` 1..10 **带 `dive`**（所以奖项字段真的被校验了）。
  每个 prize：`name` 1..100、`description` ≤500、`image_hashes` ≤9 每项 1..128、`nsfw_hashes` ≤9、`delivery` `code|manual|point`、`point_mode` 可空 `fixed|split|random`、`point_amount` 0..10000、`slots` 1..500、`codes` ≤500 每项 1..200。
- **响应**：`OKMessage("抽奖创建成功")`，无 data，无新建 id。
- **错误路径**（全部 `233` + 中文句子）：话题不存在 404；非话题作者且无 `lottery.create_any` 403；反诈门不过 403（注册 <30 天**且** 萌萌点 <100，`internal/constants/topic.go:30-31`）；本话题抽奖数 ≥10 → 400（`count, _ :=` **又一次吞错**）；`validateShape` 的 11 条 400；trust deny 422；事务失败 500。
- **`validateShape` 的 11 条**（`lottery_service.go:194-277`）：奖项为空 / >10；单奖名额 >500；单奖图片 >9；`nsfw_hashes` 里有不在 `image_hashes` 里的；`delivery=code` 且密钥未配；`delivery=code` 且 `len(codes) != slots`；`delivery=point` 且 `point_amount<=0`；pool 模式（split/random）且 `point_amount < slots`；总名额 >500（**复用了 `MaxSlotsPerPrize` 这个常量，名字说的是单奖上限，这里当总上限用**）；总萌萌点 >100000；`draw_mode=deadline` 而无 deadline / deadline 已过；`draw_mode=threshold` 而阈值 < 总名额；`entry_mode=floor` 且 `draw_mode=threshold`；楼层规则解析失败。
- **BUG（本域最严重）· 萌萌点奖项是凭空铸币，发起人一分钱不扣。** 全树 `rg 'moemoepoint\.'` 在抽奖里只有 `afterDraw` 的 `moemoepoint.Award(winner, +N, ReasonContentApproved, ...)`（`lottery_draw.go:485-488`），**没有任何一处对发起人做 `Award(author, -N, ...)`**，创建时也不校验发起人余额（只校验参与者的 `min_moemoepoint`）。所以任何越过「注册满 30 天 **或** 有 100 萌萌点」这道极低门槛的账号，都能一次性造出 100000 萌萌点定向送人（每话题 10 个抽奖 × 不限话题数 → 无上限）。对照 W4：推（upvote）是花 10 点给作者 5 点，是真扣的。这一条是产品决策还是漏写，需要用户裁决；但按 C3「余额单源在 OAuth」，论坛这边确实是直接向 OAuth 发放，OAuth 不会拒。
- **BUG · 奖品图片不在 reference-ping 的扫描面内，会被图床 GC 掉。** `topic_lottery_prize.image_hashes` 是 **jsonb**，存的是**裸 hash**（`migrations/086_*.up.sql`；`lottery_mapper.go:269` 用 `imageclient.ResolveURL(cdnBase, hash, "")` 反推 URL 即可证明）。而 `cron.RunReferencePing` 的 `collectContentImageHashes`（`internal/infrastructure/cron/reference_ping.go:55-88`）只扫 `data_type IN ('text','character varying','character')` 的列，并且只认正则 `/image/([0-9a-f]{64})`。jsonb 列既不在类型过滤里、裸 hash 也不带 `/image/` 前缀，**两重都不命中**。草稿的 `cover_images` 是 TEXT 存 token 所以安全（迁移 042 的注释专门解释了这个设计），抽奖奖品图把这条设计整个丢了。正是 memory「content image token / GC 的 reference-ping 只认 token」那条记的坑，在新功能上复发。
- **安全设计（保留）**：`entry_mode=floor` 时故意把 seed/seedHash 置空（`lottery_service.go:144-149`），注释说明「楼层抽奖没有随机性，发一个假的承诺等于宣称一个不存在的公平性证明」。
- **可见性**：只查话题作者身份，不查 `topic.Status` / access scope。

### 2.2 `PUT /api/topic/:tid/lottery` — 更新抽奖

- handler `lottery_handler.go:68`；service `UpdateLottery`（`lottery_service.go:345`）；repo `FindByID:21` / `UpdateFields:43` / `DeletePrizes:74` / `CreatePrize:70` / `CreateCode:81`。
- **请求体** `dto.UpdateLotteryRequest`（`lottery_dto.go:251`）：`lottery_id` + 与 create 相同的全量标量；`prizes` 是 `max=10,dive`（**没有 `min=1`**，空数组 = 只改标量）。DTO 上方的注释解释了「奖项只在零参与时可改，所以整包传、不 diff」。
- **响应**：`OKMessage("抽奖更新成功")`，无 data。
- **错误**：抽奖不存在 404；非抽奖作者且无 `lottery.manage_any` 403；`status != open` → 400；传了奖项但 `entry_count > 0` → 400；`validateShape` 的各条 400；trust deny 422；事务失败 500。
- **BUG · 不传奖项时整个 `validateShape` 被跳过，可以把抽奖改成永远开不了奖的形状。** `rewritePrizes := len(req.Prizes) > 0`，`validateShape` 只在 `rewritePrizes` 时调用（`lottery_service.go:363-371`），但事务里**无条件**写 `draw_mode` / `draw_threshold` / `deadline`，`entry_count==0` 时还写 `entry_mode` / `floor_rule`（`:381-398`）。于是：
  - `draw_mode: "deadline"` + `deadline: null`（或一个解析失败的字符串）→ `ClaimDue` 的 `deadline IS NOT NULL` 永不命中（`lottery_repo.go:260`）→ **抽奖永远卡在 open，没有任何手动开奖以外的出路**；
  - `draw_mode: "threshold"` + `draw_threshold` 小于总名额 → create 时会被拦，update 时不会；
  - `entry_mode: "floor"` + 一个解析不了的 `floor_rule` → 到点扫描把它 flip 成 `drawing` → `pickFloorWinners` → `parseFloorRule` 失败 → `ReleaseDrawing` 回 `open` → **下一分钟再来一遍，每分钟一条 `slog.Warn`，无限循环**（`lottery_draw.go:223-243`）。
- **BUG · `deadline` 解析失败静默变 null**（同 1.1）：`parseLotteryDeadline`（`lottery_mapper.go:291`）`err != nil` 就 `return nil`，错误被丢弃。客户端传了个坏时间戳，服务端回 200，抽奖的截止时间被悄悄抹掉。
- **可见性**：不查话题。

### 2.3 `DELETE /api/topic/:tid/lottery` — 删除抽奖

- handler `lottery_handler.go:86`（匿名内联 struct）；service `DeleteLottery`（`lottery_service.go:425`）；repo `Delete:51`（靠 083 的 `ON DELETE CASCADE` 清 prize / code / entry）。
- **请求**：query `lottery_id` ≥1。**响应** `OKMessage("抽奖已删除")`，非 204。
- **错误**：不存在 404；非作者且无 `lottery.manage_any` 403；`status == drawn` 且非版主 → 400「已开奖的抽奖不能删除, 中奖者需要它来领取奖品」；事务失败 500。
- **BUG · 可以删一个正在开奖（`status == 'drawing'`）的抽奖。** 判断只挡 `drawn`。删除会 CASCADE 掉 `topic_lottery_entry`，而 `draw()` 的事务此刻正在写这些行 → 要么外键冲突 500、要么中奖名单刚写完就被整表删掉、而 `afterDraw` 的通知和萌萌点已经发了出去（它在事务**之后**跑，`lottery_draw.go:372`）。窗口小但真实。
- **BUG · 版主删掉已开奖的抽奖会连带删光所有托管兑换码**（CASCADE），而那些码可能已经被中奖者领走也可能没有，且**无法恢复**（明文只在 claim 那一刻存在过）。没有任何二次确认或软删。

### 2.4 `POST /api/topic/:tid/lottery/enter` — 参与

- handler `lottery_handler.go:106`；service `Enter`（`lottery_participation.go:17`）→ `entryBlocker:93`；repo `CreateEntry:187` / `SyncEntryCount:196` / `HasRepliedTo:207`。
- **请求体**：`{lottery_id}`。**响应** `OKMessage("参与成功, 祝您好运")`，无 data（客户端拿不到新的 `entry_count`，要重拉整个话题的抽奖列表）。
- **错误**：抽奖不存在 404；话题不存在 404；`requireTopicRead` 不过 404；`entryBlocker` 的 7 条（未登录 403 / 已结束 400 / 楼层抽奖无需报名 400 / 过截止 400 / 不能参加自己的 400 / 要求先回帖 400 / 萌萌点不足 403 / 注册天数不足 403）；事务失败 400。
- **BUG · 事务里任何错误都被翻译成「您可能已经参与过了」400**（`lottery_participation.go:45-47`）。唯一索引 `uq_topic_lottery_entry (lottery_id, user_id)` 撞了确实该是这句，但连接断了、磁盘满了、`SyncEntryCount` 失败也都是这句 400，真正的故障被伪装成用户错误。这是「未检查的错误静默返回」的变体。
- 设计得好的一点：`entryBlocker` 同时是按钮状态的来源和 POST 的守卫（`lottery_mapper.go:232-238` 把它的 `Message` 塞进 `enter_blocked`），注释说明了「两者不能漂移」。但见 2.11 的命名问题——它把中文句子当 API 字段下发。
- 没有幂等键。唯一索引让重放无害，但重放会得到 400 而不是 200。

### 2.5 `POST /api/topic/:tid/lottery/withdraw` — 退出

- handler `lottery_handler.go:123`；service `Withdraw`（`lottery_participation.go:70`）；repo `DeleteEntry:191`（带 `prize_id = 0` 条件，所以中奖者退不掉）/ `SyncEntryCount:196`。
- **请求体** `{lottery_id}`；**响应** `OKMessage("已退出抽奖")`。
- **错误**：不存在 404；`status != open` → 400；事务失败 500。
- **BUG · 退出路径完全不查话题可见性**（`Enter` 查了，`Withdraw` 没查）。不严重（只能删自己的行），但两个对称操作走两套规则。
- **BUG · 不查 deadline**：已过截止但扫描还没跑到的窗口里可以退出。
- **BUG · 删了 0 行也回 200**：没参与过的人调用 withdraw 得到「已退出抽奖」。按 K16 这其实是对的（撤销未置位的槽位是幂等 200），但这里是 `OKMessage` 不是状态快照。

### 2.6 `POST /api/topic/:tid/lottery/draw` — 立即开奖

- handler `lottery_handler.go:140`；service `DrawNow`（`lottery_draw.go:247`）→ `draw:281`；repo `FindByID` / 裸 SQL `UPDATE ... SET status='drawing' WHERE id=? AND status='open'` / `ReleaseDrawing:273`。
- **请求体** `{lottery_id}`；**响应** `OKMessage("开奖完成")`，无 data（拿不到中奖名单）。
- **错误**：不存在 404；非作者且无 `lottery.manage_any` 403；`status != open` → 400「该抽奖已经开过奖了」（措辞对 `cancelled` 是错的）；抢锁失败（`RowsAffected == 0`）→ 400「该抽奖正在开奖中」；`draw` 内部失败 → 透传，其中事务错误是 **500 `"开奖失败: " + txErr.Error()`——把原始 Postgres 错误字符串直接拼进响应体给终端用户**，违反 K8 第 5 条（500 的 detail 不得泄漏内部信息）。
- **做对了的**：状态 flip 就是锁，手动按钮和每分钟扫描用同一把（`lottery_draw.go:259-266`，`lottery_repo.go:251-270` 的 `FOR UPDATE SKIP LOCKED`）；`drawStaleAfter = 15min` 的僵死恢复有注释论证为什么安全。
- **BUG · `afterDraw` 在事务之外，进程在这个缝里死掉就永远补不上。** `draw()` 提交事务后调 `s.afterDraw(...)`（`lottery_draw.go:372`），里面发中奖/未中奖通知并 `moemoepoint.Award`。事务已经把 `status='drawn'`、`point_awarded=N` 写死，所以重启后不会重跑。结果：库里写着「这人赢了 500 萌萌点」、前端照此显示，**而 OAuth 那边一分没加**。
- **BUG · `moemoepoint.Award` 是 fire-and-forget 且失败只记日志**（`internal/moemoepoint/pusher.go:46-72`，`go func()` + `slog.Warn("...best-effort, skipped")`）。OAuth 抖一下，中奖者的萌萌点就永久少了，没有补偿队列。幂等键 `kungal:lottery_won:{lotteryID}_{userID}` 倒是稳定的，所以**有**人工重放的可能，但没有自动重放。
- **命名不一致**：`ref` 用的是 `fmt.Sprintf("lottery_%d", lottery.ID)`（下划线），而 `moemoepoint.Ref(kind, id)` 的约定是 `"topic_upvote:1207"`（冒号）。抽奖是全站唯一一个不走 `Ref()` 的调用点。
- **`draw()` 里 `slots[i]` 与 `winners[i]` 的对齐是隐式的**（`lottery_draw.go:311-312`、`stampPointPayouts:124`）：靠 `takeLowestRanked` 截断到 `len(slots)`、靠 `parseFloorRule` 保证楼层数 == 总名额。两个不变量都成立，但没有断言，改动一处就会静默错配奖项。
- **楼层 + `point_mode=random` 时 seed 是空串**（2.1 的设计后果）：`pointWeight(seed="", prizeID, userID)` 仍然确定，但任何人都能预先算出谁分到多少。低危，但「承诺-揭示」在这条路径上是假的。

### 2.7 `POST /api/topic/:tid/lottery/cancel` — 取消

- handler `lottery_handler.go:158`；service `Cancel`（`lottery_participation.go:51`）；repo `UpdateFields:43`。
- **请求体** `{lottery_id}`；**响应** `OKMessage("抽奖已取消")`。
- **错误**：不存在 404；非作者且无 `lottery.manage_any` 403；`status != open` → 400；更新失败 500。
- **BUG（竞态）· `Cancel` 的 UPDATE 没有状态守卫。** 它先读 `status`，判断是 `open`，然后 `UpdateFields(db, lotteryID, {"status": "cancelled"})` —— **按 id 无条件写**（`lottery_participation.go:62-66`）。对比 `DrawNow` 用的是 `WHERE id=? AND status='open'`。所以：读到 open → 扫描把它 flip 成 `drawing` → 开始发奖 → Cancel 写 `cancelled` → `draw()` 的事务最后写 `drawn`。谁后写谁赢，而中奖通知和萌萌点**已经发出去了**。修法只有一行：`WHERE id = ? AND status = 'open'` 并检查 `RowsAffected`。
- 取消后参与者不收到任何通知（`sweepDueLotteries` / `afterDraw` 都不管 cancelled）。

### 2.8 `POST /api/topic/:tid/lottery/claim` — 领取兑换码

- handler `lottery_handler.go:179`（带「绝不能变 GET」的注释）；service `ClaimCode`（`lottery_participation.go:134`）；repo `FindByID` / `FindEntry:145` / `FindCodeByID:139` / `UpdateEntryFields:203`；`secretbox.Box.Open`。
- **请求体** `{lottery_id}`；**响应** `response.OK(c, dto.LotteryClaimResponse{Code: code})` → `{"code":0,"message":"成功","data":{"code":"<明文>"}}`。**注意顶层信封的 `code` 是业务码 0，data 里的 `code` 是兑换码——同一个响应里 `code` 有两个含义**，v1 的禁用名清单正好禁了顶层 `code`，这里两个都得改（建议 `redemption_code`）。
- **错误**：抽奖不存在 404；不是中奖者（查不到 entry 或 `prize_id == 0`）→ 403；`code_id == 0` → 400「该奖项不是兑换码」；`status != drawn` → 400；`fulfillment == forfeited` → 400；`FindCodeByID` 失败 → 500；`code.ClaimedBy != userID` → 403；解密失败 → 500（日志里记了 `lottery_id` / `code_id`，不记明文，正确）。
- **鉴权**：`required`，凭证即主语。
- **BUG · 不查话题可见性**：话题被隐藏了中奖者也该能拿自己的码，所以这条大概是对的，但它是**无意的对**——没有任何注释说这是有意的。
- **BUG · 「只能揭示一次」是假的。** 网页的帮助文案写着「激活码由系统托管, **只有中奖者本人能揭示一次**」（`apps/web/app/components/topic/miniapp/registry.ts:87`），但 `ClaimCode` 每次调用都重新解密返回，`fulfillment` 只是从 `pending` 推到 `received`（`lottery_participation.go:164-168`），**之后再调还是照给**。而且那次 `UpdateEntryFields` 的返回值被 `_ =` 丢掉——状态没推进也照样返回码。要么改文案，要么真的做成一次性（后者需要「揭示后明文只在客户端」的设计）。
- **BUG · 这是一个会改状态的 POST，没有幂等键**（K12 要求所有 POST 支持）。
- 好的一面：`TakeCode`（`lottery_repo.go:118-137`）用 `UPDATE ... WHERE id=? AND claimed_by=0` 当锁，注释论证了两个并发领取不会拿到同一个码。

### 2.9 `PUT /api/topic/:tid/lottery/fulfillment` — 履约状态

- handler `lottery_handler.go:197`；service `SetFulfillment`（`lottery_participation.go:172`）；repo `FindByID` / `FindWinners:170` / `UpdateEntryFields:203`。
- **请求体** `dto.LotteryFulfillRequest`（`lottery_dto.go:273`）：`lottery_id`≥1、`entry_id`≥1、`fulfillment` `pending|shipped|received|forfeited`。
- **响应** `OKMessage("履约状态已更新")`，无 data。
- **错误**：抽奖不存在 404；`FindWinners` 失败 500；`entry_id` 不在这个抽奖的中奖名单里 404；既非作者、非版主、非中奖者本人 → 403；中奖者想设 `pending`/`shipped` → 403「中奖者只能确认收货或放弃奖品」；更新失败 500。
- **BUG · 没有状态机。** 任何允许的角色都能把任何状态改成任何状态，包括 `received → pending`、`forfeited → pending`。而 `forfeited` 对兑换码奖是**不可逆**的（`ClaimCode` 见 2.8 会拒），所以中奖者手滑点「放弃」之后，作者可以把它改回 `pending` 再让他领——也就是说这个「不可逆」其实取决于作者愿不愿意帮忙，而 UI 不会告诉任何人这件事。v1 该用 `PATCH` + `INVALID_STATE_TRANSITION`。
- **BUG · 不检查 `lottery.Status == drawn`**：理论上 `FindWinners` 只返回 `prize_id > 0` 的行，未开奖时为空，所以够不着；但这是靠巧合而不是靠检查。
- `FindWinners` 拉全部中奖者再在 Go 里线性找目标（`lottery_participation.go:181-193`），500 个名额时是 500 行换 1 行。

### 2.10 `GET /api/topic/:tid/lottery/topic` — 按话题列抽奖

- handler `lottery_handler.go:22`；service `GetLotteriesByTopic`（`lottery_mapper.go:58`）→ `buildLotteryResponse:143`；repo `FindByTopicID:27` / `FindPrizesForLotteries:61` / `FindWinnersForLotteries:177` / `CountCodesForLotteries:95` / `FindEntriesForUser:151`。
- **请求**：query `topic_id` ≥1。**另外读 `utils.IsSFW(c)`**（`lottery_handler.go:28`），也就是这个端点的结果取决于 `X-Kungal-Nsfw` 头或 `KUNGalgameSettings` cookie —— v1 §3 明确规定「偏好 cookie 不再是输入」，必须改成显式 query 参数。
- **响应** `[TopicLotteryResponse]`（`lottery_dto.go:323`），字段见 §2.11 的命名表。
- **鉴权** `optional`。**可见性：不查话题**（本域最严重那条的第三个患处）。
- **BUG · 封禁作者的抽奖照常显示，投票却会消失。** `buildLotteryResponse` 对作者只做 `if author.ID == 0 { Placeholder }`（`lottery_mapper.go:156-159`），**从不调 `IsRenderable`**；中奖者列表倒是过滤了（`:193`）。而 `buildPollResponse` 会因为作者不可渲染而整条丢弃（`mapper.go:73-75`）。同一个话题页上的两个小程序，对「作者被封」这件事给出相反的答案。
- **BUG · `enter_blocked` 是服务端产出的中文句子**（`lottery_mapper.go:234` 把 `appErr.Message` 直接塞进 DTO）。违反 §7「服务端不得产出给终端用户看的句子」，也是 App i18n 的直接障碍。v1 该下发 `{reason: "REPLY_REQUIRED", params: {...}}` 这样的结构化 token。
- **性能**：一个话题最多 10 个抽奖 × 最多 500 名额 = 最多 5000 个中奖者，全部 `Hydrate`（OAuth `/users/batch` 按 `pageSize` 分片，`userclient.go:152`）。话题详情页每次都打。`CountCodesForLotteries` 的注释说明了为什么按 prize 聚合而不是一抽奖一次查询——这条是做对了的。
- `FindByTopicID` `Order("created DESC")` **无 id 决胜键**。
- **`seed` 字段**：`buildLotteryResponse:228-230` 只在 `status == drawn` 时填 `resp.Seed`，其余时候是空串（不是 null）。承诺-揭示的半边，正确。但 DTO 的 json tag 是 `json:"seed"` 而 model 上是 `json:"-"`，同一个概念两套规则，靠人记住。

### 2.11 `GET /api/topic/:tid/lottery/entrants` — 参与者名单

- handler `lottery_handler.go:35`（匿名内联 struct）；service `GetEntrants`（`lottery_mapper.go:17`）；repo `FindEntries:164`。
- **请求**：query `lottery_id` ≥1。**响应** `[{user, reply_floor, created}]`。
- **错误**：抽奖不存在 404；`FindEntries` 失败 500。
- **BUG · `show_entrants == false` 时返回 200 + 空数组而不是 403**（`lottery_mapper.go:32-34`），同 1.6 的「静默吞掉」。
- **BUG · 完全没有分页。** `FindEntries` 是 `WHERE lottery_id = ? ORDER BY id ASC`，无 LIMIT。`draw_threshold` 上限 100000，参与人数没有任何上限。一个火爆抽奖的 `entrants` 请求会把全部参与者行 + 全部 `Hydrate` 拉一遍。v1 必须是游标集合。
- **可见性：不查话题。** 隐藏话题的参与者名单对匿名公开。

---

## 3. 话题草稿（Topic Draft）· 4 个端点

路由挂在 `api` 组上并用独立的 `topicDraftAuth := a.Authn.Auth()`，**且必须在 `/topic/:tid` 之前注册**，否则静态段 `draft` 会被参数路由吃成 `tid="draft"`（`internal/app/router.go:203-208`，注释记了这个坑）。

### 3.1 `POST /api/topic/draft` — 保存草稿

- handler `internal/topic/handler/draft_handler.go:24`；service `DraftService.Save`（`internal/topic/service/draft_service.go:25`）；repo `CountByUser:26` / `Create:32`。
- **请求体** `dto.SaveTopicDraftRequest`（`internal/topic/dto/draft_dto.go:5`）：`title` ≤233、`content` ≤100007、`category` 可空 `galgame|technique|others`、**`section`（单数键、数组值）** ≤3、`is_nsfw` bool、`cover_images` ≤9。**全部可选**。
- **响应**：`response.OK(c, id)` → `{"code":0,"message":"成功","data": 42}` —— data 是一个**裸整数**，不是对象。v1 的创建该回 201 + `Location` + 完整资源。
- **错误**：标题和正文都空白 → 400「草稿的标题和正文不能都为空」；`CountByUser` 失败 500；条数 ≥30 → 400；插入失败 500。
- **鉴权**：`required`，无权限键，纯 owner。
- **文档化的每人 30 条上限：服务端确实在执行。** `MaxDraftsPerUser = 30`（`draft_service.go:15`），`Save` 里 `count >= MaxDraftsPerUser` → 400（`:31-37`）。memory 里「手动保存、每人 30 条上限」两条都属实。
  - 但这是 check-then-insert，**没有事务、没有唯一约束、没有 `count(*) < 30` 的条件插入**，并发两个 save 可以都读到 29 然后都插入 → 31 条。低危。
  - 而且 `POST` **永远是新建，没有更新语义**：网页的「保存为草稿」每按一次就多一行（`apps/web/app/components/edit/topic/DraftModal.vue:33-48`，只有 `isSaving` 这个本地 flag 防重复点击）。写三稿就占 3/30。这解释了为什么需要 30 这个数，也意味着**30 条上限实际上是「保存 30 次」**。v1 该给草稿一个 `PUT /drafts/{draft_id}`。
- **BUG · 没有幂等键。** 这是创建用户内容的 POST，K12 要求必须携带。弱网重试 = 两份一模一样的草稿。
- **BUG · `section` 的取值不校验。** 只有 `max=3`，没有 `oneof` 也没有「section 必须属于 category」的一致性检查——而正式发帖那边有（v1 的 `SectionSlug` 封闭枚举 + 类目一致性，`internal/topic/apiv1/write_types.go:37-38`；旧 DTO `CreateTopicRequest.Sections` 至少有 `required,min=1`）。于是草稿里可以躺着一组发不出去的 section，用户载入后点发布才发现。
- **BUG · `cover_images` 的元素不校验长度**（`[]string` 只有 `max=9`，无 `dive`）。存进 TEXT 列，不会炸，但一个超长字符串会一直跟着草稿走。
- `Content` 过 `markdown.NormalizeStoredContent`（`draft_service.go:26`），与正式发帖同一条。**没有 trust gate**（草稿是私有的，合理，但没写下来）。

### 3.2 `GET /api/topic/draft` — 我的草稿列表

- handler `draft_handler.go:42`；service `List`（`draft_service.go:54`）；repo `ListByUser:36`。
- **请求**：无参数。**响应** `{"code":0,"message":"成功","data":[{id,title,summary,updated}]}`，**没有分页、没有 total**（上限 30 条，所以够用，但 v1 的集合必须声明一种分页风格）。
- `summary` 是 SQL 里的 `LEFT(content, 120)`（`draft_repo.go:39`）——**按字节还是按字符？** Postgres 的 `left()` 是按字符的，所以中文安全；但网页拿到后还要 `markdownToText` + `truncateRunes(…, 40)`（`DraftModal.vue:75-78`），也就是服务端截 120 字符、客户端再截 40 个 rune，两道截断。
- 排序 `Order("updated DESC")`，**无 id 决胜键**。
- **错误**：查询失败 500。**鉴权** `required`，`WHERE user_id = ?` 是唯一的隔离。

### 3.3 `GET /api/topic/draft/:id` — 草稿详情

- handler `draft_handler.go:55`；service `Get`（`draft_service.go:71`）；repo `GetByIDForUser:46`。
- **请求**：路径 `:id`，`strconv.Atoi` 失败 → 400「无效的草稿 ID」。**这是本域唯一一个真的用了路径参数的端点。**
- **响应** `dto.TopicDraftDetail`（`draft_dto.go:21`）：`id`、`title`、`content`（Markdown 源文）、`category`、**`section`**（单数键数组）、`is_nsfw`、`cover_images`（`/image/<hash>` token 原样）、`updated`。
- **错误**：`gorm.ErrRecordNotFound` → 404；其它 → 500。**别人的草稿也是 404**（`WHERE id = ? AND user_id = ?`），正确——不泄漏存在性。
- 做对了：`cover_images` 下发的是 token 而不是解析过的 Image 对象，因为这是编辑器要写回去的原料。v1 迁移时要保留这个「编辑语境下发源文」的区分（K13 的 `content_markdown` 正是这条）。

### 3.4 `DELETE /api/topic/draft/:id` — 删除草稿

- handler `draft_handler.go:73`；service `Delete`（`draft_service.go:91`）；repo `DeleteForUser:54`。
- **请求**：路径 `:id`。**响应** `OKMessage("草稿已删除")`，该是 204。
- **错误**：id 非数字 400；DB 失败 500；`RowsAffected == 0` → 404「草稿不存在」（别人的草稿同样 404）。这一条是本域里错误语义最干净的端点。
- 删除不做任何图片引用回收——依赖 reference-ping 的「不再出现在任何文本列里就不 ping」的自然过期，正确。

---

## 4. 与已迁移话题代码共用的东西

**共用的 model struct**（`internal/topic/model/topic.go`、`lottery.go`、`topic_draft.go`）：

- `model.Topic` —— 投票和抽奖都读它（`TopicRepository.FindByID`），并都写它的 `status_update_time`（顶帖）。v1 的 `Topic` repr 已经把这一列改名成 `bumped_at`。`model.BumpCutoff` 也共用（`poll_repo.go:125`、`lottery_service.go:341`）。
- `model.TopicAccessGrant` + `internal/topic/access`（`Snapshot` / `NeedsGrants` / `Allowed` / `CanRead`）—— 只有 `requireTopicRead` 的两个调用点在用；v1 那边是 `apiv1.visibleTopic`（`internal/topic/apiv1/visible.go:27`），**功能等价但多做了一步 `rejectUnrenderableAuthor`**。迁移时直接复用 `visibleTopic` 就能一次补上 §0.2 的全部窟窿。
- `model.StringSlice`（草稿的 `sections`）与 `model.ImageTokens`（草稿的 `cover_images`，`internal/topic/model/image_tokens.go:10`）—— `ImageTokens` 与 `model.Topic.CoverImages` 是同一个类型，reference-ping 靠它。
- `model.TopicPoll` 的 `BumpCutoff` / `topic_poll.notification_sent` —— 后者由抽奖的扫描器写（`lottery_repo.go:283`），也就是**投票的定时任务寄生在抽奖的 `LotteryDrawer.Run` 里**（`lottery_draw.go:207-221`，注释解释了原因）。迁移任何一边都会动到另一边。

**共用的 DTO**：`dto.KunUser{id,name,avatar}`（`internal/topic/dto/topic_dto.go:9`）—— 投票的 `user` / `voters` / 流水 `user`，抽奖的 `user` / 中奖者 `user` / 参与者 `user` 全用它。v1 的对应物是 `repr.UserRef`（`internal/apiv1/repr/user.go`，`{object,id,name,avatar}`）。**按 §5.2「不得复用旧 DTO」，这 8 个位置全部要换。**

**共用的 repository**：`repository.TopicRepository`（`FindByID`、`FindAccessGrants`）被 `PollService`、`LotteryService` 注入。`userRepo.StateRepository.FindByID`（读 `kungal_user_state.moemoepoint` 这个缓存视图）被抽奖的两处门槛用。`userclient.Client.Hydrate` / `User` / `CollectIDs` / `IsRenderable` / `Placeholder` 全域共用。

**共用的服务**：`gate.CheckService` / `gate.ScanService`（trust，subject kind `forum_topic_poll` / `forum_topic_lottery`，`internal/trust/gate/scan.go:16-17`）；`msgService.Notifier.EmitMany`（四种通知 `NotifyLotteryWon` / `NotifyLotteryClosed` / `NotifyLotteryExpired` / `NotifyPollClosed`）；`moemoepoint.Award`；`imageclient`（`ResolveURL` + `SetImageMetaResolver` 的逐图分级）；`secretbox.Box`。

**共用的下游**：`pkg/miniapp.Lookup`（`apps/api/pkg/miniapp/*.go`）—— 首页 / 版块 / 搜索 / 动态 / 话题详情五个面都靠它数「这个话题有没有投票 / 抽奖」，v1 的 `Topic.mini_apps` / `TopicSummary.mini_apps` 已经在用（`internal/topic/apiv1/detail.go:41`、`summary.go:66`）。**删任何一张表都要同步改它。**

**删号清理**（`internal/admin/repository/purge_repo.go`）：统计三个数（`Polls:39` / `Lotteries:40` / `Drafts:41`）；`DELETE FROM topic_poll|topic_lottery|topic_draft WHERE user_id = ?`（`:133-135`）；`topic_poll_vote` 与 `topic_lottery_entry` 在删行后重算（`:292-293`：`topic_poll_option.vote_count`、`topic_lottery.entry_count`）。**注意 `topic_lottery_code.claimed_by` 不在清理面内**——删号后被这人领走的码仍然标着他的 user_id。
**管理面**（`internal/admin/repository/topic_admin_repo.go:61-63,97,102`）：话题删除前统计 polls / lotteries / drawn lotteries。
**用户资料**（`internal/user/repository/stats_repo.go:24-25` → `dto` 的 `topic_poll` / `topic_lottery` 计数）会出现在用户页。

---

## 5. 数据库：表与读写的列

| 表 | 读 | 写 |
|---|---|---|
| `topic_poll` | 全部列（`id,title,description,type,min_choice,max_choice,deadline,status,notification_sent,result_visibility,is_anonymous,can_change_vote,topic_id,user_id,created,updated`） | 创建时插全行；`UpdatePollFields` 写 `title,description,type,min_choice,max_choice,deadline,result_visibility,is_anonymous,can_change_vote`；扫描写 `notification_sent,updated`；删除 |
| `topic_poll_option` | `id,text,poll_id,vote_count` | 插 `text,poll_id`；`UpdateOptionText` 写 `text`；`AdjustOptionVoteCount` 写 `vote_count`（盲加减）；删除 |
| `topic_poll_vote` | `id,poll_id,option_id,user_id,created` + join `topic_poll_option.text` | 插 `poll_id,option_id,user_id`；按 `(poll_id,user_id)` 删 |
| `topic_lottery` | 全部列，`seed` 只在 drawn 后出 DTO | 插全行；`UpdateFields`（标量 + `status`/`drawn_at`/`updated`）；`SyncEntryCount` 写 `entry_count`；`ClaimDue`/`ReleaseDrawing` 写 `status` |
| `topic_lottery_prize` | `id,lottery_id,name,description,image_hashes,nsfw_hashes,delivery,point_mode,point_amount,slots,sort_order` | 插全行；`DeletePrizes` 整组删重写 |
| `topic_lottery_code` | `id,prize_id,secret,claimed_by`（**`secret` 只有 `ClaimCode` 读**）；`CountCodesForLotteries` 只数行 | 插 `lottery_id,prize_id,secret`；`TakeCode` 写 `claimed_by,claimed_at`；CASCADE 删 |
| `topic_lottery_entry` | 全部列（`code_id` 只在服务端） | 插 `lottery_id,user_id,reply_floor[,prize_id,rank_key,point_awarded,won_at,fulfillment,code_id,claim_deadline]`；`UpdateEntryFields` 写 `prize_id,rank_key,point_awarded,won_at,fulfillment,code_id,claim_deadline,updated`；`ClaimExpiredCodeWins` 写 `fulfillment,updated`；按 `(lottery_id,user_id,prize_id=0)` 删 |
| `topic_draft` | `id,user_id,title,content,category,sections,cover_images,is_nsfw,updated`（+ `LEFT(content,120)`） | 插全行；按 `(id,user_id)` 删。`created` 只由默认值写；**`tags` 列已被迁移 049 删除** |
| `topic`（只写一列） | `id,user_id,status,access_scope,created` | `status_update_time`（建投票 / 建抽奖时顶帖） |
| `topic_reply`（只读） | `floor,user_id,topic_id,status` | — |
| `kungal_user_state`（只读） | `moemoepoint` | 由 `moemoepoint.Award` 的回写路径写 |

**迁移状态**：投票三表来自 `000_baseline`（+ `002` 加 `vote_count`，+ `022` 转 timestamptz）。草稿是 `042`（+ `049` 删 `tags`）。抽奖是 `083`（+ `085` `claim_deadline`、`086` `point_mode`/`image_hashes`/`point_awarded`、`087` `nsfw_hashes`）。**`086` 的注释明说「drop 在这里安全只是因为 083 还没上生产」**——需要确认 083–087 在生产是否已跑（见 §6）。本次普查是只读的，不产生新迁移。

---

## 6. 代码读不出来、需要生产库数据的问题

按用户的要求逐条列出想要的计数（都是单条 SQL，供用户自己跑）：

1. `SELECT status, count(*) FROM topic_poll GROUP BY 1;` —— 确认 §1.7：`status` 是不是恒为 `'open'`。如果有 `'closed'` 行，说明 Node 时代写过它，v1 要保留这个状态；如果全是 `'open'`，直接删字段。
2. `SELECT result_visibility, count(*) FROM topic_poll GROUP BY 1;` 与 `SELECT is_anonymous, count(*) FROM topic_poll GROUP BY 1;` —— 决定 §1.6「权限不足回空数组」影响多少投票，以及 §1.2 的「追溯性去匿名」有多少历史投票暴露在风险里。
3. `SELECT count(*) FROM topic_poll_vote v JOIN topic_poll_option o ON o.id = v.option_id WHERE o.poll_id <> v.poll_id;` —— **这是 §1.4 跨投票投票 BUG 是否已经被利用的直接证据。** 非零就要连带跑一次 `vote_count` 的全量重算。
4. `SELECT count(*) FROM topic_poll_option o WHERE o.vote_count <> (SELECT count(*) FROM topic_poll_vote v WHERE v.option_id = o.id);` —— `vote_count` 漂移了多少行（W4 在 reaction 上做过同样的对账）。
5. `SELECT count(*) FROM topic_poll_option WHERE length(text) = 0 OR length(text) > 100;` —— §1.1 缺 `dive` 的实际后果。
6. `SELECT count(*) FROM topic_poll p JOIN topic t ON t.id = p.topic_id WHERE t.status = 1 OR t.access_scope <> 'public';` 与 `topic_lottery` 的同一句 —— §0.2 有多少子资源挂在不可见话题上，决定 v1 上线时有多少东西会「突然消失」。
7. `SELECT status, count(*) FROM topic_lottery GROUP BY 1;` 以及 `SELECT count(*) FROM topic_lottery WHERE status = 'drawing';` —— 有没有卡死在 `drawing` 的；`SELECT count(*) FROM topic_lottery WHERE status='open' AND draw_mode='deadline' AND deadline IS NULL;` —— §2.2 那条「永远开不了奖」是否已经发生。
8. `SELECT entry_mode, count(*) FROM topic_lottery GROUP BY 1;` / `SELECT draw_mode, count(*) FROM topic_lottery GROUP BY 1;` / `SELECT delivery, point_mode, count(*) FROM topic_lottery_prize GROUP BY 1,2;` —— 每个枚举成员的行数（§5.1 第 1 条要求）。
9. `SELECT coalesce(sum(point_awarded),0) FROM topic_lottery_entry WHERE point_awarded > 0;` —— **§2.1 凭空铸币至今发了多少萌萌点**，以及 `SELECT count(DISTINCT user_id) FROM topic_lottery_entry WHERE point_awarded > 0;`。
10. `SELECT count(*) FROM topic_lottery_code;` / `WHERE claimed_by <> 0;` / `SELECT fulfillment, count(*) FROM topic_lottery_entry WHERE prize_id > 0 GROUP BY 1;` —— 托管码的规模与履约分布，决定 §2.3「版主删除连带删码」的风险面。
11. `SELECT count(*) FROM topic_lottery_prize WHERE jsonb_array_length(image_hashes) > 0;` 和其中 distinct hash 的总数 —— **§2.1 有多少张奖品图正在等着被图床 GC 掉**。这一条还需要拿这些 hash 去 image service 查一次 `width/thumbhash` 是否还在（memory 的「galgame image lifecycle」给了方法）。
12. `SELECT count(*) FROM topic_draft;`、`SELECT count(*) FROM (SELECT user_id FROM topic_draft GROUP BY 1 HAVING count(*) >= 30) x;`、`SELECT count(*) FROM topic_draft WHERE sections <> '' AND sections NOT IN (...合法 section 组合...);` —— 草稿的规模、有多少人已经顶到 30 条上限（§3.1 说明这个上限实际是「保存次数」）、有多少草稿的 section 发不出去。
13. 生产迁移版本：`SELECT * FROM _migrations ORDER BY 1 DESC LIMIT 10;` —— 确认 083–087 是否已应用。**如果没有，整个抽奖功能在生产上是不存在的**，这会大幅改变 v1 的优先级排序。
14. `SELECT count(*) FROM topic_poll WHERE deadline IS NOT NULL AND extract(hour from deadline AT TIME ZONE 'Asia/Shanghai') = 23 AND extract(minute from deadline AT TIME ZONE 'Asia/Shanghai') = 59;` 与 `topic_lottery` 的同一句 —— §9.5 的「打开再保存挪动截止时间」有多少行已经被压到当天 23:59:59（不是铁证，但 23:59:59 占比异常高就说明编辑弹窗改过它们）。
15. `SELECT count(*) FROM topic_poll_vote v WHERE NOT EXISTS (SELECT 1 FROM topic_poll_option o WHERE o.id = v.option_id);` —— 应为 0（外键还在）；非零说明 `topic_poll_vote_option_id_fkey` 在某次迁移里被误删，那样 §1.4 的攻击面会从「站内任意已存在的选项」扩大到「任意整数」。

代码读不出来的其它两件事：

- **`KUN_LOTTERY_CODE_KEY` 在生产是否真的设了。** 代码只能证明「没设就拒 code 奖项」，设没设要看部署环境。若已有 code 奖项行（§6.10 非零）则它当时一定设过，但**不能证明现在的 key 还是当时那把**。
- **图床那边对未 ping 的 hash 到底多久回收。** §2.1 的推论在论坛侧是确定的（ping 永远覆盖不到 jsonb 里的裸 hash），但「会不会真的被删、多久删」属于 image service 的策略，要去 infra 侧确认。

---

## 7. 命名问题汇总（v1 对照表的增量）

| 旧 | 问题 | v1 |
|---|---|---|
| 路径 `:tid` | 被忽略；且是禁用名 | `{topic_id}`，且必须真的用它 |
| body/query 的 `topic_id` / `poll_id` / `lottery_id` | int；与路径重复 | 路径段，字符串 id |
| `option`（`TopicPollResponse`） | 单数键装数组 | `options` |
| `user`（投票作者 / 抽奖作者） | 禁用名，一条资源里有好几个人 | `author` |
| `created` / `updated` | 禁用名 | `created_at` / `updated_at` |
| `status`（poll / lottery 生命周期，整数式思维的字符串） | 与 problem 的 `status` 同名不同型 | `state` |
| `deadline` | 是时间戳却不以 `_at` 结尾 | `closes_at` / `ends_at` |
| `drawn_at` / `won_at` / `claimed_at` / `claim_deadline` | 前三个对，`claim_deadline` 不对 | `claim_expires_at` |
| `has_voted` / `is_voted` / `has_entered` / `can_enter` / `enter_blocked` / `my_*`（8 个） | 随查看者变化却在顶层 | 全部进 `viewer{}` |
| `enter_blocked` | 服务端产出的中文句子 | `viewer.enter_blocked_reason` 封闭枚举 + `params` |
| `vote_count`（选项） / `vote_count`（投票总数） / `voters_count` | 同名两义 + 一个不带 `_count` 的概念 | 选项 `vote_count`、投票 `total_vote_count`、人数 `voter_count` |
| `codes_loaded` | 不是 `_count` 结尾，且含义是「一共载入过几个码」而不是「还剩几个」 | `code_count` |
| `total_slots` / `slots` | 计数不带 `_count` | `slot_count` |
| `point_amount` + `point_mode` + `point_total` | `point_amount` 的含义随 `point_mode` 变（per-winner 还是池子），必须读注释才知道 | 拆成 `point_per_winner` / `point_pool`，或保留 `point_mode` 但改名 `point_amount` → `point_budget` |
| `image_hashes` + `image_urls` + `nsfw_hashes` + `machine_nsfw_hashes` | 四个平行数组靠下标对齐 | `images: [Image]`，每项带 `sexual` 与 `is_author_marked_adult` |
| `rank_key` / `seed_hash` / `seed` | 都是十六进制串，名字没说 | 保留（是公开证明的一部分），但 `seed` 应为 `string \| null` |
| `reply_floor` | 抽奖 DTO 里 Go 字段叫 `Floor`、json 叫 `reply_floor`、model 叫 `ReplyFloor` | 统一 `reply_floor` |
| `entry_mode` `signup/reply/floor`、`draw_mode` `deadline/manual/threshold`、`delivery` `code/manual/point`、`fulfillment` 四值、`result_visibility` 三值、poll `type` 二值 | 都是封闭枚举，但 OpenAPI 上没有任何一处声明 | 全部声明为封闭枚举，未知值 `UNKNOWN_ENUM_VALUE` |
| `delivery: "manual"` 与 `draw_mode: "manual"` | 同一个词两个含义（人工发货 / 人工开奖） | `delivery: "offline"`、`draw_mode: "manual"` |
| 草稿的 `section`（单数键数组） | 与话题 DTO 的同一个毛病 | `sections` |
| 草稿的 `content` | 是 Markdown 源文 | `content_markdown`（K13） |
| 草稿 `summary` | 是 `LEFT(content,120)` 的 Markdown 片段，不是摘要 | `content_excerpt` |
| claim 响应的 `data.code` | 与信封顶层的 `code`（业务码）撞名 | `redemption_code` |
| 抽奖的 `description` ≤1000 vs 投票的 `description` ≤500 | 同名不同约束 | 各自声明 `maxLength` |
| 萌萌点 `ref` = `lottery_5` | 全站唯一一个不走 `moemoepoint.Ref()`（冒号）的 | `topic_lottery:5` |
| `MaxSlotsPerPrize` 同时当「单奖上限」和「总名额上限」 | 一个常量两个语义 | 拆成两个 |

---

## 8. 测试覆盖

整个域只有两个单元测试文件：`internal/topic/service/lottery_images_test.go`（SFW 图片 URL withholding）与 `lottery_payout_test.go`（`pointPayouts` 的三种模式）。
**投票零测试、草稿零测试、21 个端点没有一个有 handler / 路由 / 契约测试。** 抽奖的开奖、抢锁、兑换码、履约、扫描器同样零测试。这与 W4 普查的「互动路径没有任何测试」是同一句话，而这里的钱和秘密更多。

---

## 9. 网页调用点（`apps/web`）

`apps/web/server/**`（Nitro）**零调用**——这一族没有任何服务端代理，全部是浏览器直连 `/api`。`docs/proj/app-direct-api.md` 同样零提及。

### 9.1 调用点清单

| 端点 | 调用点 |
|---|---|
| `GET /poll/topic` | `apps/web/app/composables/topic/usePoll.ts:5`（`useKunFetch`，**进 SSR `__NUXT__`**） |
| `POST /poll` | `usePoll.ts:12`（声明成 `kunFetch<TopicPoll>`，服务端其实无 data） |
| `PUT /poll` | `usePoll.ts:57` |
| `DELETE /poll` | `usePoll.ts:72` |
| `POST /poll/vote` | `usePoll.ts:79` |
| `GET /poll/log` | `apps/web/app/components/topic/poll/Log.vue` |
| `GET /lottery/topic` | `apps/web/app/composables/topic/useLottery.ts:37`（`useKunFetch`，**进 SSR `__NUXT__`**） |
| `GET /lottery/entrants` | `useLottery.ts:42` |
| `POST /lottery` | `useLottery.ts:47` |
| `PUT /lottery` | `useLottery.ts:53` |
| `DELETE /lottery` | `useLottery.ts:67` |
| `POST /lottery/enter` | `useLottery.ts:74` |
| `POST /lottery/withdraw` | `useLottery.ts:80` |
| `POST /lottery/draw` | `useLottery.ts:94` |
| `POST /lottery/cancel` | `useLottery.ts:109` |
| `POST /lottery/claim` | `useLottery.ts:118`（← `lottery/Card.vue:223`） |
| `PUT /lottery/fulfillment` | `useLottery.ts:124` |
| `GET /topic/draft` | `apps/web/app/composables/topic/useTopicDraft.ts:27` |
| `POST /topic/draft` | `useTopicDraft.ts:30` |
| `GET /topic/draft/:id` | `useTopicDraft.ts:43` |
| `DELETE /topic/draft/:id` | `useTopicDraft.ts:58` |

组件：`topic/poll/{Section,List,Log,Modal}.vue`、`topic/lottery/{Section,Card,Result,Modal}.vue`、`topic/miniapp/{registry,sections,deadline}.ts`、`edit/topic/DraftModal.vue`（← `edit/topic/Layout.vue:169`）、`topic/detail/Detail.vue:199-202` 挂载。

### 9.2 服务端发了但没人读的字段（v1 可以直接不发，A5）

- 投票：`user`（**整个作者对象无人读**——投票的归属判断完全靠话题作者）、`voters_count`、`created`、`updated`。
- 抽奖：`drawn_at`、`my_entry_id`、`my_point_awarded`（页面显示的点数来自 `winners[].point_awarded`）、`created`、`updated`；奖项的 `point_total`、`codes_loaded`；中奖者的 `rank_key`、`won_at`；参与者的 `reply_floor`、`created`。
- **参与者名单丢掉了 `reply_floor`**（`lottery/Card.vue:551-554` 只读 `user`），所以楼层抽奖的参与名单跟报名抽奖长得一模一样，服务端算出来的楼层号被扔掉。

### 9.3 BUG · 抽奖的界面权限挂在投票的权限键上

`topic/detail/Detail.vue:14-21`：

```js
const canCreateAnyPoll = useCan('poll.create_any')
const canEditAnyPoll = useCan('poll.edit_any')
const isTopicAdmin = authorId === id || canCreateAnyPoll || canEditAnyPoll
```

这一个布尔经 `Detail.vue:199-202` → `miniapp/Container.vue:24` 同时喂给 `poll/Section.vue` **和** `lottery/Section.vue`，最终决定 `lottery/Card.vue:44` 的 `canManage`。也就是说：

- `lottery.create_any` / `lottery.manage_any` / `lottery.view_restricted` / `poll.delete_any` / `poll.view_restricted` **五个键在 `useCan.ts:29-33`、`constants/permission.ts:52-56` 声明了，全站一次都没被查过**（grep 证实）。
- 一个只有 `lottery.manage_any` 的版主**看不到任何抽奖按钮**；一个只有 `poll.create_any` 的版主**能看到别人抽奖的开奖 / 取消 / 删除按钮**（点下去服务端会 403，所以不是提权，但是按钮状态与服务端判定彻底脱节——正是 memory「claim review authz：mirror test 钉死 BE↔FE 四处」要防的那类漂移，而这里连 mirror test 都没有）。
- 投票删除按钮的门是 `canViewResults && isTopicAdmin`（`poll/List.vue:208`），不是 `poll.delete_any`；顺带导致**管理员在 `after_deadline` 投票截止前既编辑不了也删不掉**。
- `useCan.ts:88-96`：`kun-perm-mine` 为空时回落到**硬编码**的 `ROLE_PERMISSIONS`（`useCan.ts:78-82`），这是第三份镜像。

v1 的 K9 `viewer.can_*` 正好是这一整块的出路：服务端算，前端不再自己镜像。

### 9.4 BUG · 编辑路径两端都不校验

Go 那边投票的 `options` 缺 `dive`（§1.1 / §1.2）。网页这边：

- **投票编辑**：`poll/Modal.vue:118-120` 校验的是 `{ ...payload, options: {} }`，而 `updatePollSchema.options` 的 `add`/`update`/`delete` 全是 `.optional().default([])`（`validations/topic-poll.ts:35-52`），字面量 `{}` 直接通过。真正的选项负载是**校验之后**在 `usePoll.ts:27-41` 拼出来的，一行没验。
- **抽奖编辑**：`lottery/Modal.vue:179-180` 把整个 `lotterySchema.safeParse` 包在 `if (rewritePrizes.value)` 里，而编辑时 `rewritePrizes` 恒为 false（`Modal.vue:108`）。于是**编辑一个抽奖时 `title` / `description` / `floor_rule` / `draw_threshold` / `min_*` / `deadline` 完全没有客户端校验**，而服务端的 `UpdateLottery` 在不传奖项时同样跳过 `validateShape`（§2.2）。**两端在同一条路径上同时放行**，这就是 §2.2 那个「改成永远开不了奖」能被普通 UI 走出来的原因，不需要手搓请求。
- 同一个 `if` 块里还藏着两条只在创建时跑的交叉校验：码数 == 名额（`Modal.vue:187-193`）、点数池 ≥ 名额（`:194-204`）。楼层数 == 总名额则**两端都没有客户端校验**，只有 `Modal.vue:266` 的一句说明文字。

### 9.5 BUG · 打开再保存会挪动截止时间

`miniapp/deadline.ts`：`deadlineFromPicker` 取本地 `23:59:59` 再 `toISOString()`（产出 `YYYY-MM-DDTHH:mm:ss.sssZ`，**永远合法 RFC3339**，所以 §1.1/§2.2 的「静默 nil」分支从 UI 走不出来，只能手搓请求触发）；`deadlineToPicker`（`:19-29`）把存的瞬间窄化回本地 `yyyy-MM-dd`。两者不是互逆的：**打开一个抽奖 / 投票的编辑弹窗，什么都不改就保存，截止时间会被改写成当天本地 23:59:59**。原本设在 `10:00` 的截止会变成 `23:59:59`。
附带：`new Date(year, ...)` 对 `year < 100` 走 JS 的 1900 偏移，`0099` 会静默变成 1999。

### 9.6 Zod 与 Go 的其它不一致（前端更严 = 用户被前端拦下，前端更松 = 服务端 400）

| 字段 | Go | 网页 Zod | 方向 |
|---|---|---|---|
| poll `deadline` | 任意 RFC3339（含 `+08:00`） | `z.iso.datetime()`，Zod v4 不带 `{offset:true}` **只收 Z 结尾** | 前端更严 |
| poll `topic_id` | 无上限 | `max(9999999)`（`topic-poll.ts:25`） | 前端独有 |
| poll `is_anonymous` / `can_change_vote` | 无 tag | 必填 boolean | 前端独有 |
| `max_choice >= min_choice`、`max_choice <= len(options)` | 无 | 无（`poll/Modal.vue:234` 的 `:min` 只是 HTML 提示） | **两端都缺** |
| lottery `codes` | ≤500 项、每项 1..200 | `z.string().default('')`——**校验的是字符串，而 `useLottery.ts:23-29` 在校验之后才 split 成数组** | 前端形同虚设 |
| lottery prize `image_hashes` 元素 | 1..128 | 只有 `max(128)`，缺 `min(1)` | 空串 hash 过前端、被服务端 400 |
| lottery `point_mode` | `omitempty` 允许空 | 必须三选一 | 前端更严 |
| lottery `nsfw_hashes` 元素 | 无长度约束 | `max(128)` | 前端独有 |
| lottery `show_entrants` | 无 tag | 必填 boolean | 前端独有 |
| lottery `topic_id` / `lottery_id` | ≥1 必填 | **`lotterySchema` 里根本没有这两个字段**，却照样发（`useLottery.ts:50,56`） | 前端不校验 |

### 9.7 承诺-揭示在网页上是「展示」不是「验证」

`seed_hash`（`lottery/Card.vue:384,406`）与 `seed`（`:410`）只被渲染；`rank_key` 声明在 `shared/types/topic-lottery.ts:30` 却**全站零读取**。grep `crypto|subtle|sha256|hmac` 在 `components/topic/` 与 `composables/topic/` 下只命中 `Card.vue:414,415,418` 三句说明文字。也就是说：客户端不算 SHA-256、不算 HMAC，「公平性凭据」面板把服务端自己的说法原样展示，并告诉读者「你可以自己验算」。而读者其实验不了——`rank_key` 不露出，参与者名单在 `show_entrants=false` 时为空、为真时又丢掉了 `reply_floor`。v1 如果要保留这个卖点，要么前端真算一遍，要么把 `rank_key` 和完整参与名单一起下发。

### 9.8 兑换码在网页侧的处置：合格，但有残留

`Card.vue:223` POST → `Card.vue:227` 写进**组件局部** `ref`（`Card.vue:35`，不是 `useState`、不是 Pinia）→ `Card.vue:537` 渲染 + `KunCopy`（`:538`）。**没有**进 store、URL、`localStorage`/`sessionStorage`/cookie、SSR 负载。对照：`getLotteries` 用 `useKunFetch`（`useLottery.ts:37`），它的结果**确实**会进 `__NUXT__`，所以 `seed`、`my_prize_name`、`my_claim_deadline`、`my_code_ready`、`voters` 都在页面源码里——码不在。**memory 的第二条断言在网页侧同样成立。**
残留：`revealedCode` 揭示后**从不清空**。`Card.vue:540-542` 写着「关闭后需要重新点击「领取兑换码」才能再次查看」，但组件里没有任何地方重置它（`:228` 的 `emits('refresh')` 只重拉列表），它是内联的 `KunInfo` 不是弹窗——所以那句话是假的，码会一直留在 DOM 和内存里直到卡片卸载。配合 §2.8「揭示一次」也是假的，这是同一句文案的两处不实。

### 9.9 网页替服务端擦的屁股

1. `lottery/Card.vue:99-113`：**「空 URL 就是隐藏信号」**——服务端不发「隐藏了几张」，前端靠 `image_hashes.length - visibleImages.length` 相减推。任何别的原因造成的空 URL（上传失败、CDN 缺口）都会被显示成「成人内容已隐藏」（`Card.vue:304,339-350`）。v1 该改成 `images: [Image]` 每项自带 `sexual`，不要平行数组。
2. `lottery/Modal.vue:64-66` + `:177`：`prizes: []` 在 update 语义下是「别动奖项和托管码」，在 create 语义下会违反 `min=1`——**同一个字段两个相反的含义，靠前端把它们分开**。
3. `useLottery.ts:18`：前端先把不在 `image_hashes` 里的 `nsfw_hashes` 滤掉再发（服务端 `validateShape` 其实也会拦，属于重复防线）。
4. `useLottery.ts:7,21`：模式用不到的字段前端主动清零（`floor_rule`、`point_amount`），否则服务端会照存。
5. `poll/List.vue:21-24` / `lottery/Card.vue:38-41`：`useState(() => Date.now())` + `onMounted` 重读，抹平 SSR 渲染时刻与浏览器时刻的差，因为「进行中 / 已结束」和倒计时是按**服务端渲染那一刻**算的。v1 若下发 `state` 就不需要这个。
6. 全域**零 `TODO`/`FIXME`/`HACK`**（grep 证实）。

---

## 10. 最严重的十条（按严重度）

1. **读面完全不查话题可见性**（§0.2）——隐藏 / 受限话题的投票题面、票数、投票流水、抽奖、参与者名单对匿名公开。四个读端点，一行 `requireTopicRead` 都没有。
2. **投票不校验 `option_id` 属于 `poll_id`**（§1.4）——任何人可以用自己的投票当宿主，给站内任意投票的任意选项加票或减票，且删掉宿主投票不回退，污染永久。
3. **抽奖的萌萌点奖是凭空铸币**（§2.1）——发起人不扣分、不校验余额，单次上限 10 万，抽奖数无上限。
4. **奖品图不在 reference-ping 的扫描面内**（§2.1）——jsonb 列 + 裸 hash，两重都不命中，正在等着被图床 GC。
5. **`UpdateLottery` 不传奖项就跳过 `validateShape`，网页同一条路径也跳过 Zod**（§2.2 + §9.4）——可以把抽奖改成永远开不了奖，或让扫描器每分钟失败一次无限循环。
6. **投票可以在收了匿名票之后改成实名**（§1.2）——追溯性去匿名，`/poll/log` 把每个人投了什么全列出来。
7. **开奖的萌萌点发放在事务之外、fire-and-forget、失败只记日志**（§2.6）——库里写着「赢了 N 点」、通知也发了，OAuth 那边可能一分没加，没有补偿。
8. **`Cancel` 的 UPDATE 没有状态守卫**（§2.7）——与开奖竞态，奖已发、通知已发，状态被改成 `cancelled`。
9. **抽奖的界面权限挂在投票的权限键上，五个权限键全站零查询**（§9.3）——BE↔FE 镜像彻底脱节，且没有 mirror test。
10. **投票选项文本两端都不校验**（§1.1 / §1.2 / §9.4）——空选项能入库，超长选项打到 `varchar(100)` 变成 500。

再加两条不算 bug 但必须在 v1 裁决的：`:tid` 被 21 个端点全部忽略（§0.1）、`topic_poll.status` 是一条只读不写的死轴（§1.7）。
