# W5 普查 · 话题评论（topic comments）

> 只读普查，2026-09-22。代码读法，没有跑库、没有起服务。对应 README 波次看板里 W5 的「评论（含评论点赞与按评论定位）」那一格。
> 数据库行数留空，末尾列出需要用户在生产库补的确切计数。

## 0. 范围与端点清单

评论族一共 **5 条旧路由**：4 条写面全部挂在 `/api/topic/:tid/comment*`，第 5 条是与回复共用的定位端点，`?comment=` 那一半属于本域。

| # | 方法 + 路径 | handler | 中间件链 |
|---|---|---|---|
| 1 | `POST /api/topic/:tid/comment` | `comment_handler.go:25` `CreateComment` | cors → NamePreference → OptionalAuth → **Auth** |
| 2 | `PUT /api/topic/:tid/comment` | `comment_handler.go:47` `UpdateComment` | 同上 |
| 3 | `PUT /api/topic/:tid/comment/like` | `comment_handler.go:67` `ToggleCommentLike` | 同上 |
| 4 | `DELETE /api/topic/:tid/comment` | `comment_handler.go:85` `DeleteComment` | 同上 |
| 5 | `GET /api/topic/:tid/reply/locate?comment=` | `reply_handler.go:57` `GetReplyLocate` | cors → NamePreference → OptionalAuth |

注册点：`apps/api/internal/app/router.go:268-271`（写面，在 **THE AUTH BOUNDARY** 之下）与 `router.go:215`（locate，在 optAuth 段）。
`routes.golden` 行号：22 / 256 / 317 / 318 与 164。

**评论的读面已经在 W2 迁走**，不在本次迁移范围内，但本域必须知道它长什么样（见 §2）：`GET /api/v1/topics/{topic_id}/replies` 与 `GET /api/v1/replies/{reply_id}` 内嵌 `Reply.comments: [Comment]`，由 `apps/api/internal/topic/repository/v1_read.go:18` `ListByReplyIDs` + `apps/api/internal/topic/apiv1/assemble.go:191` `mapComments` 产出。**旧的读路径 `CommentRepository.FindCommentsByReplyIDs`（`comment_repo.go:36`）与旧 DTO `dto.TopicCommentResponse` 仍活着**，因为 `ReplyService.buildReplyResponses`（`service/mapper.go:125`）还在服务旧的 `GET /api/topic/:tid/reply`、`/reply/detail`。这两条旧回复读面属于回复轨，但**删它们的时候会把本域的旧读路径一起带走**。

不在本域（另轨）：`/api/galgame/**` 社区原语评论、`/api/galgame-{quiz,rating,resource}`、`/api/toolset`、`/api/website` 的资源评论；`GET /api/user/:id/comments`（用户轨，但读同一张表，见 §3 末尾）；`GET /api/search` 的评论段（搜索轨，同表）。

---

## 1. 逐端点

### 1.1 `POST /api/topic/:tid/comment` — 发表评论

**链路**
`handler.CommentHandler.CreateComment`（`comment_handler.go:25`）
→ `service.CommentService.CreateComment`（`service/comment_service.go:53`）
→ `repository.CommentRepository.FindCommentByID`（`comment_repo.go:65`，仅当带 `parent_comment_id`）
→ `repository.NewTopicRepository(...).FindByID` + `requireTopicRead`（`service/read_decision.go:15`）
→ `repository.CommentRepository.CreateComment`（`comment_repo.go:77`）
→ `service.recomputeTopicCounts`（`service/interaction_helpers.go:84`）
→ `InteractionHelpers.AdjustMoemoepoint` / `CreateReplyMessage`（`interaction_helpers.go:15` / `:32`）
→ `userclient.Client.Hydrate`

**请求**

| 位置 | 名字 | 真实类型 | 必填 | 说明 |
|---|---|---|---|---|
| path | `:tid` | — | — | **完全不读**。handler 从不取 `c.Params("tid")` |
| body | `topic_id` | int | 必填 `min=1` | 真正决定可见性判定的那个 id |
| body | `reply_id` | int | 必填 `min=1` | 落库成 `topic_reply_id`，**不做任何校验** |
| body | `target_user_id` | int | 必填 `min=1` | **不做任何校验**，客户端自选 |
| body | `content` | string | 必填 `min=1,max=1007` | 服务端跑 `markdown.NormalizeStoredContent` |
| body | `parent_comment_id` | *int | 可选 `min=1` | 只验「存在且 `TopicReplyID == reply_id`」 |

DTO：`apps/api/internal/topic/dto/reply_dto.go:71`。解析器 `utils.ParseAndValidate`（`pkg/utils`），绑定失败 → `400 {code:233,message:"请求格式错误"}`，校验失败 → `400 {code:233,message:<中文逐字段串联>}`。

**响应**（`dto.TopicCommentResponse`，`reply_dto.go:88`，外面包 `response.OK` 信封 `{code:0,message:"成功",data:{…}}`）

```
id, reply_id, topic_id, parent_comment_id,
user{id,name,avatar}, target_user{id,name,avatar},
content, is_liked(恒 false), like_count(恒 0),
created, edited(恒缺席 → null)
```

**网页其实不消费这个响应体**：`Panel.vue:42` 只看它是否真值，然后 `emit('getComment', comment)`；接收方 `Comment.vue:62` `handleNewComment()` 与 `Footer.vue:26` `handleNewComment()` 都把参数丢掉，改为 `refreshReply(replyId)` 重新拉 v1 的 `GET /replies/{reply_id}`。所以这个 DTO 只剩「非 null 即成功」这一个语义。

**错误路径**

| 触发 | 状态 | 体 |
|---|---|---|
| 无会话 / 会话过期 | 401 | `{code:205,message:"用户登录失效"}` |
| 封禁 | 403 | `{code:234,…}`（Auth 中间件产出） |
| body 绑定失败 | 400 | `{code:233,message:"请求格式错误"}` |
| 字段校验失败 | 400 | `{code:233,message:"…"}` |
| `parent_comment_id` 不存在 / 不属于该 reply | 400 | `{code:233,message:"回复的评论不存在"}` |
| trust `deny` | 422 | `{code:233,message:"内容包含违禁词，无法发布"}`（`gate.ErrContentBlocked`） |
| `topic_id` 查不到 | 404 | `{code:233,message:"未找到该话题"}` |
| 话题隐藏 / 访问范围不够 | 404 | `{code:233,message:"未找到该话题"}` |
| 读 access grant 失败 | 500 | `{code:233,message:"获取话题权限失败"}` |
| **`reply_id` 不存在** | **500** | `{code:233,message:"发表评论失败"}` ← FK `topic_comment_topic_reply_id_fkey` 抛 23503，被吞成 Internal |
| 事务内任一步失败（含发通知失败） | 500 | `{code:233,message:"发表评论失败"}` |

**鉴权 / 权限**：匿名不可。发表本身不查任何权限键，只要能读到 `topic_id` 那条话题即可。没有用到 `user.Can`/`CanModerate`，也没有用 `perm.CanUser` / `role.Can*` —— 这一条是干净的，Bearer 请求走这里不会拿到 staff 能力（它压根不需要）。

**可见性**：只对 **body 里的 `topic_id`** 跑 `requireTopicRead`（status==1 需作者或 `topic.view_hidden`；`access_scope` 走 `access.Allowed`）。**不检查**：`reply_id` 是否属于该话题、`topic_reply.status`、`parent_comment.status`、`parent_comment.topic_id`、话题作者是否被封禁（v1 的 `visibleTopic` 有 `rejectUnrenderableAuthor`，这里没有）、`target_user_id` 是否真的存在。

