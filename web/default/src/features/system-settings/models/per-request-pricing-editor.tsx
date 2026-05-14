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
import { useTranslation } from 'react-i18next'
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
  IMAGE_RESOLUTION_ROWS,
  VIDEO_RESOLUTION_ROWS,
  type PerRequestMediaType,
  type PerRequestPriceRule,
  type PerRequestSubtype,
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

function createEnabledState(
  rows: readonly string[],
  prices?: Record<string, number>,
  defaultEnabled = true
) {
  return Object.fromEntries(
    rows.map((resolution) => [
      resolution,
      prices ? prices[resolution] !== undefined : defaultEnabled,
    ])
  ) as Record<string, boolean>
}

function createPriceState(
  rows: readonly string[],
  prices?: Record<string, number>
) {
  return Object.fromEntries(
    rows.map((resolution) => [
      resolution,
      prices?.[resolution] !== undefined ? String(prices[resolution]) : '',
    ])
  ) as Record<string, string>
}

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
  const [imageEnabled, setImageEnabled] = useState<Record<string, boolean>>(
    () => createEnabledState(IMAGE_RESOLUTION_ROWS)
  )
  const [imagePrices, setImagePrices] = useState<Record<string, string>>(() =>
    createPriceState(IMAGE_RESOLUTION_ROWS)
  )
  const [imageDefault, setImageDefault] = useState('1K')
  const [videoEnabled, setVideoEnabled] = useState<Record<string, boolean>>(
    () => createEnabledState(VIDEO_RESOLUTION_ROWS)
  )
  const [videoPrices, setVideoPrices] = useState<Record<string, string>>(() =>
    createPriceState(VIDEO_RESOLUTION_ROWS)
  )
  const [videoDefault, setVideoDefault] = useState('480')

  useEffect(() => {
    if (!rule?.media_type) return

    if (rule.media_type === 'image') {
      setImageEnabled(
        createEnabledState(IMAGE_RESOLUTION_ROWS, rule.prices, false)
      )
      setImagePrices(createPriceState(IMAGE_RESOLUTION_ROWS, rule.prices))
      setImageDefault(rule.default_resolution || '1K')
    } else {
      setVideoEnabled(
        createEnabledState(VIDEO_RESOLUTION_ROWS, rule.prices, false)
      )
      setVideoPrices(createPriceState(VIDEO_RESOLUTION_ROWS, rule.prices))
      setVideoDefault(rule.default_resolution || '480')
    }
  }, [rule])

  const syncRule = (
    mediaType: PerRequestMediaType,
    nextPrices: Record<string, string>,
    nextEnabled: Record<string, boolean>,
    nextDefault: string
  ) => {
    const nextRule = buildRuleFromRows(
      mediaType,
      nextPrices,
      nextEnabled,
      nextDefault
    )
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
      mediaType === 'image' ? imagePrices : videoPrices,
      mediaType === 'image' ? imageEnabled : videoEnabled,
      mediaType === 'image' ? imageDefault : videoDefault
    )
  }

  const renderRows = (
    mediaType: PerRequestMediaType,
    rows: readonly string[],
    enabled: Record<string, boolean>,
    prices: Record<string, string>,
    defaultResolution: string,
    setEnabled: (next: Record<string, boolean>) => void,
    setPrices: (next: Record<string, string>) => void,
    setDefault: (next: string) => void
  ) => {
    const selectableRows = rows.filter((resolution) => enabled[resolution])

    return (
      <div className='space-y-4'>
        <div className='rounded-lg border'>
          {rows.map((resolution) => (
            <div
              key={resolution}
              className='grid grid-cols-[auto_72px_1fr] items-center gap-3 border-b px-3 py-2 last:border-b-0'
            >
              <Switch
                checked={enabled[resolution]}
                onCheckedChange={(checked) => {
                  const nextEnabled = { ...enabled, [resolution]: checked }
                  const nextDefault =
                    checked || defaultResolution !== resolution
                      ? defaultResolution
                      : rows.find(
                          (item) => item !== resolution && nextEnabled[item]
                        ) || resolution
                  setEnabled(nextEnabled)
                  setDefault(nextDefault)
                  syncRule(mediaType, prices, nextEnabled, nextDefault)
                }}
                aria-label={`${resolution} ${t('Enabled')}`}
              />
              <div className='text-sm font-medium'>{resolution}</div>
              <Input
                value={prices[resolution] || ''}
                inputMode='decimal'
                placeholder='0.01'
                onChange={(event) => {
                  const value = event.target.value
                  if (!numericDraftRegex.test(value)) return
                  const nextPrices = { ...prices, [resolution]: value }
                  setPrices(nextPrices)
                  syncRule(mediaType, nextPrices, enabled, defaultResolution)
                }}
                disabled={!enabled[resolution]}
              />
            </div>
          ))}
        </div>

        <div className='flex items-center gap-3'>
          <Label className='text-sm'>{t('Default resolution')}</Label>
          <Select
            items={selectableRows.map((value) => ({ value, label: value }))}
            value={defaultResolution}
            onValueChange={(next) => {
              if (!next) return
              setDefault(next)
              syncRule(mediaType, prices, enabled, next)
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
          IMAGE_RESOLUTION_ROWS,
          imageEnabled,
          imagePrices,
          imageDefault,
          setImageEnabled,
          setImagePrices,
          setImageDefault
        )}

      {subtype === 'video' &&
        renderRows(
          'video',
          VIDEO_RESOLUTION_ROWS,
          videoEnabled,
          videoPrices,
          videoDefault,
          setVideoEnabled,
          setVideoPrices,
          setVideoDefault
        )}
    </div>
  )
}
