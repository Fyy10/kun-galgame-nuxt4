import { imageRefCdnUrl, parseImageRef } from '../utils/imageRef'

export default defineEventHandler((event) => {
  if (event.method !== 'GET') return
  const path = event.path.split('?')[0]!
  if (!path.startsWith('/image/')) return
  const parsed = parseImageRef(path.slice('/image/'.length))
  if (!parsed) return

  const base = useRuntimeConfig(event).imageCdnBase || ''
  return sendRedirect(event, imageRefCdnUrl(base, parsed), 302)
})