**幂等**：**没有**。`POST /topic` 与 `POST /topic/:tid/reply` 都挂了 `middleware.Idempotent`，唯独评论没有（`router.go:255` / `:265` 对比 `:268`）。弱网重试一次 = 两条评论 + 两次萌萌点 + 两条通知。

**副作用清单**（全在一个事务闭包里，但奖励是事务内 `go` 出去的）

1. 插 `topic_comment` 一行。
2. `recomputeTopicCounts(tx, topicID)`：重算 `topic.reply_count` 与 `topic.comment_count`，**只按 `topic_id` 数**。
3. 三个月窗口内 `UPDATE topic SET status_update_time = now() WHERE id = ? AND created > BumpCutoff(now)`（顶帖）。
4. `targetUserID != userID` 时：`AdjustMoemoepoint(targetUserID, +1, content_approved, ref="topic_reply:<reply_id>")`。
5. 同条件下：`CreateReplyMessage(sender=作者, receiver=target, type="commented", content=前 233 字, link="/topic/<tid>?comment=<cid>")` —— **不去重**（与点赞通知不同）。
6. 提交后：`scan.ScanBg("forum_comment", <id>, content, authorID)`。
7. DB 触发器 `trg_feed_topic_comment`（迁移 034，056 改写）写 `feed_activity` 一行 `TOPIC_COMMENT_CREATION`。

---

### 1.2 `PUT /api/topic/:tid/comment` — 编辑评论

**链路** `comment_handler.go:47` → `service/comment_service.go:148` → `FindCommentByID`（`comment_repo.go:65`）→ `UpdateCommentContent`（`comment_repo.go:81`）→ `CountCommentLikes` + `FindCommentLikeStatus` + `Hydrate`。

**请求**：`:tid` 不读。body `dto.UpdateCommentRequest`（`reply_dto.go:83`）= `{comment_id:int 必填 min=1, content:string 必填 min=1,max=1007}`。

**响应**：同 `TopicCommentResponse`，但**服务端构造时漏掉了 `ParentCommentID`**（`comment_service.go:191-202` 的结构体字面量里没有这个字段），所以编辑后返回的 `parent_comment_id` 恒为 `null`，与实际存库值不符。网页因为走 `refreshReply` 重拉，掩盖了这个 bug。

**错误路径**

| 触发 | 状态 | 体 |
|---|---|---|
| 未登录 / 封禁 | 401 / 403 | `205` / `234` |
| body 校验 | 400 | `233` |
| 评论不存在 | 404 | `{code:233,message:"未找到该评论"}` |
| 非作者且无 `comment.topic.edit` | 403 | `{code:233,message:"您没有权限编辑此评论"}` |
| trust `deny` | 422 | `{code:233,message:"内容包含违禁词，无法发布"}` |
| 写库失败 | 500 | `{code:233,message:"编辑评论失败"}` |

**鉴权 / 权限**：`user.Can(perm.CommentTopicEdit)`，在 handler 里算好再传进 service（`comment_handler.go:59`）。用的是 **`UserInfo.Can`**，正确 —— Bearer 请求 `viaBearer==true` 时 `Can` 恒 false，staff 能力不会经 Bearer 泄漏。

**可见性**：**一条都不查**。不查评论 `status`（trust 隐藏的评论照样能编辑）、不查所属话题是否隐藏/受限、不查话题作者封禁、`:tid` 忽略。

**别的毛病**
- trust 打分用的 `authorID` 是**原作者**（`comment.UserID`），不是发起编辑的人（`comment_service.go:163`）。版主改别人的评论会算到被改者头上。
- `likeCount, _ :=` 与 `likedMap, _ :=`（`:185`、`:186`）吞错误 → DB 抖动时静默返回 `like_count: 0`、`is_liked: false`。
- 更新只写 `content` 与 `edited` 两列（`comment_repo.go:81` 的 `Updates(map)`），GORM 的 `Updates(map[string]any)` 仍会自动带 `updated`，所以 `updated` 列会动，`edited` 才是「作者改过正文」的真信号（迁移 014 的原话）。

---

### 1.3 `PUT /api/topic/:tid/comment/like` — 评论点赞（切换）

**链路** `comment_handler.go:67` → `service/comment_service.go:205` → `FindCommentByIDTx` / `FindCommentLike` / `CreateCommentLike` / `DeleteCommentLike`（`comment_repo.go:85`、`:91`、`:97`、`:101`）→ `AdjustMoemoepoint` + `createDedupMessage`。

**请求**：`:tid` 不读。body `dto.CommentInteractionRequest`（`reply_dto.go:79`）= `{comment_id:int 必填 min=1}`。

**响应**：`response.OKMessage(c,"操作成功")` → `{code:0,message:"操作成功"}`，**没有 `data`**。`kunFetch` 在 `data === undefined` 时回退到 `resp.message`（`apps/web/app/utils/kunFetch.ts:219`），所以前端拿到字符串 `"操作成功"`，靠它的真值判断成功。响应体里**不含新的 `like_count` 与 `has_liked`**，客户端只能乐观更新（`Like.vue:27` 的 `revert`）。

**错误路径**

| 触发 | 状态 | 体 |
|---|---|---|
| 未登录 / 封禁 | 401 / 403 | `205` / `234` |
| body 校验 | 400 | `233` |
| 给自己的评论点赞 | 400 | `{code:233,message:"您不能给自己的评论点赞"}`（用 `gorm.ErrInvalidData` 当信号跨出事务） |
| **评论不存在** | **500** | `{code:233,message:"操作失败"}` ← `FindCommentByIDTx` 的 `ErrRecordNotFound` 没被区分 |
| 并发重复点赞撞唯一索引 | **500** | `{code:233,message:"操作失败"}` ← 23505 |
| 其它 | 500 | `{code:233,message:"操作失败"}` |

**鉴权 / 权限**：登录即可，不查任何权限键，也不查可见性。

**可见性**：**零检查**。不查评论 `status`、不查 reply `status`、不查话题 `status` / `access_scope` / 作者封禁、`:tid` 忽略。任何登录用户只要猜到 `comment_id` 就能给一条隐藏话题里的评论点赞，并给它作者 +1 萌萌点、发一条通知。和 W4 记录的「回复与评论互动什么都不查」是同一个洞，W4 明确把评论点赞留给了 W5。

**副作用**
- 点：插 `topic_comment_like`；`AdjustMoemoepoint(comment.UserID, +1, liked, ref="topic_comment:<cid>")`；`createDedupMessage(type="liked", link=BuildTopicLink(topicID,0,commentID))` —— **按 (sender,receiver,type,link) 去重**，所以同一个人第二次点赞不会再发通知，取消点赞也不会删掉已发的通知。
- 取消：删行；`AdjustMoemoepoint(comment.UserID, -1, liked, 同 ref)`。
- `like_count` **没有物化列**，每次读都是 `(SELECT COUNT(*) FROM topic_comment_like WHERE topic_comment_id = tc.id)` 相关子查询（`comment_repo.go:44`、`v1_read.go:26`）。所以点赞这一族**不存在计数漂移**，代价是每条评论一次子查询。

---

### 1.4 `DELETE /api/topic/:tid/comment` — 删除评论

