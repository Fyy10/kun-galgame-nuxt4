# 03 · 正文文档（W1）

> K13 的完整规格。2026-09-18 定稿。节点类型的唯一来源是 Go 代码 `apps/api/internal/apiv1/content/node.go`，本文解释它为什么长这样、服务端怎么从 Markdown 产出它、客户端怎么渲染它。

## §1 形状

话题、回复（以后还有评论）的正文字段叫 `content`，值是 `ContentDocument`：

```json
{
  "object": "document",
  "children": [
    { "object": "heading", "depth": 2, "anchor": "简介", "children": [{ "object": "text", "value": "简介" }] },
    { "object": "paragraph", "children": [
      { "object": "mention", "mentioned_user": { "object": "user", "id": "3", "name": "鲲", "avatar": null } },
      { "object": "text", "value": " 说得对 " },
      { "object": "reply_reference", "reply_id": "48", "floor": 2 },
      { "object": "break" },
      { "object": "image", "url": "https://…/9f3c…_320.webp", "alt": "", "image": { "url": "https://…/9f3c….webp", "hash": "9f3c…", "width": 800, "height": 600, "thumbhash": "…", "sexual": "safe" } }
    ] }
  ]
}
```

- **判别字段是 `object`**，与 v1 其余对象一致。不用 mdast 的 `type`：Problem 的 `type` 是 URI 字符串，同名不同型会被 G8 拦下（与资源生命周期叫 `state` 不叫 `status` 同理）。
- **节点名取 mdast，写成 snake_case**（F1 要求封闭枚举值是 snake_case）；唯一改名是 mdast 的 `delete` → `strikethrough`（GFM 规范的叫法，`delete` 读起来像动作）。块级与行内同族的，块级用裸名、行内加 `inline_` 前缀，与 mdast 的 `code` / `inlineCode`、`math` / `inlineMath` 同一规律：`spoiler` / `inline_spoiler`。
- **两个联合**：`BlockNode`（9 种）与 `InlineNode`（13 种），spec 里是 `oneOf` + `discriminator`。段落只能装行内节点、引用块只能装块级节点，类型系统替客户端守住这一点（`content.typetest.ts` 钉住收窄）。`list_item`、`table_row`、`table_cell` 只出现在各自的父节点下，不进联合。
- **`children` 进 G8 例外清单**：同名属性的元素类型随父节点而变，与 `items` 随列表而变同理。
- 组件名带 `Node` 后缀（`ImageNode`），因为 `Image` 已是 `repr.Image`，huma 遇到重名会 panic。
- `mention` 里的人按角色命名为 `mentioned_user`（`user` 是禁用名）。

| 块级 `object` | 字段 | 说明 |
|---|---|---|
| `paragraph` | `children: InlineNode[]` | 空数组 = 作者保留的空行（见 §3.5） |
| `heading` | `depth` 2–6、`anchor`、`children` | 源里的一级标题下发为 2：页面标题是唯一的一级 |
| `thematic_break` | — | |
| `blockquote` | `children: BlockNode[]` | |
| `list` | `is_ordered`、`start`（无序为 `null`）、`is_spread`、`children: ListItemNode[]` | |
| `list_item` | `is_checked`（非任务项为 `null`）、`children: BlockNode[]` | |
| `code` | `lang`（小写；无则 `null`）、`value` | |
| `math` | `value`（TeX，不含定界符） | |
| `table` | `children: TableRowNode[]` | **第一行是表头**（GFM 表格恰好一行表头，同 mdast） |
| `table_row` / `table_cell` | `table_cell.align`：`left` / `center` / `right` / `null` | 对齐放在单元格上，客户端不必按列号回查 |
| `spoiler` | `children: BlockNode[]` | |

