<script setup lang="ts">
import type { Reply } from '#shared/utils/api/schemas'

const props = defineProps<{
  initialReplies: Reply[]
  topicId: number
  title: string
}>()
</script>

<template>
  <div class="flex flex-col gap-4">
    <TopicReply
      v-for="reply in props.initialReplies"
      :key="reply.id"
      :reply="reply"
      :title="title"
    />
  </div>
</template>
