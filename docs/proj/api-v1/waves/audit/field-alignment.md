# /api/v1 前后端字段对齐审计（W1–W4）

只读审计，未修改仓库任何文件。所有结论都来自阅读代码，未运行服务、未查库。

---

## 0. 覆盖范围

Web 侧对 v1 的调用点用 `rg '\.GET\(|\.POST\(|\.PUT\(|\.DELETE\(|\.PATCH\('` 全量枚举，命中 32 处，去重后覆盖 **16 条路径 / 27 个 operation**。生成类型 `apps/web/shared/types/api/v1.d.ts` 共声明 18 条路径，未被 web 调用的两条是 `/problems` 与 `/problems/reasons`（元数据面，只在 CI 译文门里用）。

| # | operation | 方法 · 路径 | web 调用点 |
|---|---|---|---|
| 1 | `listTopics` | GET `/topics` | `apps/web/app/components/topic/Container.vue:90`、`apps/web/server/utils/kunSitemapSources.ts:23` |
| 2 | `getTopic` | GET `/topics/{topic_id}` | `apps/web/app/pages/topic/[id]/index.vue:39`、`apps/web/server/utils/kunOgCard.ts:85` |
| 3 | `listTopicReplies` | GET `/topics/{topic_id}/replies` | `apps/web/app/composables/topic/useTopicReplies.ts:101` |
| 4 | `getReply` | GET `/replies/{reply_id}` | `apps/web/app/composables/topic/useTopicReplies.ts:250`、`apps/web/app/composables/topic/useQuoteContent.ts:69` |
| 5 | `recordTopicView` | POST `/topics/{topic_id}/views` | `apps/web/app/components/topic/detail/Detail.vue:81` |
| 6 | `createTopic` | POST `/topics` | `apps/web/app/composables/topic/useTopicSubmitter.ts:118` |
| 7 | `updateTopic` | PATCH `/topics/{topic_id}` | `apps/web/app/composables/topic/useTopicSubmitter.ts:102`、`apps/web/app/components/topic/footer/Hide.vue:67`、`apps/web/app/components/user/Topic.vue:42` |
| 8 | `getTopicSource` | GET `/topics/{topic_id}/source` | `apps/web/app/components/topic/footer/Rewrite.vue:17` |
| 9 | `createReply` | POST `/topics/{topic_id}/replies` | `apps/web/app/components/topic/reply/PanelBtn.vue:41` |
| 10 | `updateReply` | PATCH `/replies/{reply_id}` | `apps/web/app/components/topic/reply/PanelBtn.vue:84` |
| 11 | `deleteReply` | DELETE `/replies/{reply_id}` | `apps/web/app/components/topic/reply/Delete.vue:40` |
| 12 | `getReplySource` | GET `/replies/{reply_id}/source` | `apps/web/app/components/topic/reply/Rewrite.vue:16` |
| 13 | `setTopicReaction` | PUT `/topics/{topic_id}/reactions/{reaction}` | `apps/web/app/composables/useReactions.ts:76` |
| 14 | `removeTopicReaction` | DELETE 同上 | `apps/web/app/composables/useReactions.ts:75` |
| 15 | `setReplyReaction` | PUT `/replies/{reply_id}/reactions/{reaction}` | `apps/web/app/composables/useReactions.ts:83` |
| 16 | `removeReplyReaction` | DELETE 同上 | `apps/web/app/composables/useReactions.ts:82` |
| 17 | `favoriteTopic` | PUT `/topics/{topic_id}/favorite` | `apps/web/app/components/topic/footer/Favorite.vue:29` |
| 18 | `unfavoriteTopic` | DELETE 同上 | `apps/web/app/components/topic/footer/Favorite.vue:32` |
| 19 | `upvoteTopic` | POST `/topics/{topic_id}/upvotes` | `apps/web/app/components/topic/UpvoteModal.vue:29` |
| 20 | `listTopicUpvotes` | GET `/topics/{topic_id}/upvotes` | `apps/web/app/components/topic/UpvoteRecords.vue:44` |
| 21 | `listTopicReactions` | GET `/topics/{topic_id}/reactions` | `apps/web/app/components/topic/reaction/HistoryModal.vue:32` |
| 22 | `listReplyReactions` | GET `/replies/{reply_id}/reactions` | `apps/web/app/components/topic/reaction/HistoryModal.vue:26` |
| 23 | `setBestAnswer` | PUT `/topics/{topic_id}/best-answer` | `apps/web/app/components/topic/reply/BestAnswer.vue:44` |
| 24 | `clearBestAnswer` | DELETE 同上 | `apps/web/app/components/topic/reply/BestAnswer.vue:41` |
| 25 | `pinReply` | PUT `/topics/{topic_id}/pinned-reply` | `apps/web/app/components/topic/reply/Pin.vue:31` |
| 26 | `unpinReply` | DELETE 同上 | `apps/web/app/components/topic/reply/Pin.vue:28` |
| 27 | `listGalgameMoyuPatches` | GET `/galgames/{galgame_id}/moyu-patches` | `apps/web/app/components/galgame/patch/Container.vue:24` |

---

## 1. 三条先决事实（决定了后面所有判断）

**F-1 · 多余的请求体字段一律被静默丢弃，huma 绝不拒绝。**
`apps/api/internal/apiv1/setup.go:57` 设了 `cfg.AllowAdditionalPropertiesByDefault = true`；`apps/api/internal/apiv1/extrafields_test.go:12` 的 `TestUnknownBodyFieldsAccepted` 明确钉死「发 `{"name":"alice","extra":true,"nested":{"x":1}}` 回 200」，`apps/api/internal/apiv1/extrafields_test.go:52` 还禁止整份 spec 出现 `additionalProperties: false`。
→ 方向 1（前端发了服务端不读的字段）在这个面上**永远不会以报错的形式暴露**，只能靠读代码发现。下面逐条核对过。

**F-2 · query 参数会被 huma 按 schema 校验，失败是 422。**
`huma@v2.39.1/huma.go:1007-1008` 对每个 present 的参数调 `Validate(...)`，`huma.go:899` 把 `errStatus` 定为 `http.StatusUnprocessableEntity`。所以 `from_floor=0`（schema `minimum:"1"`，`apps/api/internal/topic/apiv1/detail_ops.go:30`）会得到 422，不是被 clamp。见 **[H-4]**（§6）。

**F-3 · `viewer` 对匿名一律为 `null`，且这是唯一的空值口子。**
`topicViewer` / `replyViewer` 在 `viewer == nil` 时直接返回 nil（`apps/api/internal/topic/apiv1/engage_viewer.go:9,29`），`mapComments` 只在 `viewer != nil` 时建 `CommentViewer`（`apps/api/internal/topic/apiv1/assemble.go:208-209`），`reactionSummaries` 同理（`apps/api/internal/topic/apiv1/reactions.go:31-34`）。写面返回的 `TopicEngagement.Viewer` / `ReplyEngagement.Viewer` 是**值类型不是指针**（`engage_types.go:47,56`），而这些 operation 全是 `v1.Required` 档，所以那里不会是 null；`buildTopicEngagement` 仍然兜了一层 `if tv == nil { tv = &TopicViewer{} }`（`engage_snapshot.go:32-34`）。

