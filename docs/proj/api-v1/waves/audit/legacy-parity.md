# 旧 `/api/topic/**` 与 `/api/v1` 的逐操作对照审计（W2 / W3 / W4）

只读审计，2026-09-22。所有路径为仓库绝对路径的相对形式（根 = `/home/kun/Desktop/code/website/kun-galgame-forum`）。

共对照 **26 个操作**：W3 写面 8 个（七个裁决操作 + 隐藏/取消隐藏这条状态迁移）、W4 互动 14 个、W2 读面 4 个。

三种标注贯穿全文：

- **[已宣告]** —— 波记录（`docs/proj/api-v1/waves/w{2,3,4}-*.md`）里写明的改动，本文只核对是否真的这么实现了。
- **[未宣告 · 无害]** —— 波记录没写、但不会让用户看到错误结果的差异。
- **[未宣告 · 缺陷]** —— 波记录没写、且会改变用户可观察结果或降低可用性的差异。

---

## 0. 方法与共享判定

新写面/互动面全部复用同一套判定，先把这三个共享件钉死，后面各操作不再重复：

| 共享件 | 位置 | 语义 |
|---|---|---|
| 读可见性 | `apps/api/internal/topic/apiv1/visible.go:80-110`（`visibleTopic`）、`:112-141`（`visibleReply`）、`:143-152`（`rejectUnrenderableAuthor`） | 解析 id → `FindByID` → `access.CanRead` → 作者可渲染性。任一不过 → `NOT_FOUND`（`visible.go:72-74`） |
| 能力 | `apps/api/internal/topic/apiv1/caps.go:19-35`（`capsForTopic`）、`:43-53`（`capsForReply`） | 全部走 `user.Can(perm.*)`，无一处 `perm.CanUser` / `role.Can*` |
| 已发布闸 | `apps/api/internal/topic/apiv1/engage_ops.go:494-520` | 互动额外要求 `topic.Status == 0` |

旧面的对应件：`apps/api/internal/topic/service/read_decision.go:159-176`（`requireTopicRead`，**不查作者封禁**）、`apps/api/internal/topic/service/hide_decision.go:124-144`、以及各 handler 里手写的 `user.Can(perm.X)`。

**K2（Bearer 永无 staff 能力）核对通过**：`apps/api/internal/topic/` 下非测试代码对 `perm.CanUser(` / `role.CanModerate(` / `role.CanAdminister(` 的直接调用数为 0；能力一律经 `middleware.UserInfo.Can`（`apps/api/internal/middleware/auth.go:53-55`，内含 `!u.viaBearer`）。仓库级门 `apps/api/internal/middleware/bearer_guard_test.go:22-47` 会扫出任何回退。`apps/api/internal/topic/access/read.go:26-29` 读的是 `user.Roles`，但那是**作者授予的 role ACL**、不是 staff 能力，且 Bearer 的 roles 已在 `apps/api/internal/middleware/bearer.go:85`（`role.WithoutStaff`）剥过一遍——W2 记录把它标为等价变异，结论成立。

---

## 1. W3 · 话题与回复写面（8 个操作）

### 1.1 建话题 `POST /api/topic` → `POST /api/v1/topics`

旧：`apps/api/internal/topic/handler/topic_handler.go:92-109` → `apps/api/internal/topic/service/topic_write_service.go:107-208`
新：`apps/api/internal/topic/apiv1/write_topic_create.go:17-120`

**① 副作用顺序**

| 步骤 | 旧 | 新 | 差异 |
|---|---|---|---|
| 访问范围归一 | `topic_write_service.go:113` | `write_topic_create.go:36-37,48` | 见 ② |
| 封面推导 | `:119-126`（缺席或空数组都从正文取前 9 个 token） | `:43-47`（**仅缺席**才推导；空数组 = 不要封面） | [已宣告] |
| 信任检查 | `:128-133`，事务之前 | `:50-54`，事务之前 | 一致 |
| 锁 user_state | `:138` | `:59` | 一致 |
| 日上限 | `:143-150`，`余额/10 + 1` | `:63-69`，同式 | 一致 |
| 付费版块余额闸 | `:152-154` | `:71-73` | 一致 |
| 写 topic + 授权 + 版块关系 | `:165-181` | `:83-91` | 版块关系新增 `position`（见 ④） |
| 萌萌点 | `:188`，**事务回调里** `KeyNonce`，`Ref("topic", id)` | 记录成 `pendingAward`（`:96`），`w.flushAwards` 在**提交之后**（`:105`），键 `kungal:topic_created:topic_{id}`（`write_awards.go:53-66`） | [已宣告] |
| @ 通知 | `:189` `NotifyMentions`，**无上限** | `:92` → `write_awards.go:48-51` → `NotifyMentionsLimited(..., 10)` | [已宣告] |
| 动态流 | 触发器 | 触发器 | 一致 |
| ScanBg | `:205` | `:106` | 一致 |

**② 校验与上限**

| 字段 | 旧 | 新 | 判定 |
|---|---|---|---|
| `title` | `required,min=1,max=233`（`dto/topic_dto.go:122`），**不 trim** | `minLength 1 / maxLength 233` + `trimTitle` 后空 → `TOO_SHORT`（`write_types.go:35`、`write_topic_create.go:23-27`） | [已宣告] |
| 正文 | `required,min=1,max=100007`（`:123`） | 同上限 + 全空白 → `TOO_SHORT`（`write_topic_create.go:28-30`），按原样存 | [已宣告] |
| `category` | `oneof=galgame technique others` | 同枚举 | 一致 |
| `sections` | `required,min=1,max=3`，**不校验唯一、不校验与 category 匹配**，未知名静默丢（`taxonomy_repo.go:70-74` + `CreateSectionRelation` OnConflict） | `minItems 1 maxItems 3 uniqueItems true` + 封闭 28 项枚举（`summary.go:16-18`）+ `sectionFields` 匹配校验（`write_validate.go:74-85`） | 匹配校验 [已宣告]；**`uniqueItems` [未宣告 · 缺陷]，见 §5 A** |
| `is_nsfw` | 可缺席（零值 false） | **必填**（`write_types.go:39` 非指针无 `omitempty`） | 半宣告（W3 §3 验收提到"漏了必填的 is_nsfw"） |
| `cover_images` → `cover_image_hashes` | `omitempty,max=9` + token 正则 `^/image/[0-9a-f]{64}$`（`topic_write_service.go:83-105`），内部去重 | `maxItems 9 uniqueItems true` + `^[0-9a-f]{64}$`（`write_types.go:23-32,40`） | [已宣告]（哈希数组） |
| `access_scope` | 可缺席 → 建时默认 `public`（`access_write.go:205-207`） | **必填**枚举（`write_types.go:41`） | **[未宣告 · 无害]，见 §5 H** |
| `access_roles` | 1–8 项，去重后落库（`access_write.go:213-228`） | 1–4 项 + `uniqueItems`（`write_types.go:42`） | [已宣告]（"1–4 个不重复角色"） |
| `access_user_ids` | 1–50，`id<=0` → 400 | 1–50 + `uniqueItems`；`repr.ParseID` 拒 0 后静默跳过（`write_validate.go:198-200`，`repr/id.go:23-26`） | [未宣告 · 无害]：非法 id 由 400 变成"静默忽略"，但 pattern `^[0-9]+$` 已经挡掉绝大多数 |
| `Idempotency-Key` | `Idempotent` 中间件（路由链里） | `v1.IdempotencyRequired`（`register_writes.go:13`），缺席 → `400 INVALID_PARAMETER` + `header` 位置 | [已宣告]（K12） |

**③ 权限与可见性**：两版都只要求登录。新版无额外闸。

**④ 错误映射**

