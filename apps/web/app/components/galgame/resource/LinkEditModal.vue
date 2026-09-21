<script setup lang="ts">
import { checkGalgameResourcePublish } from '../utils/checkGalgameResourcePublish'
import {
  FILE_SIZE_UNITS,
  clampResourceSizeAmount,
  joinResourceSize,
  splitResourceSize,
  type FileSizeUnit
} from '~~/shared/utils/resourceSize'
import {
  LANGUAGE_OPTIONS,
  PLATFORM_OPTIONS,
  RESOURCE_TYPE_OPTIONS,
  RUNTIME_OPTIONS,
  VERSION_LABEL_OPTIONS,
  hasRuntimeAxis,
  resourceTypeLabel
} from '~~/shared/utils/galgameResourceVocab'

const props = defineProps<{
  galgameId: number
  resource?: GalgameResourceDetailLink | null
  refresh: () => void
}>()

const open = defineModel<boolean>({ required: true })

const nuxtApp = useNuxtApp()

const isEditing = computed(() => !!props.resource)

const modalTitle = computed(() =>
  isEditing.value ? '重新编辑资源信息' : '发布 Galgame 资源'
)
const modalSubtitle = computed(() =>
  isEditing.value
    ? '修改链接 / 提取码 / 备注等信息, 保存后立即生效。'
    : '为这部 Galgame 提交一份新的资源链接, 提交后立即对所有用户可见。'
)
const submitLabel = computed(() => (isEditing.value ? '保存修改' : '发布资源'))

interface FormShape {
  type: string
  title: string
  version_label: string
  link: string[]
  language: string
  platform: string
  languages: string[]
  platforms: string[]
  runtimes: string[]
  size: string
  code: string
  password: string
  note: string
}

const defaultForm = (): FormShape => ({
  type: 'game',
  title: '',
  version_label: '',
  link: [],
  language: 'zh-cn',
  platform: 'windows',
  languages: ['zh-cn'],
  platforms: ['win'],
  runtimes: ['native-win'],
  size: '',
  code: '',
  password: '',
  note: ''
})

const snapshotFromResource = (): FormShape => {
  const r = props.resource
  if (!r) return defaultForm()
  const languages =
    r.languages?.length ? r.languages : r.language === 'others' ? ['other'] : [r.language]
  const platforms = r.platforms?.length
    ? r.platforms
    : r.platform === 'windows'
      ? ['win']
      : r.platform === 'app'
        ? ['and']
        : r.platform === 'linux'
          ? ['lin']
          : r.platform === 'mac'
            ? ['mac']
            : r.platform === 'others'
              ? ['oth']
              : []
  const runtimes = r.runtimes?.length
    ? r.runtimes
    : r.platform === 'windows'
      ? ['native-win']
      : r.platform === 'app'
        ? ['native-and']
        : []
  return {
    type: r.type === 'others' || r.type === 'ai' ? 'other' : r.type,
    title: r.title ?? '',
    version_label: r.version_label ?? '',
    link: [...r.link],
    language: r.language,
    platform: r.platform,
    languages,
    platforms,
    runtimes,
    size: r.size,
    code: r.code,
    password: r.password,
    note: r.note
  }
}

const form = ref<FormShape>(snapshotFromResource())
const size = reactive(splitResourceSize(form.value.size))

watch(open, (isOpen) => {
  if (!isOpen) return
  form.value = snapshotFromResource()
  Object.assign(size, splitResourceSize(form.value.size))
})
watch(size, () => {
  form.value.size = joinResourceSize(size)
})

const onSizeAmount = (raw: string | number) => {
  size.amount = clampResourceSizeAmount(String(raw))
}

const isSubmitting = ref(false)