---

## 2. 逐 operation 四方向表

表中「①发了但不读」「②读了但可能空」「③改名语义」「④发了但没人读」。
只写有内容的格子；`—` 表示该方向核对后为空。

### 1. `listTopics` GET /topics

| 方向 | 结论 |
|---|---|
| ① | 无。web 只发 `limit` / `sort` / `include_nsfw` / `cursor`（`Container.vue:92-97`、`kunSitemapSources.ts:26-30`），四个都在 `listTopicsInput`（`apps/api/internal/topic/apiv1/list.go:46-51` + `collect.Page`）里被读。sitemap 只发 `limit`+`cursor`，不发 `sort`，服务端用 `default:"bumped_desc"`。 |
| ② | `items` 恒为 `[]`（`repr.NewList`，`apps/api/internal/apiv1/repr/list.go:112-117`，且 `MarshalJSON` 再兜一次）。`author.name` 对已注销/查不到的用户是 `null`（`repr.DeletedUserRef`，`apps/api/internal/apiv1/repr/user.go:24-26`），web 走 `toKunUser` 兜成 `deletedUserName`（`apps/web/app/utils/userRef.ts:8`）✅。`sections` 可为 `[]` → `BadgeGroup` 的 `v-for` 自然不渲染 ✅。 |
| ③ | `status`(int) → `state`(string) 已按 `topicState`（`summary.go:73`）映射；web 在列表面根本不读 `state`，无残留。`view`→`view_count`、`status_update_time`→`bumped_at` 均已改完（`Card.vue:46`、`kunSitemapSources.ts:38`）。 |
| ④ | **死负载**：`object`、`state`、`category`、`cover_images`、`upvoted_at`。`TopicSummary` 的唯一两个消费者是 `Card.vue` 与 `Container.vue`，两者都不读这五个字段（`cover_images` 尤其可惜，列表卡片不画封面）。`bumped_at` 只被 sitemap 读。 |

### 2. `getTopic` GET /topics/{topic_id}

| 方向 | 结论 |
|---|---|
| ① | 无（无请求体，无 query）。 |
| ② | 见下方专表，这是四方向里唯一需要逐字段列的形状。 |
| ③ | `user`→`author`、`created`→`created_at`、`edited`→`edited_at`、`is_nsfw_topic`→`is_nsfw`、`section`→`sections`、`cover_images`(hash 数组)→`[Image]`、`status`→`state`+`hidden_by`：web 全部按新名读（`Master.vue`、`Detail.vue`、`pages/topic/[id]/index.vue`），无一处残留旧名。**注意** `apps/web/app/components/search/TopicCard.vue:30` 与 `apps/web/app/components/section/Container.vue:90` 仍读 `is_nsfw_topic`——那是站内搜索与版块页的 legacy 接口，不是 v1，不算残留。 |
| ④ | **死负载**：`object`、`category`、`dislike_count`（只被写进 state 从不渲染）、`viewer.can_like`、`viewer.has_liked`、`viewer.has_disliked`、`cover_images[].sexual`。其中 `viewer.has_liked` / `has_disliked` 之所以是死的，是因为 web 的「我表过态没」只读 `reactions[].viewer.has_reacted`（`apps/web/app/utils/reactionSummary.ts:8`）——两套「我的状态」只用了一套，见 §5 末尾。 |

**`getTopic` 响应逐字段（方向②重点）**

| 字段 | Go 填充处 | 何时为空 | web 渲染 | 判定 |
|---|---|---|---|---|
| `hidden_by` | `topicLifecycle` `summary.go:84-99` | `state == "published"` 时恒 null | `Detail.vue:202` `v-if="topic.state === 'hidden'"` 才渲染，且 `?? ''` 兜底；`KUN_TOPIC_HIDDEN_BY` 三个键 author/moderator/trust 全覆盖（`apps/web/app/constants/topic.ts:174-190`） | ✅ 对齐 |
| `edited_at` | `repr.TimestampPtr(topic.Edited)` `get.go:135` | 从未编辑过 → null | `Master.vue:84` `v-if="topic.edited_at"`；`pages/topic/[id]/index.vue:150` 三元兜底成 `created_at` | ✅ |
| `upvoted_at` | `get.go:137` | 从未被推 → null | `Master.vue:66` 传给 `BadgeGroup`，`upvoteTime` prop 默认 null，`hourDiff(props.upvoteTime \|\| 0, 24)` 兜底（`BadgeGroup.vue:36-41`），徽章 `v-if="upvoteTime && isRecentlyUpvoted"` | ✅ 渲染安全，但见 **[H-3]** 写后不刷新 |
| `best_answer` / `pinned_reply` | `pickSpecials` `get.go:176-191` | 未设置时 null；**已设置但该回复作者被封禁 → `mapOne` 返回 nil（`assemble.go:158-159`）→ 仍是 null**，而 `has_best_answer`(summary 面) 依旧 true | `Master.vue:93 v-if`；`Detail.vue:113` 过滤 falsy | ✅ web 正确处理 null |
| `cover_images` | `coverImagesWithMeta` `summary.go:101-121` | 无封面→`[]`；**不可解析的 token 被跳过**（`repr.NewImageFromToken` → `markdown.ParseContentImageRef`，正则 `^/image/([0-9a-f]{64})(?:_([a-z0-9]+))?$`，`apps/api/internal/infrastructure/markdown/markdown.go:40-48`） | `Master.vue:18-20` 过滤掉正文里已出现的 hash，`CoverGrid.vue:61` 的根 `v-if="shown.length"` | ✅ 渲染安全，但见 **[H-2]** 编辑回写会丢行 |
| `cover_images[].width/height/thumbhash` | `imageFrom` `apps/api/internal/apiv1/repr/image.go:64-82` | 图床 meta 缺失 → 三者全 null | `CoverGrid.vue:26-30` `?? undefined`；`aspectOf` 返回 undefined 时不设 aspect-ratio | ✅（无宽高 → 无占位，会有 CLS，但不是空白 UI） |
| `cover_images[].sexual` | `sexualOf` `image.go:84-101` | meta 缺席或未判定 → null | **无人读** | ④ 死负载 |
| `reactions` | `reactionSummaries` `reactions.go:10-53` | 无人表态 → `[]` | `Bar.vue:13` `v-if="list.length"` | ✅ |
| `reactions[].reactors` | 同上 `reactions.go:29,45` | 采样上限 3；**被封禁者被剔除**，所以可能比 `min(count,3)` 少，甚至是 `[]` | `Bar.vue:34` `v-if="r.reactors?.length"`，否则退化成显示 `count` | ✅ 设计正确 |
| `reactions[].viewer` | `reactions.go:31-34` | 匿名 → null | `toKunReactions` `?? false`（`apps/web/app/utils/reactionSummary.ts:8`） | ✅ |
| `comments`（嵌在 best_answer/pinned_reply 里） | `mapComments` `assemble.go:191` | 恒为 `[]`，不会 null；被封禁作者的评论被剔除 | `Comment.vue:110` `v-if="comments.length"` | ✅ |
| `viewer` | `loadTopicViewer` `engage_viewer.go:43-47` | 匿名 → null | 全部调用点用 `?.` 且 `=== true` / `?? false` | ✅，但见 **[M-1]** 匿名看不到「推」按钮 |
| `author_moemoepoint` | `get.go:121`，来自 `LookupMoemoepoints`（`apps/api/internal/topic/repository/v1_read.go:35-55`） | **该用户在 `kungal_user_state` 没有行时 Go map 取零值 → 发 `0`**，而不是 null | `Detail.vue:196` → `MasterUser.vue` / `detail/User.vue:28` **无条件渲染** `{{ user.moemoepoint }}` | ⚠️ **[M-2]**：0 与「未缓存」不可区分，UI 直接显示 0 |
| `content` | `convertBodies` `assemble.go:107-115` | 空正文 → `{object:"document", children:[]}` | `Master.vue:115` `ContentDocument` 无 `v-if`，空 children 渲染成空容器（不报错） | ✅（详情正文按定义非空，`content_markdown` 有 `minLength:1`） |