**链路** `comment_handler.go:85` → `service/comment_service.go:250` → `FindCommentByID` → `CountCommentLikes` → `StateRepository.LockForUpdate` → `DeleteCommentLikesForComment` → `DeleteCommentByID` → `recomputeTopicCounts` → `AdjustMoemoepoint`。

**请求**：`:tid` 不读。**查询参数 `commentId`（camelCase！）**，`strconv.Atoi` 失败 → 400。全站唯一一处 camelCase 查询参数的写法在这里与 `/reply/detail?replyId=` 成对出现。网页照发（`Delete.vue:46` `query: { commentId: … }`）。

**响应**：`response.OKMessage(c,"评论已删除")` → `{code:0,message:"评论已删除"}`，无 `data`。

**错误路径**

| 触发 | 状态 | 体 |
|---|---|---|
| 未登录 / 封禁 | 401 / 403 | `205` / `234` |
| `commentId` 不是数字 / 缺席 | 400 | `{code:233,message:"无效的评论 ID"}` |
| 评论不存在 | 404 | `{code:233,message:"未找到该评论"}` |
| 非作者且无 `comment.topic.delete` | 403 | `{code:233,message:"您没有权限删除此评论"}` |
| **作者缓存余额 < 扣分** | 400 | `{code:233,message:"萌萌点不足, 无法删除此评论"}` |
| 其它 | 500 | `{code:233,message:"删除评论失败"}` |

**鉴权 / 权限**：`user.Can(perm.CommentTopicDelete)`（`comment_handler.go:96`），写法正确。

**可见性**：不查话题、不查评论 status。

**扣分公式**（`comment_service.go:259-263`）

```
likeCount, _ = CountCommentLikes(commentID)      // 错误被吞 → 0
penalty = 3
if comment.UserID == userID && !canModerate {    // 「我自己删，且我不是版主」
    penalty = 3 * (likeCount + 1)
}
```

被扣的**永远是评论作者** `comment.UserID`，理由 `content_removed`，ref `topic_comment:<cid>`。前端 `Delete.vue:22` 镜像了这个公式（`3 * (like_count + 1)`）并按 `isCommonUser` 分支提示文案，两边一致。

**副作用**：`LockForUpdate` 锁 `kungal_user_state` 行 → 删 `topic_comment_like` → 删 `topic_comment` → `recomputeTopicCounts(comment.TopicID)` → 扣分。DB 层还有两条兜底：`topic_comment_like_topic_comment_id_fkey ON DELETE CASCADE`（所以显式删 like 是重复动作），以及 `fk_topic_comment_parent ON DELETE SET NULL`（迁移 037）——**删父评论会把子评论重挂成顶层**，不是级联删。

---

### 1.5 `GET /api/topic/:tid/reply/locate?comment=<id>` — 按评论定位

与回复定位共用一条路由，`?comment=` 那一半是本域唯一的读端点。

**链路** `reply_handler.go:57` → `service.ReplyService.LocateReply`（`service/reply_service.go:60`）→ `TopicRepository.FindByID` + `requireTopicRead` → `ReplyRepository.FindReplyFloorByCommentID`（`repository/reply_repo.go:76`）→ `LocateReplyPageByFloor`（`reply_repo.go:58`）。

**请求**：path `:tid`（**这条真的读**）；query `reply`（int，可选）、`comment`（int，可选），两者都 ≤0 → 400。`strconv.Atoi` 的错误被丢弃（`reply_handler.go:62-63`），非数字等价于 0。`comment` 优先于 `reply`：给了 `comment` 就覆盖掉 `floor`。`limit` 写死 30（`reply_handler.go:68`）。

**响应**：`dto.ReplyLocateResponse`（`reply_dto.go:12`）= `{page,floor,reply_id,comment_id}`。

**错误路径**：话题不存在 / 不可见 → 404「未找到该话题」；评论查不到 → 404「评论不存在或已删除」；`floor<=0` → 400；SQL 失败 → 500「定位评论失败」/「定位回复失败」。

**可见性**：对话题跑了 `requireTopicRead`（比写面好），但 `FindReplyFloorByCommentID` 的 SQL **不过滤 `c.status` 也不过滤 `r.status`**，所以 trust 隐藏的评论仍能定位到楼层，而前端滚过去找不到锚点，落到 `Detail.vue:92` 的「目标回复或评论可能已被删除」。

**已经成为死重量的一半**：`LocateReplyPageByFloor` 算出的 `page` 现在没有任何消费者 —— 回复读面在 W2 换成了游标 + `from_floor`，`Detail.vue:58` 只取 `located.floor`。而且这个 `page` 本身是错的：它 `COUNT(*) FROM topic_reply WHERE topic_id=? AND floor<=?` **不过滤 `status=0`**，隐藏回复会把页码顶大。v1 化的时候这个字段应该直接不要。

---

## 2. 与已迁移话题代码共享的东西（**不得**重复实现或单方面改名）

### 2.1 v1 表示层（W2 已上线，评论读面就是它）

| 符号 | 位置 | 说明 |
|---|---|---|
| `apiv1.Comment` | `internal/topic/apiv1/reply.go:35` | `object:"comment"`、`id`/`reply_id`/`parent_comment_id` 全是 `repr.DecimalID`、`author`/`in_reply_to_user` 是 `repr.UserRef`、正文字段叫 **`text`**（不是 `content`，也不是 `content_markdown`）、`like_count`、`created_at`/`edited_at`、`viewer` |
| `apiv1.CommentViewer` | `reply.go:49` | 只有 `has_liked`。**W5 要加 `can_edit` / `can_delete` / `can_like` 就是往这里加字段**，会同时改 `Reply.comments` 的形状 |
| `apiv1.Reply.Comments` | `reply.go:21` | `json:"comments"`，`maxItems:"1000"`，doc 里已承诺「oldest first / 封禁作者的评论被剔除 / 父评论可能缺席但 `parent_comment_id` 保留」 |
| `repository.CommentListRow` + `ListByReplyIDs` | `repository/v1_read.go:5,18` | v1 的读行。`ORDER BY tc.created ASC, tc.id ASC`（**已有决胜键**，旧的 `FindCommentsByReplyIDs` 没有） |
| `CommentRepository.FindCommentLikeStatus` | `comment_repo.go:61` | v1 的 `loadReplyExtras`（`apiv1/assemble.go:54`）和旧 mapper 共用 |
| `replyPack.mapComments` | `apiv1/assemble.go:191` | 评论 → `Comment` 的唯一映射点 |
| `repr.ID` / `repr.UserRef` / `repr.DeletedUserRef` / `repr.Timestamp(Ptr)` / `optID` | `internal/apiv1/repr`、`assemble.go:228` | — |

### 2.2 service / repository 共享件

