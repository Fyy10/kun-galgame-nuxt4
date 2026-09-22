# 普查 · 用户域（`/api/user/**` + `/api/perm/**`）

> 只读普查。写于 2026-09-22，基于 `master` @ 45127518。没有连库、没有起服务，全部结论来自代码。
> 需要生产库才能确认的数字集中在 §7。

## 0. 范围与总览

`routes.golden` 里属于本域的路由共 **24 条**（18 GET / 3 POST / 3 PUT）。其中 22 条挂在 `/api/user/**`，2 条挂在 `/api/perm/**`。

| # | 方法 | 路径 | handler | 档位 | 拥有者 |
|---|---|---|---|---|---|
| 1 | GET | `/api/user/:id` | `user/handler/user_handler.go:32` | public | 本域 |
| 2 | GET | `/api/user/:id/floating` | `user_handler.go:123` | public | 本域 |
| 3 | GET | `/api/user/:id/topics` | `user_handler.go:151` | optional | 本域 |
| 4 | GET | `/api/user/:id/replies` | `user_handler.go:175` | public | 本域 |
| 5 | GET | `/api/user/:id/comments` | `user_handler.go:191` | public | 本域 |
| 6 | GET | `/api/user/:id/galgames` | `user_handler.go:135` | public | 本域 |
| 7 | GET | `/api/user/:id/galgame-comments` | `user_handler.go:223` | public | 本域 |
| 8 | GET | `/api/user/:id/resources` | `user_handler.go:207` | public | 本域 |
| 9 | GET | `/api/user/:id/ratings` | `user_handler.go:239` | public | 本域 |
| 10 | GET | `/api/user/:id/toolsets` | `toolset/handler/toolset_handler.go:39` | public | toolset 域寄挂 |
| 11 | GET | `/api/user/:id/collections` | `galgame/handler/collection_handler.go:147` | optional | galgame 域寄挂 |
| 12 | GET | `/api/user/status` | `user_handler.go:56` | required | 本域 |
| 13 | POST | `/api/user/check-in` | `user_handler.go:44` | required | 本域 |
| 14 | GET | `/api/user/moemoepoint/log` | `user_handler.go:96` | required | 本域 |
| 15 | GET | `/api/user/notification-preferences` | `user_handler.go:68` | required | 本域 |
| 16 | PUT | `/api/user/notification-preferences` | `user_handler.go:80` | required | 本域 |
| 17 | GET | `/api/user/search` | `user_handler.go:114` | required | 本域 |
| 18 | PUT | `/api/user/bio` | `user/handler/profile_handler.go:28` | required | OAuth 代理 |
| 19 | PUT | `/api/user/username` | `profile_handler.go:46` | required | OAuth 代理 |
| 20 | POST | `/api/user/avatar` | `profile_handler.go:64` | required | OAuth 代理 |
| 21 | GET | `/api/user/creator/status` | `galgame/handler/creator_handler.go:22` | required | galgame 域寄挂 |
| 22 | POST | `/api/user/creator/apply` | `creator_handler.go:42` | required | galgame 域寄挂 |
| 23 | GET | `/api/perm/mine` | `admin/handler/user_permission_handler.go:112` | required | admin 域 |
| 24 | GET | `/api/perm/bundles` | `admin/handler/role_permission_handler.go:46` | public | admin 域 |

路由注册在 `internal/app/router.go:60–85`（`/api/user/*`）、`:112`（`/perm/bundles`）、`:232`（`/user/:id/collections`）、`:247`（`/perm/mine`）。

**信封**：全部端点走旧信封 `pkg/response`。成功 `200 {code:0, message:"成功", data:…}`，`Paginated` 是 `data:{items,total}`。错误 `pkg/response.Error` → `{code, message}` + 一个 HTTP 状态码，`code` 的取值只有 `205`（401 登录失效）、`233`（400/403/404/500 混用）、`234`（403 封禁）三种，全域没有任何一个具名码。本域**零个** problem+json、零个 `object` 判别字段、零个字符串 id、零个 `viewer` 块。

**v1 现状**：`/api/v1` 下目前只有 topic / reply / galgame 三族（`routes.golden:29–342`），用户域**一条都没有迁**。

---

## 1. 公开读面：`/api/user/:id` 与 `/api/user/:id/floating`

### 1.1 `GET /api/user/:id`（个人资料）

链路：`UserHandler.GetProfile`（`user_handler.go:32`）→ `UserService.GetUserProfile`（`user/service/user_service.go:57`）→ `userclient.Client.User`（`pkg/userclient/userclient.go:175`）+ `UserStatsRepository.GetUserStats`（`user/repository/stats_repo.go:19`）+ `StateRepository.FindByID`（`user/repository/state_repo.go:29`）+ `GalgameUserStatsService.Stats` + `UserService.communityVisiblePosts`（`user/service/community_stats.go:41`）。

请求：路径参数 `:id`，`strconv.Atoi` 解析，**无查询参数、无请求体**。

响应（`dto.UserProfileDetail`，`user/dto/auth_dto.go:66`）：`id/name/avatar/roles[]/status/moemoepoint/bio/created` + 14 个计数（`topic` `topic_poll` `topic_lottery` `reply_created` `comment_created` `galgame` `contribute_galgame` `galgame_comment` `galgame_rating` `galgame_resource` `galgame_toolset` `galgame_toolset_resource` `upvote` `like` `dislike` `daily_topic_count` `daily_galgame_count`）。网页手写类型 `apps/web/shared/types/user.ts:3`，消费点 `apps/web/app/pages/user.vue:18`、`apps/web/app/components/user/ProfileHeader.vue`。

错误：`:id` 非数字 → `400/233 "无效的用户 ID"`；OAuth 不可达 → `500/233 "查询用户信息失败"`；OAuth 说没这个人 → `404/233 "未找到该用户"`；本地统计 SQL 失败 → `500/233 "获取用户统计失败"`。

鉴权：**完全匿名**，没有任何 capability 检查。

隐私：
- 不下发 email / ip / token，也不下发 OAuth `uuid`（`userclient.User` 解了 `uuid` 但 DTO 没带）。**这一点是对的。**
- 下发 `moemoepoint`。注意 infra 的立场与论坛相反：`docs/oauth/03-cross-service.md:107` 明写「响应中**不包含** `email`、`moemoepoint` 等**隐私字段**」，即 OAuth 把余额当隐私；论坛把任意用户的余额对匿名公开（这里、`/floating`、`/ranking/user` 三处）。这不是 bug，但是一条**跨服务口径分歧**，迁 v1 时要么明确它是公开数据，要么收进 `viewer`。
- 下发 `roles[]`（含 site roles 的并集，见 1.4）。陌生人能看到谁是 moderator/admin。站内本来就有角色徽标，属于设计如此。

**封禁用户走的是一条半成品分支**（`user_service.go:65`）：`u.Status != 0` 时只回 `{id, name, status}`，其余字段全部是 Go 零值。于是：
- `created` 序列化成 `"0001-01-01T00:00:00Z"`，`ProfileHeader.vue:91` 照样渲染「注册于 0001年1月1日」；
- 所有计数是 0，页面画出一个「0 话题 0 回复」的正常资料页；
- `KUN_USER_STATUS_MAP`（`apps/web/app/constants/user.ts:283`）只有 `0/1` 两个键，OAuth 契约却说「非 0 时调用方应隐藏或脱敏」——`status` 是**开放**的整数，出现第三个值页面就渲染 `undefined`。

### 1.2 `GET /api/user/:id/floating`（悬浮名片）

链路：`UserHandler.GetFloatingCard`（`user_handler.go:123`）→ `UserService.GetFloatingCard`（`user_service.go:198`）→ `userclient.User` + `UserStatsRepository.FindFloatingStats`（`stats_repo.go:82`）+ `communityVisiblePosts`。

**这个端点的路径参数是假的。** 它 `ParseQueryAndValidate` 进 `dto.FloatingCardRequest`（`user/dto/floating_dto.go:3`），字段是 `UserID int \`query:"user_id" validate:"required,min=1"\``——读的是**查询串** `?user_id=`，路径上的 `:id` **从头到尾没有被读过**。后果两条：
1. `GET /api/user/5/floating`（不带查询）→ `user_id` 为 0 → 校验失败 → `400/233`。
2. `GET /api/user/999/floating?user_id=3` → 返回 **3** 号的名片。路径段与响应内容可以不一致。

唯一的调用方因此必须两边都写：`apps/web/app/components/edit/topic/AccessUserPicker.vue:29-31` 同时拼了 `/api/user/${id}/floating` 和 `query: { user_id: id }`。它是 raw `$fetch`（绕开 `kunFetch` 的 toast），因为这个端点对封禁/注销用户回 404 而它不想弹错。