| 旧 | 新 |
|---|---|
| 日上限 → `400` 体码 233 "您今日发布的话题已达上限" | `429 TOPIC_DAILY_LIMIT_REACHED` + `limit`（`write_validate.go:39-43`） |
| 付费版块余额不足 → `400` 体码 233 | `403 MOEMOEPOINT_INSUFFICIENT` + `required: 10`（`write_validate.go:45-49`） |
| 信任 deny → `gate.ErrContentBlocked()` | `422 CONTENT_REJECTED`（`write_validate.go:35-37`） |
| 字段校验 → `400` 体码 233 | `422 VALIDATION_FAILED` + `errors[]` |

**旧的一个歧义被静默修掉**：`topic_write_service.go:192-198` 用同一个 `gorm.ErrInvalidData` 表示两种失败，再按 `hasConsumeSection` 猜——所以**在付费版块里触到日上限时，旧版报的是"萌萌点不足"**。新版两者是两个 problem。[未宣告 · 无害修复]

**⑤ 顺序**：信任检查 → 事务（锁 → 限额 → 写 → 通知）→ 提交 → 发分 → Scan。旧版唯一的顺序差别是发分在事务回调内（起 goroutine，回滚也收不回）。[已宣告]

---

### 1.2 改话题 `PUT /api/topic/:tid` → `PATCH /api/v1/topics/{topic_id}`

旧：`topic_handler.go:111-132` → `topic_write_service.go:210-301`
新：`apps/api/internal/topic/apiv1/write_topic_update.go:14-217`

**① 副作用**

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 整体置换 | `:251-260` 无条件写 `title/content/category/is_nsfw/cover_images`——**漏传 `is_nsfw` 就清 NSFW，漏传封面就清空封面** | `:132-166` 只写"在场且确有变化"的列 | [已宣告] |
| `edited` | `:258` 任何编辑都写 | `:151-154` 仅 `textChanged` | [已宣告] |
| 顶帖 | `:259` 无条件 `status_update_time = now`，**无 3 个月门槛** | `:172-176` `TouchStatusUpdateTime`（`topic_repo.go:106-111`，带 `created > BumpCutoff`） | [已宣告] |
| 授权 | `:264` 每次 `ReplaceAccessGrants` | `:177-181` 仅 `grantsTouched` 时 | [未宣告 · 无害] |
| 版块 | `:276` 每次替换 | `:182-186` 仅 `sectionsTouched` 时 | [未宣告 · 无害] |
| 付费属性差额 | `:280-286`，`KeyNonce`，事务内 | `:192-194` + 提交后 flush，`KeyNonce("topic_cost_changed", "topic_{id}")`（`write_awards.go:68-80`） | [已宣告] |
| @ 通知发件人 | `:288` `userID` = **编辑者** | `:188` `topic.UserID` = **内容作者** | [已宣告] |
| @ 通知上限 | 无 | 10 | [已宣告] |
| 信任检查 / Scan | `:242-247`、`:298` **每次都跑** | `:119-128`、`:204-206` **仅 `textChanged` 时跑** | **[未宣告 · 缺陷（轻）]，见 §5 B** |

**② 校验**：PATCH 校验的是合并结果（`write_topic_update.go:49-93`）。`mergeAccess`（`write_validate.go:229-258`）带一处修复：只授权给作者自己的 `users` 话题一条授权行都没有，旧合并会判成"scope 是 users 但没有 access_user_ids" → 422；`keepStored` 字段修掉了（注释在 `write_validate.go:136-140`）。[已宣告]（W3 §3）

**③ 权限与可见性**

| | 旧 | 新 |
|---|---|---|
| 判定 | `topic_write_service.go:222-224`：`topic.UserID != userID && !canModerate` → 403；**完全不查读可见性** | `write_topic_update.go:18-38`：先 `visibleTopic`（不可见 → 404），再 `capsForTopic(...).Edit` → 403 |
| 能力键 | `user.Can(perm.TopicEditAny)`（`topic_handler.go:127`） | `user.Can(perm.TopicEditAny)`（`caps.go:27`） |

**连带后果 [未宣告]**：被权限矩阵**撤销** `topic.view_hidden` / `topic.view_restricted`、但保留 `topic.edit_any` 的 staff，现在编辑隐藏话题或受限话题会拿到 404 而不是 200。这是"先按读判定"这条已宣告规则的必然结果，波记录没点名。见 §5 K。

**④ 错误映射**：旧 403 "您没有权限编辑此话题" → 新 `403 PERMISSION_REQUIRED`（`write_validate.go:31-33`）；旧 404 → 新 `404 NOT_FOUND`（口径扩大到隐藏/受限/作者封禁）。旧的整体置换校验（`required` 的 title/content/category/sections）在 PATCH 里全部变成可缺席——**旧的 400 "title 必填" 在新面是成功**，这是 PATCH 语义，[已宣告]。

**⑤ 顺序**：新版 `rejectContent` 在事务之前（`:119-128`），旧版同（`:242-247`）。新版多一步：能力判定在读判定之后、在任何校验之前。

---

### 1.3 隐藏 / 取消隐藏 `PUT /api/topic/:tid/hide` → `PATCH /api/v1/topics/{topic_id}` 的 `state`

旧：`topic_handler.go:248-264` → `topic_write_service.go:540-553` → `hide_decision.go:124-144`
新：`write_topic_update.go:27-38,155-166` + `caps.go:28-29`

逐条核对旧 `hideDecision` 的真值表：

| 状态 | 调用者 | 旧 | 新（`capsForTopic`） | 一致？ |
|---|---|---|---|---|
| 已发布 | 作者 | 隐藏，`hidden_by="author"` | `Hide=true`，`hiddenBy="author"`（`:158-161`） | ✔ |
| 已发布 | 持 `topic.hide` | 隐藏，`hidden_by="moderator"` | 同 | ✔ |
| 已发布 | 其他 | 403 | `Hide=false` → `PERMISSION_REQUIRED` | ✔ |
| 已隐藏 | 作者且 `hidden_by=="author"` | 取消，`hidden_by=""` | `Unhide=true`，`hidden_by=""`（`:164-166`） | ✔ |
| 已隐藏 | 作者但 `hidden_by!="author"` 且无 `topic.hide` | 403 | `Unhide=false` → 403 | ✔ |
| 已隐藏 | 持 `topic.hide` | 取消 | `Unhide=true` | ✔ |

**发当前状态是空操作**：`write_topic_update.go:29` 只在 `*patch.State != topicStateName(topic.Status)` 时置 `needHide/needUnhide`。旧版是**切换**（同一个 `PUT` 反复调会来回翻转）。[已宣告]

**`hidden_by` 的第三个取值 `trust`**：`summary.go:92-98` 接受 `author/moderator/trust`，`caps.go:29` 对 `trust` 只让持 `topic.hide` 的人撤销。与旧一致。

**[未宣告 · 无害]**：`summary.go:96-98` 对 `status==1 && hidden_by==''` 的行返回 error → `buildTopic` 500。迁移 `apps/api/migrations/096_topic_status_normalize.up.sql` 只归一了 `status`，没有回填 `hidden_by`；生产上这种行只出现在 `status ∈ {2,3}` 的三条 2024 遗留（已被 096 改成 0），所以当前不可达。若将来有 `status=1, hidden_by=''` 的行，`GET /topics/{id}` 会 500 而不是 404/200。

---

### 1.4 编辑语境读 `GET /api/v1/topics/{topic_id}/source`（新增）

新：`apps/api/internal/topic/apiv1/write_source.go:13-82`。旧面没有对应操作，编辑表单读的是旧详情 `GET /api/topic/:tid`（`topic_service.go:150-311`，`content_markdown` 对所有人公开）。

