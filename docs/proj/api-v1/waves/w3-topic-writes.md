# W3 · 话题与回复写面

> 裁决记录（中文，给人看）。写于 2026-09-19。契约（类型、七个操作的注册、新错误码、能力函数）由 Claude 亲手写，在分支 `w34-contract` 上，与 W4 同一个契约分支；实现派发给 cursor-agent（Grok 4.6 Extra High），与 W4 的实现并行，两边不碰同一个文件。

| 派发 | 范围 | 依赖 |
|---|---|---|
| 普查 `w3-census` | 旧写路径逐字段普查（只读，报告） | — |
| W3-go | §2 七个操作、迁移 098、旧建回复改用楼层计数器、DB 契约测试 | 契约 |
| W3/W4-web | 话题页、发帖页、回复框的写操作切到 v1（与 W4 合一次派发） | W3-go、W4-go |

## 1. 普查结论（2026-09-19，代码 + 生产库）

旧路径：`POST /api/topic`、`PUT /api/topic/:tid`、`POST|PUT|DELETE /api/topic/:tid/reply`，编辑表单另读旧详情与 `/reply/detail`。影响设计的事实：

- **楼层**：`MAX(floor)+1`，事务里但不加锁、无唯一约束。生产 1 组重复（话题 122 的 4 楼，id 202/203，同一用户 9 ms 内两次提交、内容相同）。删掉最高楼后下一条回复会**复用**这个楼号，旧通知里的 `?reply=N` 就指向了别人的回复。
- **`:tid` 形同虚设**：建回复用 body 里的 `topic_id`，改 / 删回复只认 `reply_id`，都不看路径；改 / 删回复也不查话题可见性。
- **萌萌点**：每次都用 `KeyNonce`（带纳秒随机后缀），不是契约 C3 的稳定键；而且在事务回调里就起 goroutine 发出，事务之后回滚也收不回来。
- **更新是整体置换**：漏传 `is_nsfw` 就清掉 NSFW，漏传封面就清空封面。
- **版块**：未知版块名静默丢掉；不校验版块与分类匹配（生产数据 100% 匹配）。
- **删回复的扣分**：公式 `3 ×（评论数 + 点赞数 + 1）`，但点赞数读的是已废弃的 `topic_reply_like` 表（生产仍有 5425 行，与现行 `like_count` 对不上）；staff 删别人的回复也要检查**被删者**余额，余额不足时 staff 删不掉。
- **编辑**：编辑话题总是顶帖（不看 3 个月门槛）；编辑中的新 @ 以编辑者名义发通知（改回复则以原作者名义），两处不一致；文档 `mention.md` 写的「每次至多通知 10 人」没有实现。
- 写路径没有任何 DB 测试。
- 数据：生产 `reply_count` 偏差 1 个话题；空白标题 / 正文 0 条；正文最长 76605 字（上限 100007），回复最长 6617（上限 10007）。

## 2. 裁决

**七个操作**（全部 `required` 档）：

| 操作 | 说明 |
|---|---|
| `POST /topics` → 201 `Topic` + `Location` | 必须带 `Idempotency-Key`（K12） |
| `PATCH /topics/{topic_id}` → 200 `Topic` | 部分更新，含状态迁移（隐藏 / 取消隐藏） |
| `GET /topics/{topic_id}/source` → `TopicSource` | 编辑语境：Markdown 源文与全部可编辑字段 |
| `POST /topics/{topic_id}/replies` → 201 `Reply` + `Location` | 必须带 `Idempotency-Key` |
| `PATCH /replies/{reply_id}` → 200 `Reply` | |
| `DELETE /replies/{reply_id}` → 204 | |
| `GET /replies/{reply_id}/source` → `ReplySource` | |

**请求体**