这同时是本域唯一一处真正的 **N+1 over `/users/batch`**：`resolveMissing`（`AccessUserPicker.vue:37`）对每个 grantee id 各发一次 HTTP，每次在后端变成一次单 id 的 `/users/batch`。10 分钟热缓存能吸收重复，首次打开编辑器时 N 个 grantee 就是 N 个往返。v1 应该给一个 `?ids=` 批量用户引用面（A9 的「出现真实消费者时再加」，这就是那个消费者）。

响应（`dto.FloatingCardResponse`）：`id/name/avatar/moemoepoint/topic_count/topic_reply_count/topic_comment_count/galgame_resource_count`。

错误：缺 `user_id` → `400/233`；OAuth 错 → `500/233`；**`!ok || u.Status != 0` → `404/233 "未找到该用户"`**。

**与 1.1 不一致**：同一个封禁用户，`/api/user/:id` 回 200（半资料），`/floating` 回 404。两个面对「封禁」的裁决不同。

`FindFloatingStats`（`stats_repo.go:82`）**吞掉了 SQL 错误**：`r.db.Raw(...).Scan(&stats)` 的返回值没有接，函数签名也没有 error。库挂了这张名片就是一张「0 话题 0 回复 0 资源」的卡片，看不出是坏了还是这人真没发过帖。这正是任务书点名的「unchecked error that silently returns zero」。

---

## 2. 个人内容列表（7 条）

7 条列表端点（topics / replies / comments / galgames / galgame-comments / resources / ratings）形状高度一致，问题也一致，先说共性再说各自的。

### 2.1 共性

**请求**：全部是 `type`（必填字符串）+ `page` + `limit` 的 query，经 `utils.ParseQueryAndValidate`（`pkg/utils/validate.go:66`）+ `go-playground/validator`。

- `Page int \`query:"page" validate:"min=1"\`` —— 零值 0 过不了 `min=1`，所以 **`page` 与 `limit` 事实上都是必填**，不填直接 `400/233 "请求参数验证失败"`。schema 上看着是可选，实际不是。
- `type` 是**开放的裸字符串**，没有 `oneof`。未知取值在 repo 层 `switch` 的 `default` 分支**静默回落**成「该用户自己发的」（`user/repository/content_repo.go:107/149/190/241`），`galgames` 则静默回落成空列表（`content_repo.go:35`）。01 §4「未知值 → `400 UNKNOWN_ENUM_VALUE`，不得静默回落」逐条被违反。实际词表（从 `switch` 读出来）：
  - topics: `topic` / `topic_like` / `topic_upvote` / `topic_favorite` / `topic_hide`
  - replies: `reply_created`(default) / `reply_target` / `reply_like`
  - comments: `comment_created`(default) / `comment_target` / `comment_like`
  - galgames: `galgame_like` / `galgame_favorite` / `galgame_publish` / `galgame_contributed`
  - galgame-comments: `galgame_comment_like` / 其余(default) = 本人所写
  - resources: `valid`(default) / `expire` / `galgame_resource_like`
  - ratings: 无 `type`

**响应**：**三种不同的分页信封**同时存在（01 §4 只允许两种，而且要在 spec 里声明）：
- `Paginated` → `data:{items,total}`：只有 galgames 一条（`user_handler.go:148`）；
- `fiber.Map` 自造键 → `{topics,total}` / `{replies,total}` / `{comments,total}`（`user_handler.go:172/188/204`）；
- 服务层自造 DTO → `{resources,total}` / `{rating_data,total}`（`dto.UserResourcesResponse` / `dto.UserRatingsResponse`）；
- galgame-comments 又是游标：`{comments, next_cursor}`（`user_handler.go:236`）。

同一个用户资料页的六个 tab，六种响应键名。网页每个 tab 各写一遍内联泛型（`apps/web/app/components/user/Topic.vue:29`、`Reply.vue:23`、`Comment.vue:23`、`Resource.vue:33`、`Rating.vue:15`、`Galgame.vue:40/62/94`、`Toolset.vue:14`、`CollectionTopic.vue:12`、`CollectionGalgame.vue:13`、`Overview.vue:12–42`）。

**分页没有 tie-breaker**：`content_repo.go` 里 6 处 `Order("… .created DESC")`（`:51 :118 :162 :203 :248 :309`）全部只按 `created` 排，没有带 `id`。`created` 是秒级以上精度的 timestamptz，批量导入/同秒创建的行在翻页时会重复出现或整行漏掉。01 §4「每个排序都带 `id` 作 tie-breaker」。

**`total` 与 `items` 不同谓词**：`GetUserResources`（`user/service/user_content_service.go:360-364`）和 `GetUserRatings`（`:415-417`）先按 SQL 的 `COUNT(*)` 拿 `total`，再在 Go 里把「catalog 拉不到 brief 的行」`continue` 掉。SFW 读者、被合并掉的作品、catalog 超时，都会让某一页返回 3 条而 `total` 说有 40 条。分页器因此永远画错。

**上游失败静默降级成空**：`GetUserGalgameCards`（`user_content_service.go:88`）catalog 报错 → 回空数组 + 原 `total`；`galgameCardsByIDs`（`:107`）同；`GetUserResources`/`GetUserRatings` 的 `briefMap, _ =` 直接丢错。catalog 抖一下，个人主页的「我发布的游戏」就是空的，没有任何信号。

**`hideTarget` 失败开放**（`user_content_service.go:44`）：

```go
func (s *UserContentService) hideTarget(ctx context.Context, userID int) bool {
	u, _, _ := s.userClient.User(ctx, userID)
	return !userclient.IsRenderable(u)
}
```

`found` 和 `error` 两个返回值都被丢掉。OAuth 不可达时 `u` 是零值 `User{Status:0}`，`IsRenderable` 为 true，于是**封禁用户的全部内容在 OAuth 故障期间照常可见**。这条「封禁用户内容在渲染层藏」的闸是 fail-open 的。

**偏好 cookie 仍是输入**：7 条里 6 条把 `utils.IsSFW(c)`（`pkg/utils/settings.go:39`）当过滤条件——它读 `X-Kungal-Nsfw` 头或 `KUNGalgameSettings` cookie。01 §3 末段「v1 不读 `KUNGalgameSettings`、不读 `X-Kungal-Nsfw`」。另外整个 `/api` 组挂了 `middleware.NamePreference`（`router.go:58`），本域虽然不直接用，但 v1 迁移时这一层要摘掉。

**一个测试都没有。** `internal/user` 下的测试只覆盖 OAuth 线协议、site-role 合并、和 galgame-comments 的游标排序（`galgame_comment_pagination_test.go`）。`GetProfile` / `CheckIn` / `GetStatus` / 通知偏好 / 萌萌点流水 / mention 搜索 / floating / 6 个内容列表 repo，**零测试**。

### 2.2 `GET /api/user/:id/topics`

唯一带 `OptionalAuth` 的列表（`router.go:80`）。`user_handler.go:160-168`：

- `type=topic_hide` 要求 `u.ID == userID || u.Can(perm.TopicViewHidden)`，用的是 **`user.Can`**（正确的 helper，Bearer 恒 false）。
- `canViewRestricted := viewer != nil && (viewer.ID == userID || viewer.Can(perm.TopicViewRestricted))`，同样正确。

repo 是 `FindUserTopics`（`content_repo.go:79`）。**这里有本域最严重的一条泄漏**：

```go
if queryType != "topic_hide" && !canViewRestricted {
    baseQuery = baseQuery.Where(topicRepo.SharedListPredicate("topic", authenticated))
}
```

`SharedListPredicate`（`topic/repository/access_repo.go:29`）只过滤 `access_scope`，**完全不看 `topic.status`**。而站内正牌的话题列表两处都写着 `Where("topic.status != 1")`（`topic/repository/list_repo.go:69`、`keyset_list.go:88`）。于是：

> **`GET /api/user/:id/topics?type=topic&page=1&limit=50` 对匿名访客返回该用户被版主隐藏的话题的 id 与标题。**

`type=topic_hide` 这条专门加了权限闸的分支，被旁边没加闸的 `type=topic` 整个绕过了。同样的洞也在 `topic_like` / `topic_upvote` / `topic_favorite` 三个分支上——它们列的是「这个人赞过/推过/收藏过的话题」，里面的隐藏话题同样带标题下发。

第二条：`canViewRestricted` 为真时整条 ACL 谓词被跳过，于是持 `topic.view_restricted` 的版主在**别人的**公开资料页里会看到那个人的受限话题。`access_repo.go:34` 的注释写着「Role and user grants never make a topic visible in shared lists」——个人资料页的列表就是一个 shared list，这条不变量在这里破了。

