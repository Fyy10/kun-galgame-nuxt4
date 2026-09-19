export const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null

export const isSafeHref = (url: string): boolean => {
  const lower = url.toLowerCase()
  return (
    lower.startsWith('https:') ||
    lower.startsWith('http:') ||
    lower.startsWith('mailto:')
  )
}

export const isSafeMediaUrl = (url: string): boolean => {
  const lower = url.toLowerCase()
  return lower.startsWith('https:') || lower.startsWith('http:')
}

export const isDecimalId = (id: string): boolean => /^[0-9]+$/.test(id)
