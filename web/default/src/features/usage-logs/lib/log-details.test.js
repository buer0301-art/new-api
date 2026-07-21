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

import { getTaskSettlement, getVideoDetailFields } from './log-details'

describe('getTaskSettlement', () => {
  test('identifies an additional charge when actual usage exceeds pre-consume', () => {
    expect(
      getTaskSettlement({
        pre_consumed_quota: 3_300_000,
        actual_quota: 4_288_680,
      })
    ).toEqual({
      preConsumedQuota: 3_300_000,
      actualQuota: 4_288_680,
      deltaQuota: 988_680,
      direction: 'consume',
    })
  })

  test('identifies a refund when actual usage is below pre-consume', () => {
    expect(
      getTaskSettlement({
        pre_consumed_quota: 5_000,
        actual_quota: 3_000,
      })
    ).toEqual({
      preConsumedQuota: 5_000,
      actualQuota: 3_000,
      deltaQuota: 2_000,
      direction: 'refund',
    })
  })
})

describe('getVideoDetailFields', () => {
  test('restores the video request metadata shown by the classic theme', () => {
    expect(
      getVideoDetailFields({
        media_type: 'video',
        video_duration: 5,
        video_resolution: '1080p',
        video_ratio: '16:9',
        video_size: '1920x1080',
        video_fps: 30,
        video_frames: 150,
        video_seed: 42,
        video_service_tier: 'default',
      })
    ).toEqual([
      { label: 'Video length in seconds', value: '5s' },
      { label: 'Video resolution', value: '1080p' },
      { label: 'Video aspect ratio', value: '16:9' },
      { label: 'Video size', value: '1920x1080' },
      { label: 'Frame rate', value: '30' },
      { label: 'Frame count', value: '150' },
      { label: 'Seed', value: '42' },
      { label: 'Service Tier', value: 'default' },
    ])
  })
})
