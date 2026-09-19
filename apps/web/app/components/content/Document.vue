<script setup lang="ts">
import {
  useContentBlurUp,
  useContentLightbox,
  useSpoilerContent
} from '@kungal/ui-vue'
import type { ContentDocument } from '#shared/utils/api/schemas'
import ContentNodes from './Nodes'

withDefaults(
  defineProps<{
    document: ContentDocument
    compact?: boolean
    className?: string
  }>(),
  {
    compact: false,
    className: ''
  }
)

const articleRef = ref<HTMLElement | null>(null)

useSpoilerContent(articleRef)
useContentBlurUp(articleRef)
const { isLightboxOpen, images, currentImageIndex } =
  useContentLightbox(articleRef)
</script>

<template>
  <div>
    <article
      ref="articleRef"
      :class="cn('kun-prose', compact && 'kun-prose-compact', className)"
    >
      <ContentNodes :nodes="document.children" />
    </article>
    <KunLightbox
      v-model:is-open="isLightboxOpen"
      :images="images"
      :initial-index="currentImageIndex"
    />
  </div>
</template>

<style scoped>
.kun-prose {
  & :deep(img) {
    cursor: zoom-in;
  }
  & :deep(.kun-spoiler) {
    transition: color var(--kun-dur-base) var(--ease-kun-standard);
  }
  & :deep(.kun-spoiler-hidden),
  & :deep(.kun-spoiler-live) {
    position: relative;
    display: inline-block;
  }
  & :deep(div.kun-spoiler-hidden),
  & :deep(div.kun-spoiler-live) {
    display: flow-root;
    width: fit-content;
  }
  & :deep(.kun-spoiler-hidden) {
    cursor: pointer;
    color: transparent !important;
    user-select: none;
    background-color: rgb(150 150 150 / 0.18);
  }
  & :deep(.kun-spoiler-hidden > :not(.kun-spoiler-canvas)) {
    visibility: hidden;
  }
  & :deep(.kun-spoiler-hidden:hover) {
    background-color: rgb(150 150 150 / 0.26);
  }
  & :deep(.kun-spoiler-hidden.kun-spoiler-live),
  & :deep(.kun-spoiler-hidden.kun-spoiler-live:hover) {
    background-color: transparent;
  }
}
</style>
