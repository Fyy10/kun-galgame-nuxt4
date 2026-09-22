<script setup lang="ts">
import type { Comment } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  comment: Comment
}>()

const { id } = usePersistUserStore()
const api = useApiClient()
const isLiked = ref(props.comment.viewer?.has_liked ?? false)
const likeCount = ref(props.comment.like_count)
const pending = ref(false)

watch(
  () => props.comment.viewer?.has_liked,
  (value) => {
    isLiked.value = value ?? false
  }
)
watch(
  () => props.comment.like_count,
  (value) => {
    likeCount.value = value
  }
)

const revert = (next: boolean) => {
  isLiked.value = !next
  likeCount.value += next ? -1 : 1
}

const onChange = async (next: boolean) => {
  if (!id) {
    useAuthModal().open()
    revert(next)
    return
  }
  if (!props.comment.viewer?.can_like) {
    useMessage(10218, 'warn')
    revert(next)
    return
  }
  pending.value = true
  const params = { params: { path: { comment_id: props.comment.id } } }
  const result = await settle(
    next
      ? api.PUT('/comments/{comment_id}/like', params)
      : api.DELETE('/comments/{comment_id}/like', params)
  )
  pending.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    revert(next)
    return
  }
  likeCount.value = result.data.like_count
  isLiked.value = result.data.viewer?.has_liked ?? next
  useMessage(next ? '点赞评论成功' : '取消点赞成功', 'success')
}
</script>

<template>
  <KunReaction
    v-model="isLiked"
    v-model:count="likeCount"
    :disabled="pending"
    size="sm"
    icon="lucide:thumbs-up"
    color="primary"
    label="点赞"
    @change="onChange"
  />
</template>