- 权限：`visibleTopic` → 404；`capsForTopic(...).Edit` → 403。[已宣告]
- 授权用户以 `repr.UserRef` 给出，封禁/注销者 `name: null`（`:53-58`）。[已宣告]
- **[未宣告 · 无害]，见 §5 G**：`AccessGrants.Users` 的契约文案写"in grant order"（`write_types.go:61`），但 `repository.ListAccessGrants`（`v1_write_grants.go:9-16`）按 `subject_type, LPAD(subject_value,20,'0')` 排——即按**用户 id 数值**排，不是授予顺序；而同一张表的另一个读 `TopicRepository.FindAccessGrants`（`access_repo.go:8-12`）按 `subject_type, subject_value` **字典序**排。`topic_access_grant` 没有 id 列（`apps/api/internal/topic/model/access_grant.go:3-7`，复合主键），所以"授予顺序"根本无法表达。契约文案与实现不符。

---

### 1.5 建回复 `POST /api/topic/:tid/reply` → `POST /api/v1/topics/{topic_id}/replies`

旧：`apps/api/internal/topic/handler/reply_handler.go:363-380` → `apps/api/internal/topic/service/reply_service.go:164-244`
新：`apps/api/internal/topic/apiv1/write_reply.go:19-96`

**① 副作用**

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 取楼号 | `reply_service.go:193` `repository.NextReplyFloor` | `write_reply.go:39` 同一函数 | **旧路径已被改成计数器版**（`v1_write_floor.go:10-25`：`UPDATE topic SET last_reply_floor = GREATEST(last_reply_floor, MAX(floor)) + 1 ... RETURNING`）。[已宣告] |
| 顶帖 | `:208` `TouchStatusUpdateTime` | `:52` 同 | 一致（两版都有 3 个月门槛） |
| 重算计数 | `:212` `recomputeTopicCounts` | `:55` `service.RecomputeTopicCounts` | 一致 |
| 他人回复 +1 | `:219-220`，`Ref("topic", topicID)`，`KeyNonce` | `:64` `repliedAward`，`Ref("topic_reply", replyID)`，键 `kungal:replied:topic_reply_{id}`（`write_awards.go:82-90`） | [已宣告]（引用从 topic 换成 topic_reply） |
| `replied` 通知 | `:221` `CreateReplyMessage`（**不去重**） | `:61` 同一函数 | 一致 |
| @ 通知 | `:226` 无上限 | `:66` 上限 10 | [已宣告] |
| 发分时机 | 事务内起 goroutine | `:78` 提交后 | [已宣告] |
| ScanBg | `:236` | `:79` | 一致 |

**② 校验**：正文 `required,max=10007`（`dto/reply_dto.go:170`）→ `minLength 1 maxLength 10007`（`write_types.go:78`）；两版都拒全空白（旧 `reply_service.go:180-182` 在归一之后判，新 `write_reply.go:27-29` 在归一之前判——`markdown.NormalizeStoredContent` 只做图片 token/贴纸归一，不会把非空变空，等价）。`topic_id` 从 body 移到路径。[已宣告]（"`:tid` 形同虚设"）

**③ 权限与可见性**：旧 `requireTopicRead`（`reply_service.go:176`）—— 查 status 与 access_scope，**不查作者封禁**；新 `visibleTopic` 查。[已宣告]。两版都允许作者/持 `topic.view_hidden` 的 staff 给**隐藏话题**发回复。

**④ 错误映射**：旧 404 "未找到该话题" → 404 `NOT_FOUND`；旧 400 "回复内容不能为空" → `422 VALIDATION_FAILED` + `/content_markdown` `TOO_SHORT`；信任 deny → `422 CONTENT_REJECTED`。新增 201 + `Location`。

**⑤ 顺序**：一致（读判定 → 空白校验 → 信任检查 → 事务 → 提交 → 发分 → Scan）。

---

### 1.6 改回复 `PUT /api/topic/:tid/reply` → `PATCH /api/v1/replies/{reply_id}`

旧：`reply_handler.go:382-398` → `reply_service.go:246-293`
新：`write_reply.go:98-164`

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 权限 | `:257-259` `reply.UserID != userID && !canEditAny`（`perm.ReplyEditAny`），**不查话题可见性、不查回复是否隐藏** | `:102-108` `visibleReply`（隐藏回复对所有人 404，`visible.go:127-129`）→ `capsForReply(...).Edit` | [已宣告] |
| 内容未变 | 照样写 `edited`、照样发 @ 通知 | `:123-132` 提前返回，不写 `edited`、不发通知、不 Scan | [已宣告]（"正文确有变化时写 edited_at"） |
| 缺 `content_markdown` | 旧是必填 | `:109-118` 空操作 200 | PATCH 语义 |
| @ 发件人 | `:280` `reply.UserID`（作者） | `:145` `row.UserID` | 一致 |
| 顶帖 | 不顶 | 不顶 | [已宣告] |
| 发分 | 无 | 无 | 一致 |

**[未宣告 · 无害]**：内容未变时新版跳过信任检查；与 §5 B 同源但影响更小（正文确实没变）。

---

### 1.7 删回复 `DELETE /api/topic/:tid/reply` → `DELETE /api/v1/replies/{reply_id}`

旧：`reply_handler.go:400-416` → `reply_service.go:295-345`
新：`write_reply.go:166-214`

**① 扣分公式**

| | 旧 | 新 |
|---|---|---|
| 判"自删" | `reply.UserID == userID && !canModerate`（`:312`） | `user.ID == row.UserID`（`:177`） |
| 自删扣分 | `3 × (commentCount + likeCount + 1)`，其中 `commentCount` = `topic_comment WHERE topic_reply_id = ?` **不过滤 status**、`likeCount` = **已废弃的 `topic_reply_like` 表**（`reply_repo.go:158-162`，生产仍有 5425 行） | `3 × (row.LikeCount + 可见评论数 + 1)`，评论过滤 `status = 0`（`:182-188`） |
| staff 删他人 | 扣 3，**但仍锁并检查被删者余额**（`:317-323`）→ 余额不足时 staff 删不掉 | 扣 3，**不锁不查余额**（`:180,189-196` 只在 `authorDelete` 时锁） |
| 余额不足 | `400` "萌萌点不足, 无法删除此回复" | `403 MOEMOEPOINT_INSUFFICIENT` + `required`（`:194`） |
| 键 | `KeyNonce`，事务内 | `kungal:reply_deleted:topic_reply_{id}`，提交后（`write_awards.go:92-100`、`write_reply.go:212`） |

全部 [已宣告]。**注意一个实质后果**：口径从废弃表换成 `like_count` 列，对持有大量 `topic_reply_like` 遗留行的老回复，扣分金额会明显下降；这是波记录明写的修复，但上线当天扣分数额会跳变。

**[未宣告]，见 §5 D**：旧代码里"自删"要求 `!canModerate`，所以**版主删自己的回复只扣 3**；新代码只看 `user.ID == row.UserID`，版主删自己的回复要按完整公式扣并查余额。裁决表的措辞（"作者删自己的" / "staff 删别人的"）与新实现一致，但这条旧行为的丢失波记录没点名。

**② 级联**：`reply_repo.go:133-152` 显式删 `topic_comment_like` / `topic_comment` / `topic_reply_like` / `topic_reply_dislike`，`topic_reply_reaction` 靠外键 `ON DELETE CASCADE`（`apps/api/migrations/084_purge_leftovers.up.sql:43-49`）。`best_answer_id` / `pinned_reply_id` 靠 `ON DELETE SET NULL`（`apps/api/migrations/000_baseline.up.sql:4786,4929`）。[已宣告]

**③ 权限**：旧 `perm.ReplyDeleteAny`；新 `capsForReply(...).Delete = author || user.Can(perm.ReplyDeleteAny)`（`caps.go:50`），前置 `visibleReply`。

**④ 一个两版共有的旧账（不是新缺陷）**：被删的回复若正好是最佳答案，外键把 `best_answer_id` 置空，但**没有任何地方收回那 +7**。新旧一致，两波都没宣告。

