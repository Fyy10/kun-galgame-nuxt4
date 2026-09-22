<script setup lang="ts">
import type { Topic } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  topic?: Topic
  topicId?: number
  favoriteCount?: number
  isFavorite?: boolean
}>()

const api = useApiClient()
const replaceTopic = inject<(topic: Topic) => void>('replaceTopic', () => {})

const topicId = computed(() =>
  props.topic ? Number(props.topic.id) : (props.topicId ?? 0)
)
const favoriteCount = computed(
  () => props.topic?.favorite_count ?? props.favoriteCount ?? 0
)
const isFavorite = computed(
  () => props.topic?.viewer?.has_favorited ?? props.isFavorite ?? false
)

const toggleFavorite = async (next: boolean) => {
  const id = props.topic?.id ?? String(topicId.value)
  const result = await settle(
    next
      ? api.PUT('/topics/{topic_id}/favorite', {
          params: { path: { topic_id: id } }
        })
      : api.DELETE('/topics/{topic_id}/favorite', {
          params: { path: { topic_id: id } }
        })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  if (props.topic) {
    replaceTopic({
      ...props.topic,
      favorite_count: result.data.favorite_count,
      viewer: result.data.viewer
    })
  }
  return true
}
</script>

<template>
  <FavoriteToggle
    :favorited="isFavorite"
    :count="favoriteCount"
    :action="toggleFavorite"
    :messages="[10230, 10231]"
  />
</template>
