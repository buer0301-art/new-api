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
import { RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { CompactDateTimeRangePicker } from '@/features/usage-logs/components/compact-date-time-range-picker'

interface ModelsDateRangeFilterProps {
  start?: Date
  end?: Date
  onChange: (range: { start?: Date; end?: Date }) => void
  onResetToday: () => void
}

export function ModelsDateRangeFilter(props: ModelsDateRangeFilterProps) {
  const { t } = useTranslation()

  return (
    <div className='flex w-full flex-wrap items-center gap-1.5 sm:w-auto sm:gap-2'>
      <div className='w-full sm:w-[24rem]'>
        <CompactDateTimeRangePicker
          start={props.start}
          end={props.end}
          onChange={props.onChange}
        />
      </div>
      <Button
        type='button'
        variant='outline'
        size='sm'
        onClick={props.onResetToday}
        className='h-9 shrink-0'
      >
        <RotateCcw className='mr-2 size-4' aria-hidden='true' />
        {t('Today')}
      </Button>
    </div>
  )
}