**⑤ 顺序**：新版把"取消绑定/重算/扣分"都放进同一个事务，发分在提交后；旧版把发分放在事务内。[已宣告]

---

### 1.8 `GET /api/v1/replies/{reply_id}/source`（新增）

`write_source.go:84-101`。`visibleReply` → 404，`capsForReply(...).Edit` → 403，只回 `content_markdown`。替代了编辑框原先读 `GET /api/topic/:tid/reply/detail` 的用法。[已宣告]

---

## 2. W4 · 互动（14 个操作）

### 2.1 表情：`PUT` / `DELETE /topics/{topic_id}/reactions/{reaction}`

旧：`topic_handler.go:134-191`（`/like`、`/dislike`）、`:152-173`（`/reaction`）→ `topic_write_service.go:316-431`
新：`apps/api/internal/topic/apiv1/engage_reactions.go:13-104`

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 语义 | 切换（重放即撤销） | `PUT` 置位 / `DELETE` 撤销，均幂等 | [已宣告] K16 |
| 计数守卫 | `HasReaction` → `AddReaction`（`ON CONFLICT DO NOTHING`）→ **无条件 `+1`**（`:368-372`） | `InsertTopicReaction` `... DO NOTHING RETURNING id`，`!inserted` 直接返回（`v1_engage.go:28-34`、`engage_reactions.go:60-63`） | [已宣告] |
| like/dislike 互斥 | `clearTopicReaction`（`:410-431`） | `applyTopicReactionRemove`（`:83-104`），连带撤销副作用 | 一致 [已宣告] |
| 自赞 | `:336-338` → 400 "您不能给自己点赞" | `:19-21` → `403 SELF_LIKE_FORBIDDEN` | [已宣告] |
| 萌萌点 | `Ref("topic", id)` + `KeyNonce`，事务内 | `kungal:liked:topic_reaction_{row}` / `kungal:unliked:...`（`:74,98`），提交后（`engage_ops.go:522-530`） | [已宣告] |
| `liked` 通知 | `createDedupMessage`（`:376-379`） | `notifyTopicLink` → `dedupMessage`（`engage_notify.go:329-355`） | 逐字一致 |
| 可见性 | 只查 `topic.Status == 1`（`:333`） | `visiblePublishedTopic`：已发布 + `getTopic` 判定（含 access_scope、作者封禁） | [已宣告] |
| 未知 token | `reactionKeys` map（33 项，`:303-314`）→ 400 | 路径枚举（`engage_types.go:9-36`，同 33 项）→ 400 | 见 §5 F' |
| 计数下探 | `AdjustLikeCount` 裸 `like_count + delta`（`topic_repo.go:171-174`） | `AdjustTopicCount` 负增量用 `GREATEST(col + ?, 0)`（`v1_engage.go:85-90`） | **[未宣告 · 无害]，见 §5 C** |

### 2.2 表情：`PUT` / `DELETE /replies/{reply_id}/reactions/{reaction}`

旧：`reply_handler.go:418-470` → `reply_service.go:360-493`
新：`engage_reactions.go:106-197`

同上，外加两条：

- **旧版对回复互动什么可见性都不查**（`reply_service.go:368-438` 全程只 `FindByIDTx`），回复不存在时 gorm 的 `ErrRecordNotFound` 落进 `default:` 分支 → **500**（`:445-446`）。新版 `visiblePublishedReply` → 404。[已宣告]
- 旧版 `:377-379` 只拦自赞，隐藏回复、隐藏话题、受限话题一律放行。新版全部 404。[已宣告]

### 2.3 收藏：`PUT` / `DELETE /topics/{topic_id}/favorite`

旧：`topic_handler.go:230-246` → `topic_write_service.go:487-538`
新：`apps/api/internal/topic/apiv1/engage_favorite.go:209-273`

| 项 | 旧 | 新 |
|---|---|---|
| 并发 | `FindTopicFavorite` → `CreateTopicFavorite`（裸 `Create`）→ 撞唯一约束 500 | `INSERT ... ON CONFLICT DO NOTHING RETURNING id`（`v1_engage.go:43-52`） [已宣告] |
| `updated` 列 | gorm 结构体写入带上 | raw INSERT 显式写 `now()`（`v1_engage.go:44-51` 带事故注释：漏写这列时每次收藏都 500 / SQLSTATE 23502） [已宣告]（W4 §3） |
| 自收藏 | 计数 +1，不发分不通知（`:506-513`） | 同（`:221-226`） ✔ |
| 键 | `Ref("topic", id)` + `KeyNonce`，与点赞**共用** `liked`/`topic:{id}` | `kungal:favorited:topic_favorite_{row}` / `kungal:unfavorited:...`（`:232,265`） [已宣告] |
| 可见性 | 只查 `Status == 1` | `visiblePublishedTopic` [已宣告] |

### 2.4 推：`PUT /api/topic/:tid/upvote` → `POST /api/v1/topics/{topic_id}/upvotes`

旧：`topic_handler.go:193-214` → `topic_write_service.go:433-485`
新：`apps/api/internal/topic/apiv1/engage_upvote.go:225-296`

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 幂等键 | 无 | 必填（`register_interactions.go:180`） | [已宣告] |
| `description` → `note` | `truncate(description, 30)` **静默截断**（`:434`） | `maxLength:"30"` 拒绝（`engage_types.go` `UpvoteCreate`），`TrimSpace` 后空 → `null`（`:211-223,294`） | [已宣告]，但见 §5 F |
| 自推 | 400 "您不能推自己的话题" | `403 SELF_UPVOTE_FORBIDDEN`（`:230-232`） | [已宣告] |
| 余额 | `stateRepo.LockForUpdate` + `< 10` → 400（`:447-453`） | `LockMoemoepoint`（`v1_engage.go:92-102`，`FOR UPDATE`）+ `403 MOEMOEPOINT_INSUFFICIENT` + `required: 10`（`:241-249`） | [已宣告] |
| 不本地扣 | 是 | 是 | [已宣告]（并发两次仍可能都过） |
| 计数 + `upvote_time` + 顶帖 | `ApplyUpvoteCountAndTime`（`topic_repo.go:185-192`） | 同一函数（`:255`） | 一致 |
| 键 | 两笔都 `Ref("topic_upvote", **topicID**)` + `KeyNonce`（`:464-467`，注意旧版引用的是 topic id 不是 upvote 行 id） | `Ref("topic_upvote", rowID)` + `kungal:upvote_sent:topic_upvote_{row}` / `kungal:upvote_received:...`（`:258-274`） | [已宣告] |
| `upvoted` 通知 | `createDedupMessage` | `notifyTopicLink`（去重） | 一致 |
| 可撤销 | 否 | 否 | [已宣告] |
| `updated` 列 | gorm 结构体 | raw INSERT 写 `now()`（`v1_engage.go:61-65`） | [已宣告]（同 §2.3 事故） |

**[未宣告 · 无害]**：响应 `created_at` 取自 INSERT 的 `RETURNING created`（`:250-254,294`），而 `upvote_time` 写的是另取的 `time.Now()`（`:255`），两者可差几微秒。

### 2.5 推记录 `GET /api/topic/:tid/upvotes` → `GET /api/v1/topics/{topic_id}/upvotes`

旧：`topic_service.go:55-79`，固定 50 条、不分页、**不查可见性**（隐藏话题的推记录对匿名可见），跳过封禁用户但不补页。
新：`apps/api/internal/topic/apiv1/engage_lists.go:243-288`，`visibleTopic` 判定、游标分页（`created DESC, id DESC`）、跳过封禁并向后读满一页（`:185-226`，`maxWindows` 上限）。

[已宣告]。注意列表面**不要求已发布**——作者/staff 可以看隐藏话题的推记录，这正是 "与 getTopic 同一判定"。

### 2.6 表情历史 `GET /api/topic/:tid/reaction/history` / `.../reply/reaction/history` → `GET /api/v1/topics/{id}/reactions`、`/replies/{id}/reactions`