### 3. `listTopicReplies` GET /topics/{topic_id}/replies

| 方向 | 结论 |
|---|---|
| ① | 无。web 发 `limit`/`sort`/`cursor`/`from_floor`，四个都在 `listTopicRepliesInput`（`detail_ops.go:26-31`）。 |
| ② | 同 `Reply` 形状（见 #4）。分页：`next_cursor` 末页**省略**（`repr/list.go:109` 的 `omitempty`），web 用 `!result.data.next_cursor` 判完（`useTopicReplies.ts:147`）✅。 |
| ③ | `sort` token `floor_asc`/`floor_desc` 两端同源（`detail_ops.go:18` vs `useTopicReplies.ts:17-18`）✅。 |
| ④ | `object`（list 判别式）无人读。 |
| **缺陷** | **[H-4] `from_floor=0` 可达 → 422**，见 §6。 |

### 4. `getReply` GET /replies/{reply_id}

| 字段 | 何时为空 | web | 判定 |
|---|---|---|---|
| `edited_at` | 从未编辑 → null（`assemble.go:186`） | `Reply.vue:159` → `detail/User.vue:48` `v-if="edited"` | ✅ |
| `comments` | 恒 `[]`（`assemble.go:193`） | `Comment.vue:110` `v-if` | ✅ |
| `comments[].parent_comment_id` | 顶层评论 → null；**父评论被剔除时 id 仍在**（Go doc 明说，`reply.go:21`） | `threadComments.ts:9-11` 同时判 `!= null` 与 `byId.has(...)`，找不到父就当根 | ✅ 正确实现了那句 doc |
| `comments[].viewer` | 匿名 → null | `comment/Like.vue:10` `?? false` | ✅ |
| `reactions` | 恒 `[]`（`assemble.go:168-170`） | ✅ | ✅ |
| `viewer` | 匿名 → null | `Rewrite.vue:12` `=== true`、`Delete.vue:58` `?.` | ✅ |
| `author_moemoepoint` | 同 **[M-2]**（`assemble.go:177`） | `Reply.vue:157` 无条件渲染 | ⚠️ |
| `content` | `updateReply` 只在正文变了才写，空文档不可能（`minLength:1`） | `Reply.vue:166` `v-if="reply.content.children.length"` | ✅ 甚至多兜了一层 |
| ① / ③ / ④ | ①无请求体。③`user`→`author`、`created`→`created_at` 已改完。④`object`、`dislike_count`、`viewer.can_like`、`viewer.has_liked`、`viewer.has_disliked` 无人读。 | | |

### 5. `recordTopicView` POST /topics/{topic_id}/views

① 无请求体、无 query、无 `Idempotency-Key`（该 op 是 `v1.Optional` 且未过 `IdempotencyRequired`，`apps/api/internal/topic/apiv1/register.go:82-92`），web 也不发（`Detail.vue:81-83`）✅。② 204 无体。③ — ④ —。
`.catch(() => undefined)` 吞掉一切失败，这是 beacon 的正确姿势。

### 6. `createTopic` POST /topics

| 方向 | 结论 |
|---|---|
| ① | **无多余字段**。`writePayload`（`apps/web/app/composables/topic/useTopicSubmitter.ts:39-61`）只产出 `title` / `content_markdown` / `category` / `sections` / `is_nsfw` / `access_scope` / 条件 `cover_image_hashes` / 条件 `access_roles` / 条件 `access_user_ids`，九个全在 `TopicCreate`（`write_types.go:34-44`）里被 `createTopic` 读到（`write_topic_create.go:17-121`）。 |
| ② | 响应就是 `getTopic` 形状（`write_topic_create.go:112-116`），同 #2。`Location` 头（`write_ops.go` 的 `createTopicOutput`）web 不读，用 body 的 `id` 导航（`useTopicSubmitter.ts:132`）——两者一致。 |
| ③ | `content`→`content_markdown` 已改；`section`→`sections` 已改；`access_user_ids` 是**十进制字符串数组**，web 用 `accessUserIds.value.map(String)`（`useTopicSubmitter.ts:58`）转出去 ✅。 |
| ④ | 与 #2 相同。 |
| **前端 Zod 与 Go schema 的差距** | `createTopicSchema`（`apps/web/app/validations/topic.ts:34-102`）漏了三处 Go 有而 Zod 没有的约束：`sections` 的 `uniqueItems`、`cover_image_hashes` 的 `uniqueItems`、`access_user_ids`/`access_roles` 的 `uniqueItems`+`minItems:1`（后者由 `superRefine` 的 `?.length` 间接覆盖）。长度上限两边一致：title 233（`apps/web/app/config/limit.ts:1` vs `write_types.go:35`）、正文 100007、role ≤4、user ≤50（`apps/web/app/constants/topic.ts:281`）、cover ≤9。重复项只会在 huma 层 422，不会静默。低危。 |

### 7. `updateTopic` PATCH /topics/{topic_id}

三个调用点行为**不同**，分开看：

**7a. 重新编辑（`useTopicSubmitter.ts:102`）**

| 方向 | 结论 |
|---|---|
| ① | 无未知字段（`TopicPatch` 是 `TopicCreate` 的超集 + `state`，`write_types.go:46-57`），但**语义上是全量替换**：`body: payload as TopicPatch` 把 `TopicCreate`（所有字段必填）原样当 patch 发。后果全在服务端 `updateTopic` 里可见：`needEdit` 恒 true（`write_topic_update.go:24-26`）、`grantsTouched` 恒 true（`write_topic_update.go:103`）→ 每次编辑都 `ReplaceAccessGrants`、`sectionsTouched` 恒 true（`write_topic_update.go:104`）→ 每次都 `ReplaceSectionRelations`。值没变时 `titleChanged`/`bodyChanged`/… 仍逐个比对（`write_topic_update.go:96-101`），所以**结果正确**，只是 PATCH 的「缺席即保留」契约在 web 上从未被用到。 |
| ② | 同 #2。 |
| ③ | 无旧名残留。`temp.id = Number(source.topic_id)`（`applyTopicSource.ts:16`）→ `String(tempStore.id)`（`useTopicSubmitter.ts:100`）是一次 string→number→string 往返，id < 2^53 时无损。 |
| ④ | 同 #2。 |
| **缺陷** | **[H-2]** 封面回写丢行、**[M-3]** author-only users 作用域被前端 Zod 卡死，均见 §6。 |