- 正文字段叫 `content_markdown`：`content` 是节点树的名字，同名不同型违反 A4。
- 标题去掉首尾空白后存；全是空白的标题或正文 → `422 VALIDATION_FAILED`，`reason: TOO_SHORT`、`params.min_length: 1`。正文按原样存（行首空白在 Markdown 里有意义）。
- `sections` 必须与 `category` 匹配（`g-` ↔ galgame、`t-` ↔ technique、`o-` ↔ others），不匹配 → `INCONSISTENT_WITH`。未知版块由枚举拒掉，不再静默丢。
- 封面用 `cover_image_hashes`（哈希数组；响应里 `cover_images` 是 Image 对象，同名不同型违反 A4）。创建时**缺席**才从正文取前 9 个图片 token，传空数组表示不要封面；PATCH 不做推导。
- 访问范围：`access_scope` + 平铺的 `access_roles` / `access_user_ids`（G8：输出里的 `access_grants` 是 `{roles, users}`，输入若同名就与之冲突）。
  - `role` 需要 1–4 个不重复角色，`users` 需要 1–50 个不重复 id，另一个必须缺席；`public` / `login` 两个都必须缺席。违反 → `INCONSISTENT_WITH` 或 `REQUIRED`，位置指向那个字段。
  - 作者从 `access_user_ids` 里剔除；只剩作者时存成零授权，话题只有作者与 staff 能读（与旧版相同）。
- **PATCH 语义**：在场的字段替换，缺席的保持。
  - 校验的是合并后的结果：分类改了而版块没传、且旧版块不属于新分类 → `INCONSISTENT_WITH`；
  - `access_scope` 变了就丢掉旧授权，新授权由请求提供。
  - `is_nsfw` / 封面缺席不再清空（修旧版整体置换的坑）。

**状态迁移**：隐藏与取消隐藏是 `PATCH` 的 `state` 字段，规则照搬 `hide_decision.go`，由 `capsForTopic` 表达：

- 已发布：作者或持 `topic.hide` 可隐藏；作者隐藏记 `hidden_by=author`，staff 记 `moderator`；
- 已隐藏：作者只能撤销自己的隐藏，持 `topic.hide` 可撤销任何隐藏；
- 发当前状态是空操作。
- 无权的迁移 → `403 PERMISSION_REQUIRED`（只有两个状态且互相可达，用不到 `INVALID_STATE_TRANSITION`）。

**权限与可见性**：

- 先按读判定：看不见 → `NOT_FOUND`（与 `getTopic` / `getReply` 同一判定，包括作者被封禁）。
- 看得见但没有能力 → `403 PERMISSION_REQUIRED`。
- 能力由 `capsForTopic` / `capsForReply` 一处计算，写操作的判定与 `viewer.can_*` 同源；Bearer 永无 staff 能力（K2）。
- 回复写操作一律按 `getReply` 的可见性判定：隐藏回复对所有人 `NOT_FOUND`，改 / 删都不行。

**楼层**：

- 新增 `topic.last_reply_floor` 计数器，建回复时在同一事务里原子地取号：`UPDATE topic SET last_reply_floor = GREATEST(last_reply_floor, (SELECT COALESCE(MAX(floor),0) …)) + 1 … RETURNING`。
  - 行锁串行化同一话题的并发回复；
  - `GREATEST` 兜住迁移与新代码上线之间旧代码插进来的行；
  - 楼号从此不复用，与 `Reply.floor` 的文档「删除留空号」一致。
- `(topic_id, floor)` 加唯一索引兜底。
- 旧建回复路径改用同一个取号函数：旧路由在网页切换前仍在服务，不改它就会和计数器撞号。

**副作用**（逐条保留，改掉的地方写明）：

| 写 | 副作用 |
|---|---|
| 建话题 | 每日上限（滚动 24 小时，含隐藏话题，`缓存萌萌点/10 + 1`）→ `429 TOPIC_DAILY_LIMIT_REACHED` + `limit`；付费版块（g-seeking、g-other、t-help）要求缓存余额 ≥ 10 → `403 MOEMOEPOINT_INSUFFICIENT` + `required: 10`；信任检查 deny → `422 CONTENT_REJECTED`，hold 照发；萌萌点 +3 或 −10；@ 通知；动态流由触发器写；提交后 `ScanBg` |
| 改话题 | 标题或正文**确有变化**时写 `edited_at`，并在 3 个月门槛内顶帖（旧版无门槛、任何编辑都顶）；付费属性变化时按差额给作者加减；新增的 @ 通知（按链接去重） |
| 建回复 | 行锁取号；3 个月内顶帖；按可见回复重算 `reply_count` 与 `comment_count`；他人回复给话题作者 +1 并发 `replied`；@ 通知；提交后 `ScanBg` |
| 改回复 | 正文确有变化时写 `edited_at`；不顶帖；新增的 @ 通知 |
| 删回复 | 硬删（评论、表情经级联删除）；楼号不复用；重算计数；置顶 / 最佳答案经外键置空；扣分见下 |