| 行内 `object` | 字段 | 说明 |
|---|---|---|
| `text` | `value` | 已解码的纯文本，按文本渲染，绝不当标记 |
| `emphasis` / `strong` / `strikethrough` | `children` | |
| `inline_code` | `value` | |
| `inline_math` | `value` | |
| `break` | — | 硬换行 |
| `link` | `url`、`children` | `url` 是绝对的 http / https / mailto |
| `image` | `url`、`alt`、`image: Image \| null` | `url` 是行内显示用的地址（可能是 `_320` 这类变体）；`image` 是图床记录，其 `url` 是原图，灯箱用它；站外图片为 `null` |
| `video` | `url` | |
| `inline_spoiler` | `children` | |
| `mention` | `mentioned_user: UserRef` | 显示为 `@name`；`name` 为 `null` 时客户端出本地化的「已注销用户」 |
| `reply_reference` | `reply_id`、`floor` | 显示为 `#floor` |

**客户端遇到未知节点**：有 `children` 就渲染子节点，有字符串 `value` 就当文本渲染，都没有就跳过（spec 的 `info.description` 与联合的 `description` 都写了）。新增节点类型是加法。

**过渡**：W1 没有任何操作返回正文，而 huma 生成 spec 时会删掉没被引用的组件，所以 `content.Register` 在文档根上挂了 `x-content-document` 引用它，节点词表才进得了提交的契约、网页才生成得出类型。W2 的话题详情引用它之后，删掉这个扩展与 `Register`。

## §2 普查（2026-09-18，生产库全量）

话题 3542 篇、回复 14785 条，共 18327 份正文；最长 76605 字符（话题上限 100007，回复 10007）。用论坛自己的 goldmark 配置逐份解析，按 AST 节点计数（节点次数 / 文档数）：

| 结构 | 次数 | 文档 | 裁决 |
|---|---|---|---|
| `<br />` 独占一行（HTML 块） | 1666 | 373 | kun-editor（Milkdown）把空段落序列化成 `<br />`，读回来就是空段落。下发空 `paragraph` |
| 行内 `<br>`（表格单元格 1129 / 段落 49） | 1178 | 12 | `break`。GFM 表格单元格里没法换行，只能写 `<br>` |
| 其它行内 HTML 标签 | 26 | 9 | 标签丢弃、文字保留；不是 HTML 元素名的「标签」（`<bucket>`、`<CID>`、`<Slice>`、`<key>`）按原文当文字——现行清洗把它们连同文字一起吞了 |
| `<img>` HTML 块 | 1 | 1 | 转成 `image` |
| 其它 HTML 块（`<p>…<br>…</p>`） | 1 | 1 | 去标签成一段 |
| 图片：图床 token | 6735 | 2473 | `image`，带 `Image` |
| 图片：`_320` 变体 token（贴纸） | 1161 | 950 | `url` 是变体，`image.url` 是原图 |
| 图片：站外 https / http | 690 | 208 | `image`，`image: null` |
| 图片：相对路径 / `data:` / 空 / 协议相对 | 18 | 8 | 协议相对补 `https:`；其余不可用，有 `alt` 就下发为文字 |
| 图片 title | 4996 | 2392 | **不下发**：全部是上传文件名（`银发教主-1765362598054-…png`），只是悬停提示 |
| 链接 `kungal-user:` | 5396 | 5311 | `mention` |
| 链接 `kungal-reply:` | 5369 | 5287 | `reply_reference`，楼层取自链接文字 `#N` |
| 链接 https / http | 2275 | 870 | `link` |
| 自动链接（URL / 邮箱） | 3008 / 7 | 1644 / 4 | `link`（邮箱为 `mailto:`） |
| 链接 `#锚点` | 15 | 1 | 解包成文字：一篇手写目录。客户端从 `heading` 自己算目录 |
| 链接：无 scheme（`kungal.com`）、`vscode-file:`、`view-source:`、`hhttps` 等 | 22 | 18 | 解包成文字：现行渲染出来就是坏链接 |
| 链接 `/image/<hash>` | 3 | 3 | 解析成图床 URL |
| 链接 title | 75 | 21 | 不下发（悬停提示） |
| `kv:` 视频 | 3 | 3 | `video` |
| 行内剧透 / 块剧透 | 215 / 0 | 72 / 0 | 块剧透编辑器能产出，照样支持 |
| 数学 行内 / 块 | 138 / 34 | 8 / 4 | `inline_math` / `math` |
| 标题 h1 / h2 / h3 / h4 / h5 / h6 | 457 / 1332 / 1269 / 392 / 80 / 102 | | h1 下发为 2 |
| 硬换行 / 软换行 | 10972 / 1990 | 1006 / 309 | `break` / 见 §3.3 |
| 列表（松散 414 个） | 1388 | 393 | `is_spread` |
| 表格（对齐 left 313、center 50、无 122） | 143 | 49 | |
| 任务项 | 5 | 2 | `is_checked` |
| 围栏代码块 / 缩进代码块 | 1264 / 2 | 296 / 2 | info 串只取第一个词（`typescript {2} showLineNumbers` 出现 2 次） |
| 嵌套引用 | 61 | 17 | |