**7b. 隐藏/取消隐藏（`Hide.vue:67`）** — body 只有 `{ state }`。`isAuthor` 用 `String(id) === props.topic.author.id` **字符串比**（`Hide.vue:15`），正确。按钮闸是 `viewer.can_hide` / `can_unhide`（`Hide.vue:19-22`），并且专门给「作者但被管理员隐藏」画了 `blocked` 文案（`:24`）。这是全站最严谨的一处。

**7c. 个人页取消隐藏（`user/Topic.vue:42`）** — body `{ state: 'published' }`，`topic_id: String(topicId)` ✅。但 **[H-5]**：这个列表来自 legacy `/user/:id/topics`，没有 `viewer`，按钮**无条件渲染**（`user/Topic.vue:96-103`）。被管理员/风纪隐藏的话题，作者在这里会看到「取消隐藏」，点下去必定 `403 PERMISSION_REQUIRED`——`capsForTopic` 的 `Unhide` 要求 `author && topic.HiddenBy == "author"`（`caps.go:29`）。同一个判断，详情页用服务端闸、个人页用「没有闸」，两处结论不一致。

### 8. `getTopicSource` GET /topics/{topic_id}/source

| 方向 | 结论 |
|---|---|
| ① | 无。 |
| ② | `access_grants.roles` / `.users` 在作用域不匹配时是 `[]`（`write_source.go:36` 显式初始化）；`access_grants.users[].name` 对封禁/注销者是 null（`write_source.go:55-58` 走 `DeletedUserRef`），web `toKunUser` 兜底 ✅。`cover_images` 同 #2 的跳过规则。 |
| ③ | `description`→无关；此处的重命名是 `cover_images` 由 hash 数组变 `[Image]`，web 用 `coverTokenFromHash(image.hash)` 反解回 token（`applyTopicSource.ts:22-24`）。 |
| ④ | `object`、`topic_id`(只被 `Number()` 存进 temp store)、`cover_images[].url/width/height/thumbhash/sexual` **全部不读**——编辑器只要 hash。 |
| 备注 | `apps/web/app/components/edit/topic/AccessUserPicker.vue:25-27` 的注释「access_grants.user_ids carries bare ids … the floating card is the only id -> name face」已经**过时**：v1 的 `access_grants.users` 是带 name 的 `UserRef`，`knownUsers` 的 immediate watcher 先于 `resolveMissing` 填满 `known`（`:39-53`），所以那条 legacy `/api/user/:id/floating` 调用在编辑路径上已成死代码。不是缺陷，是注释该改。 |

### 9. `createReply` POST /topics/{topic_id}/replies

| 方向 | 结论 |
|---|---|
| ① | body 只有 `{ content_markdown }`（`PanelBtn.vue:24-26`），等于 `ReplyCreate`（`write_types.go:77-79`）。`Idempotency-Key` 头必发且已发（`PanelBtn.vue:44`）✅。 |
| ② | 响应是完整 `Reply`（`write_reply.go:92-95`），同 #4。 |
| ③ | — |
| ④ | `Location` 头无人读。 |
| **缺陷** | **[M-4]**：写成功后 `addNewReply`（`Detail.vue:166`）只往列表里插，**不更新 `topic.reply_count`**，而服务端 `RecomputeTopicCounts`（`write_reply.go:55`）已经改了它。`TopicDetailTool :reply-count="topic.reply_count"`（`Detail.vue:219`）会一直少一条，直到刷新。 |

### 10. `updateReply` PATCH /replies/{reply_id}

① body `{ content_markdown }` = `ReplyPatch` ✅（`PanelBtn.vue:68-70`）。② 完整 `Reply`，`updateReply()` 整体替换 ✅。③ `replyRewrite.id` 来自 `getReplySource` 的 `reply_id`（十进制字符串，`Rewrite.vue:16-18`），直接当 path 参数发回，**没有过 `Number()`** ✅。④ —。

### 11. `deleteReply` DELETE /replies/{reply_id}

① 无请求体。② 204 无体。③ —。④ —。
**[M-5] 两处客户端镜像与服务端不等价**：
- 扣分公式 `3 * (comments.length + like_count + 1)`（`Delete.vue:17`）用的是 **v1 已剔除封禁作者后的** `reply.comments.length`；服务端数的是 `topic_comment WHERE status = 0` 的全部行（`write_reply.go:183-188`）。客户端预估会**偏低**，用户会看到「够」但服务端回 `MOEMOEPOINT_INSUFFICIENT`。
- 删成功后 `topic.reply_count` / `comment_count` 同样不回落（同 **[M-4]**）。
按钮闸是 `reply.viewer?.can_delete`（`Delete.vue:58`）✅。

### 12. `getReplySource` GET /replies/{reply_id}/source

① 无。② `content_markdown` 恒非空。③ —。④ `object`、`topic_id` 无人读。

### 13–16. 表情置位/撤销（topic × reply）

| 方向 | 结论 |
|---|---|
| ① | 无请求体；path 的 `reaction` 走 `ReactionInput` 封闭枚举 33 个 token（`engage_types.go:11-22`）。web 的 picker 词表需与之同源——`ReactionToken` 类型从 `operations['setTopicReaction']['parameters']['path']['reaction']` 取（`apps/web/shared/utils/api/schemas.ts`），`toggle(key: string)` 到 `write()` 时 `key as ReactionToken` 强转（`useReactions.ts:139`）。**这层强转绕过了类型检查**：若 `apps/web/app/constants/reaction.ts` 的 picker 列表里出现一个后端不认的 token，vue-tsc 不会报，运行时 422。（查了一眼两侧词表一致，但这是唯一一处用 `as` 打开的口子。） |
| ② | 响应 `TopicEngagement` / `ReplyEngagement`，`Viewer` 是值类型不为 null，`Reactions` 恒 `[]`（`engage_snapshot.go:35-37, 79-81`）。 |
| ③ | `is_liked` → `viewer.has_liked` 的改名完成了，但 web **两端都不用**：它从 `reactions[].viewer.has_reacted` 推 `mine`（`apps/web/app/utils/reactionSummary.ts:8`）。 |
| ④ | `TopicEngagement.favorite_count` / `upvote_count` / `upvoted_at`、`dislike_count` 在表情写回路径上被 `Master.applyEngagement`（`Master.vue:23-34`）**丢弃**；`ReplyEngagement.dislike_count` 被写进 state 但从不渲染。 |
| 备注 | `applyOptimistic`（`useReactions.ts:95-120`）自己实现了 like/dislike 互斥，与服务端 `applyTopicReactionSet` 的互斥（`engage_reactions.go:50-59`）一致 ✅；失败时整段回滚 ✅。 |

### 17–18. `favoriteTopic` / `unfavoriteTopic`