const handleSubmit = async () => {
  if (isSubmitting.value) return
  form.value.size = joinResourceSize(size)
  if (!checkGalgameResourcePublish(form.value)) return

  const method = isEditing.value ? 'PUT' : 'POST'
  const body = isEditing.value
    ? {
        ...form.value,
        galgame_id: props.galgameId,
        galgame_resource_id: props.resource!.id
      }
    : { ...form.value, galgame_id: props.galgameId }

  isSubmitting.value = true
  const result = await nuxtApp.runWithContext(() =>
    kunFetch(`/galgame/${props.galgameId}/resource`, { method, body })
  )
  isSubmitting.value = false

  if (result) {
    nuxtApp.runWithContext(() => {
      useMessage(isEditing.value ? 10550 : 10549, 'success')
      props.refresh()
      open.value = false
    })
  }
}

const handleCancel = () => {
  open.value = false
}

const sizeUnitOptions = [...FILE_SIZE_UNITS]
const showRuntime = computed(() => hasRuntimeAxis(form.value.type))
watch(
  () => form.value.type,
  (typ) => {
    if (!hasRuntimeAxis(typ)) form.value.runtimes = []
  }
)
const typeOptions = computed(() => {
  const opts = [...RESOURCE_TYPE_OPTIONS]
  const current = form.value.type
  if (current && !opts.some((o) => o.value === current)) {
    opts.push({ value: current, label: resourceTypeLabel(current) })
  }
  return opts
})
</script>

<template>
  <KunModal
    v-model="open"
    inner-class-name="max-w-3xl w-[92vw]"
    :is-dismissable="false"
  >
    <div class="space-y-5">
      <div class="space-y-1">
        <h2 class="text-lg font-semibold">{{ modalTitle }}</h2>
        <p class="text-default-500 text-sm">{{ modalSubtitle }}</p>
      </div>

      <GalgameResourceHelp />

      <KunTextarea
        :model-value="form.link.join(',')"
        @update:model-value="
          (v) =>
            (form.link = String(v)
              .split(',')
              .map((s) => s.trim())
              .filter(Boolean))
        "
        placeholder="资源链接 (网盘 | 磁链 | 网址); 同一资源多链接用英文逗号分隔"
      />

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="flex items-end gap-2">
          <div class="flex-1">
            <KunInput
              :model-value="size.amount"
              placeholder="资源体积"
              inputmode="decimal"
              @update:model-value="onSizeAmount"
            />
          </div>
          <div class="w-24 shrink-0">
            <KunSelect
              :model-value="size.unit"
              :options="sizeUnitOptions"
              aria-label="资源体积单位"
              @set="(v) => (size.unit = v as FileSizeUnit)"
            >
              <span>{{ size.unit }}</span>
            </KunSelect>
          </div>
        </div>
        <KunInput v-model="form.code" placeholder="提取码 (可选)" />
        <KunInput v-model="form.password" placeholder="解压码 (可选)" />
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <KunSelect
          v-model="form.type"
          label="资源类型"
          :options="typeOptions"
        >
          <span>{{ resourceTypeLabel(form.type) }}</span>
        </KunSelect>
        <KunInput v-model="form.title" placeholder="资源标题（可选）" />
        <KunSelect
          v-model="form.version_label"
          label="适配版本（可选）"
          :options="VERSION_LABEL_OPTIONS"
          clearable
        />
        <KunSelect
          v-model="form.languages"
          label="语言（可多选）"
          :options="LANGUAGE_OPTIONS"
          multiple
        />
        <KunSelect
          v-model="form.platforms"
          label="平台（可多选）"
          :options="PLATFORM_OPTIONS"
          multiple
          searchable
          search-placeholder="搜索平台"
        />
        <KunSelect
          v-if="showRuntime"
          v-model="form.runtimes"
          label="运行环境（可多选）"
          :options="RUNTIME_OPTIONS"
          multiple
        />
      </div>

      <div class="space-y-1">
        <p class="text-default-600 text-sm font-medium">
          资源备注 (可选) — 注意事项 / 介绍 / 作者信息, 支持 Markdown 与图片
        </p>
        <KunMilkdownDualEditorProvider
          :value-markdown="form.note"
          @set-markdown="(v) => (form.note = v)"
        />
      </div>

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="handleCancel">
          取消
        </KunButton>
        <KunButton
          variant="solid"
          color="primary"
          :loading="isSubmitting"
          @click="handleSubmit"
        >
          {{ submitLabel }}
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