| 符号 | 位置 | 谁在用 |
|---|---|---|
| `CommentRepository` 整个 | `internal/topic/repository/comment_repo.go` | 旧评论 service、旧回复 mapper、v1 读面、trust enforcer（`app.go:571`） |
| `service.RecomputeTopicCounts`（导出版） | `internal/topic/service/interaction_helpers.go:92` | v1 写面 `apiv1/write_reply.go:197` 已经在用；评论的增删也必须调它 |
| `InteractionHelpers.AdjustMoemoepoint` / `CreateReplyMessage` / `createDedupMessage` / `truncate` | `interaction_helpers.go:15/32/62/110` | 话题、回复、评论三家共用。**W3/W4 的 v1 侧已经不用它们了**，改成了 `pendingAward` + `App.TopicAward` + 提交后 `flushAwards`（`apiv1/write_awards.go`）。W5 必须走 v1 那一套，别再往 `InteractionHelpers` 上加东西 |
| `requireTopicRead` / `access.Allowed` / `access.Snapshot` | `service/read_decision.go:15`、`topic/access/` | 旧面用 `requireTopicRead`，v1 用 `Service.visibleTopic` / `visibleReply`（`apiv1/visible.go:27,59`）。**W5 用 `visibleReply`**，它顺带查了 reply status、reply↔topic 一致性、作者可渲染性 |
| `capsForReply` / `capsForTopic` | `apiv1/caps.go:43,19` | W4 已经接进读面。评论的 `can_edit`/`can_delete` 应当新增 `capsForComment`，与它们同文件同风格 |
| `model.TopicComment` / `model.TopicCommentLike` | `internal/topic/model/topic.go:175,195` | — |
| `moemoepoint.Ref` / `Key` / `KeyNonce` | `internal/moemoepoint/pusher.go:125,129` | W4 已经把互动改成稳定键（`kungal:liked:topic_reaction_{row}`），评论点赞现在还是 `KeyNonce` |
| `gate.SubjectKindTopicComment = "forum_comment"` | `internal/trust/gate/scan.go:15` | trust 注册表 `app.go:570` 的 key，和前端 `ReportButton :subject-kind="forum_comment"`（`Comment.vue:225`）对齐，**不能改** |
| `CommentService.ModerationRemove` | `comment_service.go:299` | trust 回调的 `Remove` 适配器（`app.go:572`）。W5 重写 service 时必须保留这个入口 |
| `perm.CommentTopicEdit` / `CommentTopicDelete` | `pkg/perm/perm.go:17,18` | 三处镜像：Go `moderatorPerms`、`apps/web/app/composables/useCan.ts:13-14`、`apps/web/app/constants/permission.ts:36-37`，由 `pkg/perm/frontend_mirror_test.go` 钉死 |

### 2.3 待删的旧件（删完要下调两个基线）

- Go：`dto.CreateCommentRequest` / `UpdateCommentRequest` / `CommentInteractionRequest` / `TopicCommentResponse`（`reply_dto.go:71-100`）、`handler.CommentHandler` 整个文件、`CommentRepository.FindCommentsByReplyIDs` + `CommentRow`（随旧回复读面一起）。
- TS：`apps/web/shared/types/topic-comment.ts`（整文件）、`apps/web/shared/types/topic-reply.ts:17` 的 `comment: TopicComment[]`。
- 基线：`apps/api/internal/app/testdata/legacy_route_baseline` = **319**（删 4 条 → 315；locate 一起删 → 314）；`apps/web/tests/api/legacy-fetch-baseline` = **324**（本域占 5 个调用点）。

---

## 3. 数据库表与列

### `topic_comment`（迁移 000 建表 / 014 加 `edited` / 037 加 `parent_comment_id` / 055 加 `status`）

| 列 | 类型 | 本域**读** | 本域**写** |
|---|---|---|---|
| `id` | `integer` PK，序列 | ✅ | 插入时由序列给 |
| `content` | `varchar(1007) NOT NULL DEFAULT ''` | ✅ | create / update |
| `topic_id` | `integer NOT NULL`，FK→`topic(id)` CASCADE | ✅ | create |
| `topic_reply_id` | `integer NOT NULL`，FK→`topic_reply(id)` CASCADE | ✅ | create |
| `user_id` | `integer NOT NULL`（FK 已被迁移 019 删） | ✅ | create |
| `target_user_id` | `integer NOT NULL`（FK 已被 019 删） | ✅ | create |
| `parent_comment_id` | `integer NULL`，FK→自身 **ON DELETE SET NULL** | ✅ | create |
| `edited` | `timestamptz NULL` | ✅ | update |
| `status` | `smallint NOT NULL DEFAULT 0`（0=正常，1=trust 隐藏） | 读面过滤 `=0`；写面**都不看** | `SetStatus`（trust `hide`） |
| `created` | `timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP`（022 转过 tz） | ✅ | GORM `autoCreateTime` |
| `updated` | **`timestamptz NOT NULL`，没有 DEFAULT** | ❌ | GORM `autoUpdateTime` |

**索引**：`idx_topic_comment_created (created DESC)`（016）、`idx_topic_comment_topic_created (topic_id, created DESC)`（038）、`idx_topic_comment_parent (parent_comment_id) WHERE NOT NULL`（037）、`idx_topic_comment_content_trgm` GIN（088）。**没有 `(topic_reply_id, created, id)` 索引** —— 而读面正是按 `topic_reply_id IN (…) ORDER BY created, id`。

**触发器**：`trg_feed_topic_comment`（`AFTER INSERT OR UPDATE OR DELETE`，034 + 056 + 090），写 `feed_activity` 的 `TOPIC_COMMENT_CREATION`，正文取 `SUBSTRING(content,1,100)`，`status<>0` 时改为 `feed_delete`。

### `topic_comment_like`（迁移 000）

| 列 | 类型 | 读 | 写 |
|---|---|---|---|
| `id` | `integer` PK | — | 序列 |
| `topic_comment_id` | `integer NOT NULL`，FK→`topic_comment(id)` **ON DELETE CASCADE** | ✅ | create / delete |
| `user_id` | `integer NOT NULL`（FK 已被 019 删） | ✅ | create / delete |
| `created` | `timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP` | — | GORM |
| `updated` | **`timestamptz NOT NULL`，没有 DEFAULT** | — | GORM |

**唯一索引** `topic_comment_like_topic_comment_id_user_id_key (topic_comment_id, user_id)` —— 在。

### 本域间接写的列

- `topic.reply_count`、`topic.comment_count`：`recomputeTopicCounts` 全量重算，**按 `topic_id`**。
- `topic.status_update_time`：三个月窗口内的顶帖。
- `kungal_user_state.moemoepoint`：`LockForUpdate` 读，`Awarder` 在 OAuth 回包后镜像写。
- `message`：`type IN ('commented','liked')` 的行。
- `feed_activity`：触发器。

### 只读同一张表的邻域（迁移时会跟着一起动）

| 位置 | 做什么 |
|---|---|
| `internal/user/repository/content_repo.go:172` `FindUserComments` | `GET /api/user/:id/comments`，三个 `type`：默认=我发的、`comment_target`=@我的、`comment_like`=我赞过的。页码分页，`ORDER BY topic_comment.created DESC` **无决胜键**，只过滤 `status=0` 与（SFW 时）`topic.is_nsfw=false`，**不查话题可见性**——隐藏/受限话题里的评论会出现在别人的个人页 |
| `internal/search/repository/search_repo.go:155` `SearchComments` | `GET /api/search`。有可见性谓词（`SharedListPredicate`），`ORDER BY relevance DESC, c.created DESC` 无决胜键 |
| `internal/activity/repository/activity_repo.go:181,714` | 动态流 |
| `internal/ranking/repository/ranking_repo.go:34` | `comment_created` 排行 |
| `internal/user/repository/stats_repo.go:27,92` | 个人统计 |
| `internal/admin/repository/purge_repo.go:33,85,86,147,242,243` | 删号；**删号后会重算 `topic_reply.comment_count`**，见 §5 的 B6 |
| `internal/admin/repository/topic_admin_repo.go:60,97` | 话题硬删与 purge-stats |
| `cmd/backfill-comment-parents`、`cmd/backfill-message-links`、`cmd/rewrite-content-image-refs`、`cmd/purge-staging-verify` | 一次性脚本 |
| `internal/infrastructure/cron/reference_ping.go:55` | 扫全库 text 列，`topic_comment.content` 里的 `/image/<hash>` token 会被当作引用 ping 上去 —— 所以正文里的图 token 是**有意义**的，不是残渣 |