| 方向 | 结论 |
|---|---|
| ① | 无请求体。`topicFavoriteInput`（`engage_ops.go:30-32`）只有 path。 |
| ② | 响应是**完整 `TopicEngagement`**。 |
| ③ | `is_favorited` → `viewer.has_favorited` ✅（`Favorite.vue:22`）。 |
| ④ | **[M-6]**：`Favorite.vue:41-45` 只取 `favorite_count` + `viewer` 两个字段，把同一份快照里的 `like_count` / `dislike_count` / `reactions` / `upvote_count` / `upvoted_at` 全丢了。这不是错误（收藏不改它们），但确实是「写回快照没被应用到读面填充的同一组字段」。 |
| 备注 | `topic_id` 路径参数用 `props.topic?.id ?? String(topicId.value)`（`Favorite.vue:14`）——组件同时支持 v1 topic 对象与 legacy 数字 id 两种入参，字符串化处理正确。 |

### 19. `upvoteTopic` POST /topics/{topic_id}/upvotes

| 方向 | 结论 |
|---|---|
| ① | body 是 `{}` 或 `{ note }`（`UpvoteModal.vue:26`），`UpvoteCreate` 只有 `Note *string`（`engage_types.go:77`）✅。`Idempotency-Key` 必发且已发 ✅。 |
| ② | 响应是 `TopicUpvote`（**不是** engagement 快照）：`id` / `topic_id` / `upvoter` / `note` / `created_at`。 |
| ③ | `description` → `note` **已在读写两侧都改完**：写面 `UpvoteModal.vue:26` 发 `note`，读面 `UpvoteRecords.vue:84` 读 `item.note`。`UpvoteRecords.vue:6-11` 里还留着一个 `LegacyUpvoteRecord { description }` 接口，但那是给 legacy `records` prop 走的另一条分支（`:63-70`），两条分支各读各的，没有混用 ✅。`user` → `upvoter` 同样改完（`:85`）。 |
| ④ | `TopicUpvote.object`、`topic_id`（只用于 `UpvoteRecords.vue:27` 的字符串比对）。`created_at` 在写回路径上**没被用来更新 `topic.upvoted_at`** → **[H-3]**。 |
| **缺陷** | **[H-1] 幂等键跨话题复用 → 409**，见 §6。**[H-3]** 推完后 `upvoted_at` 不刷新、`upvote_count` 靠客户端 `++` 估算（`Upvote.vue:53-59`）。 |

### 20. `listTopicUpvotes` GET /topics/{topic_id}/upvotes

① 只发 `limit`/`cursor` ✅。② `note` 可为 null（`notePtr`，`engage_upvote.go:24-29`），web `item.note ?? randomUpvoteDescription(Number(item.id))` 兜底 ✅；`upvoter.name` 可 null，`toKunUser` 兜底 ✅。③ 同 #19。④ `object`。
备注：游标 fingerprint 绑 `("topic_upvotes", topic.ID)`（`engage_lists.go:17-19,123`），web 每个 topic 一个 `useCursorList` key ✅。

### 21–22. `listTopicReactions` / `listReplyReactions`

① `limit`/`cursor` ✅。② `Reaction` 没有 `viewer`，`reactor.name` 可 null → `toKunUser` 兜底（`HistoryModal.vue:72-74`）✅。③ `user` → `reactor` **已改完**，`HistoryModal.vue:66,72,74` 三处都读 `item.reactor` ✅。④ `object`。
备注：`reactionAsset(item.reaction)`（`HistoryModal.vue:77`）对未知 token 的兜底要看 `constants/reaction.ts`；`ReactionToken` 是**开放**词表（`repr.OpenEnum`，`detail.go:13`），而列表接口返回的是库里存过的 token，理论上都在 `ReactionInput` 封闭表内。

### 23–26. `setBestAnswer` / `clearBestAnswer` / `pinReply` / `unpinReply`

| 方向 | 结论 |
|---|---|
| ① | set/pin 的 body 是 `{ reply_id }`（`BestAnswer.vue:46`、`Pin.vue:33`），`ReplyChoice.ReplyID` 是 `repr.DecimalID`（`engage_types.go:82`），web 直接发 `props.reply.id` **未过 `Number()`** ✅。clear/unpin 无 body ✅。 |
| ② | 四个都回**完整 `Topic`**（`engage_snapshot.go:118-126`），`replaceTopic(result.data)` 整体替换 ✅ 这是四类写里最干净的。 |
| ③ | — |
| ④ | 同 #2。 |
| 备注 | 闸门是 `pageTopic.viewer.can_set_best_answer` / `can_pin_reply`（`BestAnswer.vue:15`、`Pin.vue:13`），对应 `capsForTopic`（`caps.go:32-33`）✅。`isAuthorsOwnReply` 用 `pageTopic.author.id === props.reply.author.id` **字符串比** ✅（`BestAnswer.vue:18-20`）。写回后 `Detail.vue:147-155` 的 watcher 重算列表里每条回复的 `is_pinned`/`is_best_answer`，`undefined === '123'` 天然 false ✅。 |

### 27. `listGalgameMoyuPatches` GET /galgames/{galgame_id}/moyu-patches

① path 参数 `String(props.galgameId)` ✅（`patch/Container.vue:25`）。
② `name` / `model_name` / `note_markdown` 可 null（`nonEmpty`，`apps/api/internal/galgame/apiv1/moyu.go:191-196`），web 三处都 `v-if` ✅；`types`/`languages`/`platforms` 恒非 null（`vocabulary` 返回 `make(..., len)`）；`publisher.name` 可 null → `toKunUser` 兜底 ✅；`items` 空数组时整块 `v-if="resources.length"` 不渲染 ✅。
③ —
④ `MoyuPatch.object`、`MoyuPatch.id`、`MoyuPatchResource.object` 无人读；`MoyuPatch.web_url` 只读 `items[0]` 的（`:31`），第二个及以后的页面 URL 被丢弃——而服务端 doc 明说「moyu 按 VNDB 字符串去重，同一作品可能有两页」。这是有意的降级（只给一个「前往」链接），不是缺陷，但要知道第二页的链接确实被扔了。

---

## 3. 专项核查 A：每一处 `Number(...)` / `parseInt` 用在 v1 id 上

`rg 'Number\(|parseInt\('` 全量扫 v1 消费面，命中 26 处（已排除 `Math.floor`、`formatNumber`、lottery/poll 的 legacy 数据）。按「会不会被发回服务端」「会不会和别的 id 比较」分类：

### A-1 · 转成数字后**发回服务端**（7 处）

| 位置 | 值 | 去向 | 风险 |
|---|---|---|---|
| `apps/web/app/components/topic/detail/Detail.vue:56` | `Number(props.topic.id)` | legacy `GET /topic/{id}/reply/locate` | 无（legacy 面收数字） |
| `apps/web/app/components/topic/comment/Comment.vue:97` | `Number(comment.id)` | legacy `PUT /topic/{tid}/comment` body `comment_id` | 无 |
| `apps/web/app/components/topic/comment/Like.vue:46` | `Number(comment.id)` | legacy `PUT /topic/{tid}/comment/like` | 无 |
| `apps/web/app/components/topic/comment/Delete.vue:46` | `Number(comment.id)` | legacy `DELETE …?commentId=` | 无 |
| `apps/web/app/components/topic/comment/Panel.vue:33-37` | `replyId` / `targetUser.id` / `parentCommentId`（都源自 `Number(v1 id)`） | legacy `POST /topic/{tid}/comment` | 无 |
| `apps/web/app/composables/topic/applyTopicSource.ts:16` → `useTopicSubmitter.ts:100` | `Number(source.topic_id)` → `String(...)` | **v1** `PATCH /topics/{topic_id}` | 无损往返（id < 2^53） |
| `apps/web/app/composables/topic/useTopicSubmitter.ts:58` | `accessUsers.map(u => u.id)`（`toKunUser` 里的 `Number(ref.id)`）→ `.map(String)` | **v1** `access_user_ids` | 无损往返 |

