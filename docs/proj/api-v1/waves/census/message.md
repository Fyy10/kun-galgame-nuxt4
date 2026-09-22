# 普查 · 消息 / 通知域

> 只读普查，未跑任何数据库查询、未起服务。全部结论来自代码，行号是 `master` @ `45127518`。
> 口径同 `docs/proj/api-v1/02-governance.md` §5.1：调用方、取值、语义三项；取值里需要生产库行数的，集中列在 §8。

## 1. 面的范围与端点清单

全部挂在 `apps/api/internal/app/router.go`。域内 **14 个端点**：11 个在 `/api/message/**`，2 个通知偏好，1 个未读红点。

| # | 方法 + 路径 | handler | 注册 |
|---|---|---|---|
| 1 | `GET /api/message` | `MessageHandler.GetMessages` `message/handler/message_handler.go:24` | `router.go:289` |
| 2 | `GET /api/message/muted` | `MessageHandler.GetMutedMessages` `message_handler.go:42` | `router.go:290` |
| 3 | `DELETE /api/message/:id` | `MessageHandler.DeleteMessage` `message_handler.go:60` | `router.go:291` |
| 4 | `PUT /api/message/system/read` | `MessageHandler.MarkAllRead` `message_handler.go:115` | `router.go:292` |
| 5 | `GET /api/message/admin` | `MessageHandler.GetSystemMessages` `message_handler.go:77` | `router.go:293` |
| 6 | `PUT /api/message/admin/read` | `MessageHandler.MarkAdminRead` `message_handler.go:90` | `router.go:294` |
| 7 | `GET /api/message/nav/system` | `MessageHandler.GetNavSummary` `message_handler.go:102` | `router.go:295` |
| 8 | `GET /api/message/nav/contact` | `ChatHandler.GetNavContact` `message/handler/chat_handler.go:22` | `router.go:299` |
| 9 | `GET /api/message/chat/history` | `ChatHandler.GetChatHistory` `chat_handler.go:35` | `router.go:300` |
| 10 | `POST /api/message/chat/send` | `ChatHandler.SendChatMessage` `chat_handler.go:53` | `router.go:301` |
| 11 | `POST /api/message/chat/recall` | `ChatHandler.RecallChatMessage` `chat_handler.go:70` | `router.go:302` |
| 12 | `GET /api/user/notification-preferences` | `UserHandler.GetNotificationPreferences` `user/handler/user_handler.go:68` | `router.go:64` |
| 13 | `PUT /api/user/notification-preferences` | `UserHandler.UpdateNotificationPreferences` `user_handler.go:80` | `router.go:65` |
| 14 | `GET /api/user/status`（未读红点） | `UserHandler.GetStatus` `user_handler.go:55` | `router.go:57` |

**邻接的 4 个**（不是消息读面，但写消息域的表或状态，迁移时必须一起看）：

- `POST /api/community/wall/read` `router.go:296` → `community/engagement/engagement.go:48`，成功时写 `message.status`（`message_repo.go:114`）。
- `POST /api/community/wall/follow` `router.go:297` → `engagement.go:117`，决定镜像器将来还会不会为这个墙写消息行。
- `GET /api/community/following` `router.go:298` → `engagement.go:158`，`/message/follow` 页面的数据源，挂在消息侧栏里。
- `POST /api/image/message` `router.go:306` → `ImageHandler.UploadMessageImage:116`，私信图片。

**信封**：全部走旧信封 `{code, message, data}`（`pkg/response/response.go:9`），错误是 `{code, message}` + 一个 HTTP 状态码（`response.go:24`）。`errors.ErrBadRequest/NotFound/Forbidden/Internal` 全部把 `code` 设成同一个体码 `CodeBiz`（`pkg/errors/errors.go:77/89/93/97`），也就是 K7 表里那个「400/403/404/500 混用的 233」。这个域里**一个具名业务码都没有**。

**中间件链**（`apps/api/internal/app/testdata/routes.golden:18,125–130,185,243,245,246,310,311,336`）：

- 1–11 号在 `authed := api.Group("", a.Authn.Auth())`（`router.go:244`）之下，golden 里显示成 `OptionalAuth → Auth`：`OptionalAuth` 是前面 `optAuth` 组在 `/api` 上留下的 `Use()`，对这些路由是重复执行，没有语义作用。
- 12–14 号直接挂 `userAuth`（`router.go:55`），链里只有 `Auth`。
- 全部路由前面有 `middleware.NamePreference`。这个域不读 name preference，是白挂的一层。

## 2. 逐端点

### 2.1 `GET /api/message` —— 通知中心主列表

- **链路**：`message_handler.go:24` → `MessageService.GetMessages` `message/service/message_service.go:76` → `MessageRepository.FindMessages` `message/repository/message_repo.go:38`；沿途还打 `StateRepository.FindByID`（`message_service.go:34`，取静音集合）与 `userclient.Client.Hydrate`（`message_service.go:44`，OAuth `/users/batch`）。
- **请求**：`dto.ListMessagesRequest` `message/dto/message_dto.go:3`。

  | 参数 | 类型 | 校验 | 实际 |
  |---|---|---|---|
  | `page` | int | `min=1` | **事实上必填**：缺席绑定成 0，`min=1` 判负 → 400 |
  | `limit` | int | `min=1,max=30` | 同上，**必填** |
  | `sort_order` | string | `required,oneof=asc desc` | 必填 |
  | `type` | string | 无 | **这个端点完全不读它**（`message_service.go:81` 传 `nil` 给 `onlyTypes`）。只有 `/message/muted` 用 |

  `utils.ParseQueryAndValidate` `pkg/utils/validate.go:66`：先 `c.Bind().Query`，再 `validate.Struct`。没有 `omitempty`，所以三个参数都不能省。`docs/proj/app-direct-api.md:98` 已经写对了这一点（含「忽略 `type`」）。

- **响应**（`dto.MessageListResponse` `message_dto.go:30`，包在 `data` 里）：

  ```
  { messages: [ { id:int, sender:{id,name,avatar}, receiver_id:int, link:string,
                  content:string, status:"unread"|"read", type:string,
                  created:string, item_count:int, actor_count:int, community:bool } ],
    total: int64 }
  ```

  `created` 是 `MessageRow.CreatedAt string`（`message_repo.go:32`），由驱动把 `timestamptz` 转成字符串，不是受控的 RFC 3339 UTC。`community` 是 `(m.community_notification_id IS NOT NULL)` 的别名（`message_repo.go:50`）。

- **网页消费**：`apps/web/app/pages/message/notice.vue:14`（`useKunFetch<MessageList>('/message')`，query 是 `{page, limit:30, sort_order:'desc'}`），条目渲染在 `apps/web/app/components/message/aside/Notice.vue`，文案由 `apps/web/app/components/message/utils/getMessageI18n.ts:152` 拼。类型手写在 `apps/web/shared/types/message.ts:28/42`。App 契约 `docs/proj/app-direct-api.md:98`。

- **错误**：只有两条。参数不合法 → `400 {code:CodeBiz, message:"<中文字段名> 长度不能小于 1"}`（`validate.go:96`）；仓储返错 → `500 {code:CodeBiz, message:"获取消息列表失败"}`（`message_service.go:85`）。会话缺失由 `Auth` 中间件在进 handler 之前拦掉（`middleware/auth.go:83`）。

- **鉴权与权限**：匿名不可（`authed` 组）。**没有任何能力检查**，既不用 `user.Can` 也不用被禁的 `perm.CanUser` / `role.Can*`。Bearer 可调用且与 cookie 行为一致，`bearer_guard_test.go` 的规则在这个域无用武之地。

- **收件人如何决定**：只由凭证决定。`message_repo.go:51` 的 `Where("m.receiver_id = ?", receiverID)` 里的 `receiverID` 来自 `middleware.MustGetUser(c).ID`（`message_handler.go:25` → `message_service.go:82`）。**查询层面无法读到别人的消息**：没有任何路径把请求体或查询参数的值接到 `receiver_id` 上。