旧：`topic_service.go:83-103`（固定 300 条、不查可见性、**不跳过封禁用户**）、`reply_service.go:450-470`（同）。
新：`engage_lists.go:290-342`，游标分页 + 可见性 + 跳过封禁补页。

[已宣告]。**一处口径变化**：旧的话题表情历史对封禁用户仍下发（`topic_service.go:93-101` 没有 `IsRenderable` 过滤，回的是占位名），新版整条跳过。[已宣告]（"封禁用户的记录跳过并向后读满一页"）

### 2.7 最佳答案：`PUT` / `DELETE /topics/{topic_id}/best-answer`

旧：`topic_handler.go:266-287` → `topic_write_service.go:555-617`
新：`apps/api/internal/topic/apiv1/engage_choice.go:49-146`

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 语义 | 同一 `replyID` 再点 = 取消，**但提示总是"已设置"**（`topic_handler.go:286`） | `PUT` 设置（设当前 = 空操作，`:61-63`）/ `DELETE` 清除 | [已宣告] |
| 目标回复 | `DB().First(&reply, replyID)` + `reply.TopicID != topicID` → 400；**不查 status，隐藏回复也能设** | `choiceReply`（`:23-47`）：本话题 + `Status == 0` + 作者可渲染，否则 `422 VALIDATION_FAILED` `/reply_id` `UNKNOWN_REFERENCE` | [已宣告] |
| 换一条 | 前一条的 +7 **不收回** | `:93-101` 前任作者 −7 | [已宣告] |
| 楼主自己的回复 | 照样 +7 + 发通知（生产 2 例） | `:81-92` 跳过发分与通知 | [已宣告] |
| 顶帖 | `:585-594` 带 3 个月门槛 | `SetBestAnswerID(..., bump=true, now)`（`v1_engage.go:104-116`），同式 | 一致 |
| 清除时的顶帖 | 不顶 | `bump=false`（`:128`） | 一致 |
| 通知 | `notifier.Emit(NotifySolution)`（`:602-609`） | `emitSolution`（`engage_notify.go:357-369`） | 一致 |
| 键 | `KeyNonce` | `KeyNonce("best_answer_set"/"best_answer_cleared", ...)`，提交后 | [已宣告] |
| 权限 | `topic.UserID != userID && !user.Can(perm.TopicSetBestAnswer)` → 403 | `capsForTopic(...).SetBestAnswer = published && (author \|\| Can(TopicSetBestAnswer))`（`caps.go:32`） | 多一个 `published` 闸，[已宣告] |
| 可见性 | 不查 | `visiblePublishedTopic` | [已宣告] |

**[未宣告 · 缺陷（潜伏）]，见 §5 E**：`clearBestAnswer`（`:121-124`）对 `ErrRecordNotFound` 容错，但 `ReplyRepository.FindByID`（`apps/api/internal/topic/repository/reply_repo.go:21-25`）**即使出错也返回非空指针**，于是 `prev` 会是零值结构体（`UserID == 0`）并进入 `:131` 的 `prev.UserID != topic.UserID` 分支。`setBestAnswer` 的同一段写法是对的（`:66-72` 用 `if findErr == nil` 守住）。

### 2.8 置顶回复：`PUT /api/topic/:tid/reply/pin` → `PUT` / `DELETE /topics/{topic_id}/pinned-reply`

旧：`reply_handler.go:486-507` → `reply_service.go:495-533`
新：`engage_choice.go:148-194`

| 项 | 旧 | 新 | 判定 |
|---|---|---|---|
| 语义 | 切换 | `PUT` / `DELETE`，幂等 | [已宣告] |
| 目标回复 | `FindByID`，**不检查是否属于本话题、不检查是否隐藏** | `choiceReply`（同最佳答案） | [已宣告] |
| 通知 | `isPinning && userID != reply.UserID` → `pin-reply`，去重（`:520-526`） | `:168-171` 同 | 一致 |
| 发分 | 无 | 无 | [已宣告] |
| 权限 | `topic.UserID != userID && !user.Can(perm.ReplyPin)` | `capsForTopic(...).PinReply`（`caps.go:33`），加 `published` | [已宣告] |

**[未宣告 · 无害]**：`unpinReply`（`:190-192`）直接用 `x.db` 而不是事务——单条 UPDATE，无副作用，无所谓。

---

## 3. W2 · 读面（4 个操作）

### 3.1 `GET /api/topic/:tid` → `GET /api/v1/topics/{topic_id}`

旧：`topic_service.go:150-311`；新：`apps/api/internal/topic/apiv1/get.go:14-140`。

| 维度 | 旧 | 新 | 判定 |
|---|---|---|---|
| 谁看得见 | `requireTopicRead`（status 1 → 作者/`topic.view_hidden`；access_scope；`topic.view_restricted` 旁路）+ 作者封禁 → 404（`:222-224`） | `visibleTopic` 同一套（`access/read.go:193-223` + `rejectUnrenderableAuthor`） | 一致 [已宣告] |
| 浏览数 | **每次 GET 都 +1**（`:226` 起 goroutine，吞错误），`topic.view` + 日桶 | 不计（`get.go:14-24`），改由 `POST /views` | [已宣告] |
| 最佳答案内嵌 | `:287-306`，**不看 `reply.Status`**（隐藏的最佳答案照样出现），跳过封禁作者 | `loadSpecialReplies`（`get.go:142-174`）走 `FindRepliesByIDs`（`reply_repo.go:94-105`，`WHERE status = 0`）+ `mapOne` 对封禁作者返回 nil（`assemble.go:364-368`）→ `null` | [已宣告] |
| 置顶内嵌 | 旧详情没有这个字段（只在回复列表第 1 页插队） | `Topic.pinned_reply`，且 `loadSpecialReplies:162-164` 剔除不属于本话题的置顶 | [已宣告] |
| `content_markdown` | 对所有人下发（`dto/topic_dto.go:80`） | 不下发，改到 `/source` | [已宣告] |
| `access_grants` | 作者/`topic.edit_any` 可见（`read_decision.go:178-194`） | 不在 `Topic` 里，改到 `/source` | [已宣告] |
| 作者萌萌点 | `stateRepo.FindByID`（`:182-184`） | `LookupMoemoepoints`（`v1_read.go:35+`），同表 | 一致 |
| `viewer` | 扁平 `is_liked` / `is_favorited` / `is_upvoted` / `is_disliked` | `viewer.has_*` + 七个 `can_*`（`engage_viewer.go:264-282`、`detail.go:51-63`） | [已宣告] W3/W4 |
| 表情 | `buildReactionSummaries`，`mine` 布尔 | `reactionSummaries`（`reactions.go:10-53`），最多 3 个 reactor、跳过封禁、`viewer.has_reacted` | [已宣告] |

**[未宣告 · 缺陷（可用性）]，见 §5 I**：旧详情在用户服务失败时是 `500`；新读面是 `503 SERVICE_UNAVAILABLE`（`assemble.go:301-313` → `problem.Unavailable`）。对详情面只是码变了，但对**回复列表和全部写/互动**是真正的行为变化（下一条）。

### 3.2 `GET /api/topic/:tid/reply` → `GET /api/v1/topics/{topic_id}/replies`

旧：`reply_handler.go:312-326` → `reply_service.go:95-138` + `mapper.go:110-260`
新：`apps/api/internal/topic/apiv1/reply_ops.go:67-186`

