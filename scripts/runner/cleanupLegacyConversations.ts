import fs from 'node:fs'
import path from 'node:path'
import { prisma } from '../../prisma/prisma'
import { collectTopicReplyCascadeIds } from '../../server/utils/topicReply'

const LOG_PATH = path.resolve('deleted.log')
const logStream = fs.createWriteStream(LOG_PATH, { flags: 'w' })

const log = (message: string) => {
  const line = `[${new Date().toISOString()}] ${message}`
  console.log(line)
  logStream.write(`${line}\n`)
}

const closeLogStream = async () =>
  new Promise<void>((resolve, reject) => {
    logStream.end((err) => {
      if (err) {
        reject(err)
      } else {
        resolve()
      }
    })
  })

const describeParticipants = (
  participants: Array<{
    user_id: number
    user: { id: number; name: string } | null
  }>
) => {
  if (!participants.length) {
    return 'none'
  }
  return participants
    .map((p) => `${p.user_id}:${p.user?.name ?? 'unknown'}`)
    .join(', ')
}

const previewContent = (content: string | null) => {
  const normalized = (content ?? '').replace(/\s+/g, ' ').trim()
  if (!normalized) {
    return '<empty>'
  }
  return normalized.length > 80 ? `${normalized.slice(0, 77)}...` : normalized
}

const cleanupChatRooms = async () => {
  const candidateRooms = await prisma.chat_room.findMany({
    where: {
      OR: [{ type: 'private' }, { participant: { none: {} } }]
    },
    select: {
      id: true,
      name: true,
      type: true,
      created: true,
      participant: {
        select: {
          user_id: true,
          user: { select: { id: true, name: true } }
        }
      },
      _count: {
        select: {
          participant: true,
          message: true
        }
      }
    }
  })

  const roomsToDelete = candidateRooms.filter(
    (room) =>
      room._count.participant === 0 ||
      (room.type === 'private' && room._count.participant < 2)
  )

  if (!roomsToDelete.length) {
    log('No orphan chat rooms detected.')
    return 0
  }

  for (const room of roomsToDelete) {
    log(
      `[ChatRoom] Removing id=${room.id} name=${room.name} type=${room.type} participants=${describeParticipants(
        room.participant
      )} messageCount=${room._count.message}`
    )
  }

  await prisma.chat_room.deleteMany({
    where: {
      id: { in: roomsToDelete.map((room) => room.id) }
    }
  })

  log(`Deleted ${roomsToDelete.length} chat rooms.`)
  return roomsToDelete.length
}

const cleanupTopicReplies = async () => {
  const rows = await prisma.$queryRaw<Array<{ id: number }>>`
    SELECT id
    FROM topic_reply
    WHERE btrim(content) = ''
      AND NOT EXISTS (
        SELECT 1 FROM topic_reply_target tr WHERE tr.reply_id = topic_reply.id
      )
  `
  const rootReplyIds = Array.from(new Set(rows.map((row) => row.id)))

  if (!rootReplyIds.length) {
    log('No empty standalone topic replies detected.')
    return 0
  }

  const idsToDelete = await collectTopicReplyCascadeIds(rootReplyIds, prisma)
  if (!idsToDelete.size) {
    log('Cascade calculation produced no replies to delete.')
    return 0
  }

  const idsArray = Array.from(idsToDelete)
  const replies = await prisma.topic_reply.findMany({
    where: {
      id: { in: idsArray }
    },
    select: {
      id: true,
      topic_id: true,
      user_id: true,
      floor: true,
      created: true,
      content: true,
      user: { select: { id: true, name: true } },
      topic: { select: { id: true, title: true } }
    }
  })

  replies
    .sort((a, b) => a.id - b.id)
    .forEach((reply) => {
      log(
        `[TopicReply] Removing id=${reply.id} topic=${reply.topic?.id ?? reply.topic_id}:${reply.topic?.title ?? 'unknown'} user=${reply.user?.id ?? reply.user_id}:${
          reply.user?.name ?? 'unknown'
        } floor=${reply.floor} created=${reply.created.toISOString()} content=${previewContent(
          reply.content
        )}`
      )
    })

  await prisma.topic_reply.deleteMany({
    where: {
      id: { in: idsArray }
    }
  })

  log(
    `Deleted ${idsArray.length} topic replies (root replies: ${rootReplyIds.length}).`
  )

  return idsArray.length
}

const run = async () => {
  try {
    log('Starting legacy cleanup runner...')
    const deletedRooms = await cleanupChatRooms()
    const deletedReplies = await cleanupTopicReplies()
    log(
      `Cleanup completed. chatRoomsDeleted=${deletedRooms}, topicRepliesDeleted=${deletedReplies}`
    )
  } catch (error) {
    log(
      `Cleanup failed: ${
        error instanceof Error ? (error.stack ?? error.message) : String(error)
      }`
    )
    process.exitCode = 1
  } finally {
    await prisma.$disconnect()
    await closeLogStream()
  }
}

run()
