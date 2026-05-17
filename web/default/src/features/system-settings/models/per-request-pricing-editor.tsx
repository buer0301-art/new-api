/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  buildRuleFromRows,
  createDefaultPriceRows,
  createEmptyPriceRow,
  createPriceRowsFromRule,
  getConfiguredDefaultResolution,
  IMAGE_RESOLUTION_REFERENCE,
  type PerRequestMediaType,
  type PerRequestPriceRule,
  type PerRequestPriceRow,
  type PerRequestSubtype,
  VIDEO_RESOLUTION_REFERENCE,
} from './per-request-pricing'

type PerRequestPricingEditorProps = {
  name?: string
  price: string
  onPriceChange: (value: string) => void
  rule: PerRequestPriceRule | null
  onRuleChange: (rule: PerRequestPriceRule | null) => void
  subtype: PerRequestSubtype
  onSubtypeChange: (value: PerRequestSubtype) => void
}

const MEDIA_BY_SUBTYPE: Record<
  Exclude<PerRequestSubtype, 'fixed'>,
  PerRequestMediaType
> = {
  image: 'image',
  video: 'video',
}

const numericDraftRegex = /^(\d+(\.\d*)?|\.\d*)?$/

export function PerRequestPricingEditor({
  name,
  price,
  onPriceChange,
  rule,
  onRuleChange,
  subtype,
  onSubtypeChange,
}: PerRequestPricingEditorProps) {
  const { t } = useTranslation()
  const [imageRows, setImageRows] = useState<PerRequestPriceRow[]>(() =>
    createDefaultPriceRows('image')
  )
  const [imageDefault, setImageDefault] = useState('')
  const [videoRows, setVideoRows] = useState<PerRequestPriceRow[]>(() =>
    createDefaultPriceRows('video')
  )
  const [videoDefault, setVideoDefault] = useState('')

  useEffect(() => {
    if (rule?.media_type === 'image') {
      setImageRows(createPriceRowsFromRule('image', rule))
      setImageDefault(getConfiguredDefaultResolution('image', rule))
    } else if (rule?.media_type === 'video') {
      setVideoRows(createPriceRowsFromRule('video', rule))
      setVideoDefault(getConfiguredDefaultResolution('video', rule))
    } else {
      setImageRows(createDefaultPriceRows('image'))
      setImageDefault(getConfiguredDefaultResolution('image', null))
      setVideoRows(createDefaultPriceRows('video'))
      setVideoDefault(getConfiguredDefaultResolution('video', null))
    }
  }, [name])

  const syncRule = (
    mediaType: PerRequestMediaType,
    nextRows: PerRequestPriceRow[],
    nextDefault: string
  ) => {
    const nextRule = buildRuleFromRows(mediaType, nextRows, nextDefault)
    onRuleChange(nextRule)
  }

  const handleSubtypeChange = (value: string) => {
    const nextSubtype = value as PerRequestSubtype
    onSubtypeChange(nextSubtype)
    if (nextSubtype === 'fixed') {
      onRuleChange(null)
      return
    }
    const mediaType = MEDIA_BY_SUBTYPE[nextSubtype]
    syncRule(
      mediaType,
      mediaType === 'image' ? imageRows : videoRows,
      mediaType === 'image' ? imageDefault : videoDefault
    )
  }

  const renderRows = (
    mediaType: PerRequestMediaType,
    rows: PerRequestPriceRow[],
    defaultResolution: string,
    setRows: (next: PerRequestPriceRow[]) => void,
    setDefault: (next: string) => void
  ) => {
    const selectableRows = rows
      .filter((row) => row.enabled && row.resolution.trim())
      .map((row) => row.resolution.trim())

    const updateRows = (
      nextRows: PerRequestPriceRow[],
      nextDefault = defaultResolution
    ) => {
      setRows(nextRows)
      syncRule(mediaType, nextRows, nextDefault)
    }

    const normalizeDefault = (nextRows: PerRequestPriceRow[]) => {
      const nextSelectable = nextRows
        .filter((item) => item.enabled && item.resolution.trim())
        .map((item) => item.resolution.trim())
      if (!defaultResolution) return nextSelectable[0] || ''
      return nextSelectable.includes(defaultResolution)
        ? defaultResolution
        : nextSelectable[0] || ''
    }

    const reference =
      mediaType === 'image'
        ? IMAGE_RESOLUTION_REFERENCE
        : VIDEO_RESOLUTION_REFERENCE

    return (
      <div className='space-y-4'>
        <div className='text-muted-foreground text-xs'>
          {t(
            'Reference resolutions: {{resolutions}}. You can enter any custom resolution label.',
            { resolutions: reference }
          )}
        </div>

        <div className='rounded-lg border'>
          {rows.map((row) => (
            <div
              key={row.id}
              className='grid grid-cols-[auto_minmax(96px,0.7fr)_minmax(120px,1fr)_auto] items-center gap-3 border-b px-3 py-2 last:border-b-0'
            >
              <Switch
                checked={row.enabled}
                onCheckedChange={(checked) => {
                  const nextRows = rows.map((item) =>
                    item.id === row.id ? { ...item, enabled: checked } : item
                  )
                  const nextDefault = normalizeDefault(nextRows)
                  setDefault(nextDefault)
                  updateRows(nextRows, nextDefault)
                }}
                aria-label={`${row.resolution || t('Resolution')} ${t('Enabled')}`}
              />
              <Input
                value={row.resolution}
                placeholder={t('Resolution')}
                onChange={(event) => {
                  const value = event.target.value
                  const nextRows = rows.map((item) =>
                    item.id === row.id ? { ...item, resolution: value } : item
                  )
                  const nextDefault =
                    defaultResolution === row.resolution
                      ? value.trim()
                      : defaultResolution
                  setDefault(nextDefault)
                  updateRows(nextRows, nextDefault)
                }}
              />
              <Input
                value={row.price}
                inputMode='decimal'
                placeholder='0.01'
                onChange={(event) => {
                  const value = event.target.value
                  if (!numericDraftRegex.test(value)) return
                  const nextRows = rows.map((item) =>
                    item.id === row.id ? { ...item, price: value } : item
                  )
                  updateRows(nextRows)
                }}
                disabled={!row.enabled}
              />
              <Button
                type='button'
                variant='ghost'
                size='icon'
                onClick={() => {
                  const nextRows = rows.filter((item) => item.id !== row.id)
                  const nextDefault = normalizeDefault(nextRows)
                  setDefault(nextDefault)
                  updateRows(nextRows, nextDefault)
                }}
                aria-label={t('Delete')}
              >
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>

        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => {
            const nextRows = [...rows, createEmptyPriceRow()]
            updateRows(nextRows)
          }}
        >
          <Plus data-icon='inline-start' />
          {t('Add resolution')}
        </Button>

        <div className='flex items-center gap-3'>
          <Label className='text-sm'>{t('Default resolution')}</Label>
          <Select
            items={selectableRows.map((value) => ({ value, label: value }))}
            value={defaultResolution}
            disabled={selectableRows.length === 0}
            onValueChange={(next) => {
              if (!next) return
              setDefault(next)
              syncRule(mediaType, rows, next)
            }}
          >
            <SelectTrigger className='w-[140px]'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {selectableRows.map((value) => (
                  <SelectItem key={value} value={value}>
                    {value}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className='text-muted-foreground text-xs'>
          {t('Unknown resolution: Reject request')}
        </div>
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      {name ? (
        <div className='text-muted-foreground text-sm'>{name}</div>
      ) : null}
      <Tabs value={subtype} onValueChange={handleSubtypeChange}>
        <TabsList className='grid w-full grid-cols-3'>
          <TabsTrigger value='fixed'>{t('Fixed')}</TabsTrigger>
          <TabsTrigger value='image'>{t('Image resolution')}</TabsTrigger>
          <TabsTrigger value='video'>{t('Video resolution')}</TabsTrigger>
        </TabsList>
      </Tabs>

      {subtype === 'fixed' ? (
        <div className='space-y-2'>
          <Label>{t('Fixed price')}</Label>
          <Input
            value={price}
            inputMode='decimal'
            placeholder='0.01'
            onChange={(event) => {
              const value = event.target.value
              if (!numericDraftRegex.test(value)) return
              onPriceChange(value)
            }}
          />
          <div className='text-muted-foreground text-xs'>
            {t('Cost in USD per request, regardless of media resolution.')}
          </div>
        </div>
      ) : null}

      {subtype === 'image' &&
        renderRows(
          'image',
          imageRows,
          imageDefault,
          setImageRows,
          setImageDefault
        )}

      {subtype === 'video' &&
        renderRows(
          'video',
          videoRows,
          videoDefault,
          setVideoRows,
          setVideoDefault
        )}
    </div>
  )
}
