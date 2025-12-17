import fs from 'node:fs'
import path from 'node:path'
import matter from 'gray-matter'
import { prisma } from '../prisma/prisma'
import { markdownToHtml } from '../server/utils/remark/markdownToHtml'

const DOC_ROOT = path.resolve('content/doc')

const DOC_CATEGORY_TITLE_MAP: Record<string, string> = {
  galgame: 'Galgame',
  kun: '关于鲲',
  notice: '网站公告',
  other: '其它'
}

const CATEGORY_CACHE = new Map<string, number>()
const TAG_CACHE = new Map<string, number>()

const normalizeSlug = (value: unknown, fallback = '') =>
  (value || fallback)
    .toString()
    .trim()
    .toLowerCase()
    .replace(/[^\w-]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')

const normalizeBoolean = (value: unknown, fallback = false) => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') {
    const lower = value.trim().toLowerCase()
    if (['true', '1', 'yes', 'y', 'on'].includes(lower)) return true
    if (['false', '0', 'no', 'n', 'off'].includes(lower)) return false
  }
  return fallback
}

const normalizeNumber = (value: unknown, fallback = 0) => {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

const normalizeDate = (value: unknown, fallback = new Date()) => {
  if (value instanceof Date && !Number.isNaN(value.getTime())) {
    return value
  }
  if (typeof value === 'string') {
    const parsed = new Date(value)
    if (!Number.isNaN(parsed.getTime())) {
      return parsed
    }
  }
  return fallback
}

const parseDocFile = (filePath: string) => {
  const raw = fs.readFileSync(filePath, 'utf-8')
  const parsed = matter(raw)
  return {
    meta: parsed.data as Record<string, unknown>,
    content: parsed.content.trim()
  }
}

const ensureCategory = async (slug: string, title: string) => {
  if (CATEGORY_CACHE.has(slug)) {
    return CATEGORY_CACHE.get(slug)!
  }

  const existing = await prisma.doc_category.findUnique({
    where: { slug }
  })

  if (existing) {
    CATEGORY_CACHE.set(slug, existing.id)
    return existing.id
  }

  const created = await prisma.doc_category.create({
    data: {
      slug,
      title,
      description: '',
      icon: '',
      sort_order: CATEGORY_CACHE.size
    }
  })

  CATEGORY_CACHE.set(slug, created.id)
  return created.id
}

const ensureTags = async (
  tags: Array<{ slug?: string; title?: string; description?: string }> = []
) => {
  const tagIds: number[] = []

  for (const tag of tags) {
    const slug = normalizeSlug(tag?.slug || tag?.title)
    if (!slug) continue

    if (TAG_CACHE.has(slug)) {
      tagIds.push(TAG_CACHE.get(slug)!)
      continue
    }

    const title = tag?.title?.trim() || slug

    const existing = await prisma.doc_tag.findUnique({
      where: { slug }
    })

    if (existing) {
      TAG_CACHE.set(slug, existing.id)
      tagIds.push(existing.id)
      continue
    }

    const created = await prisma.doc_tag.create({
      data: {
        slug,
        title,
        description: tag?.description || ''
      }
    })

    TAG_CACHE.set(slug, created.id)
    tagIds.push(created.id)
  }

  return tagIds
}

const resolveAuthor = async (meta: Record<string, unknown>) => {
  const authorUid = normalizeNumber(meta.authorUid, 0)
  if (authorUid > 0) {
    const user = await prisma.user.findUnique({
      where: { id: authorUid }
    })
    if (user) {
      return user.id
    }
  }

  const authorName = (meta.authorName as string | undefined)?.trim()
  if (authorName) {
    const candidate = await prisma.user.findFirst({
      where: { name: authorName }
    })
    if (candidate) {
      return candidate.id
    }
  }

  return 1
}

const getMarkdownFiles = (dirPath: string): string[] => {
  const entries = fs.readdirSync(dirPath, { withFileTypes: true })
  return entries.flatMap((entry) => {
    const fullPath = path.join(dirPath, entry.name)
    if (entry.isDirectory()) {
      return getMarkdownFiles(fullPath)
    }
    if (entry.isFile() && entry.name.endsWith('.md')) {
      return [fullPath]
    }
    return []
  })
}

const migrateFile = async (filePath: string) => {
  const relativePath = path.relative(DOC_ROOT, filePath).replace(/\\/g, '/')
  const { meta, content } = parseDocFile(filePath)

  if (!meta.title || typeof meta.title !== 'string') {
    console.warn(`跳过 ${relativePath}: 缺少 title`)
    return
  }

  const slug =
    normalizeSlug(meta.slug) ||
    normalizeSlug(path.basename(filePath, path.extname(filePath)))
  const docPath = `/doc/${slug}`

  const categorySlug = normalizeSlug(meta.category || 'other') || 'other'
  const categoryTitle =
    DOC_CATEGORY_TITLE_MAP[categorySlug] ||
    (meta.category as string) ||
    DOC_CATEGORY_TITLE_MAP.other
  const categoryId = await ensureCategory(categorySlug, categoryTitle)
  const tagIds = await ensureTags(
    Array.isArray(meta.tags)
      ? (meta.tags as Array<{
          slug?: string
          title?: string
          description?: string
        }>)
      : []
  )
  const authorId = await resolveAuthor(meta)

  const markdown = content
  const publishedTime = normalizeDate(meta.publishedTime, new Date())
  const modifiedTime = meta.modifiedTime
    ? normalizeDate(meta.modifiedTime, new Date())
    : undefined

  const baseData = {
    title: meta.title.trim(),
    slug,
    path: docPath,
    description: (meta.description as string) || '',
    banner: (meta.banner as string) || '',
    status: normalizeNumber(meta.status, meta.draft ? 0 : 1),
    is_pin: normalizeBoolean(meta.pin, false),
    view: normalizeNumber(meta.view, 0),
    published_time: publishedTime,
    edited_time: modifiedTime,
    content_markdown: markdown.slice(0, 100007),
    category_id: categoryId,
    author_id: authorId
  }

  const existing = await prisma.doc_article.findFirst({
    where: {
      OR: [{ slug }, { path: docPath }]
    }
  })

  if (!existing) {
    const created = await prisma.doc_article.create({
      data: baseData
    })

    if (tagIds.length) {
      await prisma.doc_article_tag_relation.createMany({
        data: tagIds.map((tagId) => ({
          doc_article_id: created.id,
          doc_tag_id: tagId
        })),
        skipDuplicates: true
      })
    }
  } else {
    await prisma.doc_article.update({
      where: { id: existing.id },
      data: baseData
    })

    await prisma.doc_article_tag_relation.deleteMany({
      where: { doc_article_id: existing.id }
    })

    if (tagIds.length) {
      await prisma.doc_article_tag_relation.createMany({
        data: tagIds.map((tagId) => ({
          doc_article_id: existing.id,
          doc_tag_id: tagId
        })),
        skipDuplicates: true
      })
    }
  }

  await markdownToHtml(markdown)

  console.log(`✓ ${relativePath} -> ${slug}`)
}

const main = async () => {
  try {
    const files = getMarkdownFiles(DOC_ROOT)
    console.log(`发现 ${files.length} 篇文档，即将开始迁移……`)

    for (const file of files) {
      await migrateFile(file)
    }

    console.log('文档迁移完成 ✅')
  } catch (error) {
    console.error('文档迁移失败 ❌', error)
    process.exit(1)
  } finally {
    await prisma.$disconnect()
  }
}

main()
