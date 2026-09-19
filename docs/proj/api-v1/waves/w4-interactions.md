# W4 · 互动

> 裁决记录（中文，给人看）。写于 2026-09-19。契约与 W3 同在分支 `w34-contract`；实现派发给 cursor-agent，与 W3-go 并行。

| 派发 | 范围 | 依赖 |
|---|---|---|
| 普查 `w4-census` | 旧互动路径逐分支普查（只读，报告） | — |
| W4-go | §2 十四个操作、`viewer.can_*` 接进读面、迁移 099、删号重算修正、DB 契约测试 | 契约 |
| W3/W4-web | 话题页、回复、动态卡片的互动切到 v1 | W3-go、W4-go |

## 1. 普查结论（2026-09-19，代码 + 生产库）

旧路径：话题 `like` / `dislike` / `reaction` / `favorite` / `upvote` / `hide` / `best-answer`，回复 `like` / `dislike` / `reaction` / `pin`，评论 `like`，全部是 `PUT` 切换。影响设计的事实：

- **切换 + 重放 = 撤销**：没有幂等键，弱网重试一次就把赞取消了。
- **计数与行会漂移**：先 `HasReaction` 再 `INSERT … ON CONFLICT DO NOTHING`，再无条件 `+1`。并发时行只有一条，计数加两次、萌萌点也发两次。
  - 生产：话题 like 计数偏 1 个、dislike 偏 2 个，回复 dislike 偏 1 个；
  - 8 个话题、9 条回复有人同时点了赞和踩。
- **萌萌点键**：同 W3，全是 `KeyNonce`，事务里起 goroutine；点赞与收藏共用 `liked` / `topic:{id}`，去掉随机后缀就会互相吞掉。
- **推**：不是切换，花 10 萌萌点、给作者 5，可以对同一话题反复推。
  - 生产 930 次里有 25 对（用户，话题）推了多次（最多 6 次），2026 年就有 46 次：这是在用的功能，不是 bug。
  - 余额按缓存判断且不在事务里扣，并发时两次都能通过。
- **最佳答案**：同一回复再点一次就是取消，成功提示却总是「已设置」；换一条最佳答案时，前一条的 +7 不收回；隐藏回复也能设；生产有 2 个话题的最佳答案是楼主自己的回复（自己给自己 +7）。
- **置顶**：不检查回复是否属于本话题（v1 读面已经跳过外话题的置顶）；隐藏回复也能置顶。
- **可见性**：话题互动只拦 `status==1`，访问范围与作者封禁都不查；回复与评论互动什么都不查，`:tid` 被忽略，网页用 `/topic/0/reply/reaction`。回复不存在时回 500 而不是 404。
- **历史读**：`/upvotes` 固定 50 条、`/reaction/history` 固定 300 条，都不分页、不查可见性，隐藏话题的互动名单对匿名可见。
- **删号清理**（`purge_repo.go`）：从已废弃的 `topic_like` / `topic_dislike` / `topic_reply_like` 重算计数（生产仍有 11757 / 79 / 5425 行），然后删掉 `topic_reaction` 却不重算，删号后话题的赞数偏高。
- 网页「推」的余额提示写着 20，实际是 10。
- 互动路径没有任何测试。

## 2. 裁决

**十四个操作**（写操作 `required` 档，列表 `optional` 档）：

| 操作 | 响应 |
|---|---|
| `PUT` / `DELETE /topics/{topic_id}/reactions/{reaction}` | 200 `TopicEngagement` |
| `PUT` / `DELETE /topics/{topic_id}/favorite` | 200 `TopicEngagement` |
| `POST /topics/{topic_id}/upvotes` | 201 `TopicUpvote`，必须带 `Idempotency-Key` |
| `GET /topics/{topic_id}/upvotes` | 游标 `List<TopicUpvote>`，新到旧 |
| `GET /topics/{topic_id}/reactions` | 游标 `List<Reaction>`，新到旧 |
| `PUT` / `DELETE /topics/{topic_id}/best-answer` | 200 `Topic`；`PUT` 体 `{reply_id}` |
| `PUT` / `DELETE /topics/{topic_id}/pinned-reply` | 200 `Topic`；`PUT` 体 `{reply_id}` |
| `PUT` / `DELETE /replies/{reply_id}/reactions/{reaction}` | 200 `ReplyEngagement` |
| `GET /replies/{reply_id}/reactions` | 游标 `List<Reaction>`，新到旧 |

**K16 · 查看者槽位的置位与撤销。** 「我点的赞」「我的收藏」「本话题的最佳答案」「置顶回复」是一个槽位，不是有自己身份的资源：

- `PUT` 置位、`DELETE` 撤销，都是幂等的：置已置的、撤未置的都是 200 且无副作用。
- 两者都回 200，响应体是客户端重画所需的状态：表情与收藏回目标的 engagement 快照（计数、表情汇总、查看者状态），最佳答案与置顶回完整 `Topic`。
- 删掉有自己身份的资源（回复、草稿）才是 204。
- 路径 `/reactions/{reaction}` 指的是**调用者自己**那条该 token 的表情，凭证即主语，与 01 §5 抽奖的 `entries/me` 同理。
- 同一集合的 `GET` 列的是所有人的表情。将来要按 token 过滤就加 `?reaction=`，不在 `/reactions/{reaction}` 上加 `GET`，以免同一路径两种含义。

**表情**：

