/*
Copyright (C) 2025 QuantumNous

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

export const DEFAULT_CUSTOM_SIDEBAR_MENUS = [
  {
    title: '在线生图',
    url: 'https://model-go.com/tools/image-ui.html',
  },
];

export function normalizeEmbedUrl(value) {
  const trimmed = value?.trim() ?? '';
  if (!trimmed) return '';

  const candidate = /^[a-z][a-z\d+\-.]*:/i.test(trimmed)
    ? trimmed
    : `https://${trimmed}`;

  try {
    const url = new URL(candidate);
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return '';
    return url.toString();
  } catch {
    return '';
  }
}

export function parseCustomSidebarMenus(raw) {
  if (!raw) return DEFAULT_CUSTOM_SIDEBAR_MENUS;

  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
    if (!Array.isArray(parsed)) return DEFAULT_CUSTOM_SIDEBAR_MENUS;

    const menus = parsed
      .map((menu) => ({
        title: String(menu?.title || '').trim(),
        url: normalizeEmbedUrl(menu?.url),
      }))
      .filter((menu) => menu.title && menu.url);

    return menus;
  } catch {
    return DEFAULT_CUSTOM_SIDEBAR_MENUS;
  }
}