## §3 服务端：Markdown → 文档

转换在读时做（与现行 HTML 管线相同，库里只存 Markdown）。实现是 `content.Converter`：一次调用转换一批正文，图片元数据与用户各只查一次（W2 的回复列表一页 50 条就是一批）。

### 3.1 管线

1. 预处理与现行完全一致：`NormalizeStoredContent`（旧贴纸 URL 与图床绝对 URL 收成 `/image/<hash>` token）。
2. 用与现行 HTML 管线**同一套**解析器配置（GFM、mathjax、剧透扩展、标题 id 生成器）解析，但不挂图片元数据的 AST 变换器（它每份正文单独查一次图床）。
3. 收集全部图片 hash 与被 @ 的用户 id，各批量查一次：
   - 用户查询失败 → 整个调用失败，调用方回 `503`（与话题列表的作者查询同一规则）；查不到的用户 → `DeletedUserRef`（`name: null`）。
   - 图片元数据查询失败 → 降级：`Image` 照发，宽高 / thumbhash / `sexual` 为 `null`，记日志。元数据只是增强，不值得让整页 503。
4. 遍历 AST 产出节点。

### 3.2 文本

- `value` 是最终文本：反斜杠转义与 HTML 实体都已解码。
- 相邻文本节点合并；不产出空文本节点。

### 3.3 换行

- 硬换行（行尾两个空格或反斜杠，以及 `<br>`）→ `break`。
- **软换行由服务端处理成空白，客户端不再处理**：两侧字符都是东亚宽字符（East Asian Width 为 W / F）且都不是谚文时删除，否则变成一个空格。这是 CSS Text 3 的分段换行变换规则。现行网页把它渲染成一个空格，在中文之间看得见（Chrome 如此，Firefox 已按该规则删除）。App 的文本组件会把 `\n` 画成真换行，所以不能把 `\n` 原样交给客户端。

### 3.4 标题

- `depth` = 源级别，1 变 2。
- `anchor` 用现行 HTML 管线同一个生成器（Unicode slug，重名加 `-1`、`-2`；空文本为 `heading-N`），所以旧链接里的 `#软件线程` 仍指向同一个标题。

### 3.5 HTML

v1 不下发 `html` 节点。规则按 §2 的普查定：

- **只由 `<br>` 组成的 HTML 块** → 每个 `<br>` 一个空 `paragraph`（Milkdown 的空段落）。
- 行内 `<br>` → `break`；`<img src>` → 按 §3.6 转成 `image`（`alt` 取属性）。
- 其它 **HTML 元素**的标签（`golang.org/x/net/html/atom` 认得的名字）丢弃，标签之间的文字照常保留；HTML 注释丢弃。
- **不是 HTML 元素名**的尖括号词按原文作为文字。
- 其它 HTML 块 → 用 HTML tokenizer 拆成一段：文字、`<br>` → `break`、`<img>` → `image`，块级元素的结束标签与 `<hr>` 变成 `break`（首尾的去掉），`<script>` / `<style>` 的内容丢弃。

