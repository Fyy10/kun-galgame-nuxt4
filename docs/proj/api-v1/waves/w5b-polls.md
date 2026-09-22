# W5b · 话题投票

> 车道 B 的契约提交，2026-09-22。普查见 [census/polls-lottery-drafts.md](census/polls-lottery-drafts.md) §0–§1、§5.9、§7。
> 生产取值已实测并写进普查 §5.9，本文只引用结论。
> 本文是实现轨的**唯一依据**。轨不得自行新增错误码、迁移号或 `app.go` 字段。

## 1. 生产取值决定的三件事

- **`topic_poll.status` 35 行全是 `open`** → 这是死轴（普查 §1.7），**v1 不下发也不接受这个字段**，旧列留着等旧路由删完后再走 deploy-then-drop。
- **跨投票投票 0 行、`vote_count` 漂移 0 行、空/超长选项 0 行** → 普查指出的三个缺陷都是**潜伏**而非现存脏数据，不需要数据修复迁移。其中跨投票投票的洞我已于 2026-09-22 在旧面修掉（`poll_service.go` 的 `belongs` / `chosen` 校验）。
- **35 个投票里只有 1 个设了截止时间**，且正好落在 23:59（与普查 §9.5 的编辑弹窗 bug 吻合，n=1 不构成铁证）→ 不做数据修复，但 v1 的写面不得再改写用户给的时间。

另：读面的可见性洞（普查 §0.2）我已于 2026-09-22 修掉并上线，`requireTopicReadByID` 覆盖四个读面。**v1 不得退回去**。

## 2. 范围

8 个 v1 端点，取代 6 条旧路由（`router.go` 的 `poll` 族）。

| v1 | 说明 |
|---|---|
| `POST /api/v1/topics/{topic_id}/polls` | 建投票。**必填 `Idempotency-Key`**。201 + `Location` + `Poll` |
| `GET /api/v1/topics/{topic_id}/polls` | 列一个话题的投票。不分页（每话题上限 30） |
| `GET /api/v1/polls/{poll_id}` | 单个投票 |
| `PATCH /api/v1/polls/{poll_id}` | 改投票。**部分更新**，不是旧的整体替换 |
| `DELETE /api/v1/polls/{poll_id}` | 删投票，204 |
| `PUT /api/v1/polls/{poll_id}/vote` | **设置我的投票**（K16 槽位），200 + `Poll` |
| `DELETE /api/v1/polls/{poll_id}/vote` | 撤回我的投票，200 + `Poll`。仅当 `can_change_vote` |
| `GET /api/v1/polls/{poll_id}/votes` | 投票流水，**游标分页**，取代旧的手搓 `{logs,total}` 信封 |

## 3. 形状

### 3.1 `Poll`

`object: "poll"`；`id` / `topic_id` 是 `repr.DecimalID`；`author` 是 `repr.UserRef`。

| 字段 | 说明 |
|---|---|
| `title`、`description` | `description` 空串下发为 `""`，不是 null |
| `choice_type` | 封闭枚举 `single` \| `multiple`。**不叫 `type`**：`Problem.type` 是 URI 字符串，同名不同型过不了 G8 |
| `min_choice`、`max_choice` | 整数。`single` 时服务端强制两者为 1 |
| `closes_at` | 旧 `deadline`。RFC 3339 秒精度 UTC，可 `null` |
| `result_visibility` | 封闭枚举 `always` \| `after_vote` \| `after_deadline` |
| `is_anonymous`、`can_change_vote` | bool |
| `options` | 旧的单数键 `option`。每项 `{object:"poll_option", id, text}`——**票数不在这里** |
| `results` | `PollResults \| null`。**不可见时整块是 `null`**，而不是把每个 `vote_count` 单独发成 null |
| `created_at`、`updated_at` | |
| `viewer` | 匿名时 `null` |

`PollResults`：`total_vote_count`、`voter_count`、`options: [{option_id, vote_count}]`、`sample_voters: [UserRef]`（至多 5 人，**`is_anonymous` 时恒为空数组**）。

`PollViewer`：`has_voted`、`chosen_option_ids: [ID]`、`can_vote`、`can_change_vote`、`can_edit`、`can_delete`、`can_view_results`。

**`status` / `state` 不下发**（§1）。

### 3.2 写面请求体

- 建：`{title, description?, type, min_choice?, max_choice?, closes_at?, result_visibility, is_anonymous?, can_change_vote?, options: [{text}]}`。`topic_id` 在路径上。
- 改：所有字段可选，只改传来的；选项增删改用 **`option_changes`** `{add:[{text}], update:[{option_id, text}], remove:[option_id]}`。**不叫 `options`**：`Poll.options` 是数组，同名不同型过不了 G8。
- 投：`{option_ids: [ID]}`。

## 4. 逐条裁决（对普查 §1）