- **@ 通知**：每次写至多通知正文里前 10 个不同的被提及者（按出现顺序），发件人一律是**内容的作者**（staff 代改时不再算到 staff 头上）。去重规则不变（发件人、收件人、类型、链接）。
- **删回复的扣分**：
  - 作者删自己的：`3 ×（like_count + 可见评论数 + 1）`，余额不足 → `403 MOEMOEPOINT_INSUFFICIENT` + `required`；
  - staff 删别人的：给被删者扣 3，**不查余额**，余额可以变负。
- **萌萌点一律在事务提交之后发出**，幂等键按 C3 用稳定的 `kungal:<事件>:<引用>`：

  | 事件 | 键 | 理由 / 引用 |
  |---|---|---|
  | 建话题 | `kungal:topic_created:topic_{id}` | `content_approved` +3 或 `content_removed` −10，`topic:{id}` |
  | 改话题换付费属性 | `KeyNonce("topic_cost_changed", "topic_{id}")` | 同上，按差额。没有天然的事件行；重试时状态已变，差额为 0，不会重复 |
  | 他人回复 | `kungal:replied:topic_reply_{id}` | `content_approved` +1 给话题作者，`topic_reply:{id}` |
  | 删回复 | `kungal:reply_deleted:topic_reply_{id}` | `content_removed`，`topic_reply:{id}` |

**编辑语境读**：

- `TopicSource` 带 `content_markdown` 与全部可编辑字段；授权用户以 `UserRef` 给出（`access_grants.users`），编辑器不必再逐个查浮动卡片。
- `ReplySource` 只带源文。
- 二者都要 `can_edit`，看不见 → `NOT_FOUND`，看得见不能编辑 → `PERMISSION_REQUIRED`。
- 旧详情里的 `content_markdown` 是对所有人公开的，v1 只在这里给。

**`viewer.can_*`**：这一波起，`TopicViewer` 带 `can_edit`、`can_hide`、`can_unhide`、`can_like`、`can_upvote`、`can_set_best_answer`、`can_pin_reply`，`ReplyViewer` 带 `can_edit`、`can_delete`、`can_like`。

- 定义在 `caps.go`，由 W4-go 接进读面的组装。
- 语义是「此刻发这个请求会不会因权限或状态被拒」，余额与竞态不算。
- 其余表情与收藏对已发布话题的所有登录读者开放，没有标志。

**新错误码**（K4 注册表）：

| code | 域 | status | 扩展 |
|---|---|---|---|
| `PERMISSION_REQUIRED` | moderation（复用 infra，type URI 用 infra 的域） | 403 | — |
| `CONTENT_REJECTED` | kungal | 422 | — |
| `TOPIC_DAILY_LIMIT_REACHED` | kungal | 429 | `limit` |
| `MOEMOEPOINT_INSUFFICIENT` | kungal | 403 | `required` |

W4 另加 `SELF_LIKE_FORBIDDEN`、`SELF_UPVOTE_FORBIDDEN`（见 W4 记录）。

**不在 W3**：

- 图片上传（`POST /api/image/topic`，multipart + 每日 50 张配额）：网页继续用旧路由，另起一波；
- 删除话题（旧版没有作者删除，只有管理员硬删）；
- 草稿在发布后不删除（与旧版相同）。

**迁移 098**（部署时自动执行，先于新代码；与旧代码兼容）：

1. 重排重复楼层：按 `(floor, id)` 顺序，每行的新楼号 = max(原楼号, 前一行新楼号 + 1)。只动有重复的话题，空号尽量保留；生产只动话题 122 的 3 行。
2. `topic.last_reply_floor` 加列，回填为各话题 `MAX(floor)`。
3. `topic_reply (topic_id, floor)` 唯一索引。
4. 按可见回复重算 `topic.reply_count`（生产偏差 1 个）。