### 3.6 链接与图片

| 源 | 产出 |
|---|---|
| `[…](kungal-user:N)` | `mention`（链接文字不用，名字取当前的） |
| `[#N](kungal-reply:M)` | `reply_reference{reply_id: M, floor: N}`；文字不是 `#?数字` 或 id 非法 → 解包成文字 |
| http / https / mailto | `link`，URL 经 `net/url` 规范化（非 ASCII 百分号编码），必须通过 `format: uri` |
| `/image/<hash>[_variant]` | 链接：图床 URL；图片：`image` 带 `Image` |
| 其它以 `/` 开头 | 拼上站点域名 |
| `//host/…` | 补 `https:` |
| 其余（无 scheme、`#锚点`、未知 scheme、超过 2048 字符） | 链接解包成子节点；图片有 `alt` 就下发为文字，否则什么都不出 |
| 文字以 `kv:` 结尾、紧跟 URL 以 `.mp4` 结尾的 http(s) 链接 | `video`，去掉 `kv:` |

- `alt` 超过 512 个字符时按字符截断。
- 图片与链接的 `title` 不下发（§2）。
- 只剩不可用图片而变空的段落不下发。空段落只来自 §3.5 的 `<br>` 块。

### 3.7 其余

- 紧凑列表的 `TextBlock` 与松散列表的 `Paragraph` 都是 `paragraph`，区别由 `list.is_spread` 表达。
- 任务项的复选框变成 `list_item.is_checked`，不留节点。
- 代码块 `value` 不含末尾换行；`lang` 取 info 串第一个词并小写，缩进代码块为 `null`。
- 数学的 `value` 是定界符之间的 TeX 原文。

## §4 客户端渲染

- **网页**：`ContentDocument` 组件用渲染函数递归渲染，**不用 `v-html`**，正文 XSS 在结构上不存在。标记与现行服务端 HTML 逐一对齐（`.kun-code-container`、`.kun-table-container`、`a.kun-mention[data-uid]`、`span.kun-quote[data-reply-id][data-floor]`、`.kun-spoiler.kun-spoiler-hidden`、`img[data-thumbhash]`……）。这样三样东西不用改：
  - 现有的 prose 样式；
  - 按事件委托挂在这些类名上的交互（@ 跳转、楼层预览）；
  - KunUI 导出的 `useSpoilerContent` / `useContentLightbox` / `useContentBlurUp`。KunUI 注明它们就是给「自己写正文渲染器的应用」用的，不需要改 KunUI。

  `KunContent` 的剧透样式写在它自己的 scoped style 里，渲染器照抄一份（必要的重复）。KaTeX 由渲染器调用。
- **App**：原生渲染，规则同上表。

## §5 测试

- **差分测试**：对生产普查语料逐份比较，新文档的纯文本与现行 HTML 管线输出的纯文本，去掉全部空白后必须相等。已知的、按本文裁决产生的差异逐类列为例外并计数：
  - `kv:` 前缀；
  - 非元素尖括号词；
  - 解包的坏链接不受影响（文字相同）；
  - 不可用图片的 `alt`；
  - 用户名换成当前名字。

  语料在会话 scratchpad，不进仓库。
- **契约**：每份语料的输出都用提交的 spec 校验（`ContentDocument` 的 JSON Schema）。
- **模糊测试**：任意输入不 panic、输出通过 schema。
- **网页**：每种节点一个用例，断言精确的 HTML；未知节点三种退路各一个用例。
- **验收交叉**：Go 把语料转成 JSON，网页渲染器 SSR 出 HTML，再与现行 HTML 做同样的纯文本比较。
