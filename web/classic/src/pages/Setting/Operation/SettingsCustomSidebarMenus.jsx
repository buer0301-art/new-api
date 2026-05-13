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

import React, { useContext, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../helpers';
import { StatusContext } from '../../../context/Status';

const { Text } = Typography;

const DEFAULT_CUSTOM_MENUS = [
  {
    title: '在线生图',
    url: 'https://model-go.com/tools/image-ui.html',
  },
];

function parseCustomMenus(raw) {
  if (!raw) return DEFAULT_CUSTOM_MENUS;
  try {
    const value = JSON.parse(raw);
    if (!Array.isArray(value)) return DEFAULT_CUSTOM_MENUS;
    return value.length > 0 ? value : [{ title: '', url: '' }];
  } catch {
    return DEFAULT_CUSTOM_MENUS;
  }
}

function normalizeMenus(menus) {
  return menus
    .map((menu) => ({
      title: String(menu.title || '').trim(),
      url: String(menu.url || '').trim(),
    }))
    .filter((menu) => menu.title && menu.url);
}

export default function SettingsCustomSidebarMenus(props) {
  const { t } = useTranslation();
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [loading, setLoading] = useState(false);
  const [menus, setMenus] = useState(DEFAULT_CUSTOM_MENUS);

  useEffect(() => {
    setMenus(parseCustomMenus(props.options?.CustomSidebarMenus));
  }, [props.options?.CustomSidebarMenus]);

  const updateMenu = (index, field, value) => {
    const next = menus.map((menu, menuIndex) =>
      menuIndex === index ? { ...menu, [field]: value } : menu,
    );
    setMenus(next);
  };

  const addMenu = () => {
    setMenus([...menus, { title: '', url: '' }]);
  };

  const removeMenu = (index) => {
    const next = menus.filter((_, menuIndex) => menuIndex !== index);
    setMenus(next.length > 0 ? next : [{ title: '', url: '' }]);
  };

  const resetMenus = () => {
    setMenus(DEFAULT_CUSTOM_MENUS);
    showSuccess(t('已重置为默认配置'));
  };

  const onSubmit = async () => {
    const normalizedMenus = normalizeMenus(menus);
    setLoading(true);
    try {
      const serialized = JSON.stringify(normalizedMenus);
      const res = await API.put('/api/option/', {
        key: 'CustomSidebarMenus',
        value: serialized,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('保存成功'));
        statusDispatch({
          type: 'set',
          payload: {
            ...statusState.status,
            CustomSidebarMenus: serialized,
          },
        });
        if (props.refresh) {
          await props.refresh();
        }
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <Form.Section
        text={t('自定义侧边栏菜单')}
        extraText={t('配置后会显示在个人中心区域，点击后以内嵌页面打开。')}
      >
        <div className='flex flex-col gap-3'>
          {menus.map((menu, index) => (
            <Card key={index} bodyStyle={{ padding: 16 }}>
              <div className='grid gap-3 md:grid-cols-[1fr_2fr_auto] md:items-end'>
                <div>
                  <Text strong>{t('菜单名称')}</Text>
                  <Input
                    value={menu.title}
                    onChange={(value) => updateMenu(index, 'title', value)}
                    placeholder={t('例如：在线生图')}
                    style={{ marginTop: 8 }}
                  />
                </div>
                <div>
                  <Text strong>{t('链接地址')}</Text>
                  <Input
                    value={menu.url}
                    onChange={(value) => updateMenu(index, 'url', value)}
                    placeholder='https://model-go.com/tools/image-ui.html'
                    style={{ marginTop: 8 }}
                  />
                </div>
                <Button
                  type='danger'
                  theme='light'
                  onClick={() => removeMenu(index)}
                >
                  {t('删除')}
                </Button>
              </div>
            </Card>
          ))}
        </div>

        <Text type='secondary' size='small' style={{ display: 'block', marginTop: 12 }}>
          {t('每一项会生成一个左侧菜单，并通过 iframe 嵌入对应链接。')}
        </Text>

        <div
          style={{
            display: 'flex',
            gap: '12px',
            justifyContent: 'flex-start',
            alignItems: 'center',
            marginTop: '16px',
            paddingTop: '16px',
            borderTop: '1px solid var(--semi-color-border)',
          }}
        >
          <Button type='tertiary' onClick={addMenu}>
            {t('添加菜单')}
          </Button>
          <Button type='tertiary' onClick={resetMenus}>
            {t('重置为默认')}
          </Button>
          <Button type='primary' onClick={onSubmit} loading={loading}>
            {t('保存设置')}
          </Button>
        </div>
      </Form.Section>
    </Card>
  );
}
