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

import React, { useEffect, useState } from 'react';
import { InputNumber, Modal, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

const SetInviterModal = ({ visible, onCancel, user, refresh, t }) => {
  const [inviterId, setInviterId] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (visible) {
      setInviterId('');
    }
  }, [visible, user?.id]);

  const handleConfirm = async () => {
    if (inviterId === '' || inviterId === null || inviterId === undefined) {
      showError(t('请输入邀请人用户 ID'));
      return;
    }

    const parsedInviterId = parseInt(inviterId, 10);
    if (!Number.isInteger(parsedInviterId) || parsedInviterId <= 0) {
      showError(t('请输入有效的邀请人用户 ID'));
      return;
    }
    if (parsedInviterId === user?.id) {
      showError(t('邀请人不能是当前用户'));
      return;
    }

    setLoading(true);
    try {
      const userRes = await API.get(`/api/user/${user.id}`);
      const { success: loadSuccess, message: loadMessage, data } = userRes.data;
      if (!loadSuccess || !data) {
        showError(loadMessage || t('加载失败'));
        return;
      }

      if ((data.inviter_id || 0) > 0) {
        showError(t('仅无邀请人的用户可以在这里设置邀请人。'));
        onCancel();
        await refresh();
        return;
      }

      const res = await API.put('/api/user/', {
        id: data.id,
        username: data.username,
        display_name: data.display_name || data.username,
        group: data.group,
        remark: data.remark || '',
        inviter_id: parsedInviterId,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('用户信息更新成功！'));
        onCancel();
        await refresh();
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(error.message || t('操作失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={t('设置邀请人')}
      visible={visible}
      onCancel={onCancel}
      onOk={handleConfirm}
      confirmLoading={loading}
      okText={t('确认')}
      cancelText={t('取消')}
    >
      <div className='space-y-3'>
        <Text type='secondary'>
          {t('输入邀请人用户 ID，为该用户设置邀请人。')}
        </Text>
        <div>
          <div className='mb-1'>
            <Text size='small'>{t('用户 ID')}</Text>
          </div>
          <InputNumber
            min={1}
            precision={0}
            placeholder={t('请输入邀请人用户 ID')}
            value={inviterId}
            onChange={(value) => setInviterId(value)}
            style={{ width: '100%' }}
          />
        </div>
      </div>
    </Modal>
  );
};

export default SetInviterModal;