- **值得记的事实**：
  - **排序没有 tie-breaker**。`message_repo.go:63` 是 `Order("m.created " + sortOrder)`，仅此一列。`message.created` 是 `timestamp(3)`，同一个事务里批量发的 `mentioned` 会撞同一毫秒；镜像行的 `created` 直接取上游 `updated_at`（`community/notify/map.go:62`），同一折叠批次也会撞。翻页会漏行/重行。
  - `sortOrder` 是**字符串拼进 SQL** 的。目前靠 `oneof=asc desc` 挡住，但这是一条只由 DTO tag 守着的注入面，迁移时不能原样搬。
  - `query.Count(&total)` 的错误被丢掉（`message_repo.go:60`），失败就静默返回 0。
  - **`total` 与 `messages` 不同谓词**：`total` 在 SQL 里数，`messages` 在 Go 里被 `userclient.IsRenderable` 二次过滤（`message_service.go:49`）。封禁发信人的行计进 `total`、不出现在 `messages`。`notice.vue:87` 用 `Math.ceil(data.total / 30)` 画分页器，于是页数会多于真实可见内容，最坏情况整页空白。

### 2.2 `GET /api/message/muted` —— 被静音类型的通知

- **链路**：`message_handler.go:42` → `MessageService.GetMutedMessages` `message_service.go:90` → 同一个 `FindMessages`，但走 `onlyTypes` 分支（`message_repo.go:53`）。
- **请求**：同 2.1 的 DTO，这里 `type` 是活的：`message_service.go:103` 先检查它在不在该用户的静音集合里，不在就返回空列表（**不是 400**）。
- **响应**：与 2.1 逐字相同。
- **错误**：同 2.1。另有两条「静默空集」路径：没有静音任何类型（`message_service.go:98`）、`type` 不在静音集合里（`message_service.go:105`）。两者都是 `200 {messages:[], total:0}`，客户端无法区分「你没静音过」和「你传了个不存在的 type」。
- **鉴权**：同 2.1，仅凭证决定收件人。
- **调用方**：`apps/web/app/pages/message/muted.vue:41`（带 `tab` → `type`）、`apps/web/app/components/message/aside/MutedItem.vue:2`（`limit:1` 只为取 `total` 画角标，`server:false, lazy:true`）。App 契约 `docs/proj/app-direct-api.md:102`。
- **语义坑**：静音**不只是**熄灭红点。`GetMessages` 把静音类型整个 `NOT IN` 掉（`message_service.go:82` → `message_repo.go:56`），所以被静音的通知从主列表消失、只在 `/message/muted` 里能找到。而 `NotificationPreference.vue:86` 的文案写的是「消息仍会保留在通知中心里，你随时可以进去查看」。文案与行为对不上。

### 2.3 `DELETE /api/message/:id`

- **链路**：`message_handler.go:60` → `MessageService.DeleteMessage` `message_service.go:119` → `MessageRepository.DeleteByIDAndReceiver` `message_repo.go:71`；若删掉的是一条未读的镜像行，再异步 `forwardRead`（`message_service.go:124`）。
- **请求**：路径参数 `id`，`strconv.Atoi`，非数字 → `400 "无效的消息 ID"`（`message_handler.go:68`）。没有请求体。
- **响应**：`response.OKMessage(c, "消息已删除")` → `{code:0, message:"消息已删除"}`，**没有 `data`**。
- **错误**：`id` 非数字 → 400；仓储报错 → `500 "删除消息失败"`。
- **鉴权与寻址**：`message_repo.go:73` 是 `Where("id = ? AND receiver_id = ?", id, receiverID)`，`receiverID` 只来自凭证。**删不掉别人的消息**（查询层面证明）。但 `errors.Is(err, gorm.ErrRecordNotFound)` 那一支返回 `(nil, nil)`（`message_repo.go:74`），service 直接返回 nil，**删别人的消息 / 删不存在的 id 都回 200「消息已删除」**。这是一个 404 该在的地方回了 200，也是一个可用来探测「这条 id 是不是我的」的静默差异——不过因为成功与失败响应逐字相同，实际上探测不到，算是歪打正着。
- **调用方**：`apps/web/app/components/message/aside/Notice.vue:22`。App 契约 `docs/proj/app-direct-api.md:100`。

### 2.4 `PUT /api/message/system/read` —— 把**通知**全部标已读（名字是错的）

- **链路**：`message_handler.go:115` → `MessageService.MarkAllRead` `message_service.go:130` → `MessageRepository.MarkAllRead` `message_repo.go:89`（`UPDATE message SET status='read', updated=now() WHERE receiver_id=? AND status='unread' RETURNING community_notification_id`）→ `forwardRead` `message_service.go:139`。
- **请求**：无参数、无请求体。
- **响应**：`{code:0, message:"已标记全部已读"}`，无 `data`。客户端拿不到「标了几条」。
- **错误**：仓储报错 → `500 "标记已读失败"`。
- **鉴权与寻址**：`WHERE receiver_id = ?` 来自凭证。改不动别人的行。
- **调用方**：`apps/web/app/pages/message/notice.vue:25`。App 契约 `docs/proj/app-direct-api.md:99`。
- **问题**：
  - **路径写着 `system`，动的是 `message` 表（通知）**；真正管系统公告的是 `/message/admin/read`。两个端点的名字互换了。
  - **它会把静音类型也一并标已读**——`UPDATE` 没有任何 type 过滤，而 `GetMessages` 只给用户看非静音的。用户在 `/message/notice` 停一下，`/message/muted` 里的未读也全没了。
  - **和翻页竞争**：`notice.vue:20` 的 `onMounted` 只要**当前这一页**有未读就发这个请求，而请求把**全部**未读标掉。用户看的是第 1 页，第 5 页的未读被静默清掉且永不再提示。
  - **和封禁过滤竞争（更严重）**：如果一个用户剩下的未读全部来自被封禁的发信人，`data.messages` 里一条都没有（`message_service.go:49` 过滤掉了），`hasUnreadMessage` 为 false，这个请求**不会发出**；而 `/user/status` 的红点是直接数 SQL 行（`user/repository/stats_repo.go:43`，不过 `IsRenderable`），所以红点永远亮着、用户没有任何办法熄灭它。
  - `forwardRead`（`message_service.go:139`）起了个 `context.Background()` 的 goroutine、分 100 一批调上游，失败只 `slog.Warn` 且 `return`（`message_service.go:155`）——**剩余批次直接丢弃**，不是只丢当前批。
  - 注释（`message_repo.go:86`）记录了一次真实事故：`gorm.Scan` 到 `[]*int64` 在 UPDATE 已提交之后才报错，于是接口回「标记已读失败」而 id 从没转发出去。这条注释是「隐形约束」类，迁移时不要删。

### 2.5 `GET /api/message/admin` —— 系统公告列表

- **链路**：`message_handler.go:77` → `MessageService.GetSystemMessages` `message_service.go:162` → `MessageRepository.FindSystemMessages` `message_repo.go:133` + `GetSystemReadCursor` `message_repo.go:151` + `userclient.Hydrate`。
- **请求**：**没有任何参数**。网页发的 `page` / `limit`（`system.vue:8`）被完全忽略，`FindSystemMessages` 是 `SELECT … FROM system_message ORDER BY created DESC` 无 `LIMIT`。
- **响应**：**裸数组**（不是 `{items,total}`），`data` 直接是 `[]dto.SystemMessageResponse`（`message_dto.go:35`）：`{id, is_read, content, admin:{id,name,avatar}, created}`。
- `is_read` 不是行上的列，是 `int64(r.ID) <= cursor` 算出来的（`message_service.go:178`），游标在 `system_message_read_state`。
- **错误**：仓储报错 → `500 "获取系统消息失败"`。`GetSystemReadCursor` 的错误被丢掉（`message_service.go:168`），读不到游标就当 0，**全部公告显示成未读**。
- **鉴权与寻址**：`system_message` 是全站广播，**不按收件人过滤**；`system_message.user_id` 是**发布公告的管理员**，不是收件人（`model/message.go:32`，`message_repo.go:136` 把它 hydrate 成 `admin` 字段）。每个人看到同一份列表，只有 `is_read` 因人而异。
- **调用方**：`apps/web/app/pages/message/system.vue:13`，条目渲染 `apps/web/app/components/message/aside/System.vue`。App 契约 `docs/proj/app-direct-api.md:101`。
- **重大事实：这张表在本仓里没有任何写入方。** `rg 'system_message' --type go` 的全部命中只有读（`message_repo.go:133/144/151`、`stats_repo.go:56`）、删（`admin/repository/purge_repo.go:210`）与 model 声明。没有 admin handler、没有 cron、没有 S2S 回调能新建一条系统公告。**这个端点服务的是一张冻结的历史表。**
- `System.vue:31` 用 `v-html="message.content"` 直出。内容来自 `system_message.content`（072 迁移从 `content_zh_cn` 搬过来的），没有任何服务端清洗。目前只有历史数据，但 v1 的 K13 明确不下发 `html`，迁移时这一条必须处理掉。
- **数据损坏路径**：`purge_repo.go:210` 删号时执行 `DELETE FROM system_message WHERE user_id = ?`。`user_id` 是**作者**，于是注销一个曾经发过公告的管理员账号，会把那条**全站广播**从所有人的收件箱里抹掉。

