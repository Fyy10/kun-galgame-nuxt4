<script setup lang="ts">
const props = defineProps<{
  userId: number
}>()

const pageData = reactive({
  page: usePageQuery(),
  limit: 24,
  userId: props.userId
})

const { data, status } = await useKunFetch<{
  rating_data: GalgameRatingCard[]
  total: number
}>(`/user/${props.userId}/ratings`, { query: pageData })
</script>

<template>
  <div class="space-y-3">
    <div v-if="data && data.rating_data.length" class="space-y-3">
      <GalgameRatingCard :ratings="data.rating_data" :is-transparent="false" />

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull v-if="data && !data.rating_data.length" description="暂无评分" />
  </div>
</template>
