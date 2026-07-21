/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export type CustomSidebarMenu = {
  title: string
  url: string
}

export const DEFAULT_CUSTOM_SIDEBAR_MENUS: CustomSidebarMenu[] = [
  {
    title: '在线生图',
    url: 'https://model-go.com/tools/image-ui.html',
  },
]

export function normalizeCustomSidebarMenuUrl(value?: string | null): string {
  const trimmed = value?.trim() ?? ''
  if (!trimmed) return ''

  const candidate = /^[a-z][a-z\d+\-.]*:/i.test(trimmed)
    ? trimmed
    : `https://${trimmed}`

  try {
    const url = new URL(candidate)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return ''
    return url.toString()
  } catch {
    return ''
  }
}

export function parseCustomSidebarMenus(
  raw: unknown,
  fallback: CustomSidebarMenu[] = DEFAULT_CUSTOM_SIDEBAR_MENUS
): CustomSidebarMenu[] {
  if (raw === null || raw === undefined || raw === '') {
    return fallback.map((menu) => ({ ...menu }))
  }

  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
    if (!Array.isArray(parsed)) {
      return fallback.map((menu) => ({ ...menu }))
    }

    return parsed
      .map((menu) => {
        if (!menu || typeof menu !== 'object') return null
        const record = menu as Record<string, unknown>
        const title =
          typeof record.title === 'string' ? record.title.trim() : ''
        const url =
          typeof record.url === 'string'
            ? normalizeCustomSidebarMenuUrl(record.url)
            : ''
        return title && url ? { title, url } : null
      })
      .filter((menu): menu is CustomSidebarMenu => menu !== null)
  } catch {
    return fallback.map((menu) => ({ ...menu }))
  }
}

export function serializeCustomSidebarMenus(
  menus: CustomSidebarMenu[]
): string {
  const normalized = menus
    .map((menu) => ({
      title: menu.title.trim(),
      url: normalizeCustomSidebarMenuUrl(menu.url),
    }))
    .filter((menu) => menu.title && menu.url)

  return JSON.stringify(normalized)
}