响应项只有 `{id, title, created}`（`dto.UserTopic`，`auth_dto.go:149`）。

### 2.3 `GET /api/user/:id/replies`

`FindUserReplies`（`content_repo.go:129`）。`topic_reply.status = 0` 有过滤，但**关联的 topic 完全不查可见性**：`reply_target` 分支是 `topic_reply.topic_id IN (SELECT id FROM topic WHERE user_id = ?)`，`reply_like` 是一个纯 join，都没有 `topic.status != 1` 也没有 ACL 谓词。隐藏话题里的回复正文（`COALESCE(topic_reply.content,'')`，整段 Markdown 源文）对匿名可见。`isSFW` 时才 join `topic` 且只查 `is_nsfw`。

响应项 `repository.UserReply`（`content_repo.go:122`）：`{topic_id, floor, content, created}`。注意 `content` 是**裸 Markdown 源文**，没有渲染也没有裁剪。

### 2.4 `GET /api/user/:id/comments`

`FindUserComments`（`content_repo.go:173`）。同样只有 `topic_comment.status = 0`，不查所属 topic 的可见性。

`comment_target` 分支读 `topic_comment.target_user_id`。迁移 044 删的是 `galgame_comment.target_user_id`；`topic_comment.target_user_id` 还在（`000_baseline.up.sql:422`）。需要生产库确认这列现在还有没有新写入——如果话题评论也已经改成行内 `@` token（迁移 028 对回复做过这件事），那 `comment_target` 和 `/user/:id/comments?type=comment_target` 就是一条只能查到历史数据的死路。

### 2.5 `GET /api/user/:id/galgames`

`GetUserGalgameCards`（`user_content_service.go:49`）。四个 `type` 走三条完全不同的代码路径：

- `galgame_publish` → `GalgameUserStatsService.PublishedGIDs`（catalog 侧）；
- `galgame_contributed` → `ContributedGIDs` **拉全量再在 Go 里 `pageSlice`**（`user_content_service.go:72`），贡献多的人每翻一页都全量拉一次；
- `galgame_like` / `galgame_favorite` → 本地 SQL `FindUserGalgameIDs`（`content_repo.go:20`）。

**`type=galgame_favorite` 读的是一张冻结表。** `galgame_favorite` 在迁移 043 就被收编进 `galgame_collection`（`043_create_galgame_collection.up.sql:6-7`「Existing galgame_favorite rows are migrated into a per-user default collection …  galgame_favorite itself is kept for one release as a fallback」）。全仓对这张表的引用只剩四处：这条读路径、`galgame_merge_repo.go:36`（合并时改指向）、`purge_repo.go:295`（删号时清理）、`purge-staging-verify`。**没有任何写入方**。也就是说 2026-07 之后收藏的游戏永远不会出现在这个 tab 里，而这个 tab 还在网页上（`apps/web/app/components/user/Galgame.vue:40`，`constants/user.ts` 的 `userGalgameGroupOptions`）。这是一条**沉默的功能死亡**，需要生产库确认 `galgame_favorite` 的残留行数与最后一行的时间。

### 2.6 `GET /api/user/:id/galgame-comments`

本域唯一的游标端点。`GetUserGalgameComments`（`user_content_service.go:217`）→ `collectAuthorGalgamePosts` / `likedGalgameComments` → `paginateGalgameCommentEntries`（`user/service/galgame_comment_pagination.go:110`）。

这段代码质量明显高于本域其它部分（有注释、有测试、游标带 `(created, id)` 双键 tie-breaker）。但有三条要记进 v1：

1. **坏游标静默从头开始。** `decodeGalgameCommentCursor`（`:148`）解不开就返回 `false`，`paginate` 于是从第一条开始发。01 §4 要求 `400 INVALID_CURSOR`，且游标要绑定排序与过滤条件——现在换个 `type=` 用旧游标一样能用。
2. **每一页都把作者的整个 feed 拉一遍。** `collectAuthorGalgamePosts`（`:46`）循环最多 40 页 × 100 条 = 4000 条，**没有缓存**，而且这条路径**匿名可达**（`router.go:79` 无 auth）。一个匿名请求能放大成 40 次上游 community 调用。注释已经承认这是 stopgap（等上游能按 `created_at` 排），但放大系数值得在 v1 前处理。
3. `ContentHtml: markdown.Render(av.Post.ContentRaw)`（`:249`）服务端渲染 HTML 下发，网页 `v-html`。01 §6 / K13 要求换成结构化节点树。

游标本身是 `base64url(json{c,i})`，不带 `cur_` 前缀 —— G16 会红。

响应项 `dto.UserGalgameComment`：`{id, galgame_id, content, content_html, user{id,name,avatar}, created, deleted}`。`deleted: true` 的行 `user` 是全零对象而不是 `null`。

### 2.7 `GET /api/user/:id/resources`

`GetUserResources`（`user_content_service.go:329`）+ `FindUserResources`（`content_repo.go:226`）。

下发 `code` 与 `password`（网盘提取码 / 解压密码）。核对过：galgame 详情页的公开资源面（`galgame/service/resource_service.go:248` → `rowToCard`）也下发这两项，所以这不是本端点新开的洞，是全站既有口径。但要记一笔：**任何人可以按 user_id 遍历任意用户的全部资源提取码**，不需要登录，也不需要经过作品页。

`type=expire` 是 `WHERE user_id = ? AND status = 1` —— 别人的失效资源同样匿名可读。网页只从自己的页面链过去（`galgame/resource/Resource.vue:95`），但端点没有「只能看自己的」这一层。

`status int` 是裸整数枚举（0=有效 / 1=失效），网页 `shared/types/user.ts:56` 照抄成 `number`。

### 2.8 `GET /api/user/:id/ratings`

`GetUserRatings`（`user_content_service.go:390`）。唯一一条没有 `type` 的列表。`galgame_type` 在库里是 JSON 字符串，服务层 `json.Unmarshal` 时 `_ =` 丢错（`:421`），解不开就是空数组。`UserRatingItem` 里塞了一个 `User` 字段，而这个列表本来就是「某人的评分」——同一个人重复 N 次，纯冗余。

### 2.9 寄挂的两条

`GET /api/user/:id/toolsets`（`toolset_handler.go:39`）：本域路径、toolset 域实现。`fiber.Params[int](c,"id")` 解路径，`page`/`limit` 有默认值（这条**不**像本域其它列表那样把 page 变成必填）。`GetList` 不回 error，`response.Paginated`。**无任何鉴权**。

`GET /api/user/:id/collections`（`collection_handler.go:147`）：`optAuth` 组，`optionalUID(c)` + `middleware.GetAccessToken(c)` 透给 catalog `/v2/me/folders`。`parseCollectionPage` 自己 clamp（`page<1→1`，`limit<1→24`，`limit>50→50`），不报错。

这两条挂在 `/api/user/**` 下只是 URL 上的事，实现归属另外两个域。v1 迁移时要决定：是跟着用户域走（`/v1/users/{user_id}/toolsets`），还是归还给资源域（`/v1/toolsets?owner_id=`）。建议后者——「某人的 X」是 X 的集合上的一个过滤，不是用户资源的子资源。

---

## 3. 自己的读面：status / 萌萌点 / 通知偏好 / 搜索

### 3.1 `GET /api/user/status`

`UserHandler.GetStatus`（`user_handler.go:56`）→ `UserService.GetUserStatus`（`user_service.go:144`）。

响应 `dto.UserStatusResponse`：`{moemoepoints, is_check_in, has_new_message, daily_toolset_upload_bytes, is_creator}`。

- **`moemoepoints` 带 s。** 全站其它 11 处都叫 `moemoepoint`。同一个概念两个名字（01 §3 的「两名一义」）。
- `moemoepoints` 读的是 `kungal_user_state.moemoepoint`——**缓存列**，见 §5。
- `is_check_in` 是 `state.DailyCheckIn == 1`，库里是 int。
- `has_new_message` 把三个未读计数（`message` / `system_message` / `chat_message`）加起来判 `>0`，三次 `CountUnread*` 的 error **全部丢掉**（`user_service.go:158-162` 的 `_`），任何一张表查失败都静默算 0 未读——红点消失，用户以为没消息。
- `is_creator` 来自 `role.IsCreator(s.userClient.User(...).Roles)`，见 3.4 的缓存投毒。
- `state` 查不到时（`err == nil && state != nil` 的短路）整个块跳过，余额显示 0、未静音类型为空。

