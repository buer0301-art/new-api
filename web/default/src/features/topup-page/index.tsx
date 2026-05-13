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
import { Link } from '@tanstack/react-router'
import {
  CircleAlert,
  ExternalLink,
  Loader2,
  RefreshCw,
  ShoppingBag,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { useTopupInfo } from '../wallet/hooks/use-topup-info'
import { normalizeTopupIframeUrl } from './lib/topup-url'

function TopupPageActions(props: { url: string; onRefresh: () => void }) {
  const { t } = useTranslation()

  if (!props.url) return null

  return (
    <div className='flex gap-2'>
      <Button size='sm' variant='outline' onClick={props.onRefresh}>
        <RefreshCw className='h-4 w-4' />
        {t('Reload')}
      </Button>
      <Button
        size='sm'
        variant='outline'
        render={
          <a href={props.url} target='_blank' rel='noopener noreferrer' />
        }
      >
        <ExternalLink className='h-4 w-4' />
        {t('Open in new window')}
      </Button>
    </div>
  )
}

function TopupPageEmptyState(props: { loading: boolean; canManage: boolean }) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <Empty className='min-h-[420px] border bg-background'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Loader2 className='animate-spin' />
          </EmptyMedia>
          <EmptyTitle>{t('Loading recharge page...')}</EmptyTitle>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <Empty className='min-h-[420px] border bg-background'>
      <EmptyHeader>
        <EmptyMedia variant='icon'>
          <CircleAlert />
        </EmptyMedia>
        <EmptyTitle>{t('No top-up link configured')}</EmptyTitle>
        <EmptyDescription>
          {t('Configure Top-Up Link in system settings before embedding it.')}
        </EmptyDescription>
      </EmptyHeader>
      {props.canManage && (
        <EmptyContent>
          <Button
            size='sm'
            variant='outline'
            render={
              <Link
                to='/system-settings/billing/$section'
                params={{ section: 'quota' }}
              />
            }
          >
            {t('Configure Top-Up Link')}
          </Button>
        </EmptyContent>
      )}
    </Empty>
  )
}

export function TopupPage() {
  const { t } = useTranslation()
  const { topupInfo, loading } = useTopupInfo()
  const [frameVersion, setFrameVersion] = useState(0)
  const userRole = useAuthStore((s) => s.auth.user?.role ?? ROLE.GUEST)
  const topupUrl = useMemo(
    () => normalizeTopupIframeUrl(topupInfo?.topup_link),
    [topupInfo?.topup_link]
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Recharge Page')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <TopupPageActions
          url={topupUrl}
          onRefresh={() => setFrameVersion((value) => value + 1)}
        />
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex min-h-full flex-col'>
          {topupUrl ? (
            <div className='bg-background flex min-h-[calc(100dvh-8.5rem)] flex-1 overflow-hidden rounded-xl border shadow-sm'>
              <iframe
                key={`${topupUrl}-${frameVersion}`}
                title={t('Recharge Page')}
                src={topupUrl}
                className='h-full min-h-[calc(100dvh-8.5rem)] w-full border-0'
                referrerPolicy='strict-origin-when-cross-origin'
                sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts'
              />
            </div>
          ) : (
            <TopupPageEmptyState
              loading={loading}
              canManage={userRole >= ROLE.ADMIN}
            />
          )}

          {topupUrl && (
            <div className='text-muted-foreground mt-2 flex items-center gap-1.5 text-xs'>
              <ShoppingBag className='h-3.5 w-3.5' />
              <span>{t('Recharge content is loaded from Top-Up Link.')}</span>
            </div>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
