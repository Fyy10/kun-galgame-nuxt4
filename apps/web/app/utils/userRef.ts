import type { UserRef } from '#shared/utils/api/schemas'

export const toKunUser = (ref: UserRef): KunUser => ({
  id: Number(ref.id),
  name: ref.name ?? '已注销用户',
  avatar: ref.avatar?.url ?? ''
})
