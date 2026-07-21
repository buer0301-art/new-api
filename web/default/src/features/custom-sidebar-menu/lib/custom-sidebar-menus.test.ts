import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  normalizeCustomSidebarMenuUrl,
  parseCustomSidebarMenus,
  serializeCustomSidebarMenus,
} from './custom-sidebar-menus.ts'

describe('custom sidebar menus', () => {
  test('normalizes web URLs and rejects unsafe protocols', () => {
    assert.equal(
      normalizeCustomSidebarMenuUrl('tools.example.com/image'),
      'https://tools.example.com/image'
    )
    assert.equal(
      normalizeCustomSidebarMenuUrl('http://localhost:3000/tool'),
      'http://localhost:3000/tool'
    )
    assert.equal(normalizeCustomSidebarMenuUrl('javascript:alert(1)'), '')
  })

  test('parses only complete menu entries', () => {
    assert.deepEqual(
      parseCustomSidebarMenus(
        JSON.stringify([
          { title: ' Images ', url: 'tools.example.com/image' },
          { title: '', url: 'https://example.com' },
          { title: 'Unsafe', url: 'javascript:alert(1)' },
        ]),
        []
      ),
      [{ title: 'Images', url: 'https://tools.example.com/image' }]
    )
  })

  test('serializes normalized menu entries', () => {
    assert.equal(
      serializeCustomSidebarMenus([
        { title: ' Docs ', url: 'docs.example.com' },
        { title: '', url: 'https://example.com' },
      ]),
      '[{"title":"Docs","url":"https://docs.example.com/"}]'
    )
  })
})
