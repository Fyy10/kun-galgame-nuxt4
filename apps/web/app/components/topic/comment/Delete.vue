<script setup lang="ts">
import type { Comment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  comment: Comment
}>()

const emits = defineEmits<{
  removeComment: []
}>()

const { id } = usePersistUserStore()
const api = useApiClient()

const isAuthor = computed(() => String(id) === props.comment.author.id)

const handleDeleteComment = async () => {
  const moemoepointToDecrease = 3 * (props.comment.like_count + 1)

  const res = await useComponentMessageStore().alert(
    isAuthor.value
      ? '你这个坏萝莉, 确定删除这个评论吗?'
      : '你好萝莉管理员, 要删除这个评论吗',
    isAuthor.value
      ? `删除这个评论将会消耗 ${moemoepointToDecrease} 萌萌点, 严重注意, 删除操作不可撤销！删除消耗萌萌点计算公式为 3 × (评论被点赞数 + 1)`
      : '删除这个评论将会消耗发布评论者 3 萌萌点, 该操作不可撤销'
  )
  if (!res) {
    return
  }

  const result = await settle(
    api.DELETE('/comments/{comment_id}', {
      params: { path: { comment_id: props.comment.id } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  emits('removeComment')
  useMessage('删除评论成功', 'success')
}
</script>

<template>
  <KunButton
    v-if="comment.viewer?.can_delete"
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
