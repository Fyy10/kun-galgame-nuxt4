const WORDS_PER_MINUTE = 300

const sanitizeMarkdown = (markdown: string) => {
  if (!markdown) {
    return ''
  }
  return markdown
    .replace(/`{3}[\s\S]*?`{3}/g, ' ') // remove fenced code blocks
    .replace(/`[^`]+`/g, ' ') // remove inline code
    .replace(/!\[.*?\]\(.*?\)/g, ' ') // remove images
    .replace(/\[[^\]]*?\]\(.*?\)/g, ' ') // remove links, keep text
    .replace(/[^\w\s\u4e00-\u9fa5]/g, ' ') // remove punctuation but keep CJK
}

export const computeReadingMinute = (markdown: string) => {
  const content = sanitizeMarkdown(markdown)
  if (!content.trim()) {
    return 0
  }

  const tokens = content
    .replace(/\s+/g, ' ')
    .trim()
    .split(' ')
    .filter(Boolean)

  if (!tokens.length) {
    return 0
  }

  return Math.max(1, Math.ceil(tokens.length / WORDS_PER_MINUTE))
}

const padNumber = (value: number) => value.toString().padStart(2, '0')

export const formatDatetimeLocalInput = (
  value?: string | Date | number | null
) => {
  if (value === null || value === undefined || value === '') {
    return ''
  }

  const date =
    value instanceof Date ? value : new Date(typeof value === 'number' ? value : value.toString())

  if (Number.isNaN(date.getTime())) {
    return ''
  }

  const year = date.getFullYear()
  const month = padNumber(date.getMonth() + 1)
  const day = padNumber(date.getDate())
  const hours = padNumber(date.getHours())
  const minutes = padNumber(date.getMinutes())

  return `${year}-${month}-${day}T${hours}:${minutes}`
}
