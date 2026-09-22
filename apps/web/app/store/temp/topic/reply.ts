import { defineStore } from 'pinia'
import { ref } from 'vue'
import type {
  ReplyReference,
  ReplyRewriteData,
  ReplyStoreTemp,
  SuccessfulReplyEvent
} from '~/store/types/topic/reply'

export const useTempReplyStore = defineStore(
  'tempTopicReply',
  () => {
    const isEdit = ref<ReplyStoreTemp['isEdit']>(false)
    const isScrollToTop = ref<ReplyStoreTemp['isScrollToTop']>(false)
    const scrollToReplyId = ref<ReplyStoreTemp['scrollToReplyId']>(-1)
    const isReplyRewriting = ref<ReplyStoreTemp['isReplyRewriting']>(false)
    const replyRewrite = ref<ReplyStoreTemp['replyRewrite']>(null)
    const lastSuccessfulReply = ref<ReplyStoreTemp['lastSuccessfulReply']>(null)
    const pendingQuote = ref<ReplyStoreTemp['pendingQuote']>(null)

    const setPendingQuote = (reference: ReplyReference) => {
      pendingQuote.value = reference
    }

    const clearPendingQuote = () => {
      pendingQuote.value = null
    }

    const setRewriteData = (reply: ReplyRewriteData) => {
      isReplyRewriting.value = true
      replyRewrite.value = {
        id: reply.id,
        mainContent: reply.mainContent
      }
    }

    const resetRewriteReplyData = () => {
      replyRewrite.value = null
      isReplyRewriting.value = false
    }

    const setSuccessfulReply = (event: SuccessfulReplyEvent) => {
      lastSuccessfulReply.value = event
    }

    const clearSuccessfulReply = () => {
      lastSuccessfulReply.value = null
    }

    return {
      isEdit,
      isScrollToTop,
      scrollToReplyId,
      isReplyRewriting,
      replyRewrite,
      lastSuccessfulReply,
      pendingQuote,
      setPendingQuote,
      clearPendingQuote,
      setRewriteData,
      resetRewriteReplyData,
      setSuccessfulReply,
      clearSuccessfulReply
    }
  },
  {
    persist: false
  }
)
