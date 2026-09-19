export interface ReplyRewriteData {
  id: number
  mainContent: string
}

export interface SuccessfulReplyEvent {
  data: { id: number | string; floor: number }
  type: 'created' | 'updated' | 'deleted'
}

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