结论：**没有一处把 `Number()` 后的值原样塞进 v1 的字符串 id 字段**。两处往返都显式 `String()` 回去，vue-tsc 也会拦住（`DecimalID` 在生成类型里是 `string`）。

### A-2 · 转成数字后**与另一个 id 比较**（11 处，全部与 `usePersistUserStore().id`——一个 number——比）

`Footer.vue:53`、`Footer.vue:25`、`ActionBar.vue:75`、`reply/Footer.vue:80`、`comment/Comment.vue:18,223`、`comment/Like.vue:38`、`comment/Delete.vue:18`、`footer/Upvote.vue:35`、`detail/Detail.vue:18-20`、`useReactions.ts:127`、`useReactions.ts:91`。
两侧都是 number，`===` 成立。**另有四处反向做法**，把 store id 字符串化去比 v1 id：`Hide.vue:13`（`String(id) === topic.author.id`）、`reply/Delete.vue:13`、`BestAnswer.vue:18-20`（两个 v1 id 直接比）、`UpvoteRecords.vue:27`（`upvote.topic_id !== String(props.topicId)`）。
**两种风格混用，但每一处内部都自洽**，没有发现 `'4121' === 4121` 这类恒假比较。

### A-3 · 作为对象键 / Set 成员

全部是**字符串键**，没有与数字 id 并存的 map：
`Detail.vue:112,122`（`Set<string>` of `reply.id`）、`useTopicReplies.ts:21,75`（`Set<string>`）、`useCursorList.ts:154`（`new Set(items.map(i => i.id))`，`Item extends { id: string }`）、`threadComments.ts:4,22`（`Map<string, Comment>`，键与 `parent_comment_id` 同型）、`patch/Container.vue:41`（`Set<string>` of resource id）。
`useReactions.ts:91` 的 `r.reactors.filter(u => u.id !== id)` 两侧都是 number（`KunUser.id`）。✅

### A-4 · 只做本地用途，不回传也不比较（3 处）

`pages/topic/[id]/index.vue:182`（`Number(topic.id)` → og 卡 URL）、`UpvoteRecords.vue:84`（`Number(item.id)` 当伪随机种子）、`Detail.vue:213,257` / `Master.vue:102`（`Number(topic.id)` 传给 miniapp / legacy 组件的 number prop）。无风险。

---

## 4. 专项核查 B：`viewer.can_*` 是不是唯一的写按钮闸

### B-1 · v1 覆盖到的写操作：是的，`can_*` 是唯一闸门

| 按钮 | 闸 | 对应 Go |
|---|---|---|
| 话题重新编辑 | `topic.viewer?.can_edit === true`（`footer/Rewrite.vue:13`） | `capsForTopic.Edit` `caps.go:27` |
| 隐藏 / 取消隐藏 | `can_hide` / `can_unhide`（`footer/Hide.vue:19-22`） | `caps.go:28-29` |
| 推话题 | `can_upvote`（`footer/Upvote.vue:65`） | `caps.go:31` |
| 设/取消最佳答案 | `can_set_best_answer`（`reply/BestAnswer.vue:15`） | `caps.go:32` |
| 置顶/取消置顶回复 | `can_pin_reply`（`reply/Pin.vue:13`） | `caps.go:33` |
| 回复重新编辑 | `reply.viewer?.can_edit === true`（`reply/Rewrite.vue:12`） | `capsForReply.Edit` `caps.go:49` |
| 删除回复 | `reply.viewer?.can_delete`（`reply/Delete.vue:58`） | `caps.go:50` |

`rg "useCan\("` 全站 37 处，**没有任何一处是 `topic.edit_any` / `topic.hide` / `topic.set_best_answer` / `reply.edit_any` / `reply.delete_any` / `reply.pin`**。话题域的角色镜像在 web 上已经彻底拆干净。

### B-2 · 仍然存在的三个非 `can_*` 闸（都不是 v1 写面，但值得记）

1. **评论的编辑/删除**：`useCan('comment.topic.edit')`（`comment/Comment.vue:16`）与 `useCan('comment.topic.delete')`（`comment/Delete.vue:14`）。v1 的 `CommentViewer` **只有 `has_liked`**（`reply.go:49-51`），既没有 `can_edit` 也没有 `can_delete`，而评论的写面根本还没迁到 v1（仍走 `kunFetch('/topic/{tid}/comment')`）。所以这是**契约缺口而不是矛盾**：W3/W4 没覆盖评论，前端只能继续镜像权限。要消掉它，`Comment` 形状得补 `viewer.can_edit` / `can_delete`。
2. **`useCan('topic.view_hidden')`**（`user/Topic.vue:15`）——只控 tab 是否出现，不控写。
3. **`useCan('poll.create_any' / 'poll.edit_any')`**（`detail/Detail.vue:14-15`）——miniapp 域，不是 v1。

### B-3 · 一处 `can_*` 缺席导致的按钮错画

**[H-5]**（见 §2-7c）：`user/Topic.vue:96-103` 的「取消隐藏」无闸。这就是 B-1 那张表之外唯一一个「v1 写面的按钮没有被 `viewer` 闸住」的地方，原因是该列表根本不是 v1 拉的。

### B-4 · 三处冗余但不矛盾的客户端预判

- `footer/Upvote.vue:31-42`：`!id` → 登录弹窗、`id === Number(author.id)` → 提示、`moemoepoint < 10` → 提示。因为整个 `<template v-if="topic.viewer?.can_upvote">` 已经把「匿名」和「作者本人」排除了，前两个分支**不可达**。
  副作用是 **[M-1]**：匿名读者**看不到**「推」按钮（`viewer` 为 null → `can_upvote` 为 undefined），不再有「点一下弹登录框」的引导。对比同一排的收藏（`favorite/Toggle.vue:47-51` 会弹登录）、表情（`useReactions.ts:122-126` 会弹登录），三者对匿名的行为不一致。
- `useReactions.ts:127`：`key === 'like' && id === opts.targetUserId` 镜像 `SELF_LIKE_FORBIDDEN`（`engage_reactions.go:19-21`）。这个有用，因为表情条对匿名/作者都渲染。
- `reply/Delete.vue:17` 与 `comment/Delete.vue:22` 的扣分公式镜像——见 **[M-5]**。

---

## 5. 专项核查 C：写回的 engagement 快照有没有被完整应用