| 维度 | 旧 | 新 | 判定 |
|---|---|---|---|
| 分页 | 页码 `page`/`limit(1..30)`/`sort_order`（`dto/reply_dto.go:154-159`），`OFFSET`（`reply_repo.go:34-56`） | 游标 `(floor, id)` 键集 + `from_floor` 锚点 + `sort=floor_asc\|floor_desc`（`reply_keyset.go:22-52`、`detail_ops.go:100-105`） | [已宣告] |
| 置顶/最佳答案 | **第 1 页插到最前并从各页剔除**（`reply_service.go:108-132`）→ 第 1 页可能有 limit+2 条 | 不插队，各自在自己的楼层；顶部展示由客户端用 `Topic.pinned_reply` / `best_answer` | [已宣告] |
| 隐藏回复 | `status = 0` 过滤 | 同（`reply_keyset.go:28`） | 一致 |
| 封禁作者的回复 | `mapper.go:196-198` 跳过，**页面因此变短**（前端"不足 30 条即末页"会误判） | `gatherReplyPage`（`reply_ops.go:144-182`）跳过并向后读窗口凑满，`hasMore` 精确 | [已宣告] |
| 楼层语义 | `MAX(floor)+1`、可重复、删后复用 | 计数器 + `(topic_id, floor)` 唯一索引（迁移 100），不复用 | [已宣告] |
| 评论内嵌 | 全部评论（`mapper.go:200-227`）；**封禁作者的评论仍下发**，只是换成占位名 + **正文清空** | `ListByReplyIDs`（`v1_read.go:18-33`，`status = 0`，`created ASC, id ASC`）；封禁作者的评论**整条不下发**（`assemble.go:403-405`），子评论保留 `parent_comment_id` | [已宣告] |
| `target_user` | `kunUser(c.TargetUserID)`，找不到时是全零对象 | `in_reply_to_user`，找不到时 `repr.DeletedUserRef` | [已宣告]（命名）；零值 → 占位引用属 [未宣告 · 无害] |
| 不存在的话题 | 返回 `[]` + 200（`reply_service.go:100-103`） | 404 | [已宣告]（"不可见一律 NOT_FOUND"） |

### 3.3 `GET /api/topic/:tid/reply/detail` → `GET /api/v1/replies/{reply_id}`

旧：`reply_handler.go:328-342` → `reply_service.go:140-162`（`replyId` 走 query）
新：`reply_ops.go:188-201` + `buildOneReply:203-229`

旧版对 `status != 0` 的回复：`FindRepliesByIDs` 已过滤 → 404。新版 `visible.go:127-129` 显式 404。一致。新版额外要求回复作者可渲染（`:213-215`）。[已宣告]

### 3.4 `POST /api/v1/topics/{topic_id}/views`（新增）

`get.go:26-35`。`Optional` 档（`register.go:80-91`），允许匿名，无幂等键。`IncrementView`（`topic_repo.go:41-47`）同时写 `topic.view` 与日桶，**错误不再被吞**（旧 `topic_service.go:226` 是 `go func(){ _ = ... }()`）。[已宣告]（W2 §4）

---

## 4. 未宣告差异清单

### [未宣告 · 缺陷]

**A. `sections` 新增 `uniqueItems`，昨天能发的载荷今天 422。**
旧：`dto/topic_dto.go:125` 只有 `min=1,max=3`；重复版块名在 `taxonomy_repo.go:70-74`（`name IN ?` 去重）+ `CreateSectionRelation` 的 `OnConflict DoNothing` 里被静默折叠，请求成功。
新：`apps/api/internal/topic/apiv1/write_types.go:38,50` 带 `uniqueItems:"true"`，重复 → `422 VALIDATION_FAILED` + `DUPLICATE_ITEM`。
W3 裁决只写了"`sections` 必须与 `category` 匹配"和"未知版块由枚举拒掉"，没写唯一性。网页表单大概率产不出重复项，但 App 或脚本可以。**建议**：要么在波记录补一句，要么去掉 `uniqueItems` 并在服务端去重。

**B. 改话题时，标题与正文都没变就完全跳过信任检查与 ScanBg。**
旧：`topic_write_service.go:242-247`（`check.Decision`）与 `:298`（`ScanBg`）在**每一次** `PUT` 上都跑，哪怕只改了 `is_nsfw`。
新：`write_topic_update.go:119-128` 与 `:204-206` 都被 `if textChanged` 包住。
后果：一篇正文在创建之后才进入封禁词表的话题，作者改分类/封面/NSFW 开关现在会成功；旧版会被 `CONTENT_REJECTED` 顶回去，逼作者先改正文。正文本身没变，所以严重度有限，但这是信任闸覆盖面的收缩，W3 裁决的"改话题"一行只讲了 `edited_at` 与顶帖，没提这个。

**C. 计数器负向增量被钳到 0。**
旧：`topic_repo.go:171-186`、`reply_repo.go:170-173` 一律 `col + delta`，可以跌成负数。
新：`apps/api/internal/topic/repository/v1_engage.go:85-90,140-145` 负增量走 `GREATEST(col + ?, 0)`。
方向是对的（计数不该为负），但它同时会**吃掉一次真实的减量**：若某行因历史漂移已经是 0，撤销点赞时 `like_count` 停在 0，而 `topic_reaction` 里确实少了一行——下次跑迁移 099 式的重算才会对上。W4 裁决只写了"计数只在真的插入或删除了一行时变"，没写钳制。

**D. 版主删自己的回复，扣分公式变了。**
旧：`reply_service.go:311-314` 的 `penalty = 3 * (...)` 条件是 `reply.UserID == userID && **!canModerate**`，所以持 `reply.delete_any` 的人删自己的回复只扣 3（走 else 分支）。
新：`write_reply.go:177` 的 `authorDelete := user.ID == row.UserID` 不看 staff 身份，版主删自己的回复要按完整公式扣，并且**要过余额闸**（`:189-195`），余额不足会 `403 MOEMOEPOINT_INSUFFICIENT`——版主删不掉自己的回复。
裁决表的措辞（"作者删自己的" / "staff 删别人的"）与新代码一致，所以这更像"旧行为被无声丢弃"而不是实现走偏，但它确实是一条未点名的行为变化。

**E. `clearBestAnswer` 把 `FindByID` 在出错时仍返回的非空零值指针当成有效的前任最佳答案。**
`apps/api/internal/topic/apiv1/engage_choice.go:121-124`：
```go
prev, findErr := x.reads.replies.FindByID(*topic.BestAnswerID)
if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) { return nil, problem.Internal(findErr) }
```
`apps/api/internal/topic/repository/reply_repo.go:21-25` 是 `return &reply, err`——`ErrRecordNotFound` 时 `prev` 是 `&model.TopicReply{}`（`UserID == 0`），于是 `:131` 的 `prev != nil && prev.UserID != topic.UserID` 为真，会排一笔 `userID: 0, delta: -7, ref: "topic_reply:0"` 的发分。
同一文件的 `setBestAnswer`（`:66-72`）写法是对的（`if findErr == nil { prev = row }`），两处不对称。
**当前不可达且无害**：`topic_best_answer_id_fkey ... ON DELETE SET NULL`（`apps/api/migrations/000_baseline.up.sql:4786`）保证 `best_answer_id` 非空就一定有行；即便到了那一步，`Awarder.Award` 对 `userID <= 0` 直接 return（`apps/api/internal/moemoepoint/pusher.go:46`）。属于"下一次有人改外键或加软删就会咬人"的潜伏缺陷。

**F. 推的 `note` 长度在去首尾空白之前校验。**
W4 裁决：`note` "至多 30 字，去首尾空白"。实现是 huma 先按 `maxLength:"30"` 校验原串（`apps/api/internal/topic/apiv1/engage_types.go` 的 `UpvoteCreate.Note`），`trimUpvoteNote` 在 handler 里才跑（`engage_upvote.go:211-216,233`）。所以 `"  <30 字>  "` 会 `422 TOO_LONG`，而按裁决它应当被接受。旧版对同样的输入是静默截断到 30 字后入库。