### 2.6 `PUT /api/message/admin/read` —— 系统公告全部已读

- **链路**：`message_handler.go:90` → `MessageService.MarkAllSystemRead` `message_service.go:187` → `GetMaxSystemMessageID` `message_repo.go:142` + `UpsertSystemReadCursorForward` `message_repo.go:160`。
- **请求**：无。**响应**：`{code:0, message:"已标记全部已读"}`。
- **错误**：两处都映射成 `500 "标记已读失败"`。
- **鉴权与寻址**：游标按凭证的 `user_id` upsert，`ON CONFLICT (user_id) DO UPDATE SET last_read_message_id = GREATEST(...)`（`message_repo.go:165`）——只进不退，并发安全。这是整个域里**唯一**写得正确的已读模型。
- **调用方**：`apps/web/app/pages/message/system.vue:20`。App 契约 `docs/proj/app-direct-api.md:101`。
- 注意 `GetMaxSystemMessageID` 在表空时返回 `(0, nil)`（`message_repo.go:146`：`err != nil || maxID == nil` 合成一支），**读表失败与表为空不可区分**，两者都把游标写成 `GREATEST(旧值, 0)` = 旧值，所以失败是无害的，但这是一条被吞掉的错误。

### 2.7 `GET /api/message/nav/system` —— 侧栏两行摘要

- **链路**：`message_handler.go:102` → `MessageService.GetNavSummary` `message_service.go:198` → `MessageRepository.GetNavSummary` `message_repo.go:173`。
- **请求**：无。
- **响应**：`[]map[string]any`，**固定两个元素、靠下标区分**（`message_repo.go:214`）：`[0]` 是 `route:"notice"`（通知），`[1]` 是 `route:"system"`（系统公告）。每项的键：`chatroom_name`、`content`、`last_message_time`、`count`、`unread_count`、`route`、`title`、`avatar`。
- 网页把它当 `ChatMessageAsideItem[]` 读（`apps/web/shared/types/chat-message.ts:1`），并**按下标**取：`aside/Container.vue:30` 用 `system[0]`，`:36` 用 `system[1]`。
- **死字段**：`chatroom_name` 恒 `""`、`avatar` 恒 `""`、`title` 恒 `"zako~"`（`message_repo.go:222/232`），而 `SystemItem.vue:21` 显示的是它自己的 `title` prop，服务端那个 `title` 从没被读过。`[1]`（系统公告）的 `content` 恒 `""`。这三个键存在只是为了和私信那行 `NavContactItem` 形状一致。
- **错误**：`message_service.go:201` 有一条 `500 "获取消息概要失败"`，但 `GetNavSummary` 永远返回 `nil` error（`message_repo.go:236`），**这条错误路径是死的**。内部四个 `Count` 与 `GetSystemReadCursor` 的错误全部被丢弃（`message_repo.go:182/183/189/190/191`），任何一处失败都静默变成 0。
- **鉴权与寻址**：通知那半按 `receiver_id = ?`（凭证）；公告那半 `sysTotal` 是 `SELECT count(*) FROM system_message` **全表、与用户无关**，`sysUnread` 是 `id > cursor`。
- **调用方**：`apps/web/app/components/message/aside/Container.vue:5`。App 契约 `docs/proj/app-direct-api.md:97`。
- **真 bug：`content` 按字节切**。`message_repo.go:198`：
  ```go
  if len(latestNotice.Content) > 100 { noticeContent = latestNotice.Content[:100] }
  ```
  `len()` 是字节数，内容是中文。一刀切在多字节字符中间，`encoding/json` 会把残字节替换成 U+FFFD，侧栏预览尾巴上出现「�」。同一个文件里 `notifier.go:135` 的 `truncateNotifyContent` 是按 rune 切的，说明作者知道该怎么写，只是这里漏了。
- **计数口径与列表不一致**：`noticeUnread` / `noticeTotal` 只排静音类型，不过 `IsRenderable`。与 2.1 的 `messages` 是两个谓词，角标数会大于列表里数得出来的条数。

### 2.8 `GET /api/message/nav/contact` —— 私信会话列表

- **链路**：`chat_handler.go:22` → `ChatService.GetNavContact` `message/service/chat_service.go:31` → `ChatRepository.FindRoomsForUser` `message/repository/chat_repo.go:58` + `FindParticipantsByRoomIDs:70` + `CountUnreadByRoomIDs:79` + `CountTotalByRoomIDs:90` + `userclient.Hydrate`。
- **请求**：无。**响应**：`[]dto.NavContactItem`（`message/dto/chat_dto.go:38`）：`{chatroom_name, content, last_message_time, count, unread_count, route, title, avatar}`。
- `route` 对私聊是**对方的 user id 的字符串**（`chat_service.go:72`），网页拿它拼 `/message/user/${room.route}`（`aside/Item.vue:12`）并 `parseInt` 回来当 `KunAvatar` 的 id（`Item.vue:16`）。一个字段同时是路由段和用户 id，名字却叫 `route`。
- **错误**：`FindRoomsForUser` 报错 → `500 "查询聊天室失败"`。其余三个仓储方法**连 error 都不返回**（`chat_repo.go:70/79/90` 全部 `Scan` 后直接 return，签名里没有 error），失败就静默丢计数。
- **鉴权与寻址**：`chat_repo.go:63` `JOIN chat_room_participant crp ON crp.chat_room_id = cr.id WHERE crp.user_id = ?`，凭证决定。读不到别人的会话。
- **调用方**：`apps/web/app/components/message/aside/Container.vue:8`。App 契约 `docs/proj/app-direct-api.md:104`。
- **N+1 / 批量**：`Hydrate` 把一页里全部 participant 的 id 去重后一次 `/users/batch`（`pkg/userclient/hydrate.go:32`、`userclient.go:216`），**不是 N+1**。但 `/users/batch` 一次最多 100 个 id（C6），`userclient` 侧是否分批需要看 `Users()` 实现；会话多的用户 participant 数会超。
- `FindRoomsForUser` 过滤 `cr.last_message_sender_id != 0 AND cr.last_message_time IS NOT NULL`——这就是空房间不出现在侧栏的原因，也掩盖了 2.9 的空房间泄漏。

### 2.9 `GET /api/message/chat/history` —— 私信历史（**一个会写库的 GET**）

