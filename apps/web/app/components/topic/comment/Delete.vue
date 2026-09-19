<script setup lang="ts">
import type { Comment } from '#shared/utils/api/schemas'

const props = defineProps<{
  comment: Comment
  topicId: number
}>()

const emits = defineEmits<{
  removeComment: []
}>()

const { id, moemoepoint } = usePersistUserStore()
const canDeleteTopicComment = useCan('comment.topic.delete')

const isCommonUser = !canDeleteTopicComment.value
const canDelete = computed(
  () => id === Number(props.comment.author.id) || canDeleteTopicComment.value
)

const handleDeleteComment = async () => {
  const moemoepointToDecrease = 3 * (props.comment.like_count + 1)

  if (moemoepoint < moemoepointToDecrease && isCommonUser) {
    useMessage(
      `您的萌萌点不足, 删除这个评论将会消耗您 ${moemoepointToDecrease} 萌萌点。删除消耗萌萌点计算公式为 3 × (评论被点赞数 + 1)`,
      'warn'
    )
    return
  }

  const res = await useComponentMessageStore().alert(
    isCommonUser
      ? '你这个坏萝莉, 确定删除这个评论吗?'
      : '你好萝莉管理员, 要删除这个评论吗',
    isCommonUser
      ? `删除这个评论将会消耗 ${moemoepointToDecrease} 萌萌点, 严重注意, 删除操作不可撤销！删除消耗萌萌点计算公式为 3 × (评论被点赞数 + 1)`
      : '删除这个评论将会消耗发布评论者 3 萌萌点, 该操作不可撤销'
  )
  if (!res) {
    return
  }

  const result = await kunFetch<string>(`/topic/${props.topicId}/comment`, {
    method: 'DELETE',
    query: { commentId: Number(props.comment.id) }
  })

  if (result) {
    emits('removeComment')
    useMessage('删除评论成功', 'success')
  }
}
</script>

<template>
  <KunButton
    v-if="canDelete"
    variant="light"
    color="danger"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="handleDeleteComment"
  >
    <KunIcon class-name="text-lg" name="lucide:trash-2" />
    删除评论
  </KunButton>
</template>