---

## 4. 命名问题（照 01 §3 的对照表逐条）

| 现状 | 出处 | 问题 | v1 应当 |
|---|---|---|---|
| 路径 `:tid` | 4 条写面 + locate | 禁用名；而且**写面根本不读它** | `{topic_id}`，或者干脆不要（评论自有 id） |
| `POST/PUT/DELETE` 都打在 `/topic/:tid/comment` 这一个 URL 上 | router.go:268-271 | 集合与成员共用一个 URL，成员 id 一会儿在 body、一会儿在 query | `POST /replies/{reply_id}/comments`、`PATCH|DELETE /comments/{comment_id}` |
| `PUT .../comment/like` 是切换 | `:318` | 违反 PUT 幂等；重放撤销 | `PUT` / `DELETE /comments/{comment_id}/like`（K16 槽位） |
| query `commentId` | `comment_handler.go:91` | 全站唯一 camelCase 查询参数 | 进路径 `{comment_id}` |
| `content`（正文） | `CreateCommentRequest` / `UpdateCommentRequest` / `TopicCommentResponse` | 与话题/回复的 Markdown `content` 同名不同型（评论是纯文本） | 读面 v1 已经叫 **`text`**；请求体也应当叫 `text`，别沿用 `content` 也别叫 `content_markdown` |
| `user` / `target_user` | `TopicCommentResponse` | 01 §3 禁用名 `user`；一条资源里有两个人 | v1 已改成 `author` / `in_reply_to_user` |
| `target_user_id` | 请求体 | 名字说「目标用户」，实际含义是「父评论作者，或本层回复的作者」；而且是**客户端算的** | 请求体里**删掉**，服务端从 `parent_comment_id`/`reply_id` 推 |
| `is_liked` | `TopicCommentResponse` | 随查看者变化的字段在顶层 | v1 已改 `viewer.has_liked` |
| `created` / `edited` | 同上 | 禁用名 | v1 已改 `created_at` / `edited_at` |
| `status`（裸 smallint 0/1） | `model.TopicComment` | Problem 的 `status` 是整数，同名不同型 | `state`：`"visible"` / `"hidden"`（封闭枚举）。**注意**：读面现在直接把 status≠0 的行滤掉，所以 v1 也可以选择不下发这个字段 |
| `comment`（数组） | `TopicReplyResponse.Comments json:"comment"` | 单数命名数组 | v1 已改 `comments` |
| `reply_id` vs `topic_reply_id` | DTO 用 `reply_id`，列名 `topic_reply_id`，`CommentRow` 字段 `TopicReplyID` | 一个概念三个名字 | 统一 `reply_id`（v1 读面已是） |
| `parent_comment_id` | — | ✅ 名字对，含义对。保留 | 保留 |
| ref `topic_reply:<reply_id>` 用于「评论得分」 | `comment_service.go:111` | 萌萌点流水里一条评论奖励挂在**回复**上；同一条回复被多人评论时 ref 完全相同 | `topic_comment:<comment_id>` |
| 奖励常量 `constants.RewardReply` | `constants/moemoepoint.go:8` | 评论奖励复用回复的常量名 | 独立 `RewardComment`，或明确说明二者同值 |
| 通知 type 字符串 `"commented"` 手写 | `comment_service.go:114` | 旁边就有 `msgService.NotifyCommented`（`notifier.go:20`）没人用 | 用具名常量；更进一步按 01 §7 下发结构化事件 |

---

## 5. Bug 与可疑处（按严重度）

### A1 · `target_user_id` 完全由客户端指定 → 任意用户 +1 萌萌点 + 任意用户收通知

`CreateCommentRequest.TargetUserID` 只有 `required,min=1`（`reply_dto.go:74`），service 把它直接落库并据此发奖（`comment_service.go:90`、`:110`）：

```go
if userID != targetUserID {
    s.helpers.AdjustMoemoepoint(tx, targetUserID, constants.RewardReply, // = +1
        moemoepoint.ReasonContentApproved, moemoepoint.Ref("topic_reply", replyID))
    s.helpers.CreateReplyMessage(tx, userID, targetUserID, "commented", preview, topicID, 0, comment.ID)
}
```

服务端**从不**核对它等于 `parent_comment.UserID` 或 `topic_reply.UserID`；`topic_comment_target_user_id_fkey` 在迁移 019 里被删掉了，所以连「这个用户存在」都不保证。发表评论不花萌萌点、没有幂等键、路由上没有任何限流器（`router.go` 里唯一一条限流注释是说 check-in **故意不加**）。因此：

- **无上限地给任意 user id 增发萌萌点**（C3 说余额单源在 OAuth，这里是下游在无授权地发行）；
- 无上限地给任意用户发「xxx 评论了你」通知，且这条通知**不去重**（`CreateReplyMessage` 是裸 `tx.Create`，不像点赞走 `createDedupMessage`）；
- 渲染层把 `in_reply_to_user` 画成被指定的那个人（`Comment.vue:132` 还给他做了 `/user/<id>` 链接），等于**可以伪造「我在回复某某」**。

而且这两个值在客户端是**可推导**的：`Footer.vue:93` 传的就是 `reply.author`，`Comment.vue:57` 传的就是 `comment.author`。v1 必须把它从请求体里删掉，由服务端推。

### A2 · `reply_id` 与 `topic_id` 不互校 → 往读不到的话题里写评论

`CreateComment` 对 **body 的 `topic_id`** 跑 `requireTopicRead`，却从不验证 `reply_id` 属于这个话题（`comment_service.go:75-92`）。读面 `ListByReplyIDs` 又**只按 `topic_reply_id` 取**、不按 `topic_id` 过滤（`v1_read.go:27`）。于是：

> 构造 `{ topic_id: <我自己的公开话题>, reply_id: <目标话题某楼的 id>, target_user_id: …, content: … }`，评论就会渲染在目标话题那一楼下面 —— 哪怕目标话题 `status=1`（隐藏）或 `access_scope='users'/'role'` 且我不在授权名单里。

顺带把计数也打歪：`recomputeTopicCounts` 用的是 body 的 `topic_id`，所以被注入的那个话题 `comment_count` 不变，我自己的话题 `comment_count` 凭空 +1。`parent_comment_id` 的校验同理只比 `TopicReplyID`，不比 `topic_id`、不比 `status`。

**W5 的修法**：用 `visibleReply(reply_id)`（`apiv1/visible.go:59`）作为唯一入口，`topic_id` 从 reply 反查，请求体里不再出现 `topic_id`。

### A3 · 版主删不掉穷人的评论；删除的扣分对象是被删者且门槛读的是缓存余额

`DeleteComment`（`comment_service.go:265-292`）在事务里先 `LockForUpdate(comment.UserID)`，再 `if state.Moemoepoint < penalty { 回滚 }`。`penalty` 在「版主删别人」这一支是 3。所以**评论作者余额 < 3 时，版主的删除请求会以 `400「萌萌点不足, 无法删除此评论」` 失败** —— 一条违规评论因为作者穷而删不掉，而错误文案还在说是「您」的萌萌点不足。

