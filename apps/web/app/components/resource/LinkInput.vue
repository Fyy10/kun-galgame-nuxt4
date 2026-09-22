<script setup lang="ts">
import {
  KUN_GALGAME_PROVIDER_ICON_MAP,
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  type ProviderKey
} from '~/constants/galgameResource'
import {
  applyResourceLinkBlur,
  applyResourceLinkPaste,
  detectProviderKeyFromURL,
  parseResourceLinks,
  splitResourceLinkText
} from '~~/shared/utils/resourceLink'

const content = defineModel<string>({ default: '' })
const code = defineModel<string>('code', { default: '' })
const password = defineModel<string>('password', { default: '' })

withDefaults(
  defineProps<{
    label?: string
    description?: string
    placeholder?: string
    required?: boolean
  }>(),
  {
    label: '',
    description: '',
    placeholder: 'https://...',
    required: false
  }
)

const field = ref<{ textareaRef?: HTMLTextAreaElement | null } | null>(null)

const detectedLinks = computed(() =>
  parseResourceLinks(content.value).links.map((url) => {
    const key = detectProviderKeyFromURL(url) as ProviderKey
    return {
      url,
      key,
      label: KUN_GALGAME_PROVIDER_LABEL_MAP[key],
      icon: KUN_GALGAME_PROVIDER_ICON_MAP[key]
    }
  })
)

const selectionCoversAll = (): boolean => {
  const textarea = field.value?.textareaRef
  if (!textarea) return !content.value.trim()
  return (
    textarea.selectionStart === 0 &&
    textarea.selectionEnd === textarea.value.length
  )
}

const onPaste = (event: ClipboardEvent) => {
  const pasted = event.clipboardData?.getData('text') ?? ''
  if (!pasted.trim()) return

  const result = applyResourceLinkPaste({
    pasted,
    existingLinks: splitResourceLinkText(content.value),
    existingCode: code.value,
    existingPassword: password.value,
    replaceAll: !content.value.trim() || selectionCoversAll()
  })
  if (!result.applied) return

  event.preventDefault()
  content.value = result.links.join(', ')
  code.value = result.code
  password.value = result.password
  if (result.notify) useMessage(result.notify, 'success')
}

const onBlur = () => {
  const result = applyResourceLinkBlur(
    content.value,
    code.value,
    password.value
  )
  if (!result.applied) return
  content.value = result.links.join(', ')
  code.value = result.code
  password.value = result.password
  if (result.notify) useMessage(result.notify, 'success')
}
</script>

<template>
  <div class="space-y-1.5">
    <KunTextarea
      ref="field"
      v-model="content"
      :label="label"
      :required="required"
      :description="description"
      :placeholder="placeholder"
      @paste="onPaste"
      @blur="onBlur"
    />
    <ul v-if="detectedLinks.length" class="space-y-1">
      <li
        v-for="item in detectedLinks"
        :key="item.url"
        class="flex min-w-0 items-center gap-2"
      >
        <KunChip size="sm" color="primary" variant="flat">
          <KunIcon :name="item.icon" />
          {{ item.label }}
        </KunChip>
        <span class="text-default-500 truncate text-xs">{{ item.url }}</span>
      </li>
    </ul>
  </div>
</template>
