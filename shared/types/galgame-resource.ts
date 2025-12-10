export interface GalgameResource {
  id: number
  galgameId: number
  user: KunUser
  type: string
  language: string
  platform: string
  size: string
  status: number
  download: number
  likeCount: number
  isLiked: boolean
  linkDomain: string
  created: Date | string
}

export interface GalgameResourceDetails extends GalgameResource {
  user: KunUser
  link: string[]
  code: string
  password: string
  note: string
}

export interface GalgameResourceCard extends GalgameResource {
  galgameName: KunLanguage
}

export interface GalgameResourceSummary {
  id: number
  name: KunLanguage
  banner: string
  contentLimit: string
  resourceUpdateTime: Date | string
  view: number
  originalLanguage: string
  ageLimit: KunAgeLimit
  platform: string[]
  language: string[]
  type: string[]
}

export interface GalgameResourcePageData {
  galgame: GalgameResourceSummary
  resource: GalgameResourceDetails
  recommendations: GalgameResourceCard[]
}
