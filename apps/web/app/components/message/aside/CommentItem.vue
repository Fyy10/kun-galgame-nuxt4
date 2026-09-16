<script setup lang="ts">
const { data } = useKunFetch<{ total: number }>('/community/unread/count', {
  server: false,
  lazy: true
})
const total = computed(() => data.value?.total ?? 0)
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    to="/message/comment"
  >
    <div
      class="bg-default-100 flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
    >
      <KunIcon name="lucide:message-circle" class="text-default-500 text-xl" />
    </div>
    <div class="flex w-full flex-col justify-center">
      <span class="font-bold">未读评论</span>
      <div class="flex items-center justify-between text-sm">
        <span class="text-default-500 line-clamp-1">
          你关注的评论区有了新评论
        </span>
        <KunChip v-if="total" color="primary" class-name="whitespace-nowrap">
          {{ total }}
        </KunChip>
      </div>
    </div>
  </KunLink>
</template>
