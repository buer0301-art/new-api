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
import { useEffect, useMemo, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import {
  Check,
  Clock3,
  Copy,
  ExternalLink,
  ShieldAlert,
  X,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type {
  Web3PayChainOption,
  Web3PayOrder,
  Web3PayPaymentOption,
} from '../types'
import { getWeb3PayOrderStatus, isApiSuccess } from '../api'

type Web3PayCheckoutProps = {
  order: Web3PayOrder
  onCancel: () => void
  onPaid?: () => void
}

function parseExpireTime(value?: string): number | null {
  if (!value) return null
  const time = Date.parse(value)
  return Number.isFinite(time) ? time : null
}

function formatRemaining(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

function initials(value: string): string {
  const normalized = value.trim()
  if (!normalized) return '?'
  return normalized.slice(0, 2).toUpperCase()
}

function shortContract(contract?: string): string {
  if (!contract) return ''
  if (contract.length <= 14) return contract
  return `${contract.slice(0, 6)}...${contract.slice(-6)}`
}

function methodLabel(option: Web3PayPaymentOption): string {
  return option.code.toUpperCase()
}

export function Web3PayCheckout({
  order,
  onCancel,
  onPaid,
}: Web3PayCheckoutProps) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const [selectedToken, setSelectedToken] = useState(0)
  const [selectedChain, setSelectedChain] = useState(0)
  const expireAt = useMemo(() => parseExpireTime(order.expireTime), [order])
  const [now, setNow] = useState(Date.now())

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [])

  useEffect(() => {
    const tradeNo = order.merchantOrderNo || order.attach
    if (!tradeNo || !onPaid) return

    const timer = window.setInterval(async () => {
      const response = await getWeb3PayOrderStatus(tradeNo)
      if (isApiSuccess(response) && response.data?.status === 'success') {
        onPaid()
      }
    }, 5000)

    return () => window.clearInterval(timer)
  }, [onPaid, order.attach, order.merchantOrderNo])

  const token = order.paymentOptions[selectedToken] || order.paymentOptions[0]
  const chain: Web3PayChainOption | undefined =
    token?.chain?.[selectedChain] || token?.chain?.[0]
  const remaining = expireAt ? expireAt - now : 0
  const qrValue = chain?.address || order.payUrl || order.orderNo

  useEffect(() => {
    setSelectedChain(0)
  }, [selectedToken])

  if (!token || !chain) {
    return null
  }

  return (
    <Card className='overflow-hidden border-emerald-200/70 bg-emerald-50/30 py-0 dark:border-emerald-900/60 dark:bg-emerald-950/10'>
      <CardContent className='p-0'>
        <div className='grid lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.85fr)]'>
          <div className='space-y-5 p-4 sm:p-6'>
            <div className='flex items-start justify-between gap-3'>
              <div className='space-y-1'>
                <Button
                  variant='ghost'
                  size='sm'
                  className='h-8 px-0'
                  onClick={onCancel}
                >
                  <X className='h-4 w-4' />
                  {t('Cancel Payment')}
                </Button>
                <div className='text-muted-foreground text-sm'>
                  {t('Order')} #{order.merchantOrderNo || order.orderNo}
                </div>
              </div>
              <Badge
                variant='secondary'
                className='gap-1.5 bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-100'
              >
                <Clock3 className='h-3.5 w-3.5' />
                {expireAt ? formatRemaining(remaining) : t('Pending')}
              </Badge>
            </div>

            <div>
              <div className='text-muted-foreground text-xs font-semibold tracking-wider uppercase'>
                {t('Amount Due')}
              </div>
              <div className='mt-2 text-5xl font-bold tracking-normal text-slate-950 sm:text-6xl dark:text-slate-50'>
                {order.payAmount}
              </div>
              <div className='text-muted-foreground mt-1 text-sm'>
                {order.payCurrency || methodLabel(token)}
              </div>
            </div>

            <div className='space-y-3'>
              <div className='text-muted-foreground text-xs font-semibold tracking-wider uppercase'>
                {t('Select Payment Currency')}
              </div>
              <div className='grid grid-cols-2 gap-3'>
                {order.paymentOptions.map((option, index) => (
                  <button
                    key={option.code}
                    type='button'
                    onClick={() => setSelectedToken(index)}
                    className={cn(
                      'flex min-h-24 flex-col items-center justify-center gap-2 rounded-lg border bg-background px-4 py-3 text-center transition-colors',
                      selectedToken === index
                        ? 'border-emerald-500 bg-emerald-50 text-emerald-950 dark:bg-emerald-950/40 dark:text-emerald-50'
                        : 'hover:border-foreground/40'
                    )}
                  >
                    {option.logo ? (
                      <img
                        src={option.logo}
                        alt={methodLabel(option)}
                        className='h-8 w-8 object-contain'
                      />
                    ) : (
                      <span className='flex h-9 w-9 items-center justify-center rounded-full bg-emerald-100 text-sm font-bold text-emerald-700'>
                        {initials(methodLabel(option))}
                      </span>
                    )}
                    <span className='font-semibold'>{methodLabel(option)}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className='space-y-3 border-t pt-5'>
              <div className='text-muted-foreground text-xs font-semibold tracking-wider uppercase'>
                {t('Network')}
              </div>
              <div className='space-y-2.5'>
                {token.chain.map((item, index) => (
                  <button
                    key={`${item.chainCode}-${item.address}`}
                    type='button'
                    onClick={() => setSelectedChain(index)}
                    className={cn(
                      'grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-lg border bg-background px-4 py-3 text-left transition-colors',
                      selectedChain === index
                        ? 'border-emerald-500 bg-emerald-50/70 dark:bg-emerald-950/30'
                        : 'hover:border-foreground/30'
                    )}
                  >
                    {item.logo ? (
                      <img
                        src={item.logo}
                        alt={item.chainName}
                        className='h-9 w-9 rounded-full object-contain'
                      />
                    ) : (
                      <span className='flex h-9 w-9 items-center justify-center rounded-full bg-slate-100 text-xs font-bold text-slate-700 dark:bg-slate-800 dark:text-slate-200'>
                        {initials(item.chainName || item.chainCode)}
                      </span>
                    )}
                    <span className='min-w-0'>
                      <span className='block truncate font-semibold'>
                        {item.chainName || item.chainCode}
                      </span>
                      <span className='text-muted-foreground block truncate text-xs'>
                        {t('{{count}} block confirmations', {
                          count: item.inConfirm || 0,
                        })}
                        {item.contract
                          ? ` · ${t('Contract')} ${shortContract(item.contract)}`
                          : ''}
                      </span>
                    </span>
                    {selectedChain === index && (
                      <span className='flex h-8 w-8 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-200'>
                        <Check className='h-4 w-4' />
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className='border-t bg-background/80 p-4 sm:p-6 lg:border-t-0 lg:border-l'>
            <div className='mx-auto flex max-w-md flex-col items-center gap-5'>
              <Badge className='bg-amber-100 px-4 py-1.5 text-amber-900 hover:bg-amber-100'>
                {t('Pending Payment')}
              </Badge>
              <p className='text-muted-foreground text-center text-sm'>
                {t(
                  'Send the exact amount to the address below. The order will update after payment is detected.'
                )}
              </p>
              <div className='rounded-xl border bg-white p-5 shadow-sm'>
                <QRCodeSVG value={qrValue} size={212} level='M' />
              </div>
              <div className='w-full space-y-2'>
                <div className='text-muted-foreground text-center text-xs font-semibold tracking-wider uppercase'>
                  {t('Payment Address')} ({chain.chainName || chain.chainCode})
                </div>
                <div className='grid grid-cols-[minmax(0,1fr)_auto] rounded-lg border bg-background p-2'>
                  <code className='text-muted-foreground min-w-0 overflow-hidden px-2 py-2 text-xs break-all'>
                    {chain.address}
                  </code>
                  <Button
                    type='button'
                    onClick={() => copyToClipboard(chain.address)}
                    className='h-auto gap-2'
                  >
                    <Copy className='h-4 w-4' />
                    {t('Copy')}
                  </Button>
                </div>
              </div>

              {chain.paymentNotice && (
                <div className='flex w-full gap-2 rounded-lg border bg-muted/30 p-3 text-sm'>
                  <ShieldAlert className='text-muted-foreground mt-0.5 h-4 w-4 shrink-0' />
                  <span>{chain.paymentNotice}</span>
                </div>
              )}

              {order.payUrl && (
                <Button
                  variant='outline'
                  size='sm'
                  render={
                    <a
                      href={order.payUrl}
                      target='_blank'
                      rel='noopener noreferrer'
                    />
                  }
                >
                  <ExternalLink className='h-4 w-4' />
                  {t('Open hosted payment page')}
                </Button>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