- **链路**：`chat_handler.go:35` → `ChatService.GetChatHistory` `chat_service.go:91` → `findOrCreatePrivateRoom` `chat_service.go:216`（→ `FindPrivateRoomBetween` `chat_repo.go:100` / **`CreatePrivateRoom` `chat_repo.go:115`**）→ `FindMessagesByRoom` `chat_repo.go:141` → **`MarkMessagesRead` `chat_repo.go:191`** → `Hydrate`。
- **请求**：`dto.GetChatHistoryRequest` `chat_dto.go:3`：`receiver_id`（`required,min=1`）、`page`（`min=1`，事实必填）、`limit`（`min=1,max=50`，事实必填）。注意 `limit` 上限 50，而通知列表是 30——两个集合两个上限。
- **响应**：裸数组 `[]dto.ChatMessageItem`（`chat_dto.go:24`）：`{id, chatroom_name, sender:{id,name,avatar}, receiver_id, content, content_html, is_recall, created, recall_time, edit_time, read_by}`。**没有总数、没有游标**。
- `content_html` 由 `markdown.RenderInline`（`infrastructure/markdown/inline.go:102`）生成并用 bluemonday 清洗；`img` 的 `src` 必须匹配 `allowedImageHosts`（`inline.go:24`），这就是 memory `kungal-content-image-token` 里「inline 白名单会静默吃图」的那条闸，`inline.go:44` 的注释是它的记录，属隐形约束，别删。
- **`read_by` 恒为 `[]`**（`chat_service.go:147`）。`chat_message_read_by` 表在写、在数未读，但从来没有被读回 DTO。网页 `shared/types/chat-message.ts:16` 还给它声明了 `KunUser[]`。死字段。
- **错误**：`receiver_id == 自己` → `400 "不能给自己发送消息"`（`chat_service.go:96`，一个 GET 回这句文案）；建房失败 → `500 "查询聊天室失败"`。`FindMessagesByRoom` 与 `MarkMessagesRead` 的失败都不影响响应（后者只 `slog.Warn`，`chat_service.go:118`）。
- **鉴权与寻址**：房间由 `findOrCreatePrivateRoom(当前用户, receiver_id)` 决定，`FindPrivateRoomBetween`（`chat_repo.go:102`）要求房间同时含两个 user id，所以**读不到自己不在的会话**。
- **调用方**：`apps/web/app/components/message/pm/Container.vue:47`。App 契约 `docs/proj/app-direct-api.md:105`。
- **问题**：
  - **GET 有副作用，而且是两种**：建 `chat_room` + 两行 `chat_room_participant`（`chat_repo.go:115`），以及写 `chat_message_read_by`（`chat_repo.go:191`）。任何登录用户可以对任意 `receiver_id` 反复打这个 GET，无限制地造空房间。侧栏靠 `last_message_time IS NOT NULL` 把它们藏住，所以没人发现。
  - **建房有竞态**：`chat_room.name` 上有唯一索引（`model/message.go:50`），两个并发 GET 里输的那个撞唯一约束 → `500 "查询聊天室失败"`。没有 `ON CONFLICT`。
  - **`FindMessagesByRoom` 的 `OR`**：`chat_repo.go:148` 是 `WHERE cm.chat_room_id = ? OR cm.chatroom_name = ?`。`chatroom_name` 是 `chat_message` 上的反规范化自由文本列，没有约束能保证它与 `chat_room_id` 一致。只要历史数据里有一行 `chatroom_name` 对不上自己的 `chat_room_id`，它就会出现在别人的会话里。同时这个 `OR` 让索引用不上。
  - **分页没有 tie-breaker 但是有序**：`Order("cm.id DESC")`（`chat_repo.go:149`），主键唯一，这里是对的——全域唯一正确的一处分页。
  - **`MarkMessagesRead` 的 SQL 是手拼占位符**（`chat_repo.go:196`），`limit` 上限 50 所以最多 50 组，可控。
  - **私信不过 `IsRenderable`**：`chat_service.go:123` 只 `Hydrate` 不过滤。被封禁用户的私信照常渲染，与通知列表的行为相反。这正是 memory 里「逐 mapper 接入」的代价——这个 mapper 没接。

### 2.10 `POST /api/message/chat/send`

- **链路**：`chat_handler.go:53` → `ChatService.SendChatMessage` `chat_service.go:153` → `markdown.NormalizeStoredContent` → `findOrCreatePrivateRoom` → 事务里 `InsertChatMessage` `chat_repo.go:207` + `UpdateRoomLastMessage` `chat_repo.go:215`。
- **请求**：`dto.SendChatMessageRequest` `chat_dto.go:9`：`receiver_id`（`required,min=1`）、`content`（`required,min=1,max=1000`）。**没有 `Idempotency-Key`**（App 契约 `docs/proj/app-direct-api.md:106` 也这么写）。
- **响应**：`{code:0, message:"发送成功"}`，**不回新消息**。网页因此在发送后重新拉整个第 1 页（`pm/Container.vue:73`）。
- **错误**：发给自己 → `400 "不能给自己发送消息"`；建房或事务失败 → `500 "创建聊天室失败"` / `500 "发送消息失败"`。
- **鉴权与寻址**：发信人只能是凭证本人（`chat_handler.go:64` 传 `user.ID`）。**收件人完全由请求体决定，且不做任何存在性检查**：`receiver_id` 可以是任何正整数，包括不存在的用户。没有拉黑、没有互关要求、没有封禁检查、没有频率限制。
- `UpdateRoomLastMessage` 把**消息正文原样**写进 `chat_room.last_message_content`（`chat_repo.go:217`），侧栏预览直出。
- **调用方**：`apps/web/app/components/message/pm/Container.vue:68`。

### 2.11 `POST /api/message/chat/recall`

- **链路**：`chat_handler.go:70` → `ChatService.RecallMessage` `chat_service.go:182` → `FindMessageHeader` `chat_repo.go:163` → `MarkMessageRecalled` `chat_repo.go:175` →（若是房间最后一条）`IsLatestMessageInRoom` `chat_repo.go:182` + `UpdateRoomLastMessage`。
- **请求**：`{message_id:int}`（`chat_dto.go:14`，`required,min=1`；handler 又查了一次 `<= 0`，`chat_handler.go:80`，重复校验）。
- **响应**：`{code:0, message:"撤回成功"}`。
- **错误**：不存在 → `404 "消息不存在或已被删除"`；不是自己发的 → `403 "您只能撤回自己发送的消息"`；已撤回 → `400 "该消息已被撤回"`；更新失败 → `500 "撤回消息失败"`。**这是全域唯一把 404/403/400 分开的端点。**
- **鉴权与寻址**：`chat_service.go:191` 显式比对 `header.SenderID != userID`。改不动别人的消息。但注意 `FindMessageHeader` 用 `Scan` 而不是 `First`，`err != nil || h.ID == 0` 合成一支（`chat_repo.go:169`），**查库失败与消息不存在都回 404**，是一个 500 该在的地方回了 404。
- 撤回后的预览是服务端拼的中文句子 `fmt.Sprintf("%s撤回了一条消息", senderName)`（`chat_service.go:208`），写进 `chat_room.last_message_content`——**一句中文被持久化进数据库**，v1 §7「服务端不得产出给终端用户看的句子」在这里不止是接口问题，是数据问题。
- 没有撤回时限（微信式的 2 分钟之类），任何时候都能撤回自己任意历史消息。
- **调用方**：`apps/web/app/components/message/pm/Container.vue:193`。

### 2.12 / 2.13 `GET` / `PUT /api/user/notification-preferences`

- **链路**：`user_handler.go:68` / `:80` → `UserService.GetNotificationPreferences` `user/service/user_service.go:179` / `UpdateNotificationPreferences` `:187` → `msgService.SanitizeMutedKeys` `message/service/prefs.go:27` + `StateRepository.Ensure` / `UpdateMutedTypes`。
- **存储**：`kungal_user_state.muted_notification_types`，`jsonb NOT NULL DEFAULT '[]'`（迁移 `apps/api/migrations/053_add_notification_preferences.up.sql:8`），model `user/model/state.go:12` 用 `serializer:json`。
- **请求**：`GET` 无参数；`PUT` body `{muted_types: string[]}`（`user/dto/notification_dto.go:7`）。**`muted_types` 上没有任何 validate tag**：可以传任意长度的数组、任意字符串，`SanitizeMutedKeys` 会把不认识的丢掉（`prefs.go:31`），所以不会写脏，但一个 10 万项的数组会被完整解析一遍。
- **响应**：两者都是 `{muted_types: string[]}`，`PUT` 回的是**清洗后**的结果（`user_service.go:194`），客户端据此发现自己传的键被丢了——但没有任何错误提示。
- **错误**：`GET` **没有错误路径**：`user_service.go:181` 把 `stateRepo.FindByID` 的错误吞掉当成「没静音」。`PUT` 两处 → `500 "保存通知偏好失败"`。
- **鉴权**：`userAuth`，仅凭证。无能力检查。
- **合法键集合**（`prefs.go:7` + `:23`）：`LocalNotificationTypes` 的 18 个 + `chat`，共 **19** 个。`SanitizeMutedKeys` 顺带去重、保序。
- **`SplitMuted` `prefs.go:39`**：把 `chat` 抽成布尔，把 `wiki:` 前缀的键**静默吞掉**（退役的 wiki 通知），其余当本地 type。`prefs_test.go` 钉了这两条。
- **三处镜像**：后端 `prefs.go:7`、前端 `apps/web/app/constants/notification.ts:67`（分组 + 中文标签）、`apps/web/shared/types/message.ts:1`（`MessageType` 联合类型）。**没有任何测试钉住三者一致**。当前它们确实一致（19 个键），但 `MessageType` 多了一个永远写不出来的 `'admin'`。
- **调用方**：`apps/web/app/components/message/NotificationPreference.vue:35`（读）与 `:55`（写）、`apps/web/app/pages/message/muted.vue:12`（读，用来生成 tab）。App 契约 `docs/proj/app-direct-api.md:103`。
- `NotificationPreference.vue:45` 的 `persist` 有一个 `queued` 单飞队列，保证并发切换不会乱序；失败时 `catch` 里重新 `load()` 回滚 UI（`:64`）。这段是对的。

