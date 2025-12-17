import { prisma } from '../prisma/prisma'

const DAY = 1000 * 60 * 60 * 24

function predictResourceView(params: {
  gameView: number
  gameCreated: Date
  resourceDownload: number
  resourceCreated: Date
}) {
  const now = new Date()

  const DAY = 1000 * 60 * 60 * 24
  const gameActiveDays = Math.max(
    (now.getTime() - params.gameCreated.getTime()) / DAY,
    1
  )
  const resourceActiveDays = Math.max(
    (now.getTime() - params.resourceCreated.getTime()) / DAY,
    1
  )

  const gameViewRate = params.gameView / gameActiveDays
  const downloadRate = params.resourceDownload / resourceActiveDays

  const MAX_RATIO = 0.8
  const DOWNLOAD_VIEW_RATIO = 3
  const DOWNLOAD_RATE_WEIGHT = 0.5
  const GAME_EXPOSE_WEIGHT = 0.2

  let rawView =
    params.resourceDownload * DOWNLOAD_VIEW_RATIO +
    downloadRate * resourceActiveDays * DOWNLOAD_RATE_WEIGHT +
    gameViewRate * resourceActiveDays * GAME_EXPOSE_WEIGHT

  if (params.resourceDownload === 0) {
    rawView = gameViewRate * resourceActiveDays * 0.1
  }

  let finalView = Math.min(params.gameView * MAX_RATIO, rawView)

  finalView = Math.max(finalView, params.resourceDownload)

  finalView = Math.min(finalView, Math.max(params.gameView, finalView))

  return Math.floor(finalView)
}

async function main() {
  console.log('🚀 开始预测 galgame_resource.view')

  const games = await prisma.galgame.findMany({
    include: {
      resource: true
    }
  })

  let updateCount = 0

  for (const game of games) {
    if (game.view <= 0) continue

    for (const res of game.resource) {
      // if (res.view) continue

      const view = predictResourceView({
        gameView: game.view,
        gameCreated: game.created,
        resourceDownload: res.download,
        resourceCreated: res.created
      })

      await prisma.galgame_resource.update({
        where: { id: res.id },
        data: { view }
      })

      updateCount++
    }
  }

  console.log(`✅ 预测完成，共更新 ${updateCount} 条 galgame_resource`)
}

main()
  .catch((e) => {
    console.error('❌ 执行失败:', e)
    process.exit(1)
  })
  .finally(async () => {
    await prisma.$disconnect()
  })
