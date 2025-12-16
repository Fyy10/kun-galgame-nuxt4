<script setup lang="ts">
import { useDocEditorContext } from './context'

const { form, mode } = useDocEditorContext()

const titlePlaceholder = computed(() =>
  mode === 'rewrite' ? '更新文档标题' : '请输入新的文档标题'
)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <h3 class="text-lg font-semibold">文档标题</h3>
      <KunBadge color="primary" variant="flat">
        {{ mode === 'rewrite' ? '重新编辑' : '创建文档' }}
      </KunBadge>
    </div>

    <KunInput
      v-model="form.title"
      :placeholder="titlePlaceholder"
      label="标题"
      required
      maxlength="233"
      autofocus
    />
  </div>
</template>
