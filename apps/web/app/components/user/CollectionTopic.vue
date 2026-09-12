<script setup lang="ts">
const props = defineProps<{
  userId: number
}>()

const page = usePageQuery()
const limit = 24

const { data, status } = await useKunFetch<{
  topics: UserTopic[]
  total: number
}>(() => `/user/${props.userId}/topics`, {
  query: computed(() => ({
    page: page.value,
    limit,
    type: 'topic_favorite',
    userId: props.userId
  })),
  watch: [page]
})
</script>

<template>
  <div class="space-y-3">
    <div v-if="data && data.topics.length" class="flex flex-col space-y-3">
      <KunCard
        v-for="topic in data.topics"
        :key="topic.id"
        :href="`/topic/${topic.id}`"
      >
        <div>{{ topic.title }}</div>
        <div class="text-default-500 text-sm">
          <KunTime :time="topic.created" type="date" show-year />
        </div>
      </KunCard>

      <KunPagination
        v-if="data.total > limit"
        v-model:current-page="page"
        :total-page="Math.ceil(data.total / limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull
      v-if="data && !data.topics.length"
      description="这只笨蛋萝莉还没有收藏任何话题"
    />
  </div>
</template>
