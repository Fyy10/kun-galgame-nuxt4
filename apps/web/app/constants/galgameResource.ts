export const GALGAME_RESOURCE_TYPE_ICON_MAP: Record<string, string> = {
  game: 'lucide:box',
  collection: 'lucide:boxes',
  patch: 'lucide:puzzle',
  crack_fix: 'lucide:wrench',
  mod: 'lucide:blocks',
  tool: 'lucide:wrench',
  walkthrough: 'lucide:book-open',
  ost: 'lucide:music-4',
  voice: 'lucide:music-4',
  cg: 'lucide:images',
  wallpaper: 'lucide:image',
  artbook: 'lucide:book-open',
  video: 'lucide:video',
  other: 'lucide:ellipsis',
  image: 'lucide:image',
  ai: 'simple-icons:openai',
  others: 'lucide:ellipsis'
}

export const GALGAME_RESOURCE_PLATFORM_ICON_MAP: Record<string, string> = {
  windows: 'ant-design:windows-outlined',
  win: 'ant-design:windows-outlined',
  mac: 'iconoir:apple-mac',
  linux: 'ant-design:linux-outlined',
  lin: 'ant-design:linux-outlined',
  emulator: 'lucide:terminal',
  app: 'lucide:smartphone',
  and: 'lucide:smartphone',
  ios: 'lucide:smartphone',
  others: 'lucide:ellipsis',
  oth: 'lucide:ellipsis'
}

export type ProviderKey =
  | 'baidu'
  | 'aliyun'
  | 'quark'
  | 'pan123'
  | 'tianyiyun'
  | 'caiyun'
  | 'xunlei'
  | 'uc'
  | 'lanzou'
  | 'other'

export const PROVIDER_KEY_OPTIONS = [
  'baidu',
  'aliyun',
  'quark',
  'pan123',
  'tianyiyun',
  'caiyun',
  'xunlei',
  'uc',
  'lanzou',
  'other'
] as const satisfies ProviderKey[]

export const KUN_GALGAME_PROVIDER_LABEL_MAP: Record<ProviderKey, string> = {
  baidu: '百度网盘',
  aliyun: '阿里云盘',
  quark: '夸克网盘',
  pan123: '123盘',
  tianyiyun: '天翼云盘',
  caiyun: '和彩云',
  xunlei: '迅雷网盘',
  uc: 'UC网盘',
  lanzou: '蓝奏云',
  other: '其他 (自建网盘等不限速)'
}

export type GalgameResourceProviderBucketKey =
  | 'baidu'
  | 'quark'
  | 'caiyun'
  | 'pan123'
  | 'xunlei'
  | 'lanzou'
  | 'other'

export interface GalgameResourceProviderBucket {
  key: GalgameResourceProviderBucketKey
  label: string
  match: string[]
  icon: string
}

export const GALGAME_RESOURCE_PROVIDER_BUCKETS: readonly GalgameResourceProviderBucket[] =
  [
    {
      key: 'baidu',
      label: '百度网盘',
      match: ['百度网盘'],
      icon: 'lucide:cloud'
    },
    {
      key: 'quark',
      label: '夸克网盘',
      match: ['夸克网盘'],
      icon: 'lucide:atom'
    },
    {
      key: 'caiyun',
      label: '移动云盘',
      match: ['和彩云', '移动云盘'],
      icon: 'lucide:cloud-cog'
    },
    {
      key: 'pan123',
      label: '123云盘',
      match: ['123 云盘', '123云盘'],
      icon: 'lucide:hash'
    },
    {
      key: 'xunlei',
      label: '迅雷云盘',
      match: ['迅雷云盘', '迅雷'],
      icon: 'lucide:zap'
    },
    {
      key: 'lanzou',
      label: '蓝奏云盘',
      match: ['蓝奏云', '蓝奏'],
      icon: 'lucide:package'
    },
    { key: 'other', label: '其他下载', match: [], icon: 'lucide:ellipsis' }
  ] as const

export const bucketizeResourceProvider = (
  providerNames: string[] | undefined | null
): GalgameResourceProviderBucketKey => {
  if (!providerNames?.length) return 'other'
  for (const bucket of GALGAME_RESOURCE_PROVIDER_BUCKETS) {
    if (bucket.key === 'other') continue
    for (const pn of providerNames) {
      for (const pat of bucket.match) {
        if (pn.includes(pat)) return bucket.key
      }
    }
  }
  return 'other'
}
