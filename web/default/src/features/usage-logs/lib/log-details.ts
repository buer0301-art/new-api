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
import type { LogOtherData } from '../types'

export interface TaskSettlement {
  preConsumedQuota: number
  actualQuota: number
  deltaQuota: number
  direction: 'consume' | 'refund'
}

export interface VideoDetailField {
  label: string
  value: string
}

function isFiniteQuota(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

export function getTaskSettlement(
  other: LogOtherData | null
): TaskSettlement | null {
  if (
    !isFiniteQuota(other?.pre_consumed_quota) ||
    !isFiniteQuota(other.actual_quota)
  ) {
    return null
  }

  const deltaQuota = other.actual_quota - other.pre_consumed_quota
  if (deltaQuota === 0) return null

  return {
    preConsumedQuota: other.pre_consumed_quota,
    actualQuota: other.actual_quota,
    deltaQuota: Math.abs(deltaQuota),
    direction: deltaQuota > 0 ? 'consume' : 'refund',
  }
}

export function getVideoDetailFields(
  other: LogOtherData | null
): VideoDetailField[] {
  if (other?.media_type !== 'video') return []

  const fields: VideoDetailField[] = []
  const push = (label: string, value: unknown, suffix = '') => {
    if (value == null || value === '') return
    fields.push({ label, value: `${String(value)}${suffix}` })
  }

  push('Video length in seconds', other.video_duration, 's')
  push('Video resolution', other.video_resolution)
  push('Video aspect ratio', other.video_ratio)
  push('Video size', other.video_size)
  push('Frame rate', other.video_fps)
  push('Frame count', other.video_frames)
  push('Seed', other.video_seed)
  push('Service Tier', other.video_service_tier)

  return fields
}
