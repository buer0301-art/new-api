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

import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
  toBoolean,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { BookOpen, TriangleAlert } from 'lucide-react';

export default function SettingsPaymentGatewayWeb3Pay(props) {
  const { t } = useTranslation();
  const sectionTitle = props.hideSectionTitle
    ? undefined
    : t('Web3 Pay 设置');
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    Web3PayEnabled: false,
    Web3PayGatewayAPIBase: 'https://pay.example.com/api/gateway/v1',
    Web3PayCheckoutMode: 'inline',
    Web3PayAppKey: '',
    Web3PayApiSecret: '',
    Web3PayUnitPrice: 1.0,
    Web3PayMinTopUp: 1,
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        Web3PayEnabled: toBoolean(props.options.Web3PayEnabled),
        Web3PayGatewayAPIBase:
          props.options.Web3PayGatewayAPIBase ||
          'https://pay.example.com/api/gateway/v1',
        Web3PayCheckoutMode:
          props.options.Web3PayCheckoutMode === 'redirect'
            ? 'redirect'
            : 'inline',
        Web3PayAppKey: props.options.Web3PayAppKey || '',
        Web3PayApiSecret: props.options.Web3PayApiSecret || '',
        Web3PayUnitPrice: parseFloat(props.options.Web3PayUnitPrice) || 1.0,
        Web3PayMinTopUp: parseFloat(props.options.Web3PayMinTopUp) || 1,
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitWeb3PaySetting = async () => {
    const unitPrice = Number(inputs.Web3PayUnitPrice);
    const minTopUp = Number(inputs.Web3PayMinTopUp);
    const gatewayAPIBase = removeTrailingSlash(
      (inputs.Web3PayGatewayAPIBase || '').trim(),
    );

    if (!/^https?:\/\//.test(gatewayAPIBase)) {
      showError(t('Web3 Pay 下单地址必须以 http:// 或 https:// 开头'));
      return;
    }

    if (!Number.isFinite(unitPrice) || unitPrice <= 0) {
      showError(t('Web3 Pay 单价必须大于 0'));
      return;
    }

    if (!Number.isFinite(minTopUp) || minTopUp < 1) {
      showError(t('Web3 Pay 最低充值数量不能小于 1'));
      return;
    }

    setLoading(true);
    try {
      const options = [
        {
          key: 'Web3PayEnabled',
          value: inputs.Web3PayEnabled ? 'true' : 'false',
        },
        { key: 'Web3PayGatewayAPIBase', value: gatewayAPIBase },
        {
          key: 'Web3PayCheckoutMode',
          value:
            inputs.Web3PayCheckoutMode === 'redirect' ? 'redirect' : 'inline',
        },
        { key: 'Web3PayUnitPrice', value: unitPrice.toString() },
        { key: 'Web3PayMinTopUp', value: minTopUp.toString() },
      ];

      if (inputs.Web3PayAppKey && inputs.Web3PayAppKey.trim() !== '') {
        options.push({
          key: 'Web3PayAppKey',
          value: inputs.Web3PayAppKey.trim(),
        });
      }

      if (inputs.Web3PayApiSecret && inputs.Web3PayApiSecret.trim() !== '') {
        options.push({
          key: 'Web3PayApiSecret',
          value: inputs.Web3PayApiSecret.trim(),
        });
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={sectionTitle}>
          <Banner
            type='info'
            icon={<BookOpen size={16} />}
            description={
              <>
                {t(
                  'Web3 Pay 下单和回调验签使用 App Key 与 API Secret，请在商户后台获取并填写。',
                )}
                <br />
                {t('回调地址')}：
                {props.options.ServerAddress
                  ? removeTrailingSlash(props.options.ServerAddress)
                  : t('网站地址')}
                /api/web3-pay/webhook
              </>
            }
            style={{ marginBottom: 12 }}
          />
          <Banner
            type='warning'
            icon={<TriangleAlert size={16} />}
            description={t(
              'App Key 和 API Secret 属于敏感信息，保存后不会回显，留空表示保持当前配置不变。',
            )}
            style={{ marginBottom: 16 }}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='Web3PayEnabled'
                label={t('启用 Web3 Pay')}
                size='default'
                checkedText='｜'
                uncheckedText='〇'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='Web3PayGatewayAPIBase'
                label={t('Web3 Pay 网关 Base URL')}
                placeholder='https://pay.example.com/api/gateway/v1'
                extraText={t(
                  '填写 Web3 Pay 提供的网关 Base URL，不是商户后台地址',
                )}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Select
                field='Web3PayCheckoutMode'
                label={t('支付页面打开方式')}
                extraText={t('站内显示会在充值页展示币种、链和地址；跳转模式会打开 Web3 Pay 收银台')}
              >
                <Form.Select.Option value='inline'>
                  {t('站内显示')}
                </Form.Select.Option>
                <Form.Select.Option value='redirect'>
                  {t('跳转收银台页面')}
                </Form.Select.Option>
              </Form.Select>
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='Web3PayAppKey'
                label={t('Web3 Pay App Key')}
                placeholder={t('填写后覆盖当前 App Key，留空表示保持当前不变')}
                extraText={t('支付地址需要在 Web3 Pay 管理后台配置')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='Web3PayApiSecret'
                label={t('Web3 Pay API Secret')}
                placeholder={t(
                  '填写后覆盖当前 API Secret，留空表示保持当前不变',
                )}
                extraText={t('用于创建订单签名和回调验签')}
                type='password'
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.InputNumber
                field='Web3PayUnitPrice'
                precision={2}
                min={0}
                label={t('Web3 Pay 单价')}
                placeholder={t('例如：1，就是按 1 元人民币收款')}
                extraText={t('按 1 站内余额对应多少人民币支付金额填写')}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.InputNumber
                field='Web3PayMinTopUp'
                min={1}
                label={t('Web3 Pay 最低充值数量')}
                placeholder={t('例如：10，就是最低充值 10 元人民币')}
                extraText={t('用户单次最少可充值的人民币金额')}
              />
            </Col>
          </Row>

          <Button onClick={submitWeb3PaySetting} style={{ marginTop: 16 }}>
            {t('更新 Web3 Pay 设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
