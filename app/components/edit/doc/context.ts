import { inject, provide } from 'vue'
import type { ComputedRef, Ref } from 'vue'
import type { DocEditorForm, DocEditorMode } from './type'

export interface DocEditorContext {
  form: DocEditorForm
  categories: ComputedRef<DocCategoryItem[]>
  tags: ComputedRef<DocTagItem[]>
  mode: DocEditorMode
  isSubmitting: Ref<boolean>
  handleSubmit: () => Promise<void>
  resetForm: () => void
  markPathCustomized: () => void
}

const docEditorContextKey = Symbol('DocEditorContext')

export const provideDocEditorContext = (context: DocEditorContext) => {
  provide(docEditorContextKey, context)
}

export const useDocEditorContext = () => {
  const context = inject<DocEditorContext | null>(docEditorContextKey, null)
  if (!context) {
    throw new Error('Doc editor context is not provided')
  }
  return context
}
