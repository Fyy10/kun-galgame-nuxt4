<script setup lang="ts">
import { createToolsetResourceSchema } from '~/validations/toolset'
import { applyResourceLinkBlur } from '~~/shared/utils/resourceLink'

const props = defineProps<{
  toolsetId: number
  type: 's3' | 'user'
  uploadResult: ToolsetUploadResult
}>()

const emits = defineEmits<{
  onClose: []
  onSuccess: [ToolsetResource]
}>()

const formData = reactive({
  toolset_id: props.toolsetId,
  type: props.type,
  content: '',
  artifact_uuid: props.type === 's3' ? props.uploadResult.artifact_uuid : '',
  size:
    props.type === 's3' && props.uploadResult.size
      ? String(props.uploadResult.size)
      : '',
  code: '',
  password: '',
  note: ''
})
const isLoading = ref(false)

const sizeDisplay = computed(() => {
  if (props.type === 's3') {
    const bytes = Number(formData.size)
    return Number.isFinite(bytes) && bytes > 0 ? formatFileSize(bytes) : ''
  }
  return formData.size
})

const onSizeInput = (value: string | number) => {
  if (props.type === 'user') {
    formData.size = String(value)
  }
}

watch(
  () => props.type,
  () => {
    formData.type = props.type
    if (props.type === 's3') {
      formData.artifact_uuid = props.uploadResult.artifact_uuid
      formData.content = ''
      formData.size = props.uploadResult.size
        ? String(props.uploadResult.size)
        : ''
    } else {
      formData.artifact_uuid = ''
      formData.content = ''
      formData.size = ''
    }
  }
)

watch(
  () => props.uploadResult,
  () => {
    if (props.type === 's3') {
      formData.artifact_uuid = props.uploadResult.artifact_uuid
      formData.size = props.uploadResult.size
        ? String(props.uploadResult.size)
        : ''
    }
  }
)

const commitRecognizedLink = () => {
  if (props.type !== 'user') return
  const recognized = applyResourceLinkBlur(
    formData.content,
    formData.code,
    formData.password
  )
  if (!recognized.applied) return
  formData.content = recognized.links.join(', ')
  formData.code = recognized.code
  formData.password = recognized.password
}

const submitLink = async () => {
  commitRecognizedLink()
  const result = useKunSchemaValidator(createToolsetResourceSchema, formData)
  if (!result) {
    return
  }

  isLoading.value = true
  const ok = await kunFetch<ToolsetResource>(
    `/toolset/${props.toolsetId}/resource`,
    {
      method: 'POST',
      body: formData
    }
  )
  isLoading.value = false

  if (ok) {
    useMessage('资源发布成功', 'success')
    emits('onSuccess', ok)
    emits('onClose')
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunInput
      :placeholder="
        props.type === 'user'
          ? '大小 (如 520KB, 1007MB, 0721GB)'
          : '确认上传完成后, 自动生成文件大小'
      "
      :disabled="props.type === 's3'"
      :model-value="sizeDisplay"
      @update:model-value="onSizeInput"
    />
    <KunInput
      v-if="props.type === 'user'"
      placeholder="提取码 (可选)"
      v-model="formData.code"
    />
    <KunInput placeholder="解压密码 (可选)" v-model="formData.password" />
    <KunTextarea
      placeholder="备注 (建议写明您提供的资源的使用注意事项等)"
      v-model="formData.note"
    />
    <ResourceLinkInput
      v-if="props.type === 'user'"
      v-model="formData.content"
      v-model:code="formData.code"
      v-model:password="formData.password"
      placeholder="资源链接 (可直接粘贴分享文本；多个链接用英文逗号分隔)"
    />
    <div class="flex justify-end gap-2">
      <KunButton variant="light" color="danger" @click="emits('onClose')">
        取消
      </KunButton>
      <KunButton :loading="isLoading" :disabled="isLoading" @click="submitLink">
        提交链接
      </KunButton>
    </div>
  </div>
</template>
