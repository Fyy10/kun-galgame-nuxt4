export interface GalgameStorePersist {
  vndb_id: string
  name: KunLanguage
  introduction: KunLanguage
  content_limit: 'sfw' | 'nsfw' | ''
  age_limit: 'all' | 'r18'
  original_language: string
  aliases: string[]
  release_date: string
  release_date_tba: boolean
}

export interface GalgameEditStoreTemp {
  id: number
  vndb_id: string
  name: KunLanguage
  introduction: KunLanguage
  content_limit: 'sfw' | 'nsfw'
  age_limit: 'all' | 'r18'
  original_language: string
  alias: string[]
  tags: GalgameTagItem[]
  officials: GalgameOfficialItem[]
  engines: GalgameEngineItem[]
  links: { name: string; link: string }[]
  note: string
  title: string
  message: string
  release_date: string
  release_date_tba: boolean
  covers: GalgameCover[]
  screenshots: GalgameScreenshot[]
  covers_baseline?: string
  screenshots_baseline?: string
  links_baseline?: string
  can_direct_edit: boolean
}
