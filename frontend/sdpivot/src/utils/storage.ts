export const STORAGE_KEYS = {
  accessToken: 'sdp_access_token',
  refreshToken: 'sdp_refresh_token',
  user: 'sdp_user',
  opsAccessToken: 'sdp_ops_access_token',
  opsRefreshToken: 'sdp_ops_refresh_token',
  opsUser: 'sdp_ops_user',
  themeMode: 'sdp_theme_mode',
} as const

const LEGACY_KEYS: Record<string, string[]> = {
  [STORAGE_KEYS.accessToken]: ['access_token', 'smartknora_access_token', 'sk_access_token'],
  [STORAGE_KEYS.refreshToken]: ['refresh_token', 'smartknora_refresh_token', 'sk_refresh_token'],
  [STORAGE_KEYS.opsAccessToken]: ['ops_access_token', 'smartknora_ops_access_token', 'sk_ops_access_token'],
  [STORAGE_KEYS.opsRefreshToken]: ['ops_refresh_token', 'smartknora_ops_refresh_token', 'sk_ops_refresh_token'],
  [STORAGE_KEYS.opsUser]: ['ops_user', 'smartknora_ops_user', 'sk_ops_user'],
  [STORAGE_KEYS.themeMode]: ['smartknora_theme_mode', 'sk_theme_mode'],
}

function migrateStorage(storage: Storage) {
  for (const [key, legacyKeys] of Object.entries(LEGACY_KEYS)) {
    if (storage.getItem(key) === null) {
      const legacyKey = legacyKeys.find((candidate) => storage.getItem(candidate) !== null)
      if (legacyKey) storage.setItem(key, storage.getItem(legacyKey) as string)
    }
    legacyKeys.forEach((legacyKey) => storage.removeItem(legacyKey))
  }
}

export function migrateLegacyStorage() {
  if (typeof window === 'undefined') return
  migrateStorage(window.localStorage)
  migrateStorage(window.sessionStorage)
}
