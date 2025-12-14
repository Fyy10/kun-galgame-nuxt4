import { randomNum } from './random'
import { kungal } from '../../app/config/kungal'

const stickerCache: Record<string, string> = {}

export const getRandomSticker = (id: string) => {
  const key = `random-sticker-${id}`

  if (!stickerCache[key]) {
    const randomPackIndex = randomNum(1, 5)
    const randomStickerIndex = randomNum(1, 80)
    stickerCache[key] =
      `${kungal.domain.sticker}/stickers/KUNgal${randomPackIndex}/${randomStickerIndex}.webp`
  }

  return stickerCache[key]
}
