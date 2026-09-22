# W5a · 话题评论

> 车道 B 的契约提交，2026-09-22。普查见 [census/comments.md](census/comments.md)；生产取值已补齐，见 §1。
> 本文是实现轨的**唯一依据**：范围、形状、裁决、变异题都在这里。轨不得自行新增错误码、迁移号或 `app.go` 字段。

## 1. 生产取值（2026-09-22 实测，生产库 `kungalgame`）

| 量 | 值 | 对裁决的影响 |
|---|---|---|
| `topic_comment` 总数 | 3162 | |
| `status` 分布 | **全部 0** | trust 从未隐藏过任何一条评论；`state` 不下发（读面继续滤掉 status≠0） |
| 顶层 / 子评论 | 1874 / 1288 | 子评论占 41%，父子关系是主力路径 |
| 树深 | **最深 19 层**（depth≥3 共 632 条） | 前端只画两层，靠 `threadComments.ts` 折叠；v1 继续发**平铺数组 + `parent_comment_id`**，不发真树 |
| 单楼最多评论 | 22；超过 1000 的楼：0 | `Reply.comments` 的 `maxItems:1000` 安全 |
| `topic_comment_like` | 1106；孤儿 0；自赞 0 | |
| 跨话题评论（A2） | **0** | 服务端推导 `reply_id→topic_id` 是修 bug，不是改行为 |
| 父子跨楼 | 0 | |
| 顶层 target 指错人（A1） | **6**；子评论指错 0；指向不存在的人 0 | 服务端推导 target 是修 bug；6 条历史行由迁移 101 校正 |
| `target_user_id = user_id` | 252 | 自评自，现行逻辑跳过发分，保留 |
| `topic.comment_count` 漂移 | 0 | |
| `topic_reply.comment_count` | 1281 非零（1263 正确 / 18 错），420 该有值却是 0 | 见 §4 A10 |
| 隐藏回复下的可见评论 | 0 | |
| 评论所在话题 | public 3049 / login 1 / **status=1（隐藏）112** | 可见性判定必须真的生效 |
| 正文含 `/image/` token | **0** | K20 的 token 解析当前零行受影响，是前瞻性修正 |
| 正文含 URL / 裸 HTML / md 图片 / 行内 token | 139 / 5 / 2 / 3 | |
| 正文最长 | 1000 字符；超过 1007 的：0 | |
| Markdown 语法 | `*`或`_` 40，`**` 3，行首`#` 1，列表 0，反引号 8，引用 1，md 链接 3 | **K20 的依据**：套 Markdown 管线会静默吃掉这些字符 |
| 含换行 | **544** | `break` 节点必须保真 |
| 编辑过的评论 | 39 | 编辑面仍要做，但不值得为它优化 |
| 近 12 个月评论量 | 36–357 / 月，近三月 165/158/171 | 活跃域，不是死代码 |
| `commented` 通知 | 13108 条，其中 **12598 条 link 不含 `?comment=`** | 不是本波范围，见 §6 遗留 |

## 2. 范围

6 条 v1 端点，全部新增；对应旧路由 4 条（`router.go:267-271`）。

| v1 | 说明 |
|---|---|
| `POST /api/v1/replies/{reply_id}/comments` | 发表。**必填 `Idempotency-Key`**。201 + `Location` + `Comment` |
| `GET /api/v1/comments/{comment_id}` | 单条读，供永久链接 |
| `GET /api/v1/comments/{comment_id}/source` | 编辑上下文，回 `{ text }`，要求 `can_edit` |
| `PATCH /api/v1/comments/{comment_id}` | 编辑，200 + `Comment` |
| `DELETE /api/v1/comments/{comment_id}` | 删除，204（评论有自己的 id，按 K16） |
| `PUT` / `DELETE /api/v1/comments/{comment_id}/like` | K16 槽位，200 + `Comment`（含 `like_count` 与 `viewer.has_liked`） |

**不在本轨**：`GET /topic/:tid/reply/locate`（回复与评论共用，归读面轨）；评论的列表读面已随 W2 上线（内嵌在 `Reply.comments`）。

## 3. 形状

### 3.1 `Comment` 的破坏性改动（W2 已上线，网页必须同波改）