而且 `kungal_user_state.moemoepoint` 是 C3 明文规定的**缓存视图**，不是真余额；拿它当硬闸门，等于用可能陈旧的数字挡住一次审核动作。

（对照：trust 的 `remove` 走 `ModerationRemove`（`comment_service.go:299`），它**不扣分也不看余额**。同一个「删评论」动作，人工审核和自动处置两条路的语义不一样。）

### A4 · 萌萌点在事务提交前就发出去了

`InteractionHelpers.AdjustMoemoepoint`（`interaction_helpers.go:15`）忽略 `tx`，直接 `moemoepoint.Award(...)`，而 `Awarder.Award`（`pusher.go:45`）**立刻 `go` 一个协程**去调 OAuth。三处调用点全都在事务闭包里，而且**后面还有会失败的语句**：

- `CreateComment`：奖励在 `:110`，`CreateReplyMessage` 在 `:114`。通知插入失败 → 事务回滚 → **评论没建，目标用户却已经 +1**。
- `ToggleCommentLike`：奖励在 `:221`，`createDedupMessage` 在 `:226`。同样的窗口。
- `DeleteComment`：扣分在 `:285`，之后只剩 COMMIT；COMMIT 失败 → 评论还在，作者已被扣分。

W3 专门为这条加了回归测试（「让 message 插入失败，断言没有发分」），W4 把发分收成 `pendingAward` + 提交后 `flushAwards`。评论这三处还没改。

叠加两点让它更难自愈：幂等键是 `KeyNonce`（`pusher.go:125`，尾巴是 `time.Now().UnixNano()`），所以 OAuth 侧**没有任何去重**；`Award` 是 best-effort，失败只打 `slog.Warn`。点赞/取消反复切换时，只要有一次推送失败，作者余额就永久漂一格。

### A5 · 三条写面完全不查可见性，`:tid` 是装饰

- `UpdateComment`、`ToggleCommentLike`、`DeleteComment` 一次 `requireTopicRead` 都没有；`ToggleCommentLike` 连「评论存不存在」都靠 500 兜。
- 四条写面**都不读 `:tid`**。前端 `Like.vue:9` 与 `Comment.vue:14` 的 `inject<number>('topicId', 0)` 默认值是 0，所以一旦 provide 丢失就会打 `/topic/0/comment/like` —— 和 W4 记录的 `/topic/0/reply/reaction` 是同一件事，**服务端照收**。
- 三条写面都不看 `topic_comment.status`：trust 隐藏（`status=1`）的评论仍可编辑、仍可点赞、仍会给作者发分发通知。读面已经把它滤掉了，所以这是一条「看不见但摸得着」的资源。

### A6 · 评论不存在 / 回复不存在，回 500 而不是 404 / 422

- `ToggleCommentLike`：`FindCommentByIDTx` 的 `gorm.ErrRecordNotFound` 直接冒泡到 `errors.ErrInternal("操作失败")`（`comment_service.go:207-210`、`:244`）。
- `CreateComment`：`reply_id` 不存在时 FK 23503 → `errors.ErrInternal("发表评论失败")`。
- `ToggleCommentLike` 并发双击：唯一索引 23505 → 同样 500。W4 对表情族的裁决是 `INSERT … ON CONFLICT DO NOTHING RETURNING id`，评论点赞照抄即可。

### A7 · `UpdateComment` 的响应漏掉 `parent_comment_id`

`comment_service.go:191-202` 的返回字面量里没有 `ParentCommentID` 字段，所以编辑一条子评论后响应里 `parent_comment_id: null`。当前被前端的「编辑完重拉」掩盖，但任何按响应就地更新的客户端（例如 App）会把这条评论从子层弹回顶层。

### A8 · 吞掉的错误 → 静默的 0

| 位置 | 吞掉什么 | 后果 |
|---|---|---|
| `comment_service.go:185` `likeCount, _ :=` | `CountCommentLikes` | 编辑响应里 `like_count: 0` |
| `comment_service.go:186` `likedMap, _ :=` | `FindCommentLikeStatus` | 编辑响应里 `is_liked: false` |
| `comment_service.go:259` `likeCount, _ :=` | `CountCommentLikes` | **删除扣分从 `3*(n+1)` 悄悄降成 3** |
| `service/mapper.go:125,142` `commentMap, _ :=` | `FindCommentsByReplyIDs` / 点赞态 | 旧回复读面：DB 抖一下，整层评论直接消失、返回 200 |
| `reply_handler.go:62-63` `floor, _ := strconv.Atoi(...)` | 解析错误 | `?reply=abc` 等价于没传 |

（`CommentRow`（`comment_repo.go:19`）还声明了 `UserName` / `UserAvatar` / `TargetUserName` / `TargetAvatar` 四个 SELECT 根本不填的字段，永远是空串。v1 的 `CommentListRow` 已经去掉了。）

### A9 · 没有幂等键

评论创建是唯一一个**没挂 `middleware.Idempotent`** 的用户内容创建端点（对比 `router.go:255` 的话题、`:265` 的回复）。01 §5 K12 要求创建用户内容的 POST **必须携带** `Idempotency-Key`。

### A10 · `topic_reply.comment_count` 是一列从没被维护过的僵尸

`topic_reply.comment_count` 由迁移 002 加上（`NOT NULL DEFAULT 0`）。迁移 020 的文件名叫 `backfill_topic_reply_comment_counts`，但 SQL 里**只 UPDATE 了 `topic`**（`020…up.sql:17-19`），没碰 `topic_reply`。运行时也没有任何地方写它：全仓唯一的写点是 `internal/admin/repository/purge_repo.go:243`，即**删号清理时**才按行重算受影响的回复。

后果：这一列对绝大多数行恒为 0，而被某次删号波及过的回复是准确值 —— 同一列两种含义。`cmd/purge-staging-verify/main.go:103` 还把它当作一致性断言在校验。它目前没有读者（`TopicReplyResponse` 和 v1 `Reply` 都不发它），所以 W5 的正确处置是 **deploy-then-drop 掉这一列**，而不是去补维护它。

### A11 · 索引缺口 + 每行一次相关子查询

读面 `WHERE tc.topic_reply_id IN (…) AND tc.status = 0 ORDER BY tc.created, tc.id`，而 `topic_comment` 上**没有以 `topic_reply_id` 打头的索引**（只有 `created`、`(topic_id, created)`、`parent_comment_id` 局部、content trgm）。一页 30 条回复就是 30 个 reply id 的 IN 扫。叠加每条评论一次 `(SELECT COUNT(*) FROM topic_comment_like WHERE topic_comment_id = tc.id)`。W5 应当补 `topic_comment (topic_reply_id, created, id) WHERE status = 0`，并把点赞数改成一次 `GROUP BY` 聚合而不是 N 次子查询。

### A12 · 旧读面缺排序决胜键

`FindCommentsByReplyIDs`（`comment_repo.go:48`）`ORDER BY tc.created ASC`，没有 `id`。同毫秒写入的两条评论顺序不稳定。v1 的 `ListByReplyIDs` 已经补上 `tc.id ASC` —— 删旧读面时这个差异会自动消失，但**在此之前，旧面和 v1 面对同一批数据可能给出不同顺序**。

### A13 · `updated` 列 `NOT NULL` 且无 DEFAULT —— W4 踩过的那个坑，这里有两张表