### 2.14 `GET /api/user/status` —— 红点

- **链路**：`user_handler.go:55` → `UserService.GetUserStatus` `user/service/user_service.go:144` → `StateRepository.FindByID` + `UserStatsRepository.CountUnreadMessages` `user/repository/stats_repo.go:43` + `CountUnreadSystemMessages:54` + `CountUnreadChatMessages:65` + `userClient.User`（算 `is_creator`）。
- **请求**：无。**响应**：`dto.UserStatusResponse` `user/dto/auth_dto.go:64`：`{moemoepoints, is_check_in, has_new_message, daily_toolset_upload_bytes, is_creator}`。
- **`has_new_message` 的定义**：`(unreadMessage + unreadSystem + unreadChat) > 0`（`user_service.go:173`）。三项：通知（排静音 type）、系统公告（`id > 游标`）、私信（`chat` 被静音则跳过，`user_service.go:161`）。
- **错误**：**一条都没有**。三个 Count 的错误全部丢弃（`user_service.go:158/159/162`，`_` 接收），`stateRepo.FindByID` 的错误也丢（`:149`）。数据库挂了，红点就是「没有新消息」。
- **鉴权**：`userAuth`，仅凭证。
- **调用方**：`apps/web/app/components/kun/top-bar/Nav.vue:34`（挂载时一次，不是轮询——`docs/proj/app-direct-api.md:92` 写的「轮询」与代码不符）、`apps/web/app/plugins/validate-session.client.ts:6`、`apps/web/app/utils/kunFetch.ts:106`（把它当会话存活探针用）。
- **与列表口径不一致**：`CountUnreadMessages` 数的是 SQL 行，不过 `IsRenderable`。见 2.4 的「永远熄不灭的红点」。
- **系统公告不可静音**：`CountUnreadSystemMessages`（`stats_repo.go:54`）无条件计入，且 `system` 不在 `SanitizeMutedKeys` 的合法键里（`prefs.go:23` 只加了 `chat`）。**memory 的这条成立**。

## 3. 数据库表

### 3.1 `message`（通知收件箱）

DDL 基线 `apps/api/migrations/000_baseline.up.sql:1530`，后续 027（`content` → `text`）、093（重写 link）、095（镜像列）。

| 列 | 类型 | 读 | 写 |
|---|---|---|---|
| `id` | integer PK | 全部列表、`DELETE /:id` | 自增 |
| `content` | text（原 varchar(233)） | 列表、nav 摘要 | 全部通知写入方，按 233 rune 截断 |
| `link` | varchar(100) | 列表 | 写入方生成；`map.go:28` 按字节截 100 |
| `status` | text `'unread'\|'read'` | 列表、未读计数 | `MarkAllRead`、`MarkCommunityThreadRead`、镜像 upsert |
| `type` | text（18 个取值，见 §5） | 列表、静音过滤、未读计数 | 全部写入方 |
| `sender_id` | integer | 列表（hydrate + `IsRenderable`）、去重判据 | 全部写入方 |
| `receiver_id` | integer | **全部查询的唯一寻址列** | 全部写入方 |
| `created` | timestamp(3) → timestamptz（022） | 排序、列表、nav | 写入方；镜像行写的是上游 `updated_at` |
| `updated` | 同上 | 没有任何地方读 | `MarkAllRead`、`MarkCommunityThreadRead`、镜像 upsert |
| `community_notification_id` | bigint NULL（095） | `community` 布尔、`forwardRead` | 镜像 upsert（唯一索引冲突键） |
| `community_seq` | bigint NULL（095） | `MaxCommunitySeq`（游标 bootstrap）、upsert 的 seq 闸 | 镜像 upsert |
| `community_thread_id` | bigint NULL（095） | `MarkCommunityThreadRead` | 镜像 upsert |
| `community_post_number` | integer NULL（095） | `MarkCommunityThreadRead` | 镜像 upsert |
| `item_count` | integer NOT NULL DEFAULT 1（095） | 列表（`<1` 时兜到 1，`message_service.go:53`） | 镜像 upsert；本地写入方不写，吃默认值 |
| `actor_count` | integer NOT NULL DEFAULT 1（095） | 同上 | 同上 |

**索引**：`message_pkey`；`idx_message_type_created (type, created DESC)`（迁移 016，为首页 feed 建的）；`message_community_notification_id_key`（唯一，partial）；`message_receiver_community_thread_idx (receiver_id, community_thread_id)` partial（只覆盖镜像行）。

**没有 `receiver_id` 上的通用索引，也没有 `(receiver_id, created)`。** 收件箱的每一次列表、每一次 `COUNT`、每一次去重判据（`sender_id, receiver_id, type, link`）都在一张 016 迁移注释里写着「199k 行」的表上做。partial 索引只救镜像行。

**触发器**：`trg_feed_message AFTER INSERT OR UPDATE OR DELETE ON message`（迁移 `034_create_feed_activity.up.sql:231`，090 重写了函数）——每一条通知的增删改都会触发首页 feed 的同步写。`MarkAllRead` 是一条 `UPDATE … WHERE status='unread'`，会把该用户全部未读行逐行过一遍这个触发器。

### 3.2 `system_message`（系统公告）

`000_baseline.up.sql:1595`。列：`id`、`content`（072 加、073 之后是唯一正文列）、`content_{en_us,ja_jp,zh_cn,zh_tw}`（073 已 drop）、`user_id`（= **作者**）、`created`、`updated`。`status` 列已被迁移 012 删掉。

读：`FindSystemMessages`（全表，无 LIMIT）、`GetMaxSystemMessageID`、`CountUnreadSystemMessages`、`GetNavSummary` 的 `sysTotal`。
写：**无**。删：`purge_repo.go:210`（按作者删，见 2.5）。

### 3.3 `system_message_read_state`（公告已读游标）

迁移 `012_system_message_read_state.up.sql`。列 `user_id` PK、`last_read_message_id`、`updated_at`。读 `GetSystemReadCursor`、`stats_repo.go:57` 的子查询；写 `UpsertSystemReadCursorForward`（GREATEST 前进）。删 `purge_repo.go:211`。迁移 012 在 `cmd/migrate/main.go:25` 的默认 `--exclude` 里，必须在 OAuth `migrate-users` 之后 `--only=012`。

### 3.4 `kungal_user_state.muted_notification_types`

迁移 053。jsonb 数组。只有 §2.12/2.13 两个端点写，`message_service.go:34`、`user_service.go:153/181` 读。迁移 053 的注释现在**有两处过时**：它说键包含「stream pseudo keys (system/chat)」——`system` 从来没进过 `SanitizeMutedKeys` 的白名单；它说包含「namespaced wiki keys (wiki:approved/…)」——已随 wiki 退役，`prefs.go:44` 现在把它们吞掉。

### 3.5 私信五张表

`chat_room`（`000_baseline.up.sql:148`）、`chat_room_participant`（`:213`）、`chat_message`（`:50`）、`chat_message_read_by`（`:117`）、`chat_room_admin`（`:165`）、`chat_message_reaction`（`:86`）。

- `chat_room`：读 `id,name,avatar,type,last_message_content,last_message_time`；写 `name,type,created,updated`（建房）与 `last_message_{content,time,sender_id,name},updated`（发送/撤回）。`avatar` 只读不写（恒 `''`）。
- `chat_room_participant`：读 `chat_room_id,user_id`；建房时写两行。
- `chat_message`：读 `id,chatroom_name,sender_id,receiver_id,content,is_recall,created,recall_time,edit_time,chat_room_id`；写全部（发送）与 `is_recall,recall_time,updated`（撤回）。`edit_time` **只读不写**——私信没有编辑功能，这一列和 DTO 里的 `edit_time` 都是死的。
- `chat_message_read_by`：写 `chat_message_id,user_id,created,updated`（`ON CONFLICT DO NOTHING`）；读只在两个 `NOT IN` 子查询里（`chat_repo.go:84`、`stats_repo.go:70`），**从不 SELECT 出来给用户**（见 2.9 的 `read_by`）。
- `chat_room_admin` / `chat_message_reaction`：**代码里除了 model 声明和删号 DELETE 之外没有任何读写**。`chat_room_admin` 在 `purge_repo.go:169`，`chat_message_reaction` 在 `:167`。两张死表。