- 点赞、点踩就是 token `like` / `dislike`，不另开路径（旧版 `PUT /like` 与 `PUT /reaction` 本来就是同一个函数、同一行）。
- 路径参数是封闭枚举：服务端现行的 33 个 token，未知 → `400 INVALID_PARAMETER` + `UNKNOWN_VALUE`；响应里的 `reaction` 仍是开放词表。
- like 与 dislike 互斥：置一个就撤另一个，连同被撤那个的副作用。
- **计数只在真的插入或删除了一行时变**（`INSERT … ON CONFLICT DO NOTHING RETURNING id` / `DELETE … RETURNING id`），并发的两次点赞只有一次生效。
- 赞自己的话题或回复 → `403 SELF_LIKE_FORBIDDEN`；踩与其余表情允许（与旧版同）。

**收藏**：

- 自己的话题可以收藏，不给分、不发通知（与旧版同）。
- 插入用 `ON CONFLICT DO NOTHING`：旧版并发时第二条撞唯一约束，回 500。

**推**：

- 新建一条记录，可重复、不可撤销，每次都收费（产品行为，保留）。
- 必须带幂等键：这是花钱的 `POST`，重试不能扣两次。
- `note`（原 `description`）至多 30 字，去首尾空白，空白算没写，响应里是 `null`。旧版静默截断，v1 超长 → `TOO_LONG`。
- 推自己 → `403 SELF_UPVOTE_FORBIDDEN`；缓存余额 < 10 → `403 MOEMOEPOINT_INSUFFICIENT` + `required: 10`。
- 余额不在本地扣（C3：缓存只写 OAuth 返回的值），并发两次仍可能都过，余额可能短暂变负。与旧版相同，记录在案。
- 顶帖、`upvoted_at`、`upvote_count`、`upvoted` 通知（去重）、动态流（触发器），都与旧版相同。

**最佳答案**：

- 需要 `can_set_best_answer`。
- 回复必须是本话题的**可见**回复，否则 → `422 VALIDATION_FAILED`，`/reply_id` 上 `UNKNOWN_REFERENCE`。
- 设为当前那条：空操作；换一条：前一条的作者 −7，新的作者 +7；清除：作者 −7。
- **楼主自己的回复可以设，但不给分、不发通知**：修掉自己给自己 +7。
- 设置时在 3 个月门槛内顶帖（与旧版同）。

**置顶**：

- 需要 `can_pin_reply`；回复规则同最佳答案。
- 换一条是替换；不给分；非自己置顶时给回复作者发 `pin-reply`（去重）。

**可见性**：

- 话题互动与最佳答案、置顶：话题必须**已发布**且调用者按 `getTopic` 的判定看得见，否则 `NOT_FOUND`。所以隐藏期间撤不了赞（与旧版同）。
- 回复互动：按 `getReply` 的判定，且话题已发布。
- 列表：与 `getTopic` / `getReply` 同一判定，封禁用户的记录跳过并向后读满一页（与话题列表同一机制）。

**萌萌点键**（提交之后发出，C3 稳定键；表情与收藏以行 id 为引用，取消再点会插入新行、拿到新 id，所以正好再给一次）：

| 事件 | 键 | 理由 / 增减 / 引用 |
|---|---|---|
| 点赞话题 / 取消 | `kungal:liked:topic_reaction_{row}` / `kungal:unliked:topic_reaction_{row}` | `liked` ±1 给作者，`topic:{id}` |
| 点赞回复 / 取消 | `kungal:liked:topic_reply_reaction_{row}` / `kungal:unliked:…` | `liked` ±1 给回复作者，`topic_reply:{id}` |
| 收藏 / 取消 | `kungal:favorited:topic_favorite_{row}` / `kungal:unfavorited:…` | `liked` ±1 给作者（自己的不给），`topic:{id}` |
| 推 | `kungal:upvote_sent:topic_upvote_{row}` / `kungal:upvote_received:topic_upvote_{row}` | 发起者 `content_removed` −10，作者 `content_approved` +5，`topic_upvote:{id}` |
| 最佳答案 设 / 撤 | `KeyNonce("best_answer_set"…)` / `KeyNonce("best_answer_cleared"…)` | ±7，`topic_reply:{id}`。没有事件行；重试时状态已变，不会重复 |

**`viewer.can_*` 接进读面**：

- W4-go 把 `capsForTopic` / `capsForReply` 接到 `getTopic`、`listTopicReplies`、`getReply` 的 `viewer`，W3 的写操作复用同一个函数。
- 网页从此按 `viewer.can_*` 显示按钮，不再用 `useCan` 自己镜像权限。

**新错误码**：`SELF_LIKE_FORBIDDEN`（kungal 403）、`SELF_UPVOTE_FORBIDDEN`（kungal 403）；另用 W3 的 `PERMISSION_REQUIRED`、`MOEMOEPOINT_INSUFFICIENT`。

**迁移 099**（部署时自动执行，与旧代码兼容）：

1. 按行重算 `topic.like_count` / `dislike_count` / `favorite_count` / `upvote_count` 与 `topic_reply.like_count` / `dislike_count`（只改不相等的行）。
2. 历史列表的索引：`topic_upvote (topic_id, created DESC, id DESC)`、`topic_reaction (topic_id, created DESC, id DESC)`、`topic_reply_reaction (topic_reply_id, created DESC, id DESC)`。
3. 同时有赞和踩的 17 对不动：无从得知本意，下次置位时服务端会撤掉另一个。

**删号清理**：`purge_repo.go` 改为从 `topic_reaction` / `topic_reply_reaction` 重算赞踩计数，且在删行**之后**重算。

**不在 W4**：

- 评论点赞：随评论域进 W5。
- `GET /api/topic/interactions/mine`：只有动态卡片在用，随动态流迁移。
- 旧的 `topic_like` 等四张废弃表的删除：删号清理改完之后就没有读者了，另起 deploy-then-drop 迁移。
