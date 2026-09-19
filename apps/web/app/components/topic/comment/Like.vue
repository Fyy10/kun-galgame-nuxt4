<script setup lang="ts">
import type { Comment } from '#shared/utils/api/schemas'

const props = defineProps<{
  comment: Comment
}>()

const { id } = usePersistUserStore()
const topicId = inject<number>('topicId', 0)
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
  if (id === Number(props.comment.author.id)) {
    useMessage(10218, 'warn')
    revert(next)
    return
  }
  pending.value = true
  const result = await kunFetch<string>(`/topic/${topicId}/comment/like`, {
    method: 'PUT',
    body: { comment_id: Number(props.comment.id) }
  })
  pending.value = false
  if (!result) {
    revert(next)
    return
  }
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