旧建回复在网页切换前仍在服务，已改用计数器；两条路径并发时由唯一索引兜底，最坏是一次 500。

**旧路由的删除**：网页切到 v1、上线实测之后，删掉：

- `POST /api/topic`、`PUT /api/topic/:tid`；
- `POST|PUT|DELETE /api/topic/:tid/reply`；
- `GET /api/topic/:tid`（旧详情，只剩编辑在用）；
- `GET /api/topic/:tid/reply/detail`。

`/reply/locate` 等 W5。

**已于 2026-09-22 上线验证后执行**，连同 W4 的互动旧路由共 22 条，见 `../CHANGELOG.md`。随之而去的还有：旧 `TopicWriteService` 整个文件、`ReplyService` 的写面与表情方法、`TopicService.GetDetail`、`reaction_repo.go` 的 11 个方法、`internal/middleware/idempotency.go` 与 19 个 DTO 类型。

## 3. 验收（2026-09-22）

派发 `w3-go` 在会话中断时被杀，只留下了工作树里的代码：报告与任务书随 `/tmp` 一起没了，`check.sh` 的沙箱证明也没了，所以这一波的验收是逐文件读代码 + 自己补测试，而不是核对报告。

**交付时的状态**：两个编译错误（Fiber v3 的 `Test` 第二参数、缺 import），回复写面的 DB 测试**一条都没有**（书里要求的那一组还没写）。话题创建/更新/拒绝的测试写了，但从没跑过。

**验收修的**：

- 一条断言写成了自我比较（`coverStored != x && coverStored != x`），等于什么都没查；
- 夹具把种子话题的 `created` 设成 `now`，于是「24 小时内发帖数」把它们全算进去，第一条创建就 429；
- 多处创建载荷漏了必填的 `is_nsfw`；`g-other` 是付费版块，被当成免费版块断言 +3；
- 消息断言用没有 gorm 列名标签的结构体接 `receiver_id`，永远是 0；
- 测试用的发分函数是包级全局变量，改成 `App.TopicAward` 字段；
- `topic_access_grant` 没有顺序列，授权用户按 `ctid` 读出来，顺序不稳定：改成按 id 排序，并在契约里写明。

**验收发现的产品/代码 bug**：

- **只授权给作者自己的 `users` 话题改不动**：这种话题一条授权行都不存，PATCH 合并时看到「scope 是 users 但没有 `access_user_ids`」，直接 422。现在范围没被碰时保留库里的授权。
- **版块顺序从来没有顺序**：`topic_section_relation` 没有顺序列，两处读也没有 `ORDER BY`，而 API 文档写的是「按存储顺序」。迁移 100 加 `position` 列，老行按版块 id 编号，两处读按它排序。
- **楼层复用**：删掉最高楼后下一条回复会拿到同一个楼号（`MAX(floor)+1`）。计数器 + 唯一索引修掉，并且旧建回复路径也改用同一个取号函数，否则过渡期两条路径会撞号。

**补写的测试**（`v1_topic_writes_reply_test.go`）：建回复的值与副作用、楼主自己回复不发分、3 个月顶帖门槛、楼号不复用、并发建回复楼号不重、旧路由也走计数器、计数器落后于已有行时向前跳、编辑权限（陌生人 403 / Bearer staff 403 / cookie staff 200 / 空白 422 / 不顶帖）、删除扣分公式与余额不足、staff 删不查余额、置顶与最佳答案指针被清、评论与计数、两个 source 读面、授权用户列表（封禁者 `name: null`）、只授权作者的话题可编辑。

**变异**：10 个里 10 个被杀。其中「提交前就发分」第一版没杀掉——失败点在加入奖励之前，测试看不见；改成让第二条消息（@ 提及）插入失败，再跑就杀掉了，那条测试也留了下来。

**W2 的一处连带修改**：W2 的夹具故意在同一楼层种了两条回复来测排序，迁移 100 之后这种数据不可能存在，改成两个不同楼层。
