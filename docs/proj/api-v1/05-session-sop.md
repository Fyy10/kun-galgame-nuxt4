# 05 · 独立会话作业手册

> 2026-09-22 立。W0–W5b 是「一个督查 + 若干子代理」串着跑出来的，剩下 300 条旧路由按那个节奏要两三个月。本文把那套流程改写成一份**任何一个新开的会话都能独立执行**的作业手册：自己的 worktree、自己的临时库、自己的 PR。
>
> 读者是一个**没有任何上下文**的会话。读完本文 + [01](01-standard.md) + [02](02-governance.md) + 自己域的普查，就该能把一个域从头做到合并。

## 0. 五句话

1. **一个域 = 一个分支 = 一个 worktree = 一个临时库 = 一个 PR。**
2. **契约先写、变异题先写，实现后写。** 顺序由 git 历史证明，不由自觉保证。
3. **九条闸全绿才开 PR**（§5）。缺一条不算完成，不许"下次补"。
4. **合并 PR 就是上线**（master 一推就构建部署）。合之前确认没有别的 PR 正在合。
5. **共享面不属于任何一条轨**（§4）。要改共享面，要么先单独发一个小 PR 把它改掉，要么停下来问。

## 1. 开工

### 1.1 认领

看板在 [README](README.md#波次看板)。**远端分支就是认领表**，没有别的地方需要登记：

```bash
git ls-remote --heads origin 'api-v1/*'   # 已经有人在做的域
gh pr list --state open                    # 正在等合并的
```

挑一个没人做的域，立刻把空分支推上去占位（这一步花 5 秒，省掉两个会话撞车的两小时）：

```bash
git fetch origin && git branch api-v1/w6-user origin/master
git push -u origin api-v1/w6-user
```

分支名一律 `api-v1/<波次>-<域>`，小写连字符。

### 1.2 worktree

**绝不在主检出里做**（铁律 13）。主检出 `/home/kun/Desktop/code/website/kun-galgame-forum` 永远留在 master 上给验收和部署用。

```bash
git worktree add ../kun-galgame-forum-w6-user api-v1/w6-user
cd ../kun-galgame-forum-w6-user
pnpm install --frozen-lockfile
```

做完合并后自己收摊：`git worktree remove <路径>`，`git branch -d`，`git push origin --delete`。**历史上留下过 21 个死 worktree**，别再加。

### 1.3 自己的临时库

DB 测试**只**能打一个专属的空库。名字带上波次，别人一眼看得出是谁的：

```bash
createdb kungal_test_w6_user
export TEST_DATABASE_DSN='postgres://<user>@127.0.0.1:5432/kungal_test_w6_user?sslmode=disable'
export KUN_REQUIRE_TEST_DB=1
bash apps/api/scripts/testdb-bootstrap.sh    # 把空库建成生产形状
```

铁的三条：

- **绝不**把 `TEST_DATABASE_DSN` 指向开发库或生产库。bootstrap 脚本会拒绝非空的 schema，但那是最后一道防线，不是许可。
- **绝不**从 `.env` 里读 DSN，**绝不**把 DSN 打到输出里。
- **不设 `KUN_REQUIRE_TEST_DB=1`，没有 DSN 时 DB 测试会静默跳过并装作通过。** W4 的收藏和推在这上面翻过车：每一次调用都 500，而测试全绿。

做完删库：`dropdb kungal_test_w6_user`。

## 2. 必读

| 读什么 | 为什么 |
|---|---|
| [01-standard.md](01-standard.md) | 面与凭证、错误与 i18n、表示层、集合、写面、命名表。**这是契约本身** |
| [02-governance.md](02-governance.md) §5 | 逐端点迁移清单，八步，一步不能少 |
| [03-content-doc.md](03-content-doc.md) | 只要你的域有"正文"，就必须读 |
| [04-parallel-tracks.md](04-parallel-tracks.md) | 为什么是这么并行的；共享面清单 |
| `waves/census/` 里自己的域 | 已经做过的普查。**没有就自己做**（§3.1） |
| `waves/w5b-polls.md` | 最近一波的波次文档，照它的样子写自己的 |
| infra `refs/api-v2/` 01·02·04·05·06·07·10 | 上游规范，默认全部适用。在源工作区只读，**不许复制进 worktree** |

## 3. 五步

### 3.1 A · 普查（先于一切）

**不查就写契约 = 把旧 bug 原样搬进 v1。** 四份普查报告在 `waves/census/`，里面已经抓出过：投票/抽奖四个读面完全不查可见性（隐藏话题的投票匿名可读）、评论 target 洞。

对自己域的每一条旧路由，查清楚并写进波次文档：

1. **调用方**：`rg` 网页组件 + `apps/web/server/`（Nitro 服务端调用，最容易漏）+ `../kungal-apps`（Flutter App，文档里那张路由表是前瞻契约，不是事实）。
2. **取值**：每个枚举成员在**生产库**里的行数。没有一行的分支不要建模；只有两行的功能可以排到最后。
   生产库只读：`ssh kungal-neo "sudo -n docker exec -i kun-visual-novel-infra-vqvqbc-postgres-1 psql -U postgres -d kungalgame -X -q"`
3. **语义**：字段到底是什么意思，**不凭名字猜**。`status` 是裸整数、`section` 是数组、`is_nsfw_topic` 与 `is_nsfw` 同义不同名——这些就是立项原因。
4. **权限与可见性**：谁能读、谁能写、隐藏/封禁/NSFW 各走哪条闸。旧读面**普遍不查**，别照抄。
5. **疑似 bug**：照实记，别顺手修——修不修是契约阶段的决定。

### 3.2 B · 契约（提交 #1，必须独立成一个提交）

契约写进 `docs/proj/api-v1/waves/<波次>-<域>.md`，内容：

- 每个 v1 操作：方法、路径、operationId、请求体、响应体、**全部**状态码与错误 code；
- 类型走 v1 表示层（id 是十进制字符串、`Image`、`UserRef`、`viewer` 块、RFC 3339 UTC 秒精度、数组永不为 null）；
- 新增的 K 决定（只增不改，理由写在旁边）；
- 迁移号（从看板分到的号段里取，§5）；
- 删除哪些旧路由，`legacy_route_baseline` 降到多少；
- **变异题 8–12 条**（见 3.3）。

**契约提交里不许有一行实现代码。**

### 3.3 变异题（与契约同一个提交，早于实现）

原本的规矩是「变异题由督查在派发前出，执行者不能给自己出题」——否则测的是执行者自己的理解，不是契约。独立会话既是执行者又是督查，**唯一的替代品是顺序**：

> 变异题必须在写实现之前提交到分支上。PR 里 reviewer 看得见提交顺序。**事后补的变异题一律不算数。**

每条题是一行语义改动，形如「把 X 改成 Y，某个测试必须变红」。三条经验：

- **编译不过的变异不是变异，是废题。** 删掉跳过封禁用户的 `if` 会让变量变成未使用 → 改成恒真/恒假才是有效题。
- **排序测试的数据里必须有并列的排序键。** 每行一个不同的秒，把游标的 `id` 决胜键整个删掉，三个分页测试照样全绿。
- 变异要打在**契约语义**上（少一个权限判断、游标少一个键、错误码换一个、时间精度多三位），不是打在语法上。

实现完成后逐条执行，把「改了什么 / 哪个测试红了」写进 PR 正文。**有一条杀不掉，就是测试不够，补测试，不是改题。**

### 3.4 C · 实现

- 只写**自己域的新文件**：`internal/<域>/apiv1/*`、`internal/<域>/repository/v1_*.go`、`internal/app/v1_<域>_*_test.go`、自己域的网页目录。
- **不得复用旧 DTO**。映射从 model / service 结果直接出。
- 旧 handler 与 v1 **共用 service**；service 返回领域结构，不返回任何一代的 DTO。
- 错误一律具名 problem，不再有 `233`。
- 注释按仓库规矩：**默认不写**。只有真的踩过的坑才留一条，写结论不写机制。

### 3.5 网页

- 调用点切到生成的类型化客户端；手写类型删掉或改成生成物的别名。
- 新错误码的 `zh-CN` 译文进 `apps/web/i18n/locales/zh-CN/problem.json`，**一个都不能少**（`tests/api/problemCatalog.spec.ts` 会抓）。
- 页面根节点只能有一个真实元素，不能有前导注释/空白/兄弟节点（铁律见根 CLAUDE.md #11）。
- UI 一律先找 KunUI（`@kungal/ui-*`），**不改 KunUI 本体**；**没有渐变背景**。

### 3.6 删旧路由

新面上线并实测通过之后，**同一个 PR 里或紧跟一个清理 PR**：

1. 删旧路由、handler、DTO、手写 TS 类型；
2. `rg` 证明零调用方，含 `apps/web/server/` 与 `../kungal-apps`；
3. 死代码：`go run golang.org/x/tools/cmd/deadcode@latest -test ./...`（根集合**必须**含所有包的测试，只用 `./cmd/...` 会误杀只被内部测试用到的函数），删一轮跑一轮直到不动点；
4. `TableName()` 永远被报成不可达（GORM 反射调用），是假阳性——判断模型死活要看**类型**有没有被引用；
5. 删文件前先看里面有没有还活着的东西（`write_moderation_test.go` 当时有一半测的是活路径）；
6. `routes.golden` 重新生成、`legacy_route_baseline` 下调。

删完注意：旧命名空间下未匹配的路径返回 **401 不是 404**（鉴权是 `/api` 上的 `Use()`），这是既有行为；只有 `/api/v1` 回规范的 404 problem+json。

## 4. 能碰什么

| 位置 | 谁的 |
|---|---|
| `internal/<自己的域>/**`、自己域的网页目录、自己的波次文档 | **你的**，随便改 |
| 一个**跨前缀的老 handler**（如 `ResourceCommentHandler` 管 5 个前缀） | **归它整个连通分量的那一轨**。看板按共用 handler 切分，不按 URL 前缀——`/api/admin/topic/*` 是话题轨的，`/api/user/:id/toolsets` 是 toolset 轨的。**共用 handler 要等它服务的所有前缀都迁完才能删**，删早了别的轨当场 404 |
| `pkg/problem/registry.go` + `registry_test.go` + `i18n/locales/zh-CN/problem.json` | **共享**，见下 |
| `internal/app/app.go` 字段、`internal/app/router.go` 挂载点 | **共享**，只加自己的一行，不重排别人的 |
| `internal/apiv1/**`（表示层、`collect` 游标） | **别碰**。要改停下来问——它是所有域的地基 |
| `apps/api/migrations/**` | 只用**分到的号段**里的号 |
| `testdata/routes.golden`、`legacy_route_baseline` | **生成物**，冲突一律重新生成，永不手工合并 |
| `docs/{oauth,image_service,artifact}/` | infra 镜像，**本仓任何人都不改** |
| KunUI（`@kungal/ui-*`） | 上游库，**不改**；有 bug 报给用户 |

**错误码的三处一译**：加一个 code 要同时改 `registry.go`（常量 + `Codes`）、`registry_test.go`（`requiredCodes`）、`problem.json`（zh-CN）。`registry_test` 是**精确计数**，漏一边就红。两条轨同时加 code 会在这三个文件上冲突——**冲突一律两边都留**，永远不要因为冲突丢掉谁的 code。

**迁移号**：W3 和 W4 撞过 098，就是两条轨同时挑号。按看板的号段取，号段用完了再要。

## 5. 九条闸

交付前自己全跑完。**没跑就不算做完**，PR 正文里逐条勾。

| # | 闸 | 命令 |
|---|---|---|
| 1 | lint + vet 零输出 | `cd apps/api && GOTOOLCHAIN=go1.26.1 make lint` |
| 2 | 契约门 G2–G17 / F1–F10 | 随 `go test` 跑 |
| 3 | spec 一致性 + 无漂移 | `go test ./...`（`openapi_test.go` 比对生成物） |
| 4 | DB 测试 | `GOTOOLCHAIN=go1.26.1 go test -count=1 -p 1 ./...`，自己的库，`KUN_REQUIRE_TEST_DB=1` |
| 5 | 分页全量遍历 | 小 `limit` 翻完所有页，与直接 SQL 排序逐条相等，无重无漏，**数据里要有并列的排序键** |
| 6 | 变异全杀 | 逐条执行 3.3 的清单 |
| 7 | 每个声明的错误码至少一个用例 | — |
| 8 | 网页 | `cd apps/api && make openapi` → `pnpm -F web gen:api` → `pnpm lint` → **`pnpm typecheck`** → `pnpm -F web test` |
| 9 | 浏览器实测 | 开发环境真的点一遍，匿名 / 普通用户 / 有权限者各一遍 |

> **`pnpm vue-tsc --noEmit` 在本仓库什么都不检查。** `apps/web/tsconfig.json` 是 `{"references": […], "files": []}`，不带 `-b` 就是编译一个空程序，永远 exit 0——往 `Like.vue` 塞 `const x: number = "not a number"` 它照样过。真闸是 **`pnpm typecheck`**（`vue-tsc -b --force`）。W5a 之前所有"typecheck 通过"都是空转。

> **`GOTOOLCHAIN=go1.26.1` 不是可选项。** 系统 Go 1.27 会让 errcheck 假失败，而且内联不同会让路由 golden 本地绿 CI 红。

> **闸 9 没有替代品。** 类型检查永远不解析模板里的组件；单测的假数据只覆盖你想到的分支。W5b 的「还没有人投票」显示在实时票数旁边，四道自动闸全绿，是人点出来的。开发浏览器会话的建法见记忆 `kungal-dev-browser-signed-in-session`。

## 6. PR

### 6.1 开

```bash
gh pr create --base master --title 'feat(api-v1): move <域> onto /api/v1 (W6)' --body-file /tmp/pr-body.md
```

标题用英文 conventional commit，与提交信息同一风格。正文用仓库的 `.github/pull_request_template.md`（`gh pr create` 不带 `--body*` 时会自动带出来），**九条闸逐条勾，变异清单贴执行结果**。

提交信息与代码注释**全部英文**。提交一律给明确路径：`git commit -- <paths>`，**不在仓库根用 `git add -A`**（会漏掉根 `pnpm-lock.yaml`，也会捎上别人的东西）。

### 6.2 CI

PR 上自动跑三个作业，它们是闸 1/3/4/8 的机械复核：

- `unit (build · vet · errcheck · test)`
- `db (bootstrap · go test)` —— 自带 postgres 服务和干净的 bootstrap
- `web (lint · typecheck · test)` —— 含 `gen:api` 后的 `git diff --exit-code`，生成类型不同步直接红

**CI 绿不等于九条闸过**：5、6、7、9 没有任何自动化能替你跑。

### 6.3 合

- **合并 = 上线。** 推进 master 就构建镜像并 curl Dokploy webhook。`build-and-push` **没有路径过滤**——改一行文档合进 master 也会重建镜像并重新部署，所以文档 PR 一样占用部署档期。
- **一次只合一个。** 合之前 `gh pr list` + `gh run list --branch master --limit 5`，确认上一次部署已经完成。master 的构建 concurrency 是 cancel-in-progress，两个 PR 连着合会让前一个的构建被取消。
- 用 squash 合并，保持 master 线性。
- 合完自己盯部署（§7），不要合了就走。

## 7. 上线后

1. **迁移**：普通迁移随部署自动跑。**deploy-then-drop 类的绝不能随部署跑**——迁移在新容器之前执行，旧二进制会撞上改过的表。四步舞和「DROP COLUMN 之后必须立刻重启 API（pgx 缓存 prepared statement，不重启会一直回 SQLSTATE 0A000）」写在 `cmd/migrate/main.go` 的 exclude 注释里，照着做。
2. **Dokploy webhook 回 2xx ≠ 真的部署了**，这是反复出现的故障。盯容器创建时间，几分钟没重建就手动接管：
   ```
   cd /etc/dokploy/compose/kun-visual-novel-forum-iunwa9/code
   sudo docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9 pull kungal-api web
   sudo docker compose -f docker-compose.prod.yml -p kun-visual-novel-forum-iunwa9 up -d migrate kungal-api web
   ```
3. **实测**：线上打一遍自己的新端点（匿名 + 登录），SSR 拉一遍改过的页面，看 API 日志有没有新错误。
4. **收尾**：更新 [README](README.md) 看板状态、[CHANGELOG](CHANGELOG.md)（App 可见的变化必须写）、波次文档补一节"验收时抓到的"。删 worktree、删分支、删临时库。

## 8. 一条都不许破

- 不改 `docs/{oauth,image_service,artifact}/`，不改 KunUI，不用渐变背景。
- 不读 `.env`，不打印 DSN 或任何密钥。
- DB 测试只打专属的空库，`-count=1 -p 1`。
- 不在主检出里做事；一个任务一个分支一个 worktree。
- 不用 `pkill -f`（会杀到别的会话）。要停进程按 PID。
- `git commit -- <paths>`，不在根 `git add -A`。
- Bearer 请求**永远不持有管理权限**：能力判断用 `user.Can` / `user.CanModerate`，**绝不**用 `perm.CanUser` / `role.Can*` 读 `user.Roles`（`bearer_guard_test.go` 钉着）。
- 改了表结构就**必须**在结尾明确告诉用户：要不要在生产跑迁移、跑哪条命令、打哪个库。

## 9. 照抄清单：反复踩过的坑

- `pnpm vue-tsc --noEmit` 是空转，真闸是 `pnpm typecheck`。
- 没设 `KUN_REQUIRE_TEST_DB=1`，DB 测试静默跳过并装作通过。
- 系统 Go 1.27 让 errcheck 假失败、让路由 golden 本地绿 CI 红。
- `repr.DateTime` 是**定长 20 字符秒精度**，`toISOString()` 的毫秒会 422。网页侧统一走 `components/topic/miniapp/deadline.ts`。
- `KunDatePicker` 只吐 `yyyy-MM-dd`，撞上要求完整时间戳的字段就是"点了没反应"。
- `viewer` 对**匿名访问者恒为 null**。凡是匿名用户也合法拥有的能力，就**不能**表达成 viewer 标志——加了反而会把按钮藏掉。
- 服务端对所有匿名投票下发空 `sample_voters`：「空数组 ⇒ 没有人」是假命题，只有计数字段能下这个判断。
- 审计/普查给的"调用方清单"不可尽信：它通常只 grep 了 `apps/web/**`，看不见 Nitro 服务端调用，也看不见 Flutter App。
- `deadcode` 把 `TableName()` 全报成不可达（GORM 反射），假阳性。
- 旧读面**普遍不查可见性**。照抄旧逻辑 = 把信息泄漏搬进 v1。
- 上游 `/users/batch` 不可用时写面失败关闭回 503（K17）——这是契约，不是 bug。