| 写 | 响应形状 | web 应用到 | 丢弃的字段 | 判定 |
|---|---|---|---|---|
| set/removeTopicReaction | `TopicEngagement`（7 字段） | `useReactions.ts:146` 设 `list`；`Master.vue:27-32` 设 `like_count` `dislike_count` `viewer` `reactions` | `favorite_count` `upvote_count` `upvoted_at` | 可接受（该写不改它们），但**快照里是新鲜值却被扔** |
| set/removeReplyReaction | `ReplyEngagement`（6 字段） | `Reply.vue:39-45` 设 `like_count` `dislike_count` `viewer` `reactions` | 无（其余只有 `object`/`reply_id`） | ✅ 完整 |
| favorite/unfavoriteTopic | `TopicEngagement`（7 字段） | `Favorite.vue:41-45` 设 `favorite_count` `viewer` | `like_count` `dislike_count` `reactions` `upvote_count` `upvoted_at` | **[M-6]** 丢 5/7 |
| upvoteTopic | `TopicUpvote`（**不是**快照） | `Upvote.vue:53-59` 客户端 `upvote_count++` + 手改 `viewer.has_upvoted`；`UpvoteRecords.vue:21-36` 把整条 upvote 插进列表 | `created_at` 没被用来更新 `topic.upvoted_at` | **[H-3]** |
| updateTopic / best-answer / pinned-reply ×4 | 完整 `Topic` | `replaceTopic(result.data)` 整体替换 | 无 | ✅ 完整 |
| createReply / updateReply | 完整 `Reply` | `addNewReply` / `updateReply` | `topic.reply_count` 未更新 | **[M-4]** |
| deleteReply | 204 | `removeReply(id)` | `topic.reply_count` / `comment_count` 未回落 | **[M-4]** |

**一致性检查**：`viewer` 被整体替换的两处（`Master.vue:31`、`Favorite.vue:44`）不会把 `reactions[].viewer.has_reacted` 弄脏，因为 web 的「我表过态没」只读 `reactions[].viewer`（`apps/web/app/utils/reactionSummary.ts:8`），而 `TopicViewer.has_liked/has_disliked` 从不被读。所以 `TopicViewer` 与 `reactions[].viewer` 这两套「我的状态」在 web 上**永远只有一套被使用**，不存在互相打架的可能。

---

## 6. 发现清单（按严重度）

### [H-1] 幂等键在 UpvoteModal 里跨话题复用 → `409 IDEMPOTENCY_KEY_REUSED`

- `<LazyTopicUpvoteModal />` 在 `apps/web/app/app.vue:152` **全局挂一次**，整个 SPA 生命周期内是同一个组件实例，因此 `const createKey = useIdempotencyKey()`（`UpvoteModal.vue:8`）里的 key ref 跨路由存活。
- `useIdempotencyKey.take(payload)`（`apps/web/app/composables/useIdempotencyKey.ts:5-12`）**只用 `JSON.stringify(body)` 当指纹**，body 里没有 topic id。`clear()` 只在成功后调用（`UpvoteModal.vue:42`）。
- 服务端指纹是 `sha256(method + " " + path + "\n" + body)`（`apps/api/internal/apiv1/idempotency.go:136-138`），**包含路径**。
- 复现路径：话题 A 上推一次失败（403 `SELF_UPVOTE_FORBIDDEN` / `MOEMOEPOINT_INSUFFICIENT`，或 422，或网络错误）→ key 未清 → 去话题 B 推、留**同样的**留言（含「不留言」即 `{}`）→ `take` 认为指纹没变，复用同一个 uuid → 服务端 key 存在但指纹不符 → `409 IDEMPOTENCY_KEY_REUSED`。
- 409 不会被写回 redis（`storeIdempotentStatus` 排除 409 与 429，`idempotency.go:194-202`），但**原记录还在，24 小时窗口内每次都 409**（`idempotencyReplayWindow = 24h`，`:35`）。用户只能改留言文案绕开。
- 用户看到的是「这次提交与之前的提交冲突，请刷新页面后重试」（`apps/web/i18n/locales/zh-CN/problem.json:14`）——刷新页面确实能解（组件重建，key 重置），所以文案歪打正着，但根因在前端。
- 顺带：同样的 key 生命周期让 `403 MOEMOEPOINT_INSUFFICIENT` 被**存成可重放响应**（403 在 200–499 且不是 409/429），用户补足萌萌点后用同样留言再推，会拿到重放的 403。`createTopic`/`createReply` 有相同结构，但它们的组件是随页面创建的，跨目标复用不可能；只有 upvote 的模态框是全局单例。

### [H-2] 编辑话题会永久删掉无法解析的封面 token，并把 `_variant` 归一化

- `getTopicSource` 的 `cover_images` 走 `coverImagesWithMeta`（`write_source.go:78`），其中 `repr.NewImageFromToken` 对不匹配 `^/image/([0-9a-f]{64})(?:_([a-z0-9]+))?$` 的 token 返回 nil 并被 `continue` 跳过（`summary.go:114-118`）。Go doc 也明说「Tokens that do not parse are skipped」。
- web 把剩下的 `image.hash` 反解成 `/image/<hash>`（`applyTopicSource.ts:22-24`），**重新编辑时无条件把这份列表当 `cover_image_hashes` 发回**（`useTopicSubmitter.ts:49-53`：`if (mode === 'rewrite') body.cover_image_hashes = hashes`），服务端 `coversFromHashes` 用它整体替换（`write_topic_update.go:73-75` → `write_validate.go:94-104`）。
- 结果：① 存量行里任何非 `/image/<hash>` 形态的封面（按 memory「cover-hash migration — 041 + 回填仍待做」，理论上存在旧 CDN URL 形态）在作者点一次「重新编辑→保存」后被**从库里抹掉**；② 存成 `/image/<hash>_mini` 的 token 会被归一化成裸 hash（渲染上无害，`_mini` 是 16:9 裁切，反而更对）。
- 我**无法从代码判定线上是否真有这样的行**——那要查 `topic.cover_images` 列。见 §7。

### [H-3] 推完话题后 `upvoted_at` 不刷新，`upvote_count` 是客户端估算

- 服务端 `upvoteTopic` 会 `ApplyUpvoteCountAndTime`（`engage_upvote.go:61`），即同时改 `upvote_count` 与 `upvote_time`（→ `upvoted_at`）。
- 响应只有 `TopicUpvote`（`engage_upvote.go:94-101`），web 拿到后 `upvoteCount.value++` 并手写 `viewer.has_upvoted = true`（`Upvote.vue:53-59`），**没有动 `topic.upvoted_at`**。
- 直接后果：`Master.vue:66` 的 `:upvote-time="topic.upvoted_at"` 仍是旧值。若这是该话题的**第一次**被推，`upvoted_at` 是 `null`，「该话题被推」徽章（`BadgeGroup.vue:55`）在刷新前不会出现。
- 这是 operation 形状本身的选择（POST 建资源就回资源），但 `TopicUpvote.created_at` 已经是新的 `upvoted_at`，客户端有料却没用。

### [H-4] `from_floor=0` 可达，服务端 422

