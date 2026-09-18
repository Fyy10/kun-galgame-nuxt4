# 02 · 契约治理

## §1 单一来源

- **spec 由运行中的代码产出**（infra A14）。v1 的每个操作用 huma 注册，路由、输入输出类型、描述、状态码在同一处声明；handler 就是被声明的那个函数。
- 注册**不得**依赖活的数据库或 Redis：`go run ./cmd/openapi` 在零依赖下构建同一个 huma API，写出：
  - `apps/api/openapi/kungal-v1.json` —— OpenAPI 3.1，`info.x-stability: preview`；
  - `apps/api/openapi/problems.json` —— 错误码与 reason 注册表的机读形式。
- 两个文件提交进仓，输出确定（键序稳定、两空格缩进、末尾换行），重跑无 diff。
- `make openapi` 是唯一的生成入口。改了 v1 的任何东西，同一个提交里必须带上重新生成的这两个文件（门 G1）。
- 不对外提供 `/api/v1/openapi.json` 与 `/docs`：仓里的文件就是契约。App 仓按论坛的 commit 钉住它。

## §2 客户端

### 2.1 网页（W0b 落地）

- **类型**：`openapi-typescript` 从 `apps/api/openapi/kungal-v1.json` 生成 `apps/web/shared/types/api/v1.d.ts`（提交进仓；eslint / prettier 排除）。版本进 `devDependencies` 与 lockfile，不用 `pnpm dlx`。
- **客户端**：`openapi-fetch`，按「路径 + 方法」推导参数、请求体、响应与错误的类型。写错一个字段名，`vue-tsc` 就报错。snake_case 迁移时那约 40 处「发出的字段名不对」，在这套机制下全部是编译错误。
- **SSR**：用 openapi-fetch 的中间件做三件事——服务端 / 浏览器两套 base URL、SSR 时转发 `kungal_session` cookie、服务端超时。页面级数据走 `useAsyncData` 包一层（`useApi`），沿用 Nuxt 的 payload 水合，不在客户端重复请求。
- **错误**：客户端把非 2xx 统一收成 `Problem` 对象。展示只走 `problemMessage(problem, locale)`：按 `code` / `reason` / `params` 查 `i18n/locales/zh-CN/problem.json`，未知 code 按 status 兜底。调用点可以按 code 接管展示，例如 404 渲染自己的空态，或表单把 `errors[].pointer` 挂到对应控件下。**不得**把 `title` / `detail` 显示给用户。
- **幂等键**：创建类 `POST` 由客户端在「用户点一次提交」时生成一个 UUIDv7，同一次提交的重试复用它。
- **手写类型**：v1 操作的请求与响应**不得**再手写 TS 类型。需要具名类型时，从生成物里别名：`type TopicSummary = components['schemas']['TopicSummary']`。旧的手写类型随对应旧端点一起删。
- **旧调用的棘轮**：`kunFetch` / `useKunFetch` 的调用点数只减不增（门 F4）。

### 2.2 App（kungal-apps）

- tonik 从论坛仓 `apps/api/openapi/kungal-v1.json` 生成 Dart 客户端，按论坛 commit 钉版。
- 只绑定 `/api/v1`；旧 `/api/*` 对 App 视为不存在。
- 错误展示遵守 [01 §2 K8](01-standard.md)：ARB 的键从 `problems.json` 来，App 仓自己做覆盖检查。
- 请求带 `User-Agent: kungal-app/<版本> (<平台>)`。论坛按它统计各版本流量，这是退役旧形状的依据（§4）。

## §3 CI 门

G 编号沿用 infra 07 §2 的同名门，F 编号是论坛补的。**每道门必须有阳性对照**：测试里先构造一个违规的 schema 或样本，证明门会红，再对真实产物断言。对照要走和被断言者**同一条构造路径**（infra 第九轮教训：路径字面量搜索漏掉了把 `/api/v1` 放进环境变量的客户端，而阳性对照照样通过）。

