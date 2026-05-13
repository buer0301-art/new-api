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
import {
  Banner,
  Button,
  Card,
  Col,
  Row,
  Space,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import { CheckCircle2, Clock3, Copy, ExternalLink, X } from 'lucide-react';
import { API, copy, showSuccess } from '../../helpers';

const { Text, Title } = Typography;

function parseExpireTime(value) {
  if (!value) return null;
  const time = Date.parse(value);
  return Number.isFinite(time) ? time : null;
}

function formatRemaining(ms) {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

function initials(value) {
  const normalized = (value || '').trim();
  return normalized ? normalized.slice(0, 2).toUpperCase() : '?';
}

function shortContract(contract) {
  if (!contract) return '';
  return contract.length > 14
    ? `${contract.slice(0, 6)}...${contract.slice(-6)}`
    : contract;
}

function tokenLabel(option) {
  return (option?.code || '').toUpperCase();
}

export default function Web3PayCheckout({ t, order, onCancel, onPaid }) {
  const [selectedToken, setSelectedToken] = useState(0);
  const [selectedChain, setSelectedChain] = useState(0);
  const expireAt = useMemo(() => parseExpireTime(order?.expireTime), [order]);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    const tradeNo = order?.merchantOrderNo || order?.attach;
    if (!tradeNo || !onPaid) return undefined;

    const timer = window.setInterval(async () => {
      const res = await API.get(
        `/api/user/web3-pay/order/${encodeURIComponent(tradeNo)}`,
        { skipErrorHandler: true },
      );
      if (res.data?.success && res.data?.data?.status === 'success') {
        onPaid();
      }
    }, 5000);

    return () => window.clearInterval(timer);
  }, [onPaid, order]);

  useEffect(() => {
    setSelectedChain(0);
  }, [selectedToken]);

  if (!order?.paymentOptions?.length) return null;

  const token =
    order.paymentOptions[selectedToken] || order.paymentOptions[0] || {};
  const chain = token.chain?.[selectedChain] || token.chain?.[0];
  if (!chain) return null;

  const qrValue = chain.address || order.payUrl || order.orderNo;
  const remaining = expireAt ? expireAt - now : 0;

  return (
    <Card className='!rounded-xl w-full border border-emerald-200 bg-emerald-50/40'>
      <div className='flex items-center justify-between gap-3 mb-4'>
        <div>
          <Title heading={5} style={{ margin: 0 }}>
            {t('Web3 Pay 支付')}
          </Title>
          <Text type='tertiary'>
            {t('订单号')}：{order.merchantOrderNo || order.orderNo}
          </Text>
        </div>
        <Space>
          <Tag color='amber' prefixIcon={<Clock3 size={14} />}>
            {expireAt ? formatRemaining(remaining) : t('待支付')}
          </Tag>
          <Button icon={<X size={16} />} onClick={onCancel}>
            {t('取消')}
          </Button>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Space vertical style={{ width: '100%' }}>
            <div>
              <Text type='tertiary'>{t('支付金额')}</Text>
              <div className='mt-2'>
                <Text strong style={{ fontSize: 36 }}>
                  {order.payAmount}
                </Text>
                <Text style={{ marginLeft: 8 }}>
                  {order.payCurrency || tokenLabel(token)}
                </Text>
              </div>
            </div>

            <div>
              <Text strong>{t('选择支付币种')}</Text>
              <div className='grid grid-cols-2 sm:grid-cols-3 gap-2 mt-2'>
                {order.paymentOptions.map((option, index) => (
                  <Button
                    key={option.code || index}
                    theme={selectedToken === index ? 'solid' : 'outline'}
                    type={selectedToken === index ? 'primary' : 'tertiary'}
                    onClick={() => setSelectedToken(index)}
                    className='!h-auto !py-3'
                    icon={
                      option.logo ? (
                        <img
                          src={option.logo}
                          alt={tokenLabel(option)}
                          style={{ width: 22, height: 22, objectFit: 'contain' }}
                        />
                      ) : undefined
                    }
                  >
                    {tokenLabel(option) || initials(option.code)}
                  </Button>
                ))}
              </div>
            </div>

            <div>
              <Text strong>{t('选择链')}</Text>
              <Space vertical style={{ width: '100%', marginTop: 8 }}>
                {token.chain.map((item, index) => (
                  <Card
                    key={`${item.chainCode}-${item.address}`}
                    className='!rounded-lg cursor-pointer'
                    bodyStyle={{ padding: 12 }}
                    style={{
                      border:
                        selectedChain === index
                          ? '1px solid var(--semi-color-primary)'
                          : '1px solid var(--semi-color-border)',
                    }}
                    onClick={() => setSelectedChain(index)}
                  >
                    <div className='flex items-center justify-between gap-3'>
                      <div className='flex items-center gap-3 min-w-0'>
                        {item.logo ? (
                          <img
                            src={item.logo}
                            alt={item.chainName}
                            style={{
                              width: 32,
                              height: 32,
                              objectFit: 'contain',
                            }}
                          />
                        ) : (
                          <span className='flex h-8 w-8 items-center justify-center rounded-full bg-slate-100 text-xs font-bold'>
                            {initials(item.chainName || item.chainCode)}
                          </span>
                        )}
                        <div className='min-w-0'>
                          <Text strong ellipsis={{ showTooltip: true }}>
                            {item.chainName || item.chainCode}
                          </Text>
                          <div>
                            <Text type='tertiary' size='small'>
                              {t('确认数')}：{item.inConfirm || 0}
                              {item.contract
                                ? ` · ${t('合约')} ${shortContract(item.contract)}`
                                : ''}
                            </Text>
                          </div>
                        </div>
                      </div>
                      {selectedChain === index && (
                        <CheckCircle2
                          size={20}
                          color='var(--semi-color-primary)'
                        />
                      )}
                    </div>
                  </Card>
                ))}
              </Space>
            </div>
          </Space>
        </Col>

        <Col xs={24} lg={10}>
          <div className='flex flex-col items-center gap-4 rounded-xl border bg-white p-4'>
            <Tag color='green'>{t('等待支付')}</Tag>
            <Text type='tertiary' style={{ textAlign: 'center' }}>
              {t('请按页面显示的币种和链，向下方地址支付准确金额。')}
            </Text>
            <div className='rounded-xl border p-4 bg-white'>
              <QRCodeSVG value={qrValue} size={210} level='M' />
            </div>
            <div className='w-full'>
              <Text strong>
                {t('支付地址')}（{chain.chainName || chain.chainCode}）
              </Text>
              <div className='mt-2 rounded-lg border bg-slate-50 p-3'>
                <Text
                  code
                  style={{ wordBreak: 'break-all', whiteSpace: 'normal' }}
                >
                  {chain.address}
                </Text>
              </div>
              <Button
                block
                icon={<Copy size={16} />}
                style={{ marginTop: 8 }}
                onClick={async () => {
                  if (await copy(chain.address)) {
                    showSuccess(t('复制成功'));
                  }
                }}
              >
                {t('复制地址')}
              </Button>
            </div>

            {chain.paymentNotice && (
              <Banner
                type='warning'
                description={chain.paymentNotice}
                closeIcon={null}
              />
            )}

            {order.payUrl && (
              <Button
                theme='outline'
                icon={<ExternalLink size={16} />}
                onClick={() =>
                  window.open(order.payUrl, '_blank', 'noopener,noreferrer')
                }
              >
                {t('打开托管支付页')}
              </Button>
            )}
          </div>
        </Col>
      </Row>
    </Card>
  );
}