**F'. 未知 reaction token 的 problem code 与 W4 裁决写的不同。**
W4：「未知 → `400 INVALID_PARAMETER` + `UNKNOWN_VALUE`」。
实际：路径枚举违规经 `apps/api/pkg/problem/huma.go:98-105`——`allParamOrHeader && allUnknown` → `CodeUnknownEnumValue`。状态码一样是 400（`apps/api/pkg/problem/registry_test.go:14`），`errors[].reason` 一样是 `UNKNOWN_VALUE`，而且 `UNKNOWN_ENUM_VALUE` 在 `apps/web/i18n/locales/zh-CN/problem.json` 里有译文，所以用户侧无影响。是波记录写错了，实现遵循的是 01 §4「封闭词表的未知值 → 400 UNKNOWN_ENUM_VALUE」。

**I. 写面与互动面现在会因为 OAuth 用户服务不可用而整体 503。**
链路：任何写/互动 → `visibleTopic` / `visibleReply` → `rejectUnrenderableAuthor`（`apps/api/internal/topic/apiv1/visible.go:106-108,137-139,143-152`）→ `lookupUsers`（`apps/api/internal/topic/apiv1/assemble.go:301-313`）→ `userclient.Client.Users` 返回 error 时 → `problem.Unavailable` = `503`。
旧面：`Create` / `Update` / `CreateReply` / `UpdateReply` / `DeleteReply` / 全部互动**一次都不碰 `userclient`**（`requireTopicRead` 只读本地表），用户服务挂掉时照常成功；旧读面用的是 `userclient.Hydrate`（`apps/api/pkg/userclient/hydrate.go:32-46`），它**吞掉错误并回落成 `Placeholder`**，所以回复列表只是名字变占位、不会失败。
`userclient` 有 ~10 分钟热缓存，所以实践中热门话题的作者多半命中缓存；但"上游 `/users/batch` 抖动 → 全站发帖/回复/点赞 503"这条耦合是新的，W3/W4 都没宣告。（`getTopic` 是例外：旧详情在 `errgroup` 里对 `userClient.User` 的错误是 `return e` → 500，所以那一条只是码从 500 变 503。）

### [未宣告 · 无害]

- **G. 授权用户的顺序**：契约说 "in grant order"（`write_types.go:61`），`ListAccessGrants`（`v1_write_grants.go:9-16`）按用户 id 数值排，`FindAccessGrants`（`access_repo.go:8-12`）按字典序排，`topic_access_grant` 根本没有顺序列（`model/access_grant.go:3-7`）。三者互不相符。W3 §3 写的"改成按 id 排序"指的是用户 id，不是行 id。建议把契约文案改成"按用户 id 升序"。
- **H. `access_scope` 在创建时变成必填**（`write_types.go:41` 非指针无 `omitempty`），旧版缺席时默认 `public`（`access_write.go:205-207`）。
- **J. 建话题的错误歧义被修掉**（见 §1.1 ④）。
- **K. 读判定前置带来的 staff 404**：权限矩阵撤销 `topic.view_hidden` / `topic.view_restricted` 但保留 `topic.edit_any` / `topic.hide` 的 staff，现在编辑或取消隐藏会拿 404。这是 W3 宣告的"先按读判定"的必然结果，只是没点名。
- **L. `updateTopic` 少写了无谓的行**：授权与版块关系只在被 PATCH 触及时才替换（旧版每次 `PUT` 都 delete+insert）。
- **M. 推的 `created_at` 与 `upvote_time` 取自两个 `now()`**（`engage_upvote.go:250-255`）。
- **N. 迁移号的文档漂移**：W3 §2 写"迁移 098"，实际楼层计数器 + 唯一索引 + `topic_section_relation.position` 在 `apps/api/migrations/100_topic_reply_floor_unique.up.sql`；`098_resource_axes` 是另一波的内容。W3 §3 自己用的是 100。W4 的"迁移 099"对得上（`099_topic_engagement_recount.up.sql`）。

### 已宣告项的落地核对（全部通过）

| 宣告 | 实现 | 结论 |
|---|---|---|
| 幂等 `PUT`/`DELETE` 取代切换 | `register_interactions.go` 的十个 `PUT`/`DELETE`；各 apply 函数在"已置位/未置位"时直接返回 | ✔ |
| 计数由 `RETURNING` 行守卫 | `v1_engage.go:12-26,28-59,125-138` + 调用点 `!inserted` / `!deleted` 早退 | ✔ |
| 发分在提交之后 | `write_awards.go:39-46`（写面）、`engage_ops.go:522-530`（互动面）；全部 `pendingAward` 在事务闭包内只做 `append` | ✔ |
| 自赞被拒 | `engage_reactions.go:19-21,112-114` → `SELF_LIKE_FORBIDDEN`；自推 `engage_upvote.go:230-232` | ✔ |
| 回复楼层计数器 | `v1_write_floor.go:10-25`；**旧路径也改用它**（`reply_service.go:193`）；唯一索引在迁移 100 | ✔ |
| 服务端推导的版块顺序 | 迁移 100 加 `position`；写 `taxonomy_repo.go:22-27`，两处读都 `ORDER BY position`（`:41,60`） | ✔ |
| `hidden_by` | `write_topic_update.go:155-166` 与 `hide_decision.go` 真值表逐格一致 | ✔ |
| 稳定幂等键 | `write_awards.go:53-100`、`engage_reactions.go:74,98,167,191`、`engage_favorite.go:232,265`、`engage_upvote.go:265,272`；只有最佳答案与付费属性差额保留 `KeyNonce`（无天然事件行，裁决已写明理由） | ✔ |
| @ 通知上限 10、发件人 = 内容作者 | `write_validate.go:16`、`write_awards.go:48-51`；`write_topic_update.go:188` 传 `topic.UserID` | ✔ |
| Bearer 永无 staff 能力 | `caps.go` 全部 `user.Can`；`bearer_guard_test.go` 是仓库级门 | ✔ |
| 新错误码与状态 | `registry_test.go:31-36`：`PERMISSION_REQUIRED`(moderation,403)、`CONTENT_REJECTED`(kungal,422)、`TOPIC_DAILY_LIMIT_REACHED`(kungal,429)、`MOEMOEPOINT_INSUFFICIENT`(kungal,403)、`SELF_LIKE_FORBIDDEN`(403)、`SELF_UPVOTE_FORBIDDEN`(403) | ✔ |

---

## 5. 旧路由去留

以 `apps/api/internal/app/testdata/routes.golden` 的注册表为准，在 `apps/web/app/**` 与 `apps/web/server/**` 中逐条查调用者。两个前端取数口径：`kunFetch` / `useKunFetch`（`apps/web/app/utils/kunFetch.ts:176-179,204` 与 `:136-143`，自动拼 `${apiBaseUrl}/api`，调用点写裸 `/topic/...`）与 v1 的 `openapi-fetch` 客户端（`apps/web/shared/utils/api/client.ts:41`，`baseUrl = /api/v1`，调用点写 `/topics`、`/replies/{reply_id}` 这类复数路径）。后者不算旧调用者。

### 5.1 零调用者，可以删（21 条）

> **2026-09-22 已执行，实际删了 22 条。** 本节漏了 `GET /api/topic/:tid/reply/reaction/history`（v1 替代是 `GET /api/v1/replies/{reply_id}/reactions`）；重新从 `router.go` 全量枚举 49 条 `/topic/**` 再逐条查调用者，才对上 22。基线 `legacy_route_baseline` 从 319 降到 297，差值正好 22。清单与替代关系见 `../../CHANGELOG.md` 的 2026-09-22 条目。
>
> 本节的口径还有一处不足：只 grep 了 `apps/web/**`，看不见 Flutter App。删之前另查了 `../../../../../kungal-apps`（App 仓库里一个 API 调用都还没写）、`assetlinks.json` 的指纹仍是全 0 占位、`/api/app/version` 是 0.1.0，且 `docs/proj/app-direct-api.md` 顶部横幅已写明 App 只绑 `/api/v1`。

