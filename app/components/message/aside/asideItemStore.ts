export const asideItems = ref<ChatMessageAsideItem[]>([])

const getAsideMessagePreview = (message: ChatMessage) => {
  if (message.isRecall) {
    return `${message.sender.name}撤回了一条消息`
  }
  return message.content
}

export const replaceAsideItem = (message: ChatMessage) => {
  const targetIndex = asideItems.value.findIndex(
    (item) => item.chatroomName === message.chatroomName
  )

  if (targetIndex !== -1) {
    asideItems.value[targetIndex]!.content = getAsideMessagePreview(message)
    asideItems.value[targetIndex]!.lastMessageTime = message.created
    if (!message.isRecall) {
      asideItems.value[targetIndex]!.count++
    }
  }

  asideItems.value.sort((a, b) => {
    const timeA = new Date(a.lastMessageTime).getTime()
    const timeB = new Date(b.lastMessageTime).getTime()
    return timeB - timeA
  })
}