| 字段 | W2 现状 | W5a |
|---|---|---|
| `text` | `string`，doc 称纯文本 | **删除**，换成 `content: ContentDocument` |
| `content` | — | 新增。**受限文档**，按 K20：只产出 `paragraph` / `text` / `break` / `image` / `mention` / `reply_reference`，不跑 Markdown 解析 |
| `in_reply_to_user` | 原样回显客户端传的 `target_user_id`（A15 的契约谎言） | 由服务端推导：有父评论则取父评论作者，否则取所在回复的作者 |
| `viewer` | 只有 `has_liked` | 加 `can_edit` / `can_delete` / `can_like` |

`id` / `reply_id` / `parent_comment_id` / `author` / `like_count` / `created_at` / `edited_at` 不变。

### 3.2 `capsForComment`

放 `internal/topic/apiv1/caps.go`，与 `capsForTopic` / `capsForReply` 同文件同风格：

- `can_edit`：作者本人，或持 `perm.CommentTopicEdit`；话题必须可见且未隐藏；
- `can_delete`：作者本人，或持 `perm.CommentTopicDelete`；
- `can_like`：已登录**且不是自己的评论**（`SELF_LIKE_FORBIDDEN` 的镜像）。

### 3.3 写面请求体

- 发表：`{ "text": "…", "parent_comment_id": "12" | null }`。`reply_id` 在路径上；**`topic_id` 与 `target_user_id` 不在请求体里**，服务端推导。
- 编辑：`{ "text": "…" }`。
- `text` 的 `maxLength` 是 **1000**，按 K19 作用于原始值。

## 4. 逐条裁决（对普查 §5）

| 普查 | 裁决 |
|---|---|
| A1 target 客户端指定 | 服务端推导。旧面已于 2026-09-22 修掉；v1 请求体里根本没有这个字段。6 条历史行由迁移 101 校正 |
| A2 `reply_id` 与 `topic_id` 不互校 | 统一走 `visibleReply`，它一并查 reply status、reply↔topic 一致性、作者可渲染性。**不得另写一套 `requireTopicRead`** |
| A3 删评论的扣分门槛 | **用户拍板（2026-09-22）：扣分永不阻断删除。** 删除照常发生，扣分尽力而为、扣到 0 为止，不再因余额不足回滚。作者自删与版主删都适用 |
| A4 发分在提交前 | 收成 `pendingAward`，提交后 `flushAwards`，复用 `App.TopicAward`（W3/W4 那一套）。**不得再用 `InteractionHelpers.AdjustMoemoepoint`** |
| A5 写面不查可见性 | 同 A2 |
| A6 评论/回复不存在回 500 | `404 NOT_FOUND`；父评论不属于本楼回 `422 VALIDATION_FAILED`，`errors[]` 指向 `parent_comment_id` |
| A7 `UpdateComment` 漏 `parent_comment_id` | v1 编辑回完整 `Comment`，不存在这个问题 |
| A8 吞错误 → 静默 0 | 一律上抛；读失败是 `500`，不是 0 |
| A9 没有幂等键 | 发表必填 |
| A10 `topic_reply.comment_count` | 生产实测：迁移 004/020 在 2026-06-05/06 回填过一次，此后**代码里无人维护**（全仓无一处写它），之后新建的 14865 条回复里只有 1 条带非零值（id 14751，原因未查明，值恰好正确）。当前 420 条该有值却是 0、18 条值是错的。**走 deploy-then-drop**：迁移 101 不动它，上线后手动跑 102 删列 |
| A11 索引缺口 + 每行子查询 | 迁移 101 加 `topic_comment (topic_reply_id, created, id) WHERE status = 0`；`like_count` 改一次聚合 |
| A12 旧读面缺决胜键 | v1 的 `ListByReplyIDs` 已有 `created ASC, id ASC`，不动 |
| A13 `updated` NOT NULL 无默认 | **两张表都有这个坑**。裸 SQL 插入必须显式写 `created, updated`——这正是 W4 让收藏和推每次 500 的那个坑 |
| A14 `text` 纯文本承诺与实际不符 | **用户拍板（2026-09-22）：承认 token 并结构化解析。** 即 K20 |
| A15 `in_reply_to_user` 是契约谎言 | 服务端推导，见 §3.1 |
| A16 通知去重不一致 | **本波不改**。`createDedupMessage` 是全站共用件，改它是跨域行为变更。照现状实现，在此记下 |
| A17 `locate` 的 `page` 算错 | 不搬进 v1。locate 归读面轨，届时只发 `floor` |
| A18 trust 检查在可见性之前 | 调序：先判可见性与权限，再跑内容检查。按 K18，编辑时正文未变则完全不跑 |

