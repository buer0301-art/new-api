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

import React, { useContext, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import { Empty } from '@douyinfe/semi-ui';
import { IconAlertCircle } from '@douyinfe/semi-icons';
import { StatusContext } from '../../context/Status';
import { parseCustomSidebarMenus } from '../../helpers/customSidebarMenus';

const CustomSidebarMenuPage = () => {
  const { t } = useTranslation();
  const { index } = useParams();
  const [statusState] = useContext(StatusContext);
  const menuIndex = Number(index);
  const menus = useMemo(
    () => parseCustomSidebarMenus(statusState?.status?.CustomSidebarMenus),
    [statusState?.status?.CustomSidebarMenus],
  );
  const menu = Number.isInteger(menuIndex) ? menus[menuIndex] : undefined;

  return (
    <div className='mt-[64px] h-[calc(100vh-64px)] bg-[#f7fbfb] p-6'>
      <div className='flex h-full flex-col overflow-hidden rounded-xl border border-semi-color-border bg-semi-color-bg-0 shadow-sm'>
        {menu?.url ? (
          <div className='flex-1 overflow-hidden bg-white'>
            <iframe
              title={menu.title}
              src={menu.url}
              className='h-full w-full border-0'
              referrerPolicy='strict-origin-when-cross-origin'
              sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-same-origin allow-scripts'
            />
          </div>
        ) : (
          <div className='flex flex-1 items-center justify-center'>
            <Empty
              image={<IconAlertCircle size='extra-large' />}
              title={t('菜单不存在')}
              description={t('请在系统设置中检查自定义侧边栏菜单配置。')}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default CustomSidebarMenuPage;