调用方：`apps/web/app/components/kun/top-bar/Nav.vue:34`（红点轮询）、`apps/web/app/plugins/validate-session.client.ts:6`、`apps/web/app/utils/kunFetch.ts:106`（**它自己**——`kunFetch` 收到 `205` 后 1.5 秒起一个 raw `$fetch` 探这条路来判断会话是真死了还是抖了）。`docs/proj/app-direct-api.md:96` 把它列为 App 的红点轮询唯一入口。

v1 迁这条时要注意：`kunFetch.ts` 的会话探针依赖「旧信封 `code === 0`」这个判据，换成 problem+json 后那段逻辑要一起改，否则会话失效判定会一直认为会话是活的。

### 3.2 `POST /api/user/check-in`

`UserHandler.CheckIn`（`user_handler.go:44`）→ `UserService.CheckIn`（`user_service.go:117`）→ `StateRepository.CheckIn`（`state_repo.go:43`）+ `moemoepoint.Award`。

无请求体、无幂等键（01 §5 K12 要求全部 POST 支持 `Idempotency-Key`）。响应是一个**裸整数** `data: 7`（`response.OK(c, points)`）——2xx 顶层不是对象，G6 会红。网页 `top-bar/UserInfo.vue:55` `kunFetch<number>`。

`state_repo.go:43` 是 `UPDATE … WHERE user_id = ? AND daily_check_in = 0`，`RowsAffected == 0` 一律翻译成 `400/233 "您今天已经签到过了"`。**没有行**（`kungal_user_state` 里没这个 user）也会走进这条 → 一个从未在论坛出现过的用户第一次签到被告知「今天已经签到过了」。实际路径上 `Ensure` 在 OAuth callback（`auth_service.go:68`）和 Bearer 首见（`app.go:395`）都会跑，所以这条只在 Redis 会话还在但本地行被删过的边角上出现。

**本域最严重的一条 bug 在这里。** 幂等键是：

```go
moemoepoint.Key("checkin", strconv.Itoa(userID), time.Now().Format("2006-01-02"))
```

`time.Now()` 是**进程本地时区**。生产的 api 容器**没有 `TZ`**（`docker-compose.prod.yml` 的 `x-kungal-api-env` 里没有这一项；只有 `web` 服务显式设 `TZ: Asia/Shanghai`，`migrations/022` 的文件头也逐字写着「`kungal-api` runs in UTC (it has no TZ env; only `web` sets TZ=Asia/Shanghai)」）。所以这个日期串是 **UTC 日期**。

而清 `daily_check_in` 的定时任务钉的是 **Asia/Shanghai**（`internal/infrastructure/cron/cron.go:16` `scheduleTZ = "Asia/Shanghai"`，`:44` `"0 0 * * *"`）。

两个时钟差 8 小时，于是每天北京时间 **00:00–08:00** 这个窗口里：
- 闸门（`daily_check_in`）已经在北京 00:00 被重置，用户能签；
- 幂等键里的 UTC 日期**还是昨天**，与昨天北京 08:00 之后那次签到的键**逐字相同**。

而签到的 `delta` 是 `rand.IntN(8)`（`constants.CheckinMaxReward = 7`）。同键不同 body，OAuth 按 `docs/oauth/06-moemoepoint.md:107` 回 **`400/16004`**（「幂等键已存在但请求体不一致」）。`Awarder.Award`（`internal/moemoepoint/pusher.go:59`）只 `slog.Warn` 一行就返回，缓存列也不更新。

净效果：**每天北京 00:00–08:00 签到的用户，`daily_check_in` 被烧掉（当天不能再签），萌萌点一分没加，接口却照样返回一个 1–7 的随机数并弹「签到成功 +N」。** 只有随机数恰好撞上昨天那次（1/8）才会变成正常的幂等重放。这个洞每天开 8 小时，从签到功能上线起一直在。

两条次生问题：
- `points` 在 `Award` 之前就算好并返回，`Award` 是 `go func()` 的 best-effort。就算没有时区问题，OAuth 抖一下用户也会看到一个没到账的数字。
- 幂等键 `kungal:checkin:<uid>:<date>` 是四段，C3 写的是 `<app>:<event>:<ref>` 三段。语义上 `<ref>` = `<uid>:<date>` 说得通，但跟 `kungal:liked:topic_1207` 的构词不一致。

### 3.3 `GET /api/user/moemoepoint/log`

`user_handler.go:96` → `UserService.GetMoemoepointLog`（`user_service.go:132`）→ `userclient.MoemoepointLog`（`userclient.go:330`）→ OAuth `GET /users/{id}/moemoepoint/log`。

**这条是 C3 的正面典型**：流水完全不落地，逐次回源 OAuth，本地一个字节都不存。

请求：`limit`（`fiber.Query[int]`，默认 20，`<1 || >50` **静默夹到 20**，不报错）、`before_id`（`max(…, 0)`，负数静默变 0）、`reason`（裸字符串，**不校验直接透给 OAuth**）。`userID` 强制取自会话，看不了别人的。

响应：`userclient.MoemoepointLogPage` 原样下发 → `{items:[{id, delta, reason, source_app, ref, created_at, is_local}], has_more}`。`is_local` 是论坛算的：`SourceApp == c.cfg.ClientID`（`userclient.go:353`）——注意它比的是 **OAuth client_id**（一串 hex），不是 app 名。

错误：OAuth 任何失败 → `500/233 "获取萌萌点明细失败"`。上游 400（比如 `reason` 非法）在这里变成论坛的 500，把「你传错了」说成「我们坏了」。

分页风格是 `before_id` + `has_more`，既不是 01 §4 的游标（`next_cursor`）也不是页码。第三种。

调用方：`apps/web/app/components/kun/top-bar/MoemoepointLog.vue:145`。前端把 `content_removed` 的标签改写成「被移除」（memory `kungal-moemoepoint-debit-reason`）。

### 3.4 `GET /api/user/search`

`UserHandler.SearchMention`（`user_handler.go:114`）→ `UserService.SearchMentionUsers`（`user_service.go:227`）→ `userclient.SearchUsers`（`userclient.go:184`）→ OAuth `GET /users/search`。

请求 `q`（trim 后空 → 直接回 `[]`，不报错）、`limit`（`fiber.Query[int]` 默认 8，服务层 `<=0 || >20 → 8`，客户端 `>50` 在 userclient 再夹一次）。**要求登录**（`router.go:67` `userAuth`），匿名 `401/205`。

响应 `[]dto.MentionUser` → `[{id,name,avatar}]`。`u.Status != 0` 的行在服务层被过滤掉。

调用方：`apps/web/app/composables/useKunEditorAdapters.ts:27`（编辑器 @ 补全）、`apps/web/app/components/edit/topic/AccessUserPicker.vue:55`（话题 ACL 选人）。

**这条违反 C6 的一个子条款，而且造成一个真实 bug。** `docs/oauth/03-cross-service.md:124` 明写「搜索结果**不应缓存**」。但 `SearchUsers`（`userclient.go:202-207`）把返回的每个用户写进 `c.hot`，TTL 10 分钟：

```go
c.mu.Lock()
for _, u := range sd.Users {
    c.hot[u.ID] = cacheEntry{user: u, expire: now.Add(c.hotTTL)}
}
c.mu.Unlock()
```

问题不只是「不该缓存」。`/users/batch` 的返回在 `fetchBatch`（`userclient.go:223`）里要做一步 `role.Union(Roles, SiteRoles)`；`/users/search` 的返回**没有做**，而契约里 `/users/search` 的响应样例（`03-cross-service.md:110-117`）**根本不含 `site_roles`**。所以一次 @ 搜索会把命中用户的资料以**缺失 site roles** 的形态写进同一张热缓存，之后 10 分钟内任何 `userClient.User(id)` 拿到的都是它。具体后果：

- `GetUserStatus` 的 `is_creator`（`user_service.go:167`）对「靠 site role 拿到 creator」的用户变成 `false` —— 创作者入口消失 10 分钟；
- `CreatorService.Status` 的 `is_creator`（`galgame/service/creator_service.go:79`）同；
- `GET /api/user/:id` 的 `roles[]` 少掉 site 角色 —— 徽标消失；
- `UserPermissionService.targetRoles`（`admin/service/user_permission_service.go:89`）读的也是这张缓存，于是 `/admin/user-permissions/:uid` 的 `role_effective` 与 `Rank` 比较会在这 10 分钟里按缩水的角色算。

会话里的 `UserInfo.Roles` 走的是 Redis 会话（`middleware/auth.go:169` 的 `role.Union(info.Roles, info.SiteRoles)`），**不受影响**——所以真正的权限闸是安全的，坏掉的是「展示层的角色」和 `is_creator`。修法只有一条：`SearchUsers` 不要写 `c.hot`。