`chat_message_read_by` 与 `chat_message_reaction` 各有一个 `(chat_message_id, user_id[, reaction])` 唯一索引（`000_baseline.up.sql:3546/3552`）；`chat_message` 有指向 `chat_room` 与 `"user"` 的外键（`:3873/3928/3933`）。

## 4. 消息行的全部创建者（迁移时按这张表逐个搬）

一共 **6 条写入路径**，四种去重写法，没有一处共用。

| # | 位置 | 写法 | 去重判据 | type |
|---|---|---|---|---|
| 1 | `message/service/notifier.go:66` `notifier.Emit` | `Count` + `Create` | `sender_id, receiver_id, type, **content**, link` | `solution` `requested` `merged` `declined` `lottery-won` `lottery-closed` `lottery-expired` `poll-closed` |
| 2 | `topic/service/interaction_helpers.go:62` `createDedupMessage` | `Count` + `Create` | `sender_id, receiver_id, type, link`（**不含 content**） | `liked` `upvoted` `favorite` `mentioned` `pin-reply` |
| 3 | `topic/service/interaction_helpers.go:32` `CreateReplyMessage` | 直接 `Create` | **不去重** | `replied` `commented` |
| 4 | `topic/apiv1/engage_notify.go:33` `dedupMessage` | `Count` + `Create` | `sender_id, receiver_id, type, link` | `upvoted` `liked` `favorite` `pin-reply` `replied`（v1 写面） |
| 5 | `galgame/service/interaction.go:18/44/70` | `Count` + `Create` ×3 份复制 | `sender_id, receiver_id, type, link` | `liked` `favorite` `expired` `quiz-answered` `mentioned` |
| 6 | `galgame/service/resource_comment_write.go:308` `notifyDeduped` | `Count` + `Create`，**不在事务里**，失败只 `slog.Warn`（`:328`） | `sender_id, receiver_id, type, content, link` | `commented` |
| 7 | `message/repository/community_mirror.go:7` `UpsertCommunityMirror` | `INSERT … ON CONFLICT (community_notification_id) DO UPDATE … WHERE seq <` | 上游 notification id | `replied` `commented` `mentioned` `followed` `liked`（由 `community/notify/map.go:71` 的 `mapKind` 决定） |

调用点（全部 `Emit`/helper 的实参）：

- `notifier.Emit`：`topic/apiv1/engage_notify.go:65`（solution）、`topic/service/topic_write_service.go:602`（solution）、`topic/service/lottery_draw.go:476/500/531/564`（四个 miniapp type，经 `EmitMany` `:505/536/570`）、`galgame/handler/edit_handler.go:192`（`notifyDecision`，被 `:858` merged / `:898` declined 调用）、`edit_handler.go:429`（requested）。
- 话题旧路径：`topic_write_service.go:376`(liked) `:468`(upvoted) `:509`(favorite)、`reply_service.go:221`(replied) `:521`(pin-reply)、`comment_service.go:114`(commented)、`interaction_helpers.go:55/103`(mentioned)。
- 话题 v1：`engage_upvote.go:81`(upvoted) `engage_reactions.go:76/169`(liked) `engage_favorite.go:37`(favorite) `engage_choice.go:171`(pin-reply) `write_reply.go:61`(replied)。
- galgame：`galgame_service.go:92`(liked) `collection_service.go:288`(favorite) `resource_service.go:552`(liked) `:607`(expired) `rating_service.go:357`(liked) `quiz_service.go:335`(quiz-answered) `community_comment_write.go:109`(mentioned) `resource_comment_write.go:159`(commented)。
- 镜像：`community/notify/poller.go:131`，由 `poller.go:36` `Start()` 每 15 秒跑一次（`poller.go:18`）。

**链接生成**：本地写入方统一走 `msgService.BuildTopicLink`（`notifier.go:110`）或手拼 `fmt.Sprintf("/galgame/%d", …)` / `/galgame-quiz/%d` / `/toolset/%d` / `/website/<domain>` / `src.pageLink`（`galgame/service/resource_comment_service.go:60`）。镜像走 `anchor.Target.Link`，对 `site_game` 覆盖成 `/galgame/<id>?comment=<post_id>`（`map.go:26`）。

**link 指向死路由的历史**：迁移 093 修过一次——136 条 `message.link` 指向 `/galgame-resource/:id`，那是一条从未存在过的 Nuxt 页面，靠 route-rule 重定向活着，而 Nuxt 客户端导航不展开 `**`（memory `kungal-route-rule-client-wildcard`）。这类行只能靠迁移逐条修，v1 迁移必须重新扫一遍（§8 第 8 项）。

## 5. `message.type` 词表

代码能写出来的**恰好 18 个**：

```
upvoted  liked  favorite  replied  commented  mentioned  followed
solution  pin-reply  quiz-answered  expired  requested  merged  declined
lottery-won  lottery-closed  lottery-expired  poll-closed
```

三处镜像：后端 `message/service/prefs.go:7`（静音白名单）、前端 `apps/web/app/constants/notification.ts:67`（分组 + 标签）、前端 `apps/web/shared/types/message.ts:1`（联合类型）。

**`admin` 是死值**：常量 `NotifyAdmin` 定义在 `notifier.go:25` 但**零调用**；`MessageType` 联合里有它（`message.ts:15`）；`getMessageI18n.ts:142` 还给它准备了文案「系统消息」。没有任何代码能写出 `type='admin'` 的行。生产库里可能有远古数据（§8 第 1 项）。

**大小写与构词**：全部小写，但分隔符不统一——`pin-reply` / `quiz-answered` / `lottery-won` 是 kebab-case，其余是单词。v1 §3 要求封闭枚举是 snake_case，18 个里有 6 个要改拼写。

**语义与名字对不上的**：

- `liked` 同时表示「话题被赞」「回复被赞」「galgame 被赞」「资源被赞」「评分被赞」，靠 `link` 区分。
- `favorite` 是动词化的过去式序列里唯一的名词（其余是 `upvoted` / `liked` / `replied`）。
- `followed` 不是「有人关注了你」，是「**你关注的评论区**有新评论」（`constants/notification.ts:88` 的标签才说清楚）。这个名字在 App 上会被直接理解反。
- `expired` 不是「你的什么东西过期了」，是「**有人报告**你的资源链接失效」（`resource_service.go:607`）。
- `solution` 与 `pin-reply` 一个是名词一个是动宾。
- `mentioned` 在网页上被**改写成 `replied` 的文案**：`getMessageI18n.ts:153`——非社区来源且有正文的 `mentioned` 显示成「回复了您!」。服务端的 type 与用户看到的句子不是一回事。

## 6. 命名问题（按 K7 / F1 逐条）

| 现状 | 问题 | v1 |
|---|---|---|
| `created` | 禁用名；镜像行里装的其实是上游 `updated_at` | `created_at`，镜像行需要重新想清楚语义 |
| `updated` | 禁用名；**没有任何读者** | 删或 `updated_at` |
| `status`（`'unread'/'read'`） | K7 要求资源生命周期叫 `state`；而且这是「查看者状态」不是资源状态 | `viewer.is_read`（K9） |
| `sender` / `receiver_id` | K7：指人的字段按角色命名，`sender` 尚可，但通知的「发起人」在 infra 词汇里是 `actor` | `actor` + 收件人不出现（凭证即收件人） |
| `receiver_id` 出现在响应里 | 恒等于调用者自己，纯冗余 | 删 |
| `community: bool` | 不是布尔属性，是「这条来自哪个来源」 | `source: "local" \| "community"`，或干脆不发 |
| `item_count` / `actor_count` | `_count` 结尾合规，但 `item` 指的是「上游帖子数」，名不副实 | `post_count` / `actor_count` |
| `link: string` | 一个前端路由路径，不是资源引用；跨端不可用（App 没有 `/topic/123`） | 结构化 target：`{object, id}` + 楼层/评论锚点 |
| `content: string` | 是**快照**的 markdown 预览，不是正文 | `preview` 或按 K13 结构化 |
| `admin`（`SystemMessageResponse.admin`） | 指的是公告作者 | `author` |
| `is_read`（系统公告） | 合规 | 进 `viewer` |
| `NavContactItem.route` | 实际是对方 user id | `peer: UserRef` |
| `NavContactItem.count` | 是会话里的消息总数 | `message_count` |
| `NavContactItem.title` / `avatar` / `chatroom_name` | nav/system 那两行里恒为占位值 | 删 |
| `nav/system` 返回 `[]map[string]any` | 无类型、靠下标区分两种含义 | 两个具名对象或两个端点 |
| `ChatMessageItem.content_html` | v1 不下发 HTML | 结构化 content（K13） |
| `ChatMessageItem.read_by` | 恒 `[]` | 删或真的实现 |
| `ChatMessageItem.edit_time` | 恒 `null`，没有编辑功能 | 删 |
| `sort_order=asc\|desc` | K11 要求 `sort=<键>_<方向>` 的单 token | `sort=created_desc` |
| `PUT /message/system/read` | 路径说 system，动的是通知 | `PUT /notifications/read`（或按 K16 用槽位） |
| `GET /message/admin` | 路径说 admin，返回的是系统公告 | `GET /announcements` |
| `POST /message/chat/send` / `/recall` | 动词路径（B7） | `POST /conversations/{id}/messages`、`DELETE …/messages/{id}`（或 `PATCH` 改 `state`） |
| `GET /message/nav/contact` | 「nav」是 UI 概念 | `GET /conversations` |
| 裸数组响应（`/message/admin`、`/chat/history`、`/nav/*`） | 顶层不是 `{object:"list", items:[…]}` | K11 的 list 容器 |
| id 全是整数 | 违反 §3「全部 id 是字符串」 | 十进制字符串 |

