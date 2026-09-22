# W1–W4 回归与对齐验证（2026-09-22）

> 用户要求：「完成的模块需要进行细致的测试，确保所有的功能都没有发生改变，前后端数据字段真的对齐了。」
> 这份记录只写**跑出来的证据**，不写推断。环境：本地 dev 栈（web 2333 / api 2334），dev 库已跑到迁移 100。

## 1. 读面：旧接口与 v1 并排逐字段对比

同一个话题（dev `topic 4121`，25 层回复、10 收藏、1 推、1 个表情汇总、有最佳答案与置顶回复），匿名身份同时打两个面。

**话题详情**（`GET /api/topic/4121` ↔ `GET /api/v1/topics/4121`）：22 个共享字段全部相等。差异只有契约里声明过的那几类：

| 差异 | 旧 | v1 | 依据 |
|---|---|---|---|
| id / author.id | `4121`（数字） | `"4121"`（十进制字符串） | 01 §110 |
| 话题状态 | `status: 0` | `state: "published"` | 命名表 |
| 未隐藏 | `hidden_by: ""` | `hidden_by: null` | 空串不是取值 |
| 空数组 | `mini_apps: null` | `mini_apps: []` | 数组永不 null |
| 正文 | `content_markdown` + `content_html` | `content` 节点树 | 03 |
| 查看者字段 | `is_liked` / `is_disliked` / `is_favorited` / `is_upvoted` | `viewer.has_*` | K9 |
| 时间戳 | 本地偏移 + 毫秒 | UTC 秒精度 `Z` | 01 §110 明写「秒精度」 |
| 新增 | — | `object` / `comment_count` / `pinned_reply` / `viewer` / `author_moemoepoint` | — |

`view` 相差 1，因为两次调用各自计了一次浏览——两个面都计数，行为一致。

**回复列表**（`GET /api/topic/4121/reply?...limit=30&sort_order=asc` ↔ `GET /api/v1/topics/4121/replies?limit=30`）：

- 楼层集合完全相同（各 25 层，`floors_only_legacy` 与 `floors_only_v1` 皆空）；
- 按楼层配对后，**id、作者、like_count、dislike_count、评论条数、表情条数、is_pinned、is_best_answer 逐条零差异**；
- 唯一的顺序差异是旧接口把置顶/最佳答案顶到第一页首位，v1 是纯楼层序——这是 W2 有意移除的「首页特例」（旧做法让「最后一页」的判据失效）。

## 2. 游标分页全量遍历

`GET /api/v1/topics/4121/replies`，`limit=2` 翻到底：**13 页、25 条，与一次性 `limit=30` 的结果顺序完全一致，0 重复、0 遗漏**。

## 3. 跨面一致性

取话题列表前 10 条，逐条再打详情面，比较两面都有的 14 个字段（标题、分类、状态、NSFW、五个计数、浏览数、时间、作者、版块）：**零差异**。

## 4. 匿名态

- `viewer` 在话题、回复、表情汇总三层都是 `null`（不是空对象、不是默认值）；
- 互动历史列表匿名可读并回 200（公开话题，与 `getTopic` 同一判定，符合 W4 裁决）；
- 匿名写 → 401。

## 5. 浏览器实测

见 [w4-interactions.md §4](w4-interactions.md)。收藏置位/撤销、推（201 + 备注去空白 + 余额 −10）、表情置位/撤销、表情历史游标列表、楼主能力位、重新编辑拉 source 回填，全部实测通过。

## 6. 构建与套件

- `pnpm -F web build` 生产构建通过（`vue-tsc` 不解析模板里的组件，只有真正构建一遍才会暴露组件引用断掉）；
- `pnpm -F web lint` / `typecheck` / `test`（383 条）全绿；
- `GOTOOLCHAIN=go1.26.1 make lint` 零输出，`go test -count=1 -p 1 ./...` 全绿（独立临时库）。

## 7. 本次同时修掉的两个生产缺陷

见 commit `b91dbbb7`。评论的 `target_user_id` 改为服务端推导、`reply_id` 与 `topic_id` 绑定；投票校验选项属于本投票且不可重复。6 条变异全部被杀。生产库查证：跨话题评论 0、子评论目标错配 0、跨投票投票 0、选项计数漂移 0——两个洞都没有被利用过。

dev 上通过完整中间件链实测：传 `target_user_id: 1`（陌生人），服务端返回的 `target_user` 是楼层作者；跨话题 `reply_id` 回 404。

## 8. 顺带记下的、留给 W5 的小事

- `DELETE /api/topic/:tid/comment` 从 query string 读 **`commentId`（驼峰）**，与全站 snake_case 不一致。
- 网页仍然发送 `target_user_id`，服务端已忽略它。没有一并删掉是为了不让网页与 API 产生部署先后依赖；评论随 W5 进 v1 时这个字段整个消失。