另外 `Placeholder(id)`（`userclient.go:211`）硬编码中文 `"已注销用户"`；F8 门只扫 v1 目录，这里不红，但 v1 迁移时 `UserRef.name` 要发 `null`（02 §3 F8 一栏已经记过这件事）。

### 3.5 `GET/PUT /api/user/notification-preferences`

`user_handler.go:68/80` → `UserService.GetNotificationPreferences`（`user_service.go:179`）/ `UpdateNotificationPreferences`（`:187`）→ `StateRepository.UpdateMutedTypes`（`state_repo.go:53`）。

GET 响应 `{muted_types: string[]}`；PUT 请求体同形，**无 `validate` tag**（`dto.UpdateNotificationPreferenceRequest`，`notification_dto.go:14`）——没有 `max` 没有 `dive`，一个 10 万项的数组会原样 `json.Marshal` 成 jsonb 写进 `muted_notification_types`。清洗只有 `msgService.SanitizeMutedKeys`（丢掉不认识的键），没有条数上限。

PUT 是幂等的（整体替换），语义上对；但它返回 200 + 清洗后的值，而 01 §5 对这种「整体替换一个子资源」没有异议——v1 可以直接留成 `PUT /v1/me/notification-preferences`。

错误：`Ensure` 或 `UpdateMutedTypes` 失败 → `500/233 "保存通知偏好失败"`。GET 里 `FindByID` 的 error 被 `err == nil &&` 短路吞掉 → 库挂了就回「一个都没静音」，用户以为设置丢了。

调用方：`apps/web/app/components/message/NotificationPreference.vue:35/55`、`apps/web/app/pages/message/muted.vue:12`。`docs/proj/app-direct-api.md:103` 列为 App 端点。

---

## 4. 写面三条：bio / username / avatar（OAuth 代理）

`ProfileHandler`（`user/handler/profile_handler.go`）。三条都是「取会话里的 OAuth access token → 打 OAuth → 把 OAuth 的响应体原样回吐 → `userClient.Invalidate(user.ID)`」。

| 端点 | 请求 | 上游 | 校验 |
|---|---|---|---|
| `PUT /api/user/bio` | `{bio}` `validate:"max=107"` | `PATCH /auth/me {bio}` | 本地 max=107，上游 ≤107 |
| `PUT /api/user/username` | `{username}` `validate:"required,min=1,max=17"` | `PATCH /auth/me {name}` | 本地 1–17，上游 1–17 + 全局唯一 + 字符白名单 |
| `POST /api/user/avatar` | 裸 body + `Content-Type` 透传 | `POST /auth/me/avatar` | **只查 body 非空、Content-Type 非空** |

**C1/C2/C6 核对：这三条是对的。** 论坛不写本地用户表（论坛根本没有本地用户表——迁移 007 把 `"user"` 拆了，只留 `kungal_user_state`），身份与资料全部由 OAuth 落库，论坛只代理 + 清缓存。

要记的几条：

1. **响应是 OAuth 的 `UserResponse` 原样透传**（`response.OK(c, json.RawMessage(data))`）。论坛对这个 body 里有什么没有任何控制：OAuth 明天给 `/auth/me` 加一个字段，论坛就跟着把它发给浏览器。其中包含 `email` 键——按 `docs/oauth/02-user-profile.md:77-87` 的 scope 门控，论坛的 token scope（`apps/web/app/utils/oauth-auth.ts:49`：`openid profile catalog:read catalog:edit playtime:read playtime:write folder:read folder:write`）**不含 `email`**，所以值是 `""`。但这是靠上游的门控保证的，不是靠论坛。而且它只回给本人，不是跨用户泄漏。v1 应该把这三条的响应收成论坛自己定义的形状，不再透传。
2. 网页其实根本不读这个 body：`Bio.vue:19` 和 `Username.vue:19` 只判真假，`Avatar.vue:58` 只取 `result.url`。所以收窄响应零成本。
3. **改名扣萌萌点，本地缓存不更新。** OAuth 在改名时按 `oauth:name_change:<userId>:<第几次>` 扣分（`docs/oauth/02-user-profile.md:125`），论坛前端写着 17 分（`apps/web/app/constants/moemoepoint.ts:14`）。`UpdateUsername` 只 `Invalidate` 了 userclient 的内存缓存，**没有碰 `kungal_user_state.moemoepoint`**。于是改完名，顶栏的余额仍是扣款前的数字，一直到该用户下一次因为别的事件触发 `moemoepoint.Award` 为止（可能几天，可能永远）。这是 C3「本地列是缓存」的一个直接后果：所有**不经过论坛**的余额变动（改名、OAuth 管理员调整、摸鱼站的发放）都不会同步过来。唯一的修复路径是手工跑 `cmd/sync-moemoepoint`。
4. `UploadAvatar`（`profile_handler.go:64`）把整个 body `c.Body()` 读进内存、把客户端的 `Content-Type` 原样转发给 OAuth，**不校验 MIME、不校验大小**。4 MiB 的上限只在前端（`Avatar.vue:36`）和 fiber 的默认 body limit 上。一个 curl 可以直接把任意 Content-Type 打到 OAuth 的 `/auth/me/avatar`。
5. `mapOAuthError`（`profile_handler.go:101`）把 OAuth 的体码原样当论坛的体码回吐：`errors.New(oe.Code, oe.Message, oe.HTTPStatus)`。于是浏览器会收到 `code: 15001` 这种**论坛注册表里不存在**的码，`message` 是 OAuth 产的**中文句子**。01 §2 K4「code 在整个平台唯一，一个 code 只有一个 type URI」在这里是靠运气成立的。
6. 三条都没有幂等键。头像上传是个非幂等的 POST。

---

## 5. 创作者两条

`CreatorHandler.Status`（`galgame/handler/creator_handler.go:22`）/ `Apply`（`:42`）→ `CreatorService`（`galgame/service/creator_service.go`）。

两条都要求 `middleware.GetAccessToken(c) != ""`，否则 `401/205`。Bearer 请求带的是 App 自己的 OP token（`middleware/bearer.go:88` 把原 token 放进 `AccessToken`），所以 App 也能用——这是对的，`/creator/applications/me` 本来就收终端用户 JWT。

`GET /api/user/creator/status` 响应：`{eligibility: CreatorEligibility, application: CreatorApplication|null, is_creator: bool}`。
`POST /api/user/creator/apply` 请求 `{message}`（**无长度校验**，`json.Unmarshal` 进一个匿名 struct，`creator_handler.go:51`），响应是 `userclient.CreatorApplication` 原样。无幂等键。

**C3 正面**：`eligibility`（`creator_service.go:50`）用 `s.userClient.GetMoemoepoint(ctx, userID)` **直接问 OAuth 要余额**，不读本地缓存列。这是全域唯一一处这么做的地方，也是唯一正确的做法。

**但错误被丢掉**：`moe, _ := s.userClient.GetMoemoepoint(...)`。OAuth 不可达 → `moe = 0` → `Eligible` 的四个 or 分支里萌萌点那一条永远假，而且 `Apply` 写进 `evidence` 的 `moemoepoint` 也是 0。一个够格的用户在 OAuth 抖动时会被告知「尚不满足创作者申请条件」（`403/233`），而且如果他恰好靠另外三条之一够格，提交上去的证据里余额是 0。

`Apply` 的错误映射：`*userclient.OAuthError` → `400/233` 带 OAuth 的中文 message（同 §4.5 的问题）；其余 → `500/233`。

调用方：`apps/web/app/components/kun/top-bar/CreatorApply.vue:37/110`。

---

## 6. 权限两条

### 6.1 `GET /api/perm/mine`

`UserPermissionHandler.GetMine`（`admin/handler/user_permission_handler.go:112`）。`authed` 组（`router.go:247`，即 `OptionalAuth → Auth`）。

```go
if user.ViaBearer() {
    return response.OK(c, dto.MyPermissionsResponse{Permissions: []string{}})
}
return response.OK(c, h.svc.MyPermissions(user.ID, user.Roles))
```

**Bearer 恒空**，与 `docs/proj/app-direct-api.md:42` 逐字一致。这条是对的。

但这个保证**放错了层**。`MyPermissions`（`admin/service/user_permission_service.go:80`）调 `perm.EffectiveForUser` → `perm.CanUser` —— 正是 `bearer_guard_test.go` 禁止的那个函数。守卫没红，是因为它的正则是 `perm\.CanUser\(|role\.Can(Moderate|Administer)\(`（`middleware/bearer_guard_test.go:86`），只认字面量，**认不出经 `perm.EffectiveForUser` 的一层间接**。今天靠 handler 里那个 `if` 挡着；明天谁在别处调一次 `MyPermissions` 或 `EffectiveForUser`，Bearer 就拿回了个人 override。建议 v1 把 Bearer 判断下沉到 `perm` 包，或者把 `EffectiveForUser` 也加进守卫的正则。

