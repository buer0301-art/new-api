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
import { describe, expect, test } from 'bun:test'

import { getTaskPlatformLabel, getTaskResultUrl } from './task-log'

describe('getTaskPlatformLabel', () => {
  test('maps numeric task platforms through channel types', () => {
    expect(getTaskPlatformLabel('1')).toBe('OpenAI')
    expect(getTaskPlatformLabel('50')).toBe('Kling')
  })

  test('keeps named task platforms readable', () => {
    expect(getTaskPlatformLabel('suno')).toBe('suno')
  })
})

describe('getTaskResultUrl', () => {
  test('prefers the dedicated result URL for successful video tasks', () => {
    expect(
      getTaskResultUrl({
        result_url: 'https://example.com/result.mp4',
        fail_reason: 'https://legacy.example.com/result.mp4',
      })
    ).toBe('https://example.com/result.mp4')
  })

  test('falls back to a legacy URL stored in fail_reason', () => {
    expect(
      getTaskResultUrl({
        fail_reason: 'https://legacy.example.com/result.mp4',
      })
    ).toBe('https://legacy.example.com/result.mp4')
  })

  test('ignores non-http result values', () => {
    expect(
      getTaskResultUrl({
        result_url: 'data:video/mp4;base64,abc',
        fail_reason: 'generation failed',
      })
    ).toBeUndefined()
  })
})
