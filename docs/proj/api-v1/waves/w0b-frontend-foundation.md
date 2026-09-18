# W0b · 前端地基 + 话题列表切到 v1

> 裁决记录（中文，给人看）。执行者：cursor-agent，模型 Grok 4.6 Extra High，在沙箱里跑。验收：Claude。写于 2026-09-18。分工同 W0a：执行者写代码、跑不需要数据库的门；依赖安装、DB 测试、浏览器实测、提交由 Claude 做。
>
> | 派发 | 范围 | 依赖 |
> |---|---|---|
> | W0b-1 `client` | §1 生成类型、类型化客户端、错误目录、F2 / F4 / F5 / F6 | W0a |
> | W0b-2 `topic-page` | §2 `/topic` 页与 sitemap 切到 v1，删网页侧旧代码 | W0b-1 |
> | 部署后 | §3 删旧路由 `GET /api/topic` | W0b-2 上线 |

## 1. W0b-1：客户端地基

规定见 [02 §2.1](../02-governance.md) 与 [01 §2 K8](../01-standard.md)，落地形状写在 02 §2.1 末尾。本波另外的裁决：

- **依赖**：`openapi-fetch@0.17.0`、`openapi-typescript@7.13.0`，都精确钉版（后者的输出由 F5 比对，版本就是生成物的一部分）。由 Claude 在派发前 `pnpm add` 并单独提交：沙箱够不到 pnpm store。
- **`additionalProperties: true` 从文档里去掉**。试生成时每个 schema 都带 `& { [key: string]: unknown }`：huma 在 `AllowAdditionalPropertiesByDefault = true`（请求体接受未知字段所需）下给每个结构体写 `additionalProperties: true`，openapi-typescript 把它渲染成索引签名，于是 `topic.titel` 的类型是 `unknown` 而不是编译错误，对象字面量也不做多余属性检查。JSON Schema 里 `true` 与缺省同义，huma 的校验器只在它是 `false` 或 schema 时才动作，所以在 `sealDocument` 里把 `true` 置空，运行时行为不变。映射类型的值 schema 不动。
- **错误目录的文案由 Claude 定**，任务书逐字给出。reason 按参数分变体（`OUT_OF_RANGE` 可能只带 `minimum` 或只带 `maximum`，`UNKNOWN_VALUE` 只在封闭词表时带 `allowed`），缺参数时退回 `default`，不会出现「不能小于 {minimum}」这种半截文案。
- **暂不做**：会话失效 / 封禁的全局副作用、UUIDv7 幂等键。W0b 唯一的 v1 调用是可选鉴权的 GET，用不到它们；等第一个用得到的调用一起做。

### W0b-1 验收（2026-09-18）

Grok 用了 1481 秒、211 次调用。范围内的东西全部照任务书落地，只写了允许的路径，自己跑的变异都是真的。它照要求在 F6 命中 `server/utils/kunOgCard.ts` 时停下来报告，没有自作主张去压制：那是 OG 卡片服务的 `${ogBaseUrl}/v1/og/…`，不是论坛的 API，验收时逐行豁免。

验收改了什么：

- `useApi` 原来在 `useAsyncData` 的 handler **里面**调 `useApiClient()`。它能工作，只是因为 Nuxt 4.4 首次执行时同步调用 handler；服务端的 `useRequestHeaders` 离开 setup 上下文就拿不到。改为在 setup 里同步取一次客户端。顺手删掉它给 promise 重新定义 `then` / `catch` / `finally` 的多余代码。
- `message.ts` 查目录改用 `Object.hasOwn`：`in` 会顺着原型链命中 `constructor` 这类键。
- 它给 `walkSchema` 加的映射值递归删掉了：结构体值 huma 会注册成 component，本来就会被走到；标量值带不了 enum tag。变异证明它没有任何测试能区分。
- 补两条测试：`application/json` 里长得像 problem 的体仍算 `http`；设了超时后调用方自己的 signal 仍能取消请求（否则 Nuxt 换 key 时取消不了服务端的旧请求）。
- `web.yml` 头注释补一行说明 F5（它的编辑被 cursor 的钩子拦了）。

验证：

- Go：`make lint`、`go test ./...` 绿；`make openapi` 两次输出一致，diff 只有 11 行 `"additionalProperties": true` 的删除；一次性库上 `KUN_REQUIRE_TEST_DB=1` 全量 DB 套件绿（契约测试用新文档校验每条响应）。
- 网页：F5、`lint`、`typecheck`、`test`（26 个文件 205 条，另加两条）全绿。
- 变异：18 个里 17 个被杀（`settle` 的媒体类型判断、超时分类、`errors` 与 `request_id` 的取值、cookie 过滤与有无、调用方 signal、变体挑选、数字千分位、列表分隔符、`network` 文案、`useApi` 的 problem 与 key 响应式、F2 缺 504、F4 基线差一、F6 模板选择器）；唯一存活的是上面删掉的死代码。抽查两个，确认是断言失败而不是编译失败。

## 2. W0b-2：`/topic` 页与 sitemap

- **加载更多，不做自动加载**。用户 2026-06-24 把 `/topic` 从滚动无限加载改成了分页器；列表改游标后分页器没法保留（跳页需要页码，K11），就改成一个「加载更多」按钮，只在点击时加载，页脚始终可达。
- **后退 / 前进恢复**：从话题详情后退到 `/topic`，已加载的全部条目原样显示、不发请求，浏览器的滚动恢复才能落回原处；其他方式进入（链接、菜单、刷新）都取新的第一页。实现是一个客户端插件记下「这次导航是不是 popstate」，列表在浏览器内存里按 key 存快照（最多 10 个 key，不进 SSR payload）。
- 排序沿用现有的六个键与升降序按钮，URL 只剩 `?sort=<token>`；不在这十二个 token 里的值当作默认值，绝不发给 API。`include_nsfw` 取自设置里的内容分级。每页 50 条。
- 删号作者显示「已注销用户」：服务端发 `name: null`（F8），由客户端出文案。
- **sitemap 的话题源从 2026-06-24 起一直是空的**：旧 `GET /api/topic` 那天从数组改成了 `{ topics, total }`，sitemap 的 `pick` 仍按数组取，每页都得到 `[]`。改成沿 v1 游标遍历（每页 100 条，匿名，最多 200 页）。

## 3. 部署之后：删旧路由

旧 `GET /api/topic` 在 W0b-2 里**不删**。同一次部署里删掉它，会有两个窗口出错：新 API 先上线而旧网页还在时，`/topic` 的 SSR 直接失败；部署前打开、还没刷新的标签页在客户端导航到 `/topic` 时也会失败。等 W0b-2 的网页上线后再删：路由、`TopicHandler.GetList`、`TopicService.GetList`、`FindList`、`TopicListResponse`，重生成 `routes.golden`，`legacy_route_baseline` 320 → 319，同时把 F3 改成与 F4 一样的「等于基线」。
