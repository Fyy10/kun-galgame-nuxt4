<script setup lang="ts">
import type { Topic } from '#shared/utils/api/schemas'

const props = defineProps<{
  topic?: Topic
  topicId?: number
  favoriteCount?: number
  isFavorite?: boolean
}>()

const topicId = computed(() =>
  props.topic ? Number(props.topic.id) : (props.topicId ?? 0)
)
const favoriteCount = computed(
  () => props.topic?.favorite_count ?? props.favoriteCount ?? 0
)
const isFavorite = computed(
  () => props.topic?.viewer?.has_favorited ?? props.isFavorite ?? false
)
</script>

<template>
  <FavoriteToggle
    :favorited="isFavorite"
    :count="favoriteCount"
    :endpoint="`/topic/${topicId}/favorite`"
    :messages="[10230, 10231]"
  />
</template>
