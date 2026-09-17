export type KunAppPlatform = 'android' | 'ios' | 'windows' | 'linux'

export interface KunAppRelease {
  min_version: string
  latest_version: string
  notes: string
  downloads: Record<KunAppPlatform, string>
}
