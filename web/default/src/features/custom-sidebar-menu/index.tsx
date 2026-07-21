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
import { CircleAlert, ExternalLink, RefreshCw, Settings } from 'lucide-react'
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
import { useStatus } from '@/hooks/use-status'

import { parseCustomSidebarMenus } from './lib/custom-sidebar-menus'

type CustomSidebarMenuPageProps = {
  index: string
}

export function CustomSidebarMenuPage(props: CustomSidebarMenuPageProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [frameVersion, setFrameVersion] = useState(0)
  const menus = useMemo(
    () => parseCustomSidebarMenus(status?.CustomSidebarMenus),
    [status?.CustomSidebarMenus]
  )
  const menuIndex = Number(props.index)
  const menu = Number.isInteger(menuIndex) ? menus[menuIndex] : undefined

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {menu?.title || t('Custom menu')}
      </SectionPageLayout.Title>
      {menu && (
        <SectionPageLayout.Actions>
          <div className='flex gap-2'>
            <Button
              size='sm'
              variant='outline'
              onClick={() => setFrameVersion((value) => value + 1)}
            >
              <RefreshCw />
              {t('Reload')}
            </Button>
            <Button
              size='sm'
              variant='outline'
              render={
                <a href={menu.url} target='_blank' rel='noopener noreferrer' />
              }
            >
              <ExternalLink />
              {t('Open in new window')}
            </Button>
          </div>
        </SectionPageLayout.Actions>
      )}
      <SectionPageLayout.Content>
        {menu ? (
          <div className='bg-background flex min-h-[calc(100dvh-8.5rem)] overflow-hidden rounded-lg border'>
            <iframe
              key={`${menu.url}-${frameVersion}`}
              title={menu.title}
              src={menu.url}
              className='h-full min-h-[calc(100dvh-8.5rem)] w-full border-0'
              referrerPolicy='strict-origin-when-cross-origin'
              // Embedded tools commonly need scripts and their own origin for
              // authentication; the configured URL is restricted to HTTP(S).
              // oxlint-disable-next-line react/iframe-missing-sandbox
              sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts'
            />
          </div>
        ) : (
          <Empty className='bg-background min-h-[420px] border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <CircleAlert />
              </EmptyMedia>
              <EmptyTitle>{t('Menu not found')}</EmptyTitle>
              <EmptyDescription>
                {t(
                  'Check the custom sidebar menu configuration in system settings.'
                )}
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button
                size='sm'
                variant='outline'
                render={
                  <Link
                    to='/system-settings/site/$section'
                    params={{ section: 'custom-sidebar-menus' }}
                  />
                }
              >
                <Settings />
                {t('Custom sidebar menus')}
              </Button>
            </EmptyContent>
          </Empty>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
