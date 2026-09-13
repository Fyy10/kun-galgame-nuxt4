// The platform account layer, renamed from 鲲 Galgame OAuth on 2026-09-13 when
// the OP moved to account.nextmoe.com. Only the account layer was rebranded:
// this forum keeps its own product name, so a sweep that also renames
// 鲲 Galgame 论坛 is wrong (infra docs/auth/06-nextmoe-rebrand-domain-split.md
// D6). The name lives here because the last rename had to touch six components.
export const nextmoe = {
  brand: 'NextMoe·未萌',
  account: 'NextMoe·未萌 账号',
  admin: 'NextMoe·未萌 管理台',
  logo: '/nextmoe.webp'
} as const