`topic_comment.updated` 与 `topic_comment_like.updated` 都是 `timestamptz NOT NULL`，**没有 DEFAULT**（`000_baseline.up.sql:1696`、`:1726`）。现在靠 GORM 的 `autoUpdateTime` 填。W4 验收时 `topic_favorite` / `topic_upvote` 因为 v1 改用裸 SQL INSERT 不写这一列，**第一次真跑就是 6 条红**（`23502`）。W5 若把插入改成 `INSERT … ON CONFLICT`，**必须显式写 `created, updated`**。

### A14 · `Comment.text` 的「纯文本」承诺与实际存的内容不符

v1 的 `Comment.Text` doc 写着「Plain text, not Markdown; render it as text」，前端也确实是 `{{ comment.text }}`（`Comment.vue:168`）。但写面对正文跑了 `markdown.NormalizeStoredContent`（`comment_service.go:60`、`:154`），它会把贴纸站 URL 和图床绝对 URL 重写成 `/image/<hash>` token（`markdown/image_ref.go:95,112`）。所以库里存的可能是 `/image/9f3c…_320` 这种字符串，界面上原样显示成一行路径。

这不是渲染 bug，是**契约描述与数据不一致**：要么承认正文里有 token 并在 v1 里解析它（与 K13 的图片节点一致），要么在写面拒绝/剥离它。顺带一提，这些 token **是**被 `cron.RunReferencePing`（扫全库 text 列）当成引用 ping 上去的，所以不能简单当垃圾删掉，否则图会被 GC 掉。

### A15 · `Comment.in_reply_to_user` 的 doc 是一句数据不保证的话

`reply.go:41` 写着「The user the comment answers: the parent comment's author, or the reply's author for a top-level comment」。实际它就是客户端传的 `target_user_id` 原样回显（A1）。这是一条**已经随 W2 上线的契约谎言**，App 会按它推断关系。

### A16 · 「commented」通知不去重，「liked」通知去重且不可撤

- `commented` 走裸 `tx.Create`（`interaction_helpers.go:42`），每条评论一条通知。
- `liked` 走 `createDedupMessage`（`interaction_helpers.go:62`），按 `(sender_id, receiver_id, type, link)` 去重；取消点赞**不删**这条通知。所以「点赞 → 取消 → 再点赞」只会产生一条通知，而萌萌点来回加减两次。两种行为在同一个域里并存，没有写下理由。
- 两条都绕开了 `msgService.Notifier` 的 `Spec` 通道（`notifier.go`），所以任何在 Notifier 里做的策略（偏好、批处理）对评论域不生效。

### A17 · `locate` 的 `page` 字段既没人用又算错

见 §1.5。`LocateReplyPageByFloor`（`reply_repo.go:58`）`COUNT(*) … WHERE topic_id=? AND floor<=?` **不过滤 `status=0`**，隐藏回复计入页码。当前唯一消费者 `Detail.vue:58` 只读 `floor`，所以这个错误是哑的 —— 但 v1 化时不要把它照搬过去。

### A18 · trust 检查在可见性检查之前

`CreateComment` 先 `s.check.Decision`（一次跨服务调用，`comment_service.go:70`），再查话题、再判可见性（`:75-83`）。对一个根本不可见的话题发评论，会先付一次 trust 往返再拿 404。顺序反了。`UpdateComment` 用原作者 id 而不是编辑者 id 去打分（`:163`）。

---

## 6. W5 实现时的几条硬约束（给实现轨）

1. **入口统一走 `visibleReply` / 新增的 `visibleComment`**，不要再写第二套 `requireTopicRead`。
2. **请求体里删掉 `topic_id` 与 `target_user_id`**，两者都由服务端从 `reply_id` / `parent_comment_id` 推导。
3. **`viewer.can_*` 新增 `capsForComment`**，放 `apiv1/caps.go`，与 `capsForTopic`/`capsForReply` 同风格；`CommentViewer` 加 `can_edit` / `can_delete` / `can_like`（会改 `Reply.comments` 形状，G8 同名 property 一致性需要复查）。
4. **点赞改 `PUT`/`DELETE /comments/{comment_id}/like`**，插入 `ON CONFLICT DO NOTHING RETURNING id`，删除 `DELETE … RETURNING id`，**只有真的动了行才发分**；萌萌点键改 W4 的稳定键式样（`kungal:liked:topic_comment_like_{row}`）。
5. **奖励收成 `pendingAward`，提交后 `flushAwards`**，复用 W3 的 `App.TopicAward`，不要再用 `InteractionHelpers.AdjustMoemoepoint`。
6. **创建必须带 `Idempotency-Key`**。
7. **删除的扣分门槛**：要么改成不阻断（只扣到 0 / 允许为负），要么至少让版主删除完全不走这个门槛。这是一条产品裁决，需要用户拍板。
8. **插入若改裸 SQL，必须显式写 `created, updated`**（A13）。
9. 补索引 `topic_comment (topic_reply_id, created, id) WHERE status = 0`；把 `like_count` 的 N 次子查询改成一次聚合。
10. 迁移里顺手：`topic_reply.comment_count` 走 deploy-then-drop（A10）；如果决定保留 `status`，把它升格成 `state` 枚举。
11. `ModerationRemove` 这个 trust 入口必须保留（`app.go:572` 在引用）。
12. 删旧路由后，`legacy_route_baseline`（319）与 `legacy-fetch-baseline`（324）都要下调，`routes.golden` 重生成。

---

## 7. 调用方清单（file:line）

**网页组件 / composable**

| 文件:行 | 做什么 |
|---|---|
| `apps/web/app/components/topic/comment/Panel.vue:30` | `POST /topic/${topicId}/comment`，body `{topic_id, reply_id, target_user_id, parent_comment_id, content}` |
| `apps/web/app/components/topic/comment/Comment.vue:95` | `PUT /topic/${topicId}/comment`，body `{comment_id, content}` |
| `apps/web/app/components/topic/comment/Like.vue:44` | `PUT /topic/${topicId}/comment/like`，body `{comment_id}` |
| `apps/web/app/components/topic/comment/Delete.vue:44` | `DELETE /topic/${props.topicId}/comment?commentId=…` |
| `apps/web/app/components/topic/detail/Detail.vue:50` | `GET /topic/${topic.id}/reply/locate?comment=…`（只用返回的 `floor`） |
| `apps/web/app/components/topic/comment/threadComments.ts` | 按 `parent_comment_id` 折成两层；父缺席当顶层（有 `threadComments.spec.ts` 钉着） |
| `apps/web/app/components/topic/reply/Reply.vue:150` | `<TopicComment :reply-id="reply.id" :comments-data="reply.comments" />` —— 消费 v1 的 `Reply.comments` |
| `apps/web/app/components/topic/reply/Footer.vue:93` | 从回复底部开面板，`target-user = reply.author`，不传 `parent-comment-id` |
| `apps/web/app/composables/topic/useTopicReplies.ts:248` `refreshReply` | 评论增删改后重拉 `GET /replies/{reply_id}`（v1） |
| `apps/web/app/composables/useCan.ts:13-14` | `comment.topic.edit` / `comment.topic.delete` 的前端镜像 |
| `apps/web/app/constants/permission.ts:36-37` | 权限矩阵标签 |
| `apps/web/app/utils/topicPermalink.ts:4` `commentPermalink` | `?comment=<id>` 永久链接，被 `search/CommentCard.vue:12`、`user/Comment.vue:40`、`user/Overview.vue:72`、`activity/card/TopicComment.vue:29`、`activity/card/Topic.vue:141` 使用 |
| `apps/web/shared/types/topic-comment.ts` | 旧手写 TS 类型（`id:number`、`content`、`is_liked`、`user`/`target_user`），只剩 `Panel.vue` / `Comment.vue` 的 `kunFetch<TopicComment>` 在用 |
| `apps/web/shared/utils/api/schemas.ts:9-10` | `Comment` / `CommentViewer` 的生成类型别名（读面已在用） |

