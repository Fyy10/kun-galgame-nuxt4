<script setup lang="ts">
import { useDocEditorContext } from './context'

const { form } = useDocEditorContext()
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 class="text-lg font-semibold">正文内容</h3>
        <p class="text-default-500 text-sm">
          使用 Markdown 撰写文档，系统会自动转换为 HTML 并生成目录
        </p>
      </div>
      <div class="text-default-500 text-sm">
        预计阅读时长：
        <span class="text-primary font-semibold">
          {{ form.readingMinute || 0 }} 分钟
        </span>
      </div>
    </div>

    <KunMilkdownDualEditorProvider
      :value-markdown="form.contentMarkdown"
      @set-markdown="(value) => (form.contentMarkdown = value)"
    >
      <template #default>
        <KunLink
          underline="hover"
          to="/doc/notice/topic-publish-rule"
          target="_blank"
          size="sm"
        >
          查看文档撰写规范
        </KunLink>
      </template>
    </KunMilkdownDualEditorProvider>
  </div>
</template>
