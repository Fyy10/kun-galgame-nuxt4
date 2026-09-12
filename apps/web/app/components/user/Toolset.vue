<script setup lang="ts">
const props = defineProps<{
  userId: number
}>()

const pageData = reactive({
  page: usePageQuery(),
  limit: 24
})

const { data, status } = await useKunFetch<{
  items: ToolsetCard[]
  total: number
}>(`/user/${props.userId}/toolsets`, { query: pageData })
</script>

<template>
  <div class="space-y-3">
    <div v-if="data && data.items.length" class="space-y-3">
      <ToolsetCard :items="data.items" />

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull v-if="data && !data.items.length" description="暂无工具" />
  </div>
</template>