```
POST   /api/topic
PUT    /api/topic/:tid
GET    /api/topic/:tid
PUT    /api/topic/:tid/hide
PUT    /api/topic/:tid/like
PUT    /api/topic/:tid/dislike
PUT    /api/topic/:tid/reaction
PUT    /api/topic/:tid/favorite
PUT    /api/topic/:tid/upvote
GET    /api/topic/:tid/upvotes
GET    /api/topic/:tid/reaction/history
PUT    /api/topic/:tid/best-answer
POST   /api/topic/:tid/reply
PUT    /api/topic/:tid/reply
DELETE /api/topic/:tid/reply
GET    /api/topic/:tid/reply
GET    /api/topic/:tid/reply/detail
PUT    /api/topic/:tid/reply/like
PUT    /api/topic/:tid/reply/dislike
PUT    /api/topic/:tid/reply/reaction
PUT    /api/topic/:tid/reply/pin
```

这正好覆盖 W3 §2「旧路由的删除」列出的六条（`POST /api/topic`、`PUT /api/topic/:tid`、`POST|PUT|DELETE /api/topic/:tid/reply`、`GET /api/topic/:tid`、`GET /api/topic/:tid/reply/detail`），外加 W4 移走的全部互动路由。

**两条删除前要处理的牵连**（均已处理）：

1. `apps/web/app/components/favorite/Toggle.vue:57` 这个通用收藏组件仍保留一条 `kunFetch(endpoint)` 的旧分支；话题侧传的是 `action`（v1 函数）而不是 `endpoint`（`apps/web/app/components/topic/footer/Favorite.vue:29,32`），所以话题不会走到它，其它域（website / galgame-quiz）还在用。删 `PUT /api/topic/:tid/favorite` 不影响它，但 `apps/web/app/components/favorite/Toggle.spec.ts:25,31` 里拿 `'/topic/9/favorite'` 当样例字符串，删路由后这个测试仍会通过（它 mock 了 `kunFetch`），只是字符串变成了指向不存在路由的化石。
2. 旧 `CreateReply` 已经改用 `repository.NextReplyFloor`（`reply_service.go:193`），删掉 `POST /api/topic/:tid/reply` 之后这条路径只剩 `ModerationRemove` 等内部调用者，`ReplyService.CreateReply` 会变成死代码——可以一并清掉。

### 5.2 仍有调用者，不能删（30 条）

| 路由 | 调用者 `file:line` |
|---|---|
| `GET /api/topic/:tid/reply/locate` | `apps/web/app/components/topic/detail/Detail.vue:56` |
| `GET /api/topic/interactions/mine` | `apps/web/app/composables/useMyTopicInteractions.ts:18` |
| `GET /api/topic/draft` | `apps/web/app/composables/topic/useTopicDraft.ts:27` |
| `POST /api/topic/draft` | `apps/web/app/composables/topic/useTopicDraft.ts:30` |
| `GET /api/topic/draft/:id` | `apps/web/app/composables/topic/useTopicDraft.ts:43` |
| `DELETE /api/topic/draft/:id` | `apps/web/app/composables/topic/useTopicDraft.ts:59` |
| `POST /api/topic/:tid/comment` | `apps/web/app/components/topic/comment/Panel.vue:30` |
| `PUT /api/topic/:tid/comment` | `apps/web/app/components/topic/comment/Comment.vue:95` |
| `DELETE /api/topic/:tid/comment` | `apps/web/app/components/topic/comment/Delete.vue:44` |
| `PUT /api/topic/:tid/comment/like` | `apps/web/app/components/topic/comment/Like.vue:44` |
| `GET /api/topic/:tid/poll/topic` | `apps/web/app/composables/topic/usePoll.ts:5` |
| `POST /api/topic/:tid/poll` | `apps/web/app/composables/topic/usePoll.ts:12` |
| `PUT /api/topic/:tid/poll` | `apps/web/app/composables/topic/usePoll.ts:57` |
| `DELETE /api/topic/:tid/poll` | `apps/web/app/composables/topic/usePoll.ts:72` |
| `POST /api/topic/:tid/poll/vote` | `apps/web/app/composables/topic/usePoll.ts:79` |
| `GET /api/topic/:tid/poll/log` | `apps/web/app/components/topic/poll/Log.vue:28` |
| `POST /api/topic/:tid/lottery` | `apps/web/app/composables/topic/useLottery.ts:48`（`base` 在 `:34`） |
| `PUT /api/topic/:tid/lottery` | `apps/web/app/composables/topic/useLottery.ts:54` |
| `DELETE /api/topic/:tid/lottery` | `apps/web/app/composables/topic/useLottery.ts:67` |
| `GET /api/topic/:tid/lottery/topic` | `apps/web/app/composables/topic/useLottery.ts:37` |
| `GET /api/topic/:tid/lottery/entrants` | `apps/web/app/composables/topic/useLottery.ts:43` |
| `POST /api/topic/:tid/lottery/enter` | `apps/web/app/composables/topic/useLottery.ts:75` |
| `POST /api/topic/:tid/lottery/withdraw` | `apps/web/app/composables/topic/useLottery.ts:81` |
| `POST /api/topic/:tid/lottery/draw` | `apps/web/app/composables/topic/useLottery.ts:94` |
| `POST /api/topic/:tid/lottery/cancel` | `apps/web/app/composables/topic/useLottery.ts:109` |
| `POST /api/topic/:tid/lottery/claim` | `apps/web/app/composables/topic/useLottery.ts:119` |
| `PUT /api/topic/:tid/lottery/fulfillment` | `apps/web/app/composables/topic/useLottery.ts:129` |
| `GET /api/admin/topic/hidden` | `apps/web/app/pages/admin/topic.vue:93` |
| `GET /api/admin/topic/:tid/purge-stats` | `apps/web/app/pages/admin/topic.vue:120` |
| `DELETE /api/admin/topic/:tid` | `apps/web/app/pages/admin/topic.vue:134` |

与两份波记录的「不在本波」一致：`/reply/locate` 归 W5（W2 §2 明写"W5 前网页继续用旧 locate"），`/interactions/mine` 随动态流迁移（W4 §2），评论归 W5，投票/抽奖/草稿/管理面各有自己的波。`apps/web/server/**` 里没有任何 Nitro 路由代理旧 `/api/topic/**`；`kunSitemapSources.ts`、`kunOgCard.ts` 用的都是 v1 客户端。

---

## 6. 代码无法裁定的问题

- **`topic_comment` 是否被 GORM 软删**：`reply_repo.go:143-149` 用 `tx.Where(...).Delete(&model.TopicComment{})`。如果 `TopicComment` 带 `gorm.DeletedAt`，删回复后评论行会留在库里且 `status` 仍是 0，`recomputeTopicCounts`（`interaction_helpers.go:84-90`）算出的 `comment_count` 就会偏高。新旧两版用的是同一个函数，所以**不是本次迁移引入的差异**；要确认得看 `apps/api/internal/topic/model/` 里 `TopicComment` 的字段，或者在测试库里删一条带评论的回复再查 `SELECT count(*) FROM topic_comment WHERE topic_reply_id = ...`。
- **`huma` 的 `maxLength` 到底按 rune 还是 byte 计**：决定推的 `note` 对中文的实际上限是 30 个字还是 30 字节（§5 F 的严重度）。一条针对 `POST /api/v1/topics/{id}/upvotes`、body `{"note": "<30 个汉字>"}` 的请求即可裁定。
- **生产是否存在 `status = 1 AND hidden_by = ''` 的话题行**：决定 §1.3 末尾那条 500 是否真的不可达。一句 `SELECT count(*) FROM topic WHERE status = 1 AND hidden_by = ''` 即可裁定（须打 infra 的 postgres 容器，不是 `postgres-kungalgame` MCP）。