| 门 | 检查 | 实现 | 作业 |
|---|---|---|---|
| **G1** | ① 提交的 spec 与 `problems.json` 等于重新生成的结果。② 契约测试打真路由：每条响应都通过 spec 校验（状态码在声明集合里、body 符合 schema、`Content-Type` 正确） | ① `make openapi && git diff --exit-code`。② Go 测试（kin-openapi `openapi3filter`），带真库 | api · db |
| **G2** | 操作、参数、property、响应都有非空 `description` | spec 测试 | api |
| **G3** | 每个枚举标 `x-vocabulary-closed`；开放枚举标 `x-vocabulary` | spec 测试 | api |
| **G4** | 每个操作声明它真实会发的全部状态码，错误响应一律 `$ref` Problem。最低集合：全部操作 500；有参数 400；`required` / `optional` 档 401；路径带 id 404；有请求体 400 + 415 + 422；要求幂等键 409 | spec 测试 | api |
| **G5 / G13** | 注册表七项检查（infra 10 §7）；code ↔ type URI 双向一一对应；`problems.json` 与注册表一致；代码里构造的每个 code / reason 都在注册表里 | Go 测试 + AST 扫描 | api |
| **G6** | 2xx schema 顶层不含 `code` / `message` / `data` / `success` / `status` / `timestamp` / `error` | spec 测试 | api |
| **G7** | 名为 `id`、以 `_id` 结尾的 property 与参数是字符串；以 `_ids` 结尾的是字符串数组 | spec 测试 | api |
| **G8** | 同名 property 全 spec 内 schema 一致。例外只走具名清单（初版只有 `state`：资源生命周期，各资源各一份封闭枚举）；禁用名不出现 | spec 测试 | api |
| **G9** | 数组 / map 不允许 `null`；没有 `additionalProperties: false`；v1 包里 `omitempty` 只在指针字段上 | spec 测试 + Go AST | api |
| **G14** | 字符串必须有 `enum` / `format` / `pattern` 之一或自由文本声明，且全部有 `maxLength`；数值有 `minimum` | spec 测试 | api |
| **G16** | `request_id` 匹配 `^req_[0-9A-HJKMNP-TV-Z]{26}$`；游标匹配 `^cur_` | spec 测试 + 契约测试 | api · db |
| **G17** | 写面路径参数寻址的对象，在读面 schema 里有 `id` | spec 测试 | api |
| **F1** | 命名规则（[01 §3](01-standard.md)）：布尔前缀、`_at` ↔ date-time、`_date` ↔ date、`_count` 为非负整数、禁用名 | spec 测试 | api |
| **F2** | 注册表的每个 code 与 reason 在 `zh-CN/problem.json` 里都有译文，目录里没有多余键 | vitest，读 `problems.json` | web |
| **F3** | 旧路由数只减不增：`routes.golden` 里 `/api/v1` 以外的路由数 ≤ `legacy_route_baseline` | Go 测试 | api |
| **F4** | `kunFetch` / `useKunFetch` 调用点数 ≤ 基线 | vitest 源码扫描 | web |
| **F5** | 提交的 `v1.d.ts` 等于从提交的 spec 重新生成的结果 | `pnpm -F web gen:api && git diff --exit-code` | web |
| **F6** | 应用代码里不出现以 `/v1/` 开头的字符串字面量：v1 只能经类型化客户端调用 | eslint `no-restricted-syntax` | web |
| **F7** | DB 作业里 `testdb` 不许跳过：设了 `KUN_REQUIRE_TEST_DB=1` 而没有 DSN 就失败，而不是 skip | Go 测试辅助 | db |
| **G12** | oasdiff 对比 master 上的 spec。preview 期只报告，稳定后阻断 | CI | api |

作业：

- **api**：现有 `test.yml` 的 unit 作业，加 spec 重生成与 diff。
- **db**：新作业。起 Postgres 与 Redis 服务容器，用仓里的引导脚本从零建库，`TEST_DATABASE_DSN` 显式指向该容器，`go test -count=1 -p 1 ./...`。之前 DB 测试在 CI 里全部「因缺席而绿」，从这一波起真跑。
- **web**：现有 `web.yml`，加 F2 / F4 / F5 / F6。路径过滤加上 `apps/api/openapi/**`，spec 一变网页门就跑。

