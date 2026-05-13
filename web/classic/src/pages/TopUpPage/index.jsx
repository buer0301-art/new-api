/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Empty, Spin } from '@douyinfe/semi-ui';
import { IconAlertCircle, IconExternalOpen } from '@douyinfe/semi-icons';
import { API, isAdmin } from '../../helpers';

function normalizeTopUpUrl(value) {
  const trimmed = value?.trim() ?? '';
  if (!trimmed) return '';

  const candidate = /^[a-z][a-z\d+\-.]*:/i.test(trimmed)
    ? trimmed
    : `https://${trimmed}`;

  try {
    const url = new URL(candidate);
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return '';
    return url.toString();
  } catch {
    return '';
  }
}

const TopUpPage = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [topUpLink, setTopUpLink] = useState('');

  const topUpUrl = useMemo(() => normalizeTopUpUrl(topUpLink), [topUpLink]);

  useEffect(() => {
    let cancelled = false;

    const loadTopUpInfo = async () => {
      setLoading(true);
      try {
        const res = await API.get('/api/user/topup/info');
        if (!cancelled && res.data.success) {
          setTopUpLink(res.data.data?.topup_link || '');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    loadTopUpInfo();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className='mt-[64px] h-[calc(100vh-64px)] bg-[#f7fbfb] p-6'>
      <div className='flex h-full flex-col overflow-hidden rounded-xl border border-semi-color-border bg-semi-color-bg-0 shadow-sm'>
        <div className='flex h-16 flex-shrink-0 items-center justify-between border-b border-semi-color-border px-6'>
          <h1 className='m-0 text-xl font-semibold text-semi-color-text-0'>
            {t('充值页面')}
          </h1>
          {topUpUrl && (
            <Button
              theme='light'
              icon={<IconExternalOpen />}
              onClick={() =>
                window.open(topUpUrl, '_blank', 'noopener,noreferrer')
              }
            >
              {t('新窗口打开')}
            </Button>
          )}
        </div>

        {loading ? (
          <div className='flex flex-1 items-center justify-center'>
            <Spin size='large' tip={t('正在加载充值页面...')} />
          </div>
        ) : topUpUrl ? (
          <div className='flex-1 overflow-hidden bg-white'>
            <iframe
              title={t('充值页面')}
              src={topUpUrl}
              className='h-full w-full border-0'
              referrerPolicy='strict-origin-when-cross-origin'
              sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts'
            />
          </div>
        ) : (
          <div className='flex flex-1 items-center justify-center'>
            <Empty
              image={<IconAlertCircle size='extra-large' />}
              title={t('未配置充值链接')}
              description={t('请先在系统设置的通用设置中配置充值链接。')}
            >
              {isAdmin() && (
                <Button
                  theme='solid'
                  type='primary'
                  onClick={() => {
                    window.location.href = '/console/setting?tab=operation';
                  }}
                >
                  {t('前往配置')}
                </Button>
              )}
            </Empty>
          </div>
        )}
      </div>
    </div>
  );
};

export default TopUpPage;