| 普查 | 裁决 |
|---|---|
| §0.1 `:tid` 被忽略 | 标识符**只从路径来**。请求体里不再有 `topic_id` / `poll_id` |
| §0.2 读面不查可见性 | 已修并上线。v1 一律走 `visibleTopic` / 新增 `visiblePoll` |
| §1.1 `options` 缺 `dive` | huma schema 对数组元素天然递归；`text` 是 `minLength:1 maxLength:100`。空选项 → `422` + `/options/N/text` + `TOO_SHORT`；超长 → `TOO_LONG`。**不得再让它变成 500** |
| §1.1 `min_choice`/`max_choice` 事实必填但 schema 说可选 | v1 给默认值：`single` 恒 1/1；`multiple` 缺省 `min=1`、`max=len(options)` |
| §1.1 deadline 解析失败被静默吞掉 | `422` + `/closes_at` + `INVALID_FORMAT`。**绝不静默变 null** |
| §1.1 `count, _ :=` 吞错误导致上限静默通过 | 错误上抛 500；30 个的上限保留 |
| §1.1/§1.2 建/改不查话题可见性 | 走 `visibleTopic`；版主也不例外 |
| §1.2 PUT 是伪装的整体替换 | 改成真 `PATCH`，只改传来的字段 |
| §1.2 `options` 子树零校验 | schema 覆盖每一层；`option_id` 必须属于本投票，否则 `422` + `UNKNOWN_REFERENCE` |
| §1.3 删除不回退 `vote_count` | 级联删掉选项本身，无需回退；但删除必须在一个事务里 |
| §1.3 DELETE 用 query 传 id | 进路径 |
| §1.4 `option_id` 不校验归属 | 旧面已修。v1 必须自带同样的校验：不属于本投票 → `422` + `UNKNOWN_REFERENCE`；重复 id → `DUPLICATE_ITEM`（**不得靠唯一索引炸成 500**） |
| §1.4 计数与行数漂移 | 插入/删除都用 `RETURNING id`，**只有真的动了行才改计数**（W4 的做法） |
| §1.4 改投用事务外读出的旧选项 | 全程在事务内、同一个 `tx` 句柄 |
| §1.4 没有幂等键 | 投票改成 `PUT` 槽位，天然幂等；建投票必填幂等键 |
| §1.5 `voters` 无 `ORDER BY` | `sample_voters` 按 `created ASC, id ASC` 取前 5 |
| §1.5 匿名时白查 `HasUserVoted(pollID, 0)` | 匿名直接跳过，`viewer` 为 `null` |
| §1.5 `FindByTopicID` 无 id 决胜键 | `created DESC, id DESC` |
| §1.6 权限不足回 200 + 空数组 | **回 `403 PERMISSION_REQUIRED`**。K7 要根除的就是这种兜底 |
| §1.6 `total` 与 `logs` 口径不一 | 游标分页，不下发 `total` |
| §1.6 `ORDER BY created DESC` + OFFSET | 键集分页，排序键 `(created, id)` |
| §1.7 `status` 是死轴 | 不下发、不接受。旧列等旧路由删完后另走 deploy-then-drop |

## 5. 预分配（轨只填，不得自行挑号）

- **迁移号：103**，且只有 103。内容：`topic_poll_vote (poll_id, created, id)` 索引供流水键集分页；`topic_poll (topic_id, created DESC, id DESC)` 索引。**纯加、幂等、不动任何数据。**
- **错误码：已加齐两个，轨不得再加。** 契约提交里已注册：
  - `POLL_CLOSED`（kungal，409）——过了 `closes_at`。请求本身没有任何问题，所以不是 `VALIDATION_FAILED`。
  - `VOTE_ALREADY_CAST`（kungal，409）——已投过且 `can_change_vote` 为假。
  其余全部复用：`NOT_FOUND`、`VALIDATION_FAILED`（含 `TOO_SHORT`/`TOO_LONG`/`TOO_FEW_ITEMS`/`TOO_MANY_ITEMS`/`UNKNOWN_REFERENCE`/`DUPLICATE_ITEM`/`INVALID_FORMAT`）、`PERMISSION_REQUIRED`、`CONTENT_REJECTED`、`SERVICE_UNAVAILABLE`、`IDEMPOTENCY_*`。**再缺就停下来报告。**
- **`app.go` 字段：不新增。** 投票属于话题域，落 `internal/topic/apiv1/`。

## 5.5 实现之后对本契约的三处修正（督查，2026-09-22 验收时）

写契约时没想到的，记在这里免得下一波照抄错的：

1. **`total_votes` 违反 01 §3**（「计数以 `_count` 结尾」）。普查提的 `total_vote_count` 才是对的，是我改坏的。已改回 `total_vote_count`。
2. **`type` 与 `options` 这两个名字过不了 G8**，实现轨报上来并自行改成了 `choice_type` 与 `option_changes`。这不是它擅自改名，是契约写的名字与已有 schema 同名不同型。
3. **`can_change_vote=false` 时重复发同一个 `PUT` 回 200，不是 409**。严格照字面读会让「响应丢失后的重试」报错，同时违反 K16 与 K12。只有**真的改变了选择**才回 `VOTE_ALREADY_CAST`。

