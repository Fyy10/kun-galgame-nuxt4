<script setup lang="ts">
const props = defineProps<{
  recommendations: GalgameResourceCard[]
}>()
</script>

<template>
  <KunCard :is-hoverable="false" :is-transparent="false" content-class="space-y-4">
    <KunHeader
      name="相关资源推荐"
      description="优先展示同一个 Galgame 的资源，如果不足则随机补充下载量不错的优质资源。"
      scale="h3"
    />

    <div
      v-if="props.recommendations.length"
      class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3"
    >
      <GalgameResourceCard
        v-for="resource in props.recommendations"
        :key="resource.id"
        :resource="resource"
      />
    </div>

    <KunNull
      v-else
      description="暂无可推荐的资源，欢迎贡献新的下载链接。"
    />
  </KunCard>
</template>
