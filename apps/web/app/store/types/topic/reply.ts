import type { Reply } from '#shared/utils/api/schemas'

export interface ReplyRewriteData {
  id: string
  mainContent: string
}

export type SuccessfulReplyEvent =
  | { type: 'created'; data: Reply }
  | { type: 'updated'; data: Reply }
  | { type: 'deleted'; data: { id: string } }

export interface ReplyReference {
  userId: number
  userName: string
  replyId: number
  floor: number
}

export interface ReplyStoreTemp {
  isEdit: boolean
  isScrollToTop: boolean
  scrollToReplyId: number
  isReplyRewriting: boolean
  replyRewrite: ReplyRewriteData | null
  lastSuccessfulReply: SuccessfulReplyEvent | null
  pendingQuote: ReplyReference | null
}

export interface ReplyStorePersist {
  mode: 'preview' | 'source'
  replyDraft: {
    mainContent: string
  }
}