另外实现轨补了两条本契约漏掉的不可变规则（`is_anonymous` 与 `choice_type` 在已有投票后不可改），依据是普查 §10.6 把「追溯性去匿名」排在本域第六严重，而我的 §4 裁决表走漏了它。**这两条采纳。**

顺带修掉一条 W2 就上线的假话：`Topic.bumped_at` 的描述写着「poll votes … set it to now」，而投票从来不顶帖——顶帖的是**建**投票、**建**抽奖和发评论。已改。

## 5.6 网页轨报上来的两条，以及验收时抓到的一条

网页轨（2026-09-22）：

1. **`closes_at` 收不下 `toISOString()`。** `repr.DateTime` 是 `minLength=maxLength=20` 加 `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`，秒精度，不带毫秒；而既有的 `deadlineFromPicker` 吐 24 个字符的 `…T15:59:59.000Z`。凡是带截止日期的投票**每一次都会 422**。本契约和端点描述都没写这条，只有 Go schema 里有。网页轨加了 `closesAtFromPicker`，旧的留给还没搬的抽奖。全站扫过一遍：`toISOString()` 的其余调用点都是 JSON-LD 与 `<time datetime>`，没有第二个 v1 写面吃日期时间。
2. **`PollViewer` 没有 `can_view_votes`。** 流水的服务端闸是 `!is_anonymous && canViewPollResults(...)`，网页按 `results !== null && !is_anonymous` 推。**这条不补，是刻意的**：`viewer` 对匿名访问者恒为 null（K16），而匿名访问者确实读得到 `always` 且实名的投票流水——真加了这个标志，登出的人就看不到按钮了。`results` 的可空性与流水闸是同一个 `canViewPollResults`，所以这个推导今天是**等值**而不是近似；要改流水闸，必须同时改 `results` 的可空性，否则这里就错了。

验收时抓到的第三条（**已修**，`791144b2`）：投票卡的投票人一栏落到一个裸 `v-else`，只要 `sample_voters` 是空就说「还没有人投票」。服务端对**所有**匿名投票都下发空 `sample_voters`，生产上 10 个「匿名 + 任何人可见结果 + 有票」的投票会把这句话显示在实时票数和「共 N 票」旁边。旧代码是 `v-else-if="!poll.is_anonymous && poll.vote_count"` 且没有 `v-else`，所以什么都不显示——新写法把「不显示」变成了「说假话」。只有 `voter_count` 能下这个判断。（第二个可达分支：实名投票但抽样到的投票人全被封禁，`IsRenderable` 会把他们滤光。）

## 6. 本波不做

- 删旧投票路由与下调基线（督查在验收后统一做）。
- `topic_poll.status` 列的 drop（旧面还在读它）。
- 抽奖与草稿（各自的波）。
- 普查 §9.5 那条「打开再保存挪动截止时间」的历史数据修复（只有 1 行，且不确定）。

## 7. 变异题（督查出题，轨不得自行增删）

| # | 改动 | 应当杀死它的断言 |
|---|---|---|
| M1 | 建投票时不查话题可见性 | 往读不到的话题建投票应得 404 |
| M2 | 投票时不校验 `option_id` 属于本投票 | 传别的投票的选项 id 应得 422 + `UNKNOWN_REFERENCE` |
| M3 | 同一请求里的重复 `option_id` 放行 | 应得 422 + `DUPLICATE_ITEM`，不是 500 |
| M4 | `closes_at` 解析失败时静默置 null | 应得 422 + `/closes_at` + `INVALID_FORMAT` |
| M5 | 计数改成无条件 `vote_count + 1` | 重复 `PUT` 同样的选票不得让票数二次自增 |
| M6 | 流水在权限不足时回空数组 | 应得 403 `PERMISSION_REQUIRED` |
| M7 | 流水的排序去掉 id 决胜键 | 全量遍历（**数据里必须有并列的 created**）应出现重复或遗漏 |
| M8 | `result_visibility=after_vote` 时对未投票者下发 `results` | `results` 必须是 null |
| M9 | `is_anonymous` 时仍下发 `sample_voters` | 必须是空数组 |
| M10 | 过了 `closes_at` 仍接受投票 | 应得 409 `POLL_CLOSED` |
| M11 | `can_change_vote=false` 时允许改投 | 应得 409 `VOTE_ALREADY_CAST` |
| M12 | `single` 类型接受两个选项 | 应得 422 + `TOO_MANY_ITEMS` |
| M13 | 空选项文本放行 | 应得 422 + `/options/N/text` + `TOO_SHORT`，不是 500 |

## 8. 九条闸

照 [04-parallel-tracks.md](../04-parallel-tracks.md) §5。特别提醒：

- 第 4 条——**自己的临时库**，`KUN_REQUIRE_TEST_DB=1`，`-count=1 -p 1`。
- 第 5 条——流水的全量遍历，**种子里必须有并列的 `created`**，否则去掉 id 决胜键照样全绿。
- 第 8 条——网页跑 **`pnpm typecheck`**，不是 `pnpm vue-tsc --noEmit`（后者在本仓库是空转，见 04 §5）。
