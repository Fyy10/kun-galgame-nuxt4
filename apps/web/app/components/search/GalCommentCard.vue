<script setup lang="ts">
defineProps<{
  comment: SearchResultGalComment
  keywords?: string
}>()
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    :to="comment.link"
    class-name="flex-col items-start w-full gap-1.5"
  >
    <div class="flex w-full items-baseline gap-2">
      <KunIcon
        class="text-primary size-3.5 shrink-0 self-center"
        name="lucide:message-circle"
      />
      <span class="min-w-0 flex-1 truncate text-sm font-medium">
        <SearchHighlight :text="comment.title" :keywords="keywords" />
      </span>
      <KunChip size="sm" color="default" class-name="shrink-0">
        {{ comment.label }}
      </KunChip>
    </div>

    <p
      class="border-primary text-default-700 line-clamp-2 w-full border-l-2 pl-2 text-sm"
    >
      <SearchHighlight :text="comment.content" :keywords="keywords" />
    </p>

    <div class="text-default-500 flex w-full items-center gap-1.5 text-xs">
      <KunAvatar size="xs" :user="comment.user" :is-navigation="false" />
      <span class="truncate">{{ comment.user.name }}</span>
      <span class="ml-auto shrink-0">
        <KunTime :time="comment.created" />
      </span>
    </div>
  </KunLink>
</template>