响应 `{permissions: string[]}`（`shared/types/permission.ts:22` 的 `KunPermMine`）。调用方 `apps/web/app/plugins/perm-mine.ts:9`（universal 插件，每个登录用户拉一次进 `useState('kun-perm-mine')`）。

新鲜度：`perm.CanUser` 读的是进程内的 `atomic.Pointer[userOverrideTable]`，由 `PermissionOverrideSync.StartRefresher(60 * time.Second)`（`app.go:683`）每 60 秒从 `user_permission_override` / `role_permission_override` 重载。写路径（`/admin/user-permissions/:uid`）会立刻 `reload.Load`，但**只在收到请求的那个实例上**。多实例部署时其余实例最长滞后 60 秒。

### 6.2 `GET /api/perm/bundles`

`RolePermissionHandler.GetBundles`（`admin/handler/role_permission_handler.go:46`）。`router.go:112`，挂在 `api` 上，**完全匿名**。

返回 `perm.EffectiveBundles()`（`pkg/perm/overrides.go` 末尾）—— `creator` / `moderator` / `admin` / `ren` 四个角色**当前生效**的权限键全集，**含管理员通过 `/admin/role-permissions/:role` 打的 override**。也就是说：任何匿名访客可以实时读出本站的版主与管理员到底被授予/撤销了哪些权限键，包括管理员刚刚改过的。

**而且没有任何调用方。** 全仓搜 `perm/bundles` 只有三处命中：路由注册、`routes.golden`、以及 `docs/proj/permissions.md:147` 的一句描述（「另供各角色的有效 bundle，只驱动 UI 显隐」）。`apps/web` 里零引用。这是一条**无人使用、把权限配置公开出去**的端点。v1 的动作应该是删，不是迁；`legacy_route_baseline` 相应下调。

---

## 7. 跨服务契约逐条核对

### C1 / C2（身份同一整数；只验签不签发）

**全域合规。** 论坛没有本地用户表（迁移 007 删了 `"user"`，019 拆了 50 个遗留外键），`kungal_user_state.user_id` 就是 OAuth 的 `users.id`，全部 `*_user_id` 外键同轴。签发只在 OAuth：`internal/user/oauth/client.go` 只做 `ExchangeCode` / `RefreshOAuthToken` / `FetchUserInfo` / `RevokeToken`；Bearer 走 `oauth.NewAccessTokenVerifier` 本地验 JWKS 签名（`middleware/bearer.go:63`）。本域没有一处铸 token。

一处要记：`GetProfile` 的注册时间优先用 OAuth 的 `u.CreatedAt`，解析失败才退回 `state.CreatedAt`（`user_service.go:88-92`）。这符合 `03-cross-service.md:70`「渲染注册时间**必须用**此字段」，回退是权宜。memory `kungal-user-state-lazy-provisioning` 记的就是这件事。

### C3（余额单源在 OAuth，本地列是缓存）

**三处违反 / 风险，按严重度排：**

1. **签到幂等键的 UTC/北京时区错位**（§3.2）——每天 8 小时窗口内签到不加分、闸门被烧、接口还谎报加了 N 分。这是本域最严重的一条。
2. **`StateRepository.Ensure` 把缓存列种成字面量 `7`**（`state_repo.go:25`，`model.KungalUserState{UserID: userID, Moemoepoint: 7}`；表 DEFAULT 也是 7，`migrations/007:35`）。7 是 OAuth 注册欢迎礼的数额（`docs/oauth/06-moemoepoint.md:152` `register_gift` +7），对「刚在 OAuth 注册、第一次来论坛」的人碰巧对；对**任何其它路径首次出现在论坛的用户**（从摸鱼站过来、老账号、被管理员调过分的），本地缓存就是一个凭空编出来的 7，一直显示到该用户下一次触发 `Award` 为止。正确做法是 `Ensure` 时 `GetMoemoepoint` 回源一次，或者干脆让缓存列可空、空时回源。
3. **凡是不经过论坛的余额变动都不会同步**：改名扣分（§4.3）、OAuth 管理员 `admin_grant`/`admin_deduct`、摸鱼站的发放。`kungal_user_state.moemoepoint` 只在 `moemoepoint.Awarder` 推送成功后被写一次（`internal/moemoepoint/pusher.go:66` / `:91`，且写的是 `res.Balance` 而不是 `+=`，这一点**是对的**，文件头的注释也钉死了）。唯一的修复通道是手跑 `apps/api/cmd/sync-moemoepoint`。

**读路径谁用了缓存、谁回源：**

| 读点 | 来源 | 判定 |
|---|---|---|
| `GET /api/user/:id` 的 `moemoepoint` | `kungal_user_state`（缓存） | 展示用，可接受，但会陈旧 |
| `GET /api/user/:id/floating` 的 `moemoepoint` | 同上 | 同上 |
| `GET /api/user/status` 的 `moemoepoints` | 同上 | 同上 |
| `GET /api/user/moemoepoint/log` | OAuth 回源 | **正确** |
| `GET /api/user/creator/status` 的 `eligibility.moemoepoint` | OAuth 回源 | **正确**（但吞错） |
| 工具集上传额度 `checkDailyUploadBudget` | `kungal_user_state`（缓存） | 用缓存余额当**配额判据**，域外，但值得记 |
| `POST /topics/:id/upvotes`（推） | 缓存余额判 ≥10（W4 已记录在案） | 域外 |

### C6（资料不落地、不当真源）

**基本合规**，两处偏差：

1. `/users/search` 的结果被写进 10 分钟热缓存（§3.4）——违反 `03-cross-service.md:124`，并且因为缺 `site_roles` 合并而造成真实的功能退化。
2. `userclient.Client` 的 TTL 缓存本身是 C6 明确允许的（「短 TTL 内存缓存是 fine 的 —— `pkg/userclient` 已有 ~10min TTL」）。`Invalidate` 在三条写面后都调了（`profile_handler.go:42/60/85`），是对的。
3. 没有任何地方把 `name` / `avatar` / `bio` 写进本地表。`kungal_user_state` 里只有 `user_id` + 站内状态 + 缓存余额 + 静音偏好。**这一点做得很干净。**

批量：`userclient.Users`（`userclient.go:118`）有 singleflight + 负缓存 + 100 分片，符合 `oauth-integration-guide.md:749` 的 L3。真正的 N+1 只有 `AccessUserPicker` 那处（§1.2）。

一个隐患：`userclient.sendEnvelope`（`userclient.go:228`）**完全不看 HTTP status**，只解信封的 `code`。OAuth 前面挡一个返回 HTML 的代理时，`json.Decode` 报错 → 变成 `"decode envelope"` 错误 → 论坛回 `500/233`，日志里看不出是 502。

---

## 8. 数据库表清单

本域读写的表，以及哪些列是别的服务的缓存。

**写（只有一张表）：**

| 表 | 列 | 操作 | 备注 |
|---|---|---|---|
| `kungal_user_state` | `user_id`, `moemoepoint` | INSERT `ON CONFLICT DO NOTHING`（`Ensure`） | **`moemoepoint` 是 OAuth 余额的缓存**；种值写死 7 |
| | `daily_check_in` | UPDATE 0→1（`CheckIn`） | 每日 Asia/Shanghai 00:00 由 cron 清零 |
| | `muted_notification_types` (jsonb) | UPDATE（`UpdateMutedTypes`） | 论坛自有数据 |
| | `moemoepoint` | UPDATE = OAuth 返回的 balance（`pusher.go:66/91`） | 唯一的缓存刷新通道 |

`kungal_user_state` 的其余列（`daily_image_count` / `daily_toolset_upload_count` / `daily_toolset_upload_bytes` / `created` / `updated`）由别的域写，本域只读 `daily_toolset_upload_bytes` 和 `created`。`created` 是「首次出现在论坛」，**不是注册时间**（memory `kungal-user-state-lazy-provisioning`）。

**读：**

