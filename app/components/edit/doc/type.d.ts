export type DocEditorMode = 'create' | 'rewrite'

export interface DocEditorForm {
  articleId: number | null
  title: string
  slug: string
  path: string
  description: string
  banner: string
  status: number
  isPin: boolean
  readingMinute: number
  contentMarkdown: string
  categoryId: number | null
  tagIds: number[]
  publishedTime: string
}