本地门禁：`GOTOOLCHAIN=go1.26.1 make lint && GOTOOLCHAIN=go1.26.1 go test ./...`（见 memory `go-1.27-errcheck-broken`），网页 `pnpm -F web lint && pnpm -F web typecheck && pnpm -F web test`。

## §4 演进、退役与 App 兼容

**两个阶段**（同 infra 07 §3.0）：

| 阶段 | 规则 |
|---|---|
| **preview**（现在） | 任何变更都允许，包括删改。但**必须**写进 `docs/proj/api-v1/CHANGELOG.md` 并告知 App 侧。preview 免的是兼容义务，不是告知义务 |
| **稳定**（用户宣布；前提是 App 首个公开版发布） | 只做加法；G12 转阻断；破坏只能走下面的流程 |

**稳定后怎么做不兼容变更**：

1. **先加**：新字段 / 新端点与旧的并存。旧的在 spec 标 `deprecated: true`，响应带 `Deprecation`、`Sunset` 与 `Link rel="deprecation"`。
2. **再迁**：发一个用新形状的 App 版本。
3. **看数据**：按 `User-Agent: kungal-app/<版本>` 统计，旧版本还有多少流量在打旧形状。
4. **收口**：旧版本流量可以接受时，调高 `/api/app/version`（迁移后为 `/api/v1/app/version`）的 `min_version`，旧 App 弹强制更新。
5. **后删**：到 `Sunset` 后旧的回 `410 GONE`。

整体性的大改（换数据模型）按单个资源开新路径，不整体升 `/api/v2`。

**客户端契约三句**（写进 App 仓文档，有约束力）：

> 客户端必须忽略响应中未知的字段。
> 客户端必须容忍开放词表中未见过的取值；遇到未知的正文节点类型，渲染其子节点或纯文本。
> 客户端必须为未知的错误 `code` 准备一个按 `status` 的兜底分支。

## §5 逐端点迁移清单

每条端点（或一组同资源端点）迁移时，下面全部做完才算完：

1. **普查**：
   - 调用方：网页组件、`server/utils` 下的 Nitro 服务端调用、App 文档里的引用；
   - 取值：每个枚举成员对应的数据库行数，写进任务书；
   - 语义：每个字段到底是什么意思，不凭名字猜。
2. **v1 操作**：
   - 声明全部状态码与错误 code；
   - 类型走 v1 的表示层（id 字符串、Image、UserRef、`viewer`）；
   - **不得复用旧 DTO**，映射从 model / service 结果直接出。
3. **服务层**：
   - 旧 handler 与 v1 共用 service；
   - service 返回领域结构而不是某一代的 DTO；
   - 错误改成具名 problem，不再返回 `233`。
4. **网页**：
   - 调用点切到类型化客户端；
   - 新 code 的 `zh-CN` 译文进目录；
   - 手写类型删掉或改成生成物别名。
5. **删旧**：
   - 删旧路由、handler、DTO、手写 TS 类型；
   - `rg` 证明零调用方，含 `server/` 下的 Nitro 调用；
   - `routes.golden` 重生成；`legacy_route_baseline` 下调。
6. **测试**：
   - 契约测试覆盖每个声明的错误 code，至少各一个用例；
   - 分页全量遍历对照：小 `limit` 翻完全部页，与直接 SQL 的排序结果逐条相等，无重复无遗漏；
   - 网页 typecheck、lint、test 全绿。
7. **Claude 验收**：
   - 读 diff；
   - 在 dev 环境实测：匿名、普通用户、Bearer、有权限者各走一遍；
   - 浏览器里走一遍网页调用点；
   - 核对 spec 与普查结论一致。
8. **文档**：
   - App 相关的变化写进 `CHANGELOG.md`；
   - 本目录波次看板更新状态。
