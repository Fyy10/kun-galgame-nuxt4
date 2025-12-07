export const asideItems = ref<ChatMessageAsideItem[]>([])

export const replaceAsideItem = (message: ChatMessage) => {
  const targetIndex = asideItems.value.findIndex(
    (item) => item.chatroomName === message.chatroomName
  )

  if (targetIndex !== -1) {
    asideItems.value[targetIndex].content = message.content
    asideItems.value[targetIndex].lastMessageTime = message.created
    asideItems.value[targetIndex].count++
  }

  asideItems.value.sort((a, b) => {
    const timeA = new Date(a.lastMessageTime).getTime()
    const timeB = new Date(b.lastMessageTime).getTime()
    return timeB - timeA
  })
}