| 表 | 读到的列 | 经由 |
|---|---|---|
| `topic` | `id, title, created, user_id, status, access_scope, is_nsfw` | `FindUserTopics`、`GetUserStats`、`FindFloatingStats`、`FindUserReplies(reply_target)` |
| `topic_poll` | COUNT by `user_id` | `GetUserStats` |
| `topic_lottery` | COUNT by `user_id` | `GetUserStats` |
| `topic_reply` | `topic_id, floor, content, created, user_id, status` | `FindUserReplies`、两处 COUNT |
| `topic_comment` | `id, topic_id, content, created, user_id, status, target_user_id` | `FindUserComments`、两处 COUNT |
| `topic_comment_like` | `topic_comment_id, user_id` | `FindUserComments(comment_like)` |
| `topic_reaction` | `topic_id, user_id, reaction` | `FindUserTopics(topic_like)`、`GetUserStats` 的 like/dislike |
| `topic_reply_reaction` | `topic_reply_id, user_id, reaction` | `FindUserReplies(reply_like)` |
| `topic_upvote` | `topic_id, user_id` | `FindUserTopics(topic_upvote)`、`GetUserStats` |
| `topic_favorite` | `topic_id, user_id` | `FindUserTopics(topic_favorite)` |
| `galgame` | `id, published, created, view, like_count, resource_update_time, creator_user_id` | `FindUserGalgameIDs`、`FindGalgameLocalStats` | `view`/`like_count` 是本地统计；作品本体在 catalog |
| `galgame_like` | `galgame_id, user_id` | `FindUserGalgameIDs(galgame_like)` |
| **`galgame_favorite`** | `galgame_id, user_id` | `FindUserGalgameIDs(galgame_favorite)` | **冻结表，无写入方**（迁移 043 之后） |
| `galgame_resource` | `id, galgame_id, type, language, platform, size, code, password, note, status, created, user_id` | `FindUserResources`、`FindResourceMetaByGalgameIDs`、`GetUserStats`、`FindFloatingStats` |
| `galgame_resource_link` | `galgame_resource_id, url` | `FindResourceLinks` |
| `galgame_resource_like` | `galgame_resource_id, user_id` | `FindUserResources(galgame_resource_like)` |
| `galgame_rating` | 15 列 + `user_id` | `FindUserRatings`、`GetUserStats` |
| `galgame_website` | COUNT by `user_id` | `GetUserStats` 的 `galgame_toolset` | 表名与字段名不一致（见 §9） |
| `galgame_toolset_resource` | COUNT by `user_id` | `GetUserStats` |
| `galgame_post_like` | `id, post_id, user_id` | `FindUserLikedPostIDs` | **community 评论的本地点赞镜像** |
| `message` | `receiver_id, status, type` | `CountUnreadMessages` | 含 community 镜像行 |
| `system_message` / `system_message_read_state` | `id` / `user_id, last_read_message_id` | `CountUnreadSystemMessages` |
| `chat_message` / `chat_room_participant` / `chat_message_read_by` | `id, sender_id, chat_room_id, user_id, chat_message_id` | `CountUnreadChatMessages` |
| `user_permission_override` / `role_permission_override` | `user_id, permission, effect` | 不直接读 —— `/perm/mine` 读的是 60 秒刷新的进程内表 |

**不在本地的**：用户资料（name/avatar/bio/status/roles/created_at）在 OAuth，逐次 `/users/batch`；萌萌点余额与流水在 OAuth；作品元数据在 catalog；评论正文在 community 原语（`galgame_comment` 表已冻结，`stats_repo.go:28-30` 的注释说明统计改走 `visible_posts`）。

---

## 9. 命名问题清单（迁 v1 时逐条执行）

| 现状 | 问题 | v1 |
|---|---|---|
| `moemoepoints`（`/user/status`） vs `moemoepoint`（其余 11 处） | 一义两名 | `moemoepoint` |
| `created` / `updated` | 01 §3 禁用名 | `created_at` / `updated_at` |
| `user`（`UserGalgameCard.user`、`UserRatingItem.user`、`UserGalgameComment.user`） | 禁用名；且 ratings 里这个人就是路径上那个人，纯冗余 | `author`，ratings 里直接删 |
| `status: int`（profile 0/1、resource 0/1） | 裸整数枚举；profile 的 `status` 按 OAuth 契约是**开放**的 | `state` 封闭字符串枚举（`active`/`banned`、`valid`/`expired`） |
| `type=`（7 条列表） | 开放裸字符串，未知值静默回落 | 各集合自己的封闭枚举 + `400 UNKNOWN_ENUM_VALUE` |
| `view: int`（`UserGalgameCard.view`、`UserRatingItem.view`） | 禁用名；而且这两个 `view` 含义完全不同（前者=浏览量，后者=评分的「世界观」分项） | `view_count` / `worldview` |
| `topic` `reply_created` `comment_created` `galgame` `upvote` `like` `dislike` `galgame_comment` … | 14 个计数没有一个以 `_count` 结尾 | `topic_count` `reply_count` … |
| `daily_topic_count` / `daily_galgame_count` | 时间窗计数应写成 `<名词>_<窗口>_count` | `topic_today_count` 或按 F1 的窗口写法 |
| `galgame_toolset` 计的是 `galgame_website` 表 | 字段名与表名两套词 | 统一叫 toolset |
| `is_check_in` | 布尔前缀对，但 `check_in` 是动词 | `has_checked_in_today` |
| `rating_data`（`UserRatingsResponse`） | 数组不叫复数名词 | `items` |
| `link: []string`（resource） | 数组用单数 | `links` |
| `galgame_type: []string` | 同 | `galgame_types` |
| `next_cursor` 的值 | 不带 `cur_` 前缀（G16） | `cur_…` |
| `before_id` + `has_more`（萌萌点流水） | 第三种分页风格 | 游标 |
| 全部 id 是 JSON number | 01 §3「全部 id 是字符串」 | 十进制字符串 |
| `/user/:id/floating` | 动词化的「浮层」不是资源；且路径参数是假的 | 并进用户引用的批量读 `/v1/users?ids=` |
| `/user/moemoepoint/log` | 单数 + 动词化 | `/v1/me/moemoepoint-entries` 之类 |
| `/user/check-in` | 动词路径（B7） | `POST /v1/me/check-ins` |
| `/user/creator/apply` | 动词路径 | `POST /v1/creator-applications` |
| `/user/:id`（复数名词规则） | 应是 `/users/{user_id}` | — |

另外：`GET /api/user/:id` 与 `GET /api/user/:id/topics` 的 `:id` 在 v1 里要叫 `{user_id}`，与响应字段同名（K1 / F10）。

---

## 10. Bug 清单（按严重度）

### 严重

1. **隐藏话题标题经个人主页对匿名泄漏。** `FindUserTopics`（`content_repo.go:79`）没有 `topic.status != 1`，而站内正牌列表两处都有（`list_repo.go:69`、`keyset_list.go:88`）。`type=topic` / `topic_like` / `topic_upvote` / `topic_favorite` 四个分支全中。旁边那条专门加了 `perm.TopicViewHidden` 闸的 `type=topic_hide` 因此形同虚设。

2. **签到每天有 8 小时不加分。** 幂等键用 UTC 日期（api 容器无 `TZ`），闸门用 Asia/Shanghai 重置（`cron.go:16/44`），北京 00:00–08:00 的签到与昨天的键相同而 `delta` 随机不同 → OAuth `400/16004` → `pusher.go:60` 只 warn。用户看到「+N」，实际 0 分，且当天不能再签。

3. **`/users/search` 的结果污染 `/users/batch` 的热缓存。** `userclient.go:202-207` 写 `c.hot` 时没有 `role.Union(Roles, SiteRoles)`（`fetchBatch:223` 有），而 `/users/search` 的响应本来就不含 `site_roles`。后果：@ 一下某人，他的 site 角色在全站展示层消失 10 分钟，`is_creator` 变 false。同时这本身违反 `03-cross-service.md:124`「搜索结果不应缓存」。

### 中

4. **`/api/user/:id/floating` 忽略路径参数**，靠 `?user_id=` 取人（`floating_dto.go:3`）。不带查询 → 400；查询与路径不一致 → 按查询走。唯一调用方被迫两边都写（`AccessUserPicker.vue:29-31`），并因此变成逐 id 的 N+1。

5. **`type=galgame_favorite` 读一张没有写入方的冻结表。** 迁移 043 之后收藏进 `galgame_collection`，`galgame_favorite` 只剩合并/删号/本条读路径。这个 tab 在网页上还在，只是内容永远停在 2026-07。

6. **`total` 与 `items` 不同谓词**（resources / ratings）：`total` 来自 SQL COUNT，`items` 在 Go 里被 catalog brief 缺失过滤过（`user_content_service.go:360-364` / `:415-417`）。分页器永远画错。

7. **分页全域没有 tie-breaker**（`content_repo.go` 六处 `ORDER BY created DESC`）。同秒行在翻页时重复或丢失。