## 7. Bug 清单（按严重度）

**S1 · 永远熄不灭的红点。** `/user/status` 数 SQL 行（`stats_repo.go:43`），`/message` 列表在 Go 里再过一次 `IsRenderable`（`message_service.go:49`），而 `notice.vue:21` 只在**列表里**看到未读才发 mark-all-read。一个用户若剩下的未读全部来自被封禁的发信人，红点亮着、列表空着、清不掉。`IsRenderable` 的 ~10 分钟 TTL 只影响它什么时候开始，不影响它会不会结束。

**S1 · `mark-all-read` 一口气吃掉三样东西。** `PUT /message/system/read`（`message_repo.go:89`）无 type 过滤、无 id 范围、无 `before` 时间戳：① 把被静音类型的未读也清了；② 把用户没翻到的后面几页清了；③ 把 SSR 渲染之后、onMounted 之前新到的清了。客户端只知道「第 1 页有未读」，服务端做的是全量。

**S1 · `system_message` 没有写入方，`purge` 却会删它。** 见 2.5。一个只读的冻结表，加上一条按**作者** `user_id` 删全站广播的删号语句（`purge_repo.go:210`）。

**S2 · 分页没有 tie-breaker。** `message_repo.go:63` 只按 `created` 排。同毫秒的批量通知（`NotifyMentions` 循环、镜像折叠同批）在翻页时会重复或丢失。`/message/chat/history` 按 `id DESC`，是对的；`/message/admin` 干脆不分页。

**S2 · `GET /message/chat/history` 是一个会建行的 GET。** 任意登录用户对任意 `receiver_id` 打这个 GET 就建一个 `chat_room` + 两行 participant（`chat_repo.go:115`），无速率限制。并发时唯一索引冲突 → 500。

**S2 · `nav/system` 的预览按字节切中文。** `message_repo.go:198` 的 `Content[:100]`，末尾出 U+FFFD。

**S2 · 镜像的 `liked` 永远标不掉已读。** `MarkCommunityThreadRead`（`message_repo.go:121`）的 `type IN ('replied','commented','mentioned','followed')` 不含 `liked`，而 `mapKind`（`map.go:82`）会写 `liked`。用户在评论墙上读完帖子，`liked` 那条镜像行留在未读里，只能靠全量 mark-all-read 清。

**S2 · `sender_id = 0` 的镜像行渲染成无名氏。** `map.go:38` 在 `n.ActorID == nil` 时写 `senderID = 0`；`CollectIDs` 跳过 `id <= 0`（`hydrate.go:13`），`userMap[0]` 是零值 `User{}`，`IsRenderable` 判 `Status == 0` 为真（`hydrate.go:48`），于是这一行**会渲染**，名字为空、头像为空、`KunLink` 指向 `/user/0`（`aside/Notice.vue:53`）。

**S3 · 六处几乎相同的去重逻辑，判据不一致。** §4 的表：`notifier.Emit` 和 `notifyDeduped` 把 `content` 计入判据，其余四处不计。同一个语义（「同一个人对同一个目标的同类通知只发一条」）有两种答案，且都是 `Count` + `Create` 的非原子读改写——并发时两条都能插进去，没有唯一约束兜底。

**S3 · 静默吞错误的地方（每一处失败都返回 0 / 空）。**
`message_repo.go:60`（列表 total）、`:182/:183/:189/:190/:191`（nav 五个 count）、`message_service.go:168`（公告已读游标）、`user_service.go:149/158/159/162`（红点四项）、`chat_repo.go:70/79/90/141/163/182`（六个方法连 error 都不返回）、`hydrate.go:37`（`/users/batch` 的错误）。
其中 `chat_repo.go:169` 把「查库失败」和「消息不存在」合并成 404，`message_repo.go:146` 把「读 MAX 失败」和「表为空」合并成 0。

**S3 · `message` 表没有 `receiver_id` 索引。** 全部收件箱查询与全部去重 `Count` 都在一张十几万行的表上做（016 迁移注释：199k 行）。现有索引只有 `(type, created DESC)` 和两个 partial。

**S3 · 私信没有任何屏蔽/封禁闸。** `SendChatMessage`（`chat_service.go:153`）不检查 `receiver_id` 是否存在、是否拉黑、是否封禁；`GetChatHistory` 的 mapper 不过 `IsRenderable`（与通知列表相反）。

**S3 · 服务端把中文句子写进数据库。** `chat_service.go:208` 的 `"%s撤回了一条消息"` 持久化进 `chat_room.last_message_content`。v1 §7 的「服务端不得产出给终端用户看的句子」在这里不是改接口就能解决的。

**S3 · `system_message` 的 `v-html`。** `aside/System.vue:31`。当前只有历史数据、且没有写入方，但迁移到 K13 前它一直是一个直出面。

**S3 · `/message/admin` 无分页、无上限。** `FindSystemMessages`（`message_repo.go:133`）全表返回，网页发的 `page`/`limit` 被忽略。

**S4 · `GET /message` 的 `type` 参数是死的**（`message_service.go:81`），DTO 里却声明着。`docs/proj/app-direct-api.md:98` 已经把它当特性写进契约了。

**S4 · `DELETE /message/:id` 删不存在的 / 别人的都回 200。**（`message_repo.go:74` → `message_service.go:120`）

**S4 · `sort_order` 字符串拼 SQL。**（`message_repo.go:63`）当前被 `oneof` 挡住，但这是一条只靠 DTO tag 守着的注入面。

**S4 · `FindMessagesByRoom` 的 `OR cm.chatroom_name = ?`**（`chat_repo.go:148`）：反规范化列没有约束保证与 `chat_room_id` 一致，且这个 `OR` 让索引失效。

**S4 · `chat_room_admin` 与 `chat_message_reaction` 是两张完全没有读写代码的死表**（只在 `purge_repo.go:167/169` 被删）。

**S4 · `forwardRead` 失败会丢弃剩余批次。**（`message_service.go:155` 的 `return` 在 for 循环里）

**S4 · 三处静音键镜像无测试。**（`prefs.go:7` / `constants/notification.ts:67` / `message.ts:1`）

**没有发现的**：N+1。`hydrateMessageRows`（`message_service.go:43`）与 `GetNavContact`（`chat_service.go:51`）都是先 `CollectIDs` 去重再一次 `Hydrate`，`/users/batch` 每页只打一次。唯一的逐个调用是 `chat_service.go:205` 的 `userClient.User`，那是撤回时取一个名字，不在循环里。

## 8. 三条 memory 的核对

**① 「通知偏好是 opt-out 静音标记，存 jsonb；静音在读时过滤、不删行；系统公告不可静音」** —— 基本成立，一处要改写。

