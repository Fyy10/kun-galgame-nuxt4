# 普查报告

车道 A（只读子代理）的产物，每域一份，是车道 B 裁决的输入。写于 2026-09-22。

这些报告是**代码普查**：端点、字段、错误、权限、可见性、调用方、命名问题、疑似 bug，全部带 `file:line`。生产库的取值由督查另行补，每份报告末尾列着要跑的 SQL。

| 文件 | 域 | 端点数 |
|---|---|---|
| [comments.md](comments.md) | 话题评论（写面 + 定位；读面已随 W2 迁走） | 5 |
| [polls-lottery-drafts.md](polls-lottery-drafts.md) | 投票 / 抽奖 / 草稿 | 21 |
| [user.md](user.md) | `/api/user` + `/api/perm` | 24 |
| [message.md](message.md) | 消息 / 通知 / 私信 | 14 |

报告里的结论**尚未裁决**，也尚未全部复核。督查已经逐条核实过的见各自的波次文档。