- `useTopicReplies.loadEarlier` 在首次回退时发 `from_floor: lowestFloor() - 1`（`apps/web/app/composables/topic/useTopicReplies.ts:187`），而 `lowestFloor()` 在 `replies` 为空时返回 **1**（`:88-93`）→ 发出 `from_floor=0`。
- `listTopicRepliesInput.FromFloor` 是 `minimum:"1"`（`detail_ops.go:30`），huma 会校验 present 的 query 参数（F-2）→ 422。
- 可达前提：深链 `?reply=N`（N ≥ 2）打开，但该话题在 floor ≥ N 处**一条可见回复都没有**（末楼小于 N，或 ≥N 的回复全被隐藏/作者被封）。此时 `hasEarlier` 被置 true（`:148`，只看 `fromFloor > 1`），`replies` 却是空的，用户点「加载更早的回复」即触发。
- 表现不是白屏，而是一个 `problemMessage` 提示 + 「重试」按钮，重试会再 422。

### [H-5] 个人页「取消隐藏」没有 `can_unhide` 闸

见 §2-7c。作者在 `/user/:id` 的「已隐藏」tab 里，对**管理员或风纪隐藏**的话题也会看到「取消隐藏」按钮（`apps/web/app/components/user/Topic.vue:96-103` 无条件渲染），点击必得 `403 PERMISSION_REQUIRED`（`capsForTopic.Unhide` 要求 `author && HiddenBy == "author"`，`caps.go:29`）。同一个决定在详情页有正确的三态处理（`footer/Hide.vue:18-29` 的 `hide`/`unhide`/`blocked`/`none`）。根因：该列表来自 legacy `/user/:id/topics`，没有 `viewer`。

### [M-1] 匿名读者看不到「推话题」按钮

`footer/Upvote.vue:65` 的 `v-if="topic.viewer?.can_upvote"` 对匿名恒 falsy。收藏与表情对匿名都会弹登录框，推不会。这是 W4 把闸从客户端搬到 `viewer` 时的行为变化。

### [M-2] `author_moemoepoint` 无法区分「0」与「本地没缓存」

`LookupMoemoepoints` 只回有行的用户（`apps/api/internal/topic/repository/v1_read.go:44-53`），Go map 取零值 → 发 `0`（`get.go:121`、`assemble.go:177`）。schema 是非空 `int`，没有 null 表达「未知」。web 在 `detail/User.vue:28` 与 `detail/MasterUser.vue` 里**无条件**渲染 `{{ user.moemoepoint }}`。按 memory「user-state lazy provisioning」，`kungal_user_state` 的行是懒建的，所以「注册了但从没在论坛留过痕」的作者会被显示成 0 萌萌点。这与 C3（本地 `moemoepoint` 只是缓存视图）一致，但 UI 上无法表达。

### [M-3] author-only `users` 作用域的话题，前端 Zod 卡死编辑

`grantsFromAccess` 会丢掉等于作者本人的授权（`write_validate.go:199`），所以「只授权给自己」的 users 话题**一条 grant 行都不存**。Go 侧为此专门加了 `keepStored`，注释写着那个事故：「every PATCH of such a topic answered 422 REQUIRED」（`write_validate.go:137-140`）。
但 web 从来不走 `keepStored` 那条路——它**总是**发全量 patch，`access_user_ids` 取自 `access_grants.users`（空数组），于是 `createTopicSchema.superRefine` 的 `data.access_scope === 'users' && !data.access_user_ids?.length` 命中（`apps/web/app/validations/topic.ts:95-101`），弹「请至少指定一位可以看到本话题的用户」，请求根本发不出去。
即：服务端修好的那个事故，在前端以另一种形态还活着。前提是线上确实存在 author-only 的 users 话题——Go 注释表明存在过。

### [M-4] 回复增删不同步话题计数

`createReply`（`write_reply.go:55`）与 `deleteReply`（`write_reply.go:200`）都调 `RecomputeTopicCounts`，但 web 只增删列表项（`Detail.vue:166,182`），`topic.reply_count` / `comment_count` 原地不动。`TopicDetailTool :reply-count`（`Detail.vue:219`）因此会短期失真。

### [M-5] 删除回复的扣分预估偏低

`Delete.vue:17` 用 `props.reply.comments.length`——这是**已剔除封禁作者评论**的数组（`assemble.go:196-197`）；服务端数 `topic_comment WHERE topic_reply_id = ? AND status = 0`（`write_reply.go:183-188`），不看封禁。回复下若有被封用户的评论，弹窗写的数字会小于真实扣分，用户可能在萌萌点不够时才被服务端拦下。

### [M-6] 收藏写回丢弃 5/7 个快照字段

见 §5 表。不产生错误显示，但「写回快照应用到读面同一组字段」这条要求在这里没做到。

### [L-1] 前端 Zod 缺 `uniqueItems`

`createTopicSchema` 对 `sections` / `cover_image_hashes` / `access_roles` / `access_user_ids` 都没查重，Go schema 四个都有 `uniqueItems:"true"`（`write_types.go:38,40,42,43`）。重复项会在 huma 层 422，不会静默。

### [L-2] `ReactionToken` 的 `as` 强转

`useReactions.ts:139` 的 `write(key as ReactionToken, held)` 是全链路唯一一处用 `as` 绕过 openapi 生成类型的地方。picker 词表（`apps/web/app/constants/reaction.ts`）与 `reactionVocabulary`（`engage_types.go:11-22`）当前一致，但这条链上没有类型保护，加 token 时两边必须手动同步。

### [L-3] `AccessUserPicker.vue:25-27` 的注释已过时

v1 的 `access_grants.users` 带 name，该文件里那条 legacy `/api/user/:id/floating` 解析在编辑路径上已成死代码。注释里说的前提（「carries bare ids」）不再成立。

---

## 7. 无法只靠代码判定的事项

1. **线上 `topic.cover_images` 里是否真有不匹配 `/image/<64hex>(_variant)?` 的 token**。这决定 **[H-2]** 是「理论风险」还是「正在丢数据」。查法：对 infra postgres 容器跑 `SELECT id, cover_images FROM topic WHERE cover_images <> '' AND cover_images !~ '^(/image/[0-9a-f]{64}(_[a-z0-9]+)?)(,/image/[0-9a-f]{64}(_[a-z0-9]+)?)*$' LIMIT 20;`（列是逗号分隔的 text，见 `apps/api/internal/topic/model/image_tokens.go`）。
2. **线上是否存在 `access_scope='users'` 且没有任何 `user` 类型 grant 行的话题**。这决定 **[M-3]** 是否正在发生。Go 注释说这个事故已经发生过，但那是 PATCH 阶段的 422，不等于现在仍有这样的行。
3. **`kungal_user_state` 里缺行的用户占比**，决定 **[M-2]** 的可见程度。
4. **`ReactionToken` 开放词表在库里的实际取值**：`listTopicReactions` 回的是库里存过的 token，若历史上写进过 `reactionVocabulary` 之外的值，`HistoryModal.vue:77` 的 `reactionAsset()` 会给什么兜底——要读 `apps/web/app/constants/reaction.ts` 的实现才能定（本次未展开，因为它不是前后端字段问题）。
5. **[H-1] 的实际触发率**：取决于用户在推失败后会不会立刻去推别的话题并留同样的话。代码上必然成立，频次不可知。
