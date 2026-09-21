export const FILE_SIZE_UNITS = [
  { value: 'MB', label: 'MB' },
  { value: 'GB', label: 'GB' }
] as const

export type FileSizeUnit = (typeof FILE_SIZE_UNITS)[number]['value']

export interface FileSizeParts {
  amount: string
  unit: FileSizeUnit
}

const strictRe = /^([0-9]{1,6})(?:\.([0-9]{1,2}))?\s*(MB|GB)$/i
const tokenRe =
  /(\d{1,6}(?:[.,]\d{1,4})?)\s*\+?\s*[.·]?\s*(MMB|GGB|MB|GB|KB)/gi

export const parseResourceSize = (text: string): string | null => {
  const m = strictRe.exec(text.trim().toUpperCase())
  if (!m) return null
  let amount = m[1]!
  if (m[2]) {
    const frac = m[2].replace(/0+$/, '')
    if (frac) amount += `.${frac}`
  }
  return `${amount} ${m[3]}`
}

export const splitResourceSize = (text: string): FileSizeParts => {
  const parsed = parseResourceSize(text)
  if (parsed) {
    const [amount, unit] = parsed.split(' ')
    return { amount: amount ?? '', unit: unit === 'MB' ? 'MB' : 'GB' }
  }
  const extracted = extractResourceSize(text)
  if (extracted) {
    const [amount, unit] = extracted.split(' ')
    return { amount: amount ?? '', unit: unit === 'MB' ? 'MB' : 'GB' }
  }
  return { amount: '', unit: 'GB' }
}

export const joinResourceSize = (parts: FileSizeParts): string =>
  parts.amount ? `${parts.amount} ${parts.unit}` : ''

export const clampResourceSizeAmount = (raw: string): string => {
  const digits = raw.replace(/[^0-9.]/g, '')
  const [whole = '', ...rest] = digits.split('.')
  const head = whole.slice(0, 6)
  return rest.length ? `${head}.${rest.join('').slice(0, 2)}` : head
}

const extractResourceSize = (raw: string): string | null => {
  const folded = raw
    .replace(/【[^】]*】/g, ' ')
    .replace(/（[^）]*）/g, ' ')
    .replace(/\([^)]*\)/g, ' ')
    .replace(/\[[^\]]*\]/g, ' ')
  let last: RegExpExecArray | null = null
  tokenRe.lastIndex = 0
  for (let m = tokenRe.exec(folded); m; m = tokenRe.exec(folded)) {
    last = m
  }
  if (!last) return null
  const amount = last[1]!.replace(',', '.')
  let unit = last[2]!.toUpperCase()
  if (unit === 'MMB') unit = 'MB'
  if (unit === 'GGB') unit = 'GB'
  if (unit === 'KB') return null
  const clipped =
    amount.includes('.') && amount.split('.')[1]!.length > 2
      ? `${amount.split('.')[0]}.${amount.split('.')[1]!.slice(0, 2)}`
      : amount
  return parseResourceSize(`${clipped} ${unit}`)
}
