# App 聚合 API 设计备忘（2026-07-01）

> **已被取代（2026-09-17）**：App 直接复用论坛现有 `/api/*` 与 `{code,message,data}` 信封，
> Bearer 鉴权已落地。现行契约见 [app-direct-api.md](./app-direct-api.md)。
> 下文另起 `/app/v1` + problem+json 的方向**不再采用**：App 消费的就是论坛现有面，
> 另起一套会让同一数据两种信封并存；将来真出现负载分叉，归宿是 mobile BFF，而不是平行 API 面。
> 以下内容仅作历史记录保留。

> 本仓自有工程笔记（**非** infra 镜像）。记录「鲲 Galgame」Flutter App / 开发者平台
> 对外 API 的架构决策，便于依赖就绪后直接开工。
> **当前状态：方向与工具链已确认可行，落地 gated 于身份层的第三方 token 鉴权。**

## 目标

一个 **Flutter 移动端 App**，聚合整个鲲 Galgame 生态的全部内容：论坛（帖子 / 本地
galgame / 工具集）、wiki（galgame 目录）、patch/moyu（资源）、身份与萌萌点、图床。
本质是一个**跨服务聚合客户端**。同一套契约未来也服务第三方（开发者平台）。

## 已确认的事实：Huma 支持 Fiber v3（2026-07 复核）

论坛后端已迁移至 **Fiber v3.3.0**。code-first OpenAPI 的平台标准是 Huma（身份层的
artifact / admin 面已采用），此前的顾虑是 Huma 的 fiber 适配器不支持 v3 —— 已作废：

- 上游 issue「Support for fiber v3」于 **2026-04-02 CLOSED / COMPLETED**。
- 当前 `humafiber` 适配器 import `fiber/v3`，`New(*fiber.App, huma.Config)` 收 v3 App。
- **实测**：`huma v2.38.0` + `humafiber.New`（v3）对 `fiber v3.3.0` 编译通过
  （一次性 throwaway module 验证；可复现）。
- 注意：humafiber 包同时含 v2/v3 两个适配文件，引入它会**顺带把 `fiber/v2` 作为
  indirect 依赖**拉进 go.mod（纯间接，无功能影响，仅多一行）。

结论：**在论坛现有 v3 栈上跑 code-first Huma 今天就可行**，与平台既有的 Huma 模式一致
（DTO 单源 → 生成 OpenAPI → 生成前端/客户端类型 + `git diff` 漂移门）。

## 核心决策：复用 service 层，不复用 BFF 的 HTTP 契约

论坛现有的 `/api/*` 是给 Nuxt SSR 网页设计的 **BFF**，直接给 Flutter 复用**不合适**：

1. **鉴权对不上**：BFF = 不透明 `kungal_session` cookie + Redis 里的 OAuth token
   （见 `internal/middleware/auth.go`）。移动端走 OAuth PKCE → Bearer token。App 直连
   需要 API 长出 **Bearer/token 鉴权**路径，即身份层的 RS256 + JWKS（见「依赖」）。
2. **信封与形状**：`{code,message,data}`、成功恒 HTTP 200（见 `pkg/response/response.go`）；
   且 BFF 端点按网页页面塑形，常为一屏过量取数，非移动端最优。
3. **只覆盖论坛一片**：App 还需 wiki / moyu / 身份，复用论坛 API 解决不了聚合。
4. **无类型化客户端**：对着无类型 JSON 手写 Dart model，重蹈网页端手写 `shared/types`
   漂移的覆辙。

**该复用的是业务逻辑，不是传输契约。** 论坛的 `internal/*/service` + 已持有的
wiki / oauth / image / artifact 客户端，是可复用资产。

### 要建的东西

一套**全新的、版本化的、OpenAPI 原生的聚合 API**：

- code-first **Huma-on-Fiber-v3** 定义（已验证可行）；
- **token 鉴权**（OAuth Bearer / RS256-JWKS），**地道 REST**（HTTP 状态码 + 类型化 body
  / problem+json），**不沿用** `{code,message,data}` 旧信封；
- 对内 **fan-out 到生态各服务**，复用论坛现有 service；
- 由 OpenAPI spec 用 `openapi-generator`（dart-dio 臂）**自动生成 Flutter 的类型化
  Dart 客户端**；
- 同一份契约同时服务 App、（未来）网页、第三方。

网页 BFF 的 `{code,message,data}` 契约**保持不动** —— 新面与旧 BFF 并存，不改前端。

## 聚合层放哪（待拍板）

- **(a) 放论坛 Go 应用里的新命名空间**（如 Huma-served `/app/v1`），复用论坛 service +
  现有跨服务客户端。**重复最少** —— 论坛已握着最多聚合胶水（wiki 代理 + 本地 galgame +
  帖子）。代价：moyu 内容需论坛新增一个 moyu 客户端。**倾向此项。**
- (b) 独立 **app-gateway 服务**：隔离更干净，但多一套运维、要重建论坛已有的客户端。

除非希望 App 后端与论坛彻底解耦，否则选 (a)。接口面清晰时再最终定。

## 依赖与排序（落地前置）

- **硬前置 · 第三方 token 鉴权**：身份层需对**程序化客户端**开放基于 token 的鉴权
  （RS256 签名 + `GET /oauth/jwks` + `.well-known/openid-configuration`）。移动/公开
  API **不能**用网页 BFF 的 cookie。**这是开工的门槛。**
- **受益 · wiki 读端点 OpenAPI 化**：wiki 若为其读端点导出 OpenAPI spec，论坛既能把
  手写的 wiki 类型换成生成的，聚合层也能据此对齐。
- 因此：**架构与工具链已就绪，落地排在身份层第三方鉴权（+ wiki OpenAPI）之后。**

## 就绪后的第一步（pilot）

拿一个 App 首先会用的**只读端点**（如 galgame 详情，按地道 REST 重塑），用 Huma 定义，
生成 OpenAPI + Dart 客户端 + `git diff --exit-code` 漂移门，跑通工具链；验证后再按 App
的取数需求逐面扩展。**不做大爆炸式改造。**

## 非目标

- 不 retrofit 现有 ~399 个 BFF handler 到 OpenAPI —— 信封与 cookie 鉴权决定了它形状不对。
- 不改动网页前端消费的 `{code,message,data}` 契约。
