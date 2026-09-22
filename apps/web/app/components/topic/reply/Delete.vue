<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  reply: Reply
}>()

const tempReplyStore = useTempReplyStore()
const { id, moemoepoint } = usePersistUserStore()
const api = useApiClient()

const isAuthor = computed(() => String(id) === props.reply.author.id)

const handleDeleteReply = async () => {
  const moemoepointToDecrease =
    3 * (props.reply.comments.length + props.reply.like_count + 1)

  if (moemoepoint < moemoepointToDecrease && isAuthor.value) {
    useMessage(
      `您的萌萌点不足, 删除这个回复将会消耗您 ${moemoepointToDecrease} 萌萌点。删除消耗萌萌点计算公式为 3 × (回复下评论数 + 回复被点赞数 + 1)`,
      'warn'
    )
    return
  }

  const res = await useComponentMessageStore().alert(
    isAuthor.value
      ? '你这个坏萝莉, 确定删除这个回复吗?'
      : '你好萝莉管理员, 要删除这个回复吗',
    isAuthor.value
      ? `删除这个回复将会消耗 ${moemoepointToDecrease} 萌萌点, 严重注意, 删除操作不可撤销！删除消耗萌萌点计算公式为 3 × (回复下评论数 + 回复被点赞数 + 1)`
      : '删除这个回复将会消耗发布回复者 3 萌萌点, 该操作不可撤销'
  )
  if (!res) {
    return
  }

  const result = await settle(
    api.DELETE('/replies/{reply_id}', {
      params: { path: { reply_id: props.reply.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  tempReplyStore.setSuccessfulReply({
    data: { id: props.reply.id },
    type: 'deleted'
  })
  useMessage('删除回复成功', 'success')
}
</script>

<template>
  <KunButton
    v-if="reply.viewer?.can_delete"
    variant="light"
    color="danger"
    size="sm"
    @click="handleDeleteReply"
    class-name="whitespace-nowrap gap-2 justify-start"
  >
    <KunIcon class-name="text-lg" name="lucide:trash-2" />
    删除回复
  </KunButton>
</template>
