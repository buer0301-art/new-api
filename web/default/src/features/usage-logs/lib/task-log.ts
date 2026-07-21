import { CHANNEL_TYPES } from '@/features/channels/constants'

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
import type { TaskLog } from '../types'

export function getTaskPlatformLabel(platform: string): string {
  const normalizedPlatform = platform.trim()
  if (!normalizedPlatform) return 'Unknown'

  if (/^\d+$/.test(normalizedPlatform)) {
    const channelType = Number(normalizedPlatform)
    return CHANNEL_TYPES[channelType as keyof typeof CHANNEL_TYPES] || 'Unknown'
  }

  return normalizedPlatform
}

export function getTaskResultUrl(
  log: Pick<TaskLog, 'result_url' | 'fail_reason'>
): string | undefined {
  const candidates = [log.result_url, log.fail_reason]

  return candidates.find(
    (candidate): candidate is string =>
      typeof candidate === 'string' && /^https?:\/\//i.test(candidate)
  )
}
