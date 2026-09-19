<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'

const props = defineProps<{
  reply: Reply
}>()

const tempReplyStore = useTempReplyStore()
const { id, moemoepoint } = usePersistUserStore()
const canDeleteAnyReply = useCan('reply.delete_any')

const isCommonUser = !canDeleteAnyReply.value
const isDisabled = computed(
  () => id !== Number(props.reply.author.id) && isCommonUser
)

const handleDeleteReply = async () => {
  const moemoepointToDecrease =
    3 * (props.reply.comments.length + props.reply.like_count + 1)

  if (moemoepoint < moemoepointToDecrease && isCommonUser) {
    useMessage(
      `您的萌萌点不足, 删除这个回复将会消耗您 ${moemoepointToDecrease} 萌萌点。删除消耗萌萌点计算公式为 3 × (回复下评论数 + 回复被点赞数 + 1)`,
      'warn'
    )
    return
  }

  const res = await useComponentMessageStore().alert(
    isCommonUser
      ? '你这个坏萝莉, 确定删除这个回复吗?'
      : '你好萝莉管理员, 要删除这个回复吗',
    isCommonUser
      ? `删除这个回复将会消耗 ${moemoepointToDecrease} 萌萌点, 严重注意, 删除操作不可撤销！删除消耗萌萌点计算公式为 3 × (回复下评论数 + 回复被点赞数 + 1)`
      : '删除这个回复将会消耗发布回复者 3 萌萌点, 该操作不可撤销'
  )
  if (!res) {
    return
  }

  const result = await kunFetch<string>(
    `/topic/${Number(props.reply.topic_id)}/reply`,
    {
      method: 'DELETE',
      query: { replyId: Number(props.reply.id) }
    }
  )

  if (result) {
    tempReplyStore.setSuccessfulReply({
      data: { id: props.reply.id, floor: props.reply.floor },
      type: 'deleted'
    })
    useMessage('删除回复成功', 'success')
  }
}
</script>

<template>
  <KunButton
    variant="light"
    color="danger"
    size="sm"
    @click="handleDeleteReply"
    class-name="whitespace-nowrap gap-2 justify-start"
    :disabled="isDisabled"
  >
    <KunIcon class-name="text-lg" name="lucide:trash-2" />
    删除回复
  </KunButton>
</template>
