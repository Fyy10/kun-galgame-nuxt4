<script setup lang="ts">
import { useDocEditorContext } from './context'

const { handleSubmit, isSubmitting, mode, resetForm } = useDocEditorContext()

const primaryLabel = computed(() =>
  mode === 'rewrite' ? '更新文档' : '创建文档'
)
</script>

<template>
  <div class="space-y-3 rounded-xl border border-default-200 p-4">
    <div class="space-y-2">
      <h4 class="text-base font-semibold">提交操作</h4>
      <p class="text-default-500 text-sm">
        提交后会立即同步到公开页面，请务必确认内容无误
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <KunButton
        color="primary"
        :loading="isSubmitting"
        :disabled="isSubmitting"
        @click="handleSubmit"
      >
        {{ primaryLabel }}
      </KunButton>

      <KunButton
        variant="flat"
        color="default"
        :disabled="isSubmitting"
        @click="resetForm"
      >
        {{ mode === 'rewrite' ? '还原为原始内容' : '清空所有内容' }}
      </KunButton>
    </div>
  </div>
</template>