## 5. 预分配（轨只填，不得自行挑号）

- **迁移号：101**，且只有 101。内容：(a) 校正 6 条 `target_user_id` 指错的顶层评论；(b) 加 `idx_topic_comment_reply_created_id`；(c) 不动 `topic_reply.comment_count`。
- **102 由督查在上线后手动跑**（deploy-then-drop 删 `topic_reply.comment_count`），不在本轨交付物里。
- **错误码：一个都不新增。** 需要的全都已注册：`NOT_FOUND`、`VALIDATION_FAILED`、`PERMISSION_REQUIRED`、`CONTENT_REJECTED`、`SELF_LIKE_FORBIDDEN`、`ACCOUNT_BANNED`、`SERVICE_UNAVAILABLE`、`IDEMPOTENCY_*`。`SELF_LIKE_FORBIDDEN` 的注册描述原文已含 "or comments"。**若发现确实缺，停下来报告，不要自己往 `registry.go` 或 `zh-CN/problem.json` 加。**
- **`app.go` 字段：不新增。** 评论属于话题域，落在 `internal/topic/apiv1/`，复用现有 `App.TopicAward`。
- **必须保留的入口**：`CommentService.ModerationRemove`（`app.go:572` 的 trust 处置适配器在引用）。

## 6. 本波不做（记下，别顺手改）

- `commented` 通知的 `?comment=` 回填（12598 条）——另起一轮 `cmd/backfill-message-links`。
- `liked` 通知的去重与不可撤语义（A16）。
- `GET /topic/:tid/reply/locate` 的 v1 化。
- `topic_reply.comment_count` 的实际 drop（102，上线后手动）。

## 7. 变异题（督查出题，轨不得自行增删）

每条都必须能让**某个测试变红**。编译不过的不算变异，要换等价破坏法。

| # | 改动 | 应当杀死它的断言 |
|---|---|---|
| M1 | `visibleReply` 里去掉 reply↔topic 一致性检查（改成恒真） | 往读不到的话题里写评论应得 404 |
| M2 | 把 `in_reply_to_user` 改回读库里的 `target_user_id` 列 | 6 条历史行之一的推导值断言 |
| M3 | 发表时父评论不属于本楼也放行 | `422` + `errors[]` 指向 `parent_comment_id` |
| M4 | 删除时把「扣分不阻断」改回「余额不足则回滚」 | 作者余额 0 时版主删除仍须 204 |
| M5 | `flushAwards` 挪到 commit 之前 | 事务失败时不得发分（照 W4 的 `TestV1EngageAwardsWaitForTheCommit` 写法） |
| M6 | 点赞插入去掉 `ON CONFLICT DO NOTHING RETURNING id` 的「真的动了行才发分」判断 | 重复 `PUT` 不得重复发分、`like_count` 不得二次自增 |
| M7 | 裸 SQL 插入去掉 `updated` 列 | 发表应 201 而不是 500（A13） |
| M8 | 内容文档管线改成跑完整 Markdown 解析 | 含 `*` 的评论必须原样出 `text` 节点，不得出 `emphasis` |
| M9 | 丢掉换行 → 不产出 `break` 节点 | 含换行的评论的文档结构断言 |
| M10 | `/image/<hash>` token 不解析、原样留在 `text` 节点里 | token 必须成为 `image` 节点（K20） |
| M11 | `can_like` 对自己的评论也返回 true | 自赞能力断言 + `SELF_LIKE_FORBIDDEN` 用例 |
| M12 | 编辑时即使正文未变也跑内容检查 | K18：正文未变不得产生 trust 调用 |
| M13 | `text` 的 `maxLength` 改成先去空白再判长度 | K19：1000 字 + 首尾空格须 `422` |

## 8. 九条闸

照 [04-parallel-tracks.md](../04-parallel-tracks.md) §5，一条不少。特别提醒：

- 第 4 条——**自己的临时库**，`KUN_REQUIRE_TEST_DB=1`，`-count=1 -p 1`。没有库的执行者只能产出草稿（W4 的收藏 500 就是这么来的）。
- 第 8 条——网页改完要**真的 SSR 渲染一遍**评论区。`vue-tsc` 永远不解析模板里的组件。
- 网页侧：`Comment.vue:168` 的 `{{ comment.text }}` 换成 `~/components/content/Document.vue`（回复已在用）；预览用 `contentPlainText()`。
- 旧路由删除与基线下调**不在本轨**，由督查在验收后统一做。