8. **`hideTarget` fail-open**（`user_content_service.go:44`，`u, _, _ :=`）。OAuth 故障时封禁用户的全部内容照常可见。

9. **一批吞掉的错误各自静默返回 0**：
   - `FindFloatingStats`（`stats_repo.go:94`）—— 名片全零；
   - `CountUnreadMessages` / `CountUnreadSystemMessages` / `CountUnreadChatMessages`（`user_service.go:158-162`）—— 红点消失；
   - `CreatorService.eligibility` 的 `GetMoemoepoint`（`creator_service.go:50`）—— 够格的人被判不够格；
   - `GetUserProfile` / `GetFloatingCard` / `GetUserStatus` / `GetNotificationPreferences` 里的 `s.stateRepo.FindByID` 全部 `_` 或短路 —— 余额显示 0；
   - catalog `GetBatchPublic` 的 error 在四处被丢（`user_content_service.go:88/107/356/407`）—— 列表变空。

10. **`/api/perm/bundles` 匿名公开当前生效的角色权限矩阵，且零调用方。**

11. **旧信封 `205` 与 `kunFetch` 的会话探针耦合**（`apps/web/app/utils/kunFetch.ts:90-122`）：探针自己打 `/api/user/status` 并判 `code !== 0`。这条端点一旦迁 v1 换成 problem+json，探针会永远认为会话是活的。迁移时必须同一提交改掉。

### 轻

12. 封禁用户的 `/api/user/:id` 回 `created = "0001-01-01T00:00:00Z"`，页面渲染「注册于 0001年1月1日」（`ProfileHeader.vue:91`）。
13. 同一个封禁用户，`/api/user/:id` 回 200，`/floating` 回 404。两个面裁决不同。
14. `page` / `limit` 因 `validate:"min=1"` 变成事实必填，schema 上看不出来。
15. `POST /api/user/check-in` 的 2xx 顶层是裸整数（G6）。
16. `PUT /api/user/notification-preferences` 的 `muted_types` 无任何长度/条数校验。
17. `POST /api/user/avatar` 不校验 MIME 与大小，`Content-Type` 原样转发上游。
18. `mapOAuthError`（`profile_handler.go:101`）把 OAuth 的体码与中文 message 原样当论坛的回吐。
19. `bearer_guard_test.go` 的正则漏掉 `perm.EffectiveForUser`（`user_permission_service.go:81`）。今天靠 handler 的 `if user.ViaBearer()` 挡着。
20. `galgame-comments` 的坏游标静默从头开始（`galgame_comment_pagination.go:148`）。
21. `/api/user/:id/galgame-comments` 匿名可达，一次请求最多放大成 40 次 community 调用（`galgame_comment_pagination.go:33/50`），无缓存。
22. `type=galgame_contributed` 每翻一页都全量拉一次 `ContributedGIDs` 再在 Go 里切片（`user_content_service.go:67-72`）。
23. `userclient.sendEnvelope` 不看 HTTP status（`userclient.go:228`），上游 502 的 HTML 会变成论坛的 500「decode envelope」。
24. `userclient.Placeholder` 硬编码「已注销用户」（`userclient.go:212`）。
25. `GetMoemoepointLog` 把上游的 400（如非法 `reason`）翻译成论坛的 500。
26. `limit` 越界一律静默 clamp（`user_handler.go:102`、`user_service.go:232`、`collection_handler.go:180-185`），01 §4 要求 `400 LIMIT_TOO_LARGE`。
27. 本域 24 条端点，零个契约测试、零个 handler 测试；`internal/user` 下只有 OAuth 线协议、site-role 合并、galgame-comments 游标三组单测。

---

## 11. 需要生产库回答的问题（我没法从代码得到）

请按下面的顺序跑，每条都直接对应上面的一个裁决：

1. **隐藏话题的暴露面**（对应 bug #1）：
   `SELECT count(*) FROM topic WHERE status = 1;`
   以及按作者分布：`SELECT count(DISTINCT user_id) FROM topic WHERE status = 1;`
   —— 决定这条是「修完就好」还是「还要通知被泄漏的作者」。

2. **`access_scope` 的取值分布**（对应 bug #1 的第二条，版主在别人主页看到受限话题）：
   `SELECT access_scope, count(*) FROM topic GROUP BY 1;`

3. **冻结表 `galgame_favorite`**（对应 bug #5）：
   `SELECT count(*) FROM galgame_favorite;`
   `SELECT max(id) FROM galgame_favorite;`（有 `created` 就取 `max(created)`）
   —— 如果最后一行早于迁移 043 的上线日，这个 tab 可以直接删而不是迁。

4. **`topic_comment.target_user_id` 还活着吗**（对应 §2.4）：
   `SELECT count(*) FROM topic_comment WHERE target_user_id IS NOT NULL;`
   `SELECT count(*) FROM topic_comment WHERE target_user_id IS NOT NULL AND created > now() - interval '90 days';`
   —— 90 天内为 0 就说明 `comment_target` 已是死路，v1 不用带它。

5. **七条列表的 `type` 实际用量**：这个代码里查不到，只能看访问日志。如果拿不到日志，退而求其次用数据侧的上限估计：
   `SELECT count(*) FROM topic_upvote;` / `topic_favorite` / `topic_reaction WHERE reaction='like'` / `topic_reply_reaction WHERE reaction='like'` / `topic_comment_like` / `galgame_like` / `galgame_resource_like` / `galgame_post_like`
   —— 每个 `type` 一行，决定哪些分支值得在 v1 留下来。

6. **`kungal_user_state` 的缓存偏差**（对应 C3 / bug #2）：
   `SELECT count(*) FROM kungal_user_state;`
   `SELECT count(*) FROM kungal_user_state WHERE moemoepoint = 7;`
   —— 第二个数如果异常大，说明有大量行是 `Ensure` 种下去之后**从来没被 Award 刷新过**的假余额。
   最有说服力的是拿这张表跟 OAuth 库对账（infra 那侧）：
   `SELECT count(*) FROM kungal_user_state s JOIN <oauth>.users u ON u.id = s.user_id WHERE s.moemoepoint <> u.moemoepoint;`
   以及偏差的分布 `SELECT ... percentile_cont(...) WITHIN GROUP (ORDER BY abs(s.moemoepoint - u.moemoepoint))`。

7. **签到黑洞的规模**（对应 bug #2，要查 **OAuth** 库）：
   `SELECT count(*) FROM moemoepoint_log WHERE reason = 'daily_checkin' AND source_app = '<kungal 的 client_id>';`
   按日分组看有没有每天固定缺一块：
   `SELECT date_trunc('day', created_at), count(*) FROM moemoepoint_log WHERE reason='daily_checkin' GROUP BY 1 ORDER BY 1 DESC LIMIT 30;`
   再跟论坛侧当天 `daily_check_in=1` 的人数对比。差值就是被吃掉的签到。
   （论坛侧只能在当天 cron 清零之前取：`SELECT count(*) FROM kungal_user_state WHERE daily_check_in = 1;`）

8. **改名造成的缓存漂移**（对应 §4.3，OAuth 库）：
   `SELECT count(*) FROM moemoepoint_log WHERE idempotency_key LIKE 'oauth:name_change:%';`
   —— 这些用户的论坛缓存余额全部偏高。

9. **`galgame_resource` 里 `code` / `password` 非空的行数**（对应 §2.7 的公开面）：
   `SELECT count(*) FROM galgame_resource WHERE coalesce(code,'') <> '' OR coalesce(password,'') <> '';`

10. **权限 override 的规模**（对应 §6）：
    `SELECT count(*) FROM user_permission_override;` / `SELECT count(*) FROM role_permission_override;`
    `SELECT permission, effect, count(*) FROM role_permission_override GROUP BY 1,2;`
    —— 决定 `/perm/bundles` 泄漏的到底是「跟代码里 Bundles 一样的静态表」还是「管理员真的改过的配置」。

11. **`galgame_post_like`（community 点赞镜像）的规模**（对应 §2.6 的 `galgame_comment_like` 分支，它先全量 `FindUserLikedPostIDs` 再分批 resolve）：
    `SELECT count(*) FROM galgame_post_like;`
    `SELECT max(c) FROM (SELECT count(*) c FROM galgame_post_like GROUP BY user_id) t;`
    —— 最大值决定单个用户打开这个 tab 会触发多少次 `/posts/resolve`（每 100 个一次）。

12. **`topic_reply.content` 的长度分布**（对应 §2.3，回复正文整段下发且不分页裁剪）：
    `SELECT percentile_cont(0.99) WITHIN GROUP (ORDER BY length(content)) FROM topic_reply WHERE status = 0;`