- opt-out jsonb：成立。`kungal_user_state.muted_notification_types jsonb DEFAULT '[]'`（迁移 053:8，model `user/model/state.go:12`），前端语义是「开关打开 = 不静音」（`NotificationPreference.vue:20/31`）。
- 不删行：成立。静音只影响查询谓词（`message_repo.go:53/57`、`stats_repo.go:47`），没有任何 DELETE。
- **「读时过滤」要改写**：静音不只是熄红点，它把消息从 `/message` 主列表整个排除（`message_service.go:82`），只能在 `/message/muted` 找到。前端文案（`NotificationPreference.vue:86`）说的是「仍会保留在通知中心里」，与行为不符。
- 系统公告不可静音：成立。`system` 不在 `SanitizeMutedKeys` 的白名单（`prefs.go:23` 只加 `chat`），`CountUnreadSystemMessages`（`stats_repo.go:54`）无条件计入。
- **迁移 053 的注释已过时两处**：它说键包含 `system` 与 `wiki:*`，前者从未被接受，后者已在 `prefs.go:44` 被吞掉。

**② 「community follow + feed 镜像把上游活动写进 message 表；取关写 1 绝不写 0；Redis 游标全实例共享」** —— 三条全部成立。

- 镜像：`community/notify/poller.go:36` 每 15 秒（`:18`）拉 `NotificationFeed`，`map.go:15` 映射成 `model.Message`，`community_mirror.go:7` upsert 进 `message`，冲突键是 `community_notification_id`，seq 闸 `WHERE message.community_seq < EXCLUDED.community_seq`（`community_mirror.go:31`，注释记录了「重放的页不能把已读翻回未读」这条事故）。迁移 095。
- 取关写 1：`engagement.go:130-135` 原样，注释也在：`level := NotificationNormal`（1），只有 `following == true` 才写 `NotificationWatching`。**绝不写 0（muted）**，因为 0 会连带屏蔽针对本人的回复与提及。
- Redis 游标：`poller.go:17` 的 `cursorKey = "community:notification:feed:after"`，单一全局键，无实例后缀（`poller.go:162/175`）。所有副本共享，空游标时从 `MaxCommunitySeq()` bootstrap（`poller.go:164`）。换 community 库测试后必须删镜像行 + 游标归零——memory 的这条运维提醒仍然有效。

**③ 「封禁用户内容在渲染层藏，`userclient.IsRenderable`，逐 mapper，~10 分钟 TTL」** —— 成立，但**这个域只有一个 mapper 接了**。

- `IsRenderable(u) = u.Status == 0`（`pkg/userclient/hydrate.go:48`）。
- 接了的：`hydrateMessageRows`（`message_service.go:49`），覆盖 `/message` 与 `/message/muted`。
- **没接的**：`GetSystemMessages` 的 `admin`（`message_service.go:176`，公告作者，不接是对的）、`GetNavContact` 的对方（`chat_service.go:66`）、`GetChatHistory` 的发信人（`chat_service.go:127`）、`GetNavSummary` 的最新通知预览（`message_repo.go:196`，直接读 `content`，根本不 hydrate）、`CountUnreadMessages`（`stats_repo.go:43`，SQL 层）。
- TTL 滞后：`pkg/userclient` 自带 ~10 分钟缓存（C6），所以封禁后最多 10 分钟内旧行还会渲染。这是**次要**问题；**主要**问题是 §7 S1：过滤发生在 Go 里而计数发生在 SQL 里，两者永久不一致。

## 9. 代码读不出来、需要生产库回答的

按优先级，每条给出可直接跑的 SQL。

1. **`type` 词表实况（最重要）**——确认 18 个取值之外还有什么，以及各自的量与未读量：
   ```sql
   SELECT type, count(*) AS rows,
          count(*) FILTER (WHERE status = 'unread') AS unread
   FROM message GROUP BY 1 ORDER BY 2 DESC;
   ```
   特别想知道：有没有 `admin`；`followed` / `liked` 里镜像行占多少（`… AND community_notification_id IS NOT NULL`）。

2. **无名氏镜像行**（§7 S2）：
   ```sql
   SELECT count(*), count(*) FILTER (WHERE status = 'unread') FROM message WHERE sender_id = 0;
   ```

3. **标不掉的镜像 `liked`**（§7 S2）：
   ```sql
   SELECT count(*) FROM message
   WHERE community_notification_id IS NOT NULL AND type = 'liked' AND status = 'unread';
   ```

4. **`created` 撞车率**（分页 tie-breaker 的实际风险）：
   ```sql
   SELECT count(*) FROM (
     SELECT receiver_id, created FROM message GROUP BY 1, 2 HAVING count(*) > 1
   ) t;
   ```

5. **`system_message` 是不是真的冻结了**：
   ```sql
   SELECT count(*), min(created), max(created), count(DISTINCT user_id) FROM system_message;
   SELECT count(*) FROM system_message_read_state;
   SELECT count(*) FROM system_message_read_state s
     WHERE s.last_read_message_id < (SELECT coalesce(max(id), 0) FROM system_message);
   ```
   如果 `max(created)` 是很久以前，这个端点可以直接在 v1 里删掉而不是迁移。

6. **静音键的实际使用**——哪些类别真的有人关、有没有残留的退役键：
   ```sql
   SELECT k, count(*) FROM kungal_user_state,
        LATERAL jsonb_array_elements_text(muted_notification_types) k
   GROUP BY 1 ORDER BY 2 DESC;
   SELECT count(*) FROM kungal_user_state WHERE muted_notification_types <> '[]'::jsonb;
   ```
   我想确认 `wiki:*` 和 `system` 还有没有留在库里（代码只在读时吞掉，从没清洗过存量）。

7. **红点卡死的实际受害面**（§7 S1）——需要先从 OAuth 拿到封禁用户 id 集合 `:banned`，然后：
   ```sql
   SELECT count(DISTINCT receiver_id) FROM message
   WHERE status = 'unread' AND sender_id = ANY(:banned)
     AND receiver_id NOT IN (
       SELECT receiver_id FROM message
       WHERE status = 'unread' AND NOT (sender_id = ANY(:banned))
     );
   ```
   这个数字 = 现在红点永远亮着、自己清不掉的人数。

8. **link 指向不存在页面的行**（093 那类问题有没有复发）：
   ```sql
   SELECT substring(link from '^/[a-z0-9-]+') AS prefix, count(*)
   FROM message GROUP BY 1 ORDER BY 2 DESC;
   SELECT count(*) FROM message WHERE length(link) >= 100;  -- 被截断的
   SELECT count(*) FROM message WHERE link = '' OR link IS NULL;
   ```

9. **正文快照的长度分布**（027 之后有没有超过 233 rune 的历史行，以及 K13 结构化时要处理多少 HTML）：
   ```sql
   SELECT count(*) FILTER (WHERE char_length(content) > 233) AS over_233,
          count(*) FILTER (WHERE content ~ '<[a-zA-Z]') AS has_html,
          count(*) FILTER (WHERE content ~ '/image/[0-9a-f]{64}') AS has_image_token
   FROM message;
   ```

10. **私信的空房间**（§7 S2 那个 GET 副作用造了多少行）：
    ```sql
    SELECT count(*) FROM chat_room WHERE last_message_time IS NULL;
    SELECT count(*) FROM chat_room;
    ```

11. **`chatroom_name` 与 `chat_room_id` 不一致的行**（§7 S4 的 `OR` 是否真的有泄漏面）：
    ```sql
    SELECT count(*) FROM chat_message m
    JOIN chat_room r ON r.id = m.chat_room_id
    WHERE m.chatroom_name <> r.name;
    ```

12. **两张死表是不是真的空**：
    ```sql
    SELECT (SELECT count(*) FROM chat_room_admin), (SELECT count(*) FROM chat_message_reaction);
    ```

13. **索引实况**（确认 `receiver_id` 上真的什么都没有，以及 016/095 有没有在生产跑过）：
    ```sql
    SELECT indexname, indexdef FROM pg_indexes WHERE tablename IN ('message','system_message');
    SELECT count(*) FROM message;  -- 016 注释写的是 199k，现在多少
    ```

14. **重复通知**（§7 S3 的非原子去重实际产生了多少）：
    ```sql
    SELECT count(*) FROM (
      SELECT sender_id, receiver_id, type, link FROM message
      WHERE sender_id <> 0
      GROUP BY 1,2,3,4 HAVING count(*) > 1
    ) t;
    ```

15. **`item_count` / `actor_count` 的分布**（决定 v1 要不要保留折叠语义）：
    ```sql
    SELECT type, max(item_count), max(actor_count),
           count(*) FILTER (WHERE actor_count > 1) AS folded
    FROM message WHERE community_notification_id IS NOT NULL GROUP BY 1;
    ```