**Nitro（`apps/web/server/**`）**：零。`rg` 只命中 `kunOgCard.spec.ts:31` / `kunSitemapSources.spec.ts:23` 里的 `comment_count` 夹具，与评论端点无关。

**`docs/proj/app-direct-api.md`**：零。全文唯一含「评论」的行是 `:108` 的 `GET /api/community/following`（社区原语评论墙，别的域）。**App 目前不消费话题评论**，所以 W5 是纯网页迁移，没有 App 兼容负担。

**测试**：`apps/api/internal/topic` 下**没有任何评论测试**；`v1_topic_detail_fix_test.go:227-242` 只是插 `topic_comment` / `topic_comment_like` 夹具来验回复读面。评论写面零覆盖。

---

## 8. 代码读不出来的 —— 请在生产库补这些计数

> 都是只读 SELECT。给数字即可，我会写进任务书的「取值」一节（02 §5.1 要求每个枚举成员的行数）。

**规模与形状**

1. `SELECT count(*) FROM topic_comment;`
2. `SELECT status, count(*) FROM topic_comment GROUP BY status;`（确认 `status=1` 的量，即 trust 隐藏过多少条）
3. `SELECT count(*) FROM topic_comment WHERE parent_comment_id IS NULL;` 与 `IS NOT NULL;`（子评论占比，决定 v1 要不要下发真正的树）
4. 层级深度：`WITH RECURSIVE` 或简化版 —— `SELECT count(*) FROM topic_comment c JOIN topic_comment p ON p.id = c.parent_comment_id WHERE p.parent_comment_id IS NOT NULL;`（超过两层的评论条数；前端只画两层）
5. `SELECT max(n), count(*) FILTER (WHERE n > 1000) FROM (SELECT topic_reply_id, count(*) n FROM topic_comment WHERE status=0 GROUP BY 1) s;`（一楼最多多少条评论；`Reply.comments` 的 `maxItems:1000` 是否已经被突破）
6. `SELECT count(*) FROM topic_comment_like;`

**A2 跨话题注入的实际证据**

7. `SELECT count(*) FROM topic_comment c JOIN topic_reply r ON r.id = c.topic_reply_id WHERE c.topic_id <> r.topic_id;`
8. 若 >0，再要：`SELECT c.id, c.topic_id, r.topic_id, c.user_id, c.created FROM topic_comment c JOIN topic_reply r ON r.id=c.topic_reply_id WHERE c.topic_id <> r.topic_id ORDER BY c.created DESC LIMIT 20;`
9. `SELECT count(*) FROM topic_comment c JOIN topic_comment p ON p.id = c.parent_comment_id WHERE p.topic_reply_id <> c.topic_reply_id;`（父子跨楼）

**A1 `target_user_id` 被乱填的实际证据**

10. 顶层评论指错人：`SELECT count(*) FROM topic_comment c JOIN topic_reply r ON r.id=c.topic_reply_id WHERE c.parent_comment_id IS NULL AND c.target_user_id <> r.user_id;`
11. 子评论指错人：`SELECT count(*) FROM topic_comment c JOIN topic_comment p ON p.id=c.parent_comment_id WHERE c.target_user_id <> p.user_id;`
12. 指向不存在的人：`SELECT count(*) FROM topic_comment c WHERE NOT EXISTS (SELECT 1 FROM kungal_user_state s WHERE s.user_id = c.target_user_id);`
13. `SELECT count(*) FROM topic_comment WHERE target_user_id = user_id;`（自评自，现在会跳过发分那一支）

> 10 和 11 的数字决定了「服务端推导 target」是**修 bug** 还是**改行为**：如果历史数据里绝大多数是一致的，就是修 bug；如果有成规模的不一致，需要先弄清楚是哪个老版本 UI 留下的。

**计数漂移**

14. `SELECT count(*) FROM topic t WHERE t.comment_count <> (SELECT count(*) FROM topic_comment c WHERE c.topic_id = t.id AND c.status = 0);`
15. 同上但不带 `status` 过滤（`recomputeTopicCounts` 带了 `status=0`，触发器/迁移 020 没带，想知道两者差多少）
16. `SELECT count(*) FROM topic_reply r WHERE r.comment_count <> (SELECT count(*) FROM topic_comment c WHERE c.topic_reply_id = r.id AND c.status = 0);` 以及 `SELECT count(*) FROM topic_reply WHERE comment_count <> 0;`（A10：确认这一列是不是几乎全 0，能不能直接 drop）
17. `SELECT count(*) FROM topic_comment_like l WHERE NOT EXISTS (SELECT 1 FROM topic_comment c WHERE c.id = l.topic_comment_id);`（应为 0，FK 在）
18. `SELECT count(*) FROM topic_comment_like l JOIN topic_comment c ON c.id=l.topic_comment_id WHERE l.user_id = c.user_id;`（自赞，服务端禁止；>0 说明有历史数据或别的写入口）

**可见性尾巴**

19. `SELECT count(*) FROM topic_comment c JOIN topic_reply r ON r.id=c.topic_reply_id WHERE r.status <> 0 AND c.status = 0;`（挂在隐藏回复下的可见评论）
20. `SELECT t.status, t.access_scope, count(*) FROM topic_comment c JOIN topic t ON t.id=c.topic_id WHERE c.status=0 GROUP BY 1,2;`（隐藏/受限话题里有多少评论；也顺带给出 `access_scope` 各取值的评论分布）

**正文形状（决定 A14 怎么裁决）**

21. `SELECT count(*) FROM topic_comment WHERE content LIKE '%/image/%';`
22. `SELECT count(*) FROM topic_comment WHERE content ~ 'https?://';`
23. `SELECT count(*) FROM topic_comment WHERE content ~ '!\[.*\]\(.*\)';`（markdown 图片语法）
24. `SELECT count(*) FROM topic_comment WHERE content ~ '<[a-zA-Z]';`（裸 HTML）
25. `SELECT max(length(content)), count(*) FILTER (WHERE length(content) > 1007) FROM topic_comment;`
26. `SELECT count(*) FROM topic_comment WHERE content ~ '@\d' OR content ~ '#\d';`（有没有人在评论里用了回复/话题的行内 token —— 如果有，`text` 就不能当纯文本发）

**通知与萌萌点**

27. `SELECT type, count(*) FROM message WHERE type IN ('commented','liked') GROUP BY 1;`
28. `SELECT count(*) FROM message WHERE type='commented' AND link NOT LIKE '%?comment=%';`（`cmd/backfill-message-links` 之后还剩多少条没回填）
29. `SELECT count(*) FROM topic_comment WHERE edited IS NOT NULL;`（编辑端点的真实使用量 —— 决定值不值得为它单独做一条 `PATCH`）

**时间分布（判断这些端点还活着没有）**

30. `SELECT date_trunc('month', created) m, count(*) FROM topic_comment WHERE created > now() - interval '12 months' GROUP BY 1 ORDER BY 1;`
31. `SELECT date_trunc('month', created) m, count(*) FROM topic_comment_like WHERE created > now() - interval '12 months' GROUP BY 1 ORDER BY 1;`
