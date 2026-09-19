import type { UserRef } from '#shared/utils/api/schemas'
import { deletedUserName } from '#shared/utils/deletedUser'

export { deletedUserName }

export const toKunUser = (ref: UserRef): KunUser => ({
  id: Number(ref.id),
  name: ref.name ?? deletedUserName,
  avatar: ref.avatar?.url ?? ''
})

export const toKunUserWithPoints = (
  ref: UserRef,
  moemoepoint: number
): KunUser & { moemoepoint: number } => ({
  ...toKunUser(ref),
  moemoepoint
})
