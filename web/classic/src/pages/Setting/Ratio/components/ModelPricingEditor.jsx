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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Empty,
  Input,
  Modal,
  Radio,
  RadioGroup,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconPlus,
  IconSave,
  IconSearch,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  PAGE_SIZE,
  PRICE_SUFFIX,
  buildSummaryText,
  hasValue,
  useModelPricingEditorState,
} from '../hooks/useModelPricingEditorState';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import TieredPricingEditor from './TieredPricingEditor';
import {
  buildRuleFromRows,
  createDefaultPriceRows,
  createEmptyPriceRow,
  createPriceRowsFromRule,
  getConfiguredDefaultResolution,
} from './perRequestPricing';

const { Text } = Typography;
const EMPTY_CANDIDATE_MODEL_NAMES = [];

const PriceInput = ({
  label,
  value,
  placeholder,
  onChange,
  suffix = PRICE_SUFFIX,
  disabled = false,
  extraText = '',
  headerAction = null,
  hidden = false,
}) => (
  <div style={{ marginBottom: 16 }}>
    <div className='mb-1 font-medium text-gray-700 flex items-center justify-between gap-3'>
      <span>{label}</span>
      {headerAction}
    </div>
    {!hidden ? (
      <Input
        value={value}
        placeholder={placeholder}
        onChange={onChange}
        suffix={suffix}
        disabled={disabled}
      />
    ) : null}
    {extraText ? (
      <div className='mt-1 text-xs text-gray-500'>{extraText}</div>
    ) : null}
  </div>
);

const MEDIA_BY_SUBTYPE = {
  image: 'image',
  video: 'video',
};

const RESOLUTION_REFERENCES = {
  image: '1K / 2K / 4K',
  video: '480 / 720 / 1080',
};

const DEFAULT_VIDEO_UNIT = 'request';

const PerRequestPricingEditor = ({
  model,
  onPriceChange,
  onSubtypeChange,
  onRuleChange,
  t,
}) => {
  const [imageRows, setImageRows] = useState(() =>
    createDefaultPriceRows('image'),
  );
  const [imageDefault, setImageDefault] = useState('');
  const [videoRows, setVideoRows] = useState(() =>
    createDefaultPriceRows('video'),
  );
  const [videoDefault, setVideoDefault] = useState('');
  const [videoUnit, setVideoUnit] = useState(DEFAULT_VIDEO_UNIT);

  useEffect(() => {
    const rule = model?.perRequestRule;

    if (rule?.media_type === 'image') {
      setImageRows(createPriceRowsFromRule('image', rule));
      setImageDefault(getConfiguredDefaultResolution('image', rule));
    } else if (rule?.media_type === 'video') {
      setVideoRows(createPriceRowsFromRule('video', rule));
      setVideoDefault(getConfiguredDefaultResolution('video', rule));
      setVideoUnit(rule.unit === 'second' ? 'second' : DEFAULT_VIDEO_UNIT);
    } else {
      setImageRows(createDefaultPriceRows('image'));
      setImageDefault(getConfiguredDefaultResolution('image', null));
      setVideoRows(createDefaultPriceRows('video'));
      setVideoDefault(getConfiguredDefaultResolution('video', null));
      setVideoUnit(DEFAULT_VIDEO_UNIT);
    }
  }, [model?.name]);

  const syncRule = (mediaType, nextRows, nextDefault, unit) => {
    onRuleChange(buildRuleFromRows(mediaType, nextRows, nextDefault, unit));
  };

  const handleSubtypeChange = (value) => {
    onSubtypeChange(value);
    if (value === 'fixed') {
      onRuleChange(null);
      return;
    }

    const mediaType = MEDIA_BY_SUBTYPE[value];
    syncRule(
      mediaType,
      mediaType === 'image' ? imageRows : videoRows,
      mediaType === 'image' ? imageDefault : videoDefault,
      mediaType === 'video' ? videoUnit : undefined,
    );
  };

  const renderRows = (
    mediaType,
    rows,
    defaultResolution,
    setRows,
    setDefault,
    unit,
    setUnit,
  ) => {
    const selectableRows = rows
      .filter((row) => row.enabled && row.resolution.trim())
      .map((row) => row.resolution.trim());

    const updateRows = (nextRows, nextDefault = defaultResolution) => {
      setRows(nextRows);
      syncRule(mediaType, nextRows, nextDefault, unit);
    };

    const normalizeDefault = (nextRows) => {
      const nextSelectable = nextRows
        .filter((row) => row.enabled && row.resolution.trim())
        .map((row) => row.resolution.trim());
      if (!defaultResolution) {
        return nextSelectable[0] || '';
      }
      return nextSelectable.includes(defaultResolution)
        ? defaultResolution
        : nextSelectable[0] || '';
    };

    return (
      <Card
        bodyStyle={{ padding: 16 }}
        style={{
          marginBottom: 16,
          background: 'var(--semi-color-fill-0)',
        }}
      >
        <div className='mb-3'>
          <div className='font-medium'>
            {mediaType === 'image' ? t('图片分辨率价格') : t('视频分辨率价格')}
          </div>
          <div className='text-xs text-gray-500 mt-1'>
            {mediaType === 'image'
              ? t('每个分辨率按次计价，价格单位为 $/张。')
              : unit === 'second'
                ? t('每个分辨率按秒计价，价格单位为 $/秒。')
                : t('每个分辨率按次计价，价格单位为 $/次。')}
          </div>
          <div className='text-xs text-gray-500 mt-1'>
            {t(
              '参考分辨率：{{resolutions}}。这里只是示例，可输入任意自定义档位。',
              {
                resolutions: RESOLUTION_REFERENCES[mediaType],
              },
            )}
          </div>
        </div>

        {mediaType === 'video' && unit && setUnit ? (
          <div className='mb-3'>
            <div className='mb-1 font-medium text-gray-700'>
              {t('计费单位')}
            </div>
            <RadioGroup
              type='button'
              value={unit}
              onChange={(event) => {
                const nextUnit = event.target.value;
                if (nextUnit !== 'request' && nextUnit !== 'second') return;
                setUnit(nextUnit);
                syncRule(mediaType, rows, defaultResolution, nextUnit);
              }}
            >
              <Radio value='request'>{t('按次')}</Radio>
              <Radio value='second'>{t('按秒')}</Radio>
            </RadioGroup>
          </div>
        ) : null}

        <div style={{ display: 'grid', gap: 8 }}>
          {rows.map((row) => (
            <div
              key={row.id}
              style={{
                display: 'grid',
                gridTemplateColumns:
                  'max-content minmax(96px, 0.8fr) minmax(120px, 1fr) max-content',
                gap: 8,
                alignItems: 'center',
              }}
            >
              <Switch
                size='small'
                checked={row.enabled}
                onChange={(checked) => {
                  const nextRows = rows.map((item) =>
                    item.id === row.id ? { ...item, enabled: checked } : item,
                  );
                  const nextDefault = normalizeDefault(nextRows);
                  setDefault(nextDefault);
                  updateRows(nextRows, nextDefault);
                }}
              />
              <Input
                value={row.resolution}
                placeholder={t('分辨率')}
                onChange={(value) => {
                  const nextRows = rows.map((item) =>
                    item.id === row.id ? { ...item, resolution: value } : item,
                  );
                  const nextDefault =
                    defaultResolution === row.resolution
                      ? value.trim()
                      : defaultResolution;
                  setDefault(nextDefault);
                  updateRows(nextRows, nextDefault);
                }}
              />
              <Input
                value={row.price}
                placeholder='0.01'
                suffix={
                  mediaType === 'image'
                    ? t('$/张')
                    : unit === 'second'
                      ? t('$/秒')
                      : t('$/次')
                }
                disabled={!row.enabled}
                onChange={(value) => {
                  if (!/^(\d+(\.\d*)?|\.\d*)?$/.test(value)) return;
                  const nextRows = rows.map((item) =>
                    item.id === row.id ? { ...item, price: value } : item,
                  );
                  updateRows(nextRows);
                }}
              />
              <Button
                type='danger'
                theme='borderless'
                icon={<IconDelete />}
                onClick={() => {
                  const nextRows = rows.filter((item) => item.id !== row.id);
                  const nextDefault = normalizeDefault(nextRows);
                  setDefault(nextDefault);
                  updateRows(nextRows, nextDefault);
                }}
              />
            </div>
          ))}
        </div>

        <Space wrap className='mt-3'>
          <Button
            icon={<IconPlus />}
            onClick={() => updateRows([...rows, createEmptyPriceRow()])}
          >
            {t('添加分辨率')}
          </Button>
          <Space>
            <Text>{t('默认分辨率')}</Text>
            <Select
              value={defaultResolution}
              style={{ width: 140 }}
              disabled={selectableRows.length === 0}
              onChange={(value) => {
                if (!value) return;
                setDefault(value);
                syncRule(mediaType, rows, value, unit);
              }}
            >
              {selectableRows.map((value) => (
                <Select.Option key={value} value={value}>
                  {value}
                </Select.Option>
              ))}
            </Select>
          </Space>
        </Space>

        <div className='mt-2 text-xs text-gray-500'>
          {mediaType === 'video' && unit === 'request'
            ? t('视频任务会按匹配分辨率每次请求计费一次。')
            : null}
          {mediaType === 'video' && unit === 'second'
            ? t('视频任务会按匹配分辨率和生成秒数计费。')
            : null}
          {mediaType !== 'video'
            ? t('未配置的分辨率会拒绝请求，不会自动套用其它档位。')
            : null}
        </div>
      </Card>
    );
  };

  const subtype = model.perRequestSubtype || 'fixed';

  return (
    <>
      <div className='mb-4'>
        <div className='mb-2 font-medium text-gray-700'>{t('按次类型')}</div>
        <RadioGroup
          type='button'
          value={subtype}
          onChange={(event) => handleSubtypeChange(event.target.value)}
        >
          <Radio value='fixed'>{t('固定价格')}</Radio>
          <Radio value='image'>{t('图片分辨率')}</Radio>
          <Radio value='video'>{t('视频分辨率')}</Radio>
        </RadioGroup>
      </div>

      {subtype === 'fixed' ? (
        <PriceInput
          label={t('固定价格')}
          value={model.fixedPrice}
          placeholder={t('输入每次调用价格')}
          suffix={t('$/次')}
          onChange={(value) => onPriceChange('fixedPrice', value)}
          extraText={t('适合不区分分辨率的任务类按次收费模型。')}
        />
      ) : null}

      {subtype === 'image'
        ? renderRows(
            'image',
            imageRows,
            imageDefault,
            setImageRows,
            setImageDefault,
          )
        : null}

      {subtype === 'video'
        ? renderRows(
            'video',
            videoRows,
            videoDefault,
            setVideoRows,
            setVideoDefault,
            videoUnit,
            setVideoUnit,
          )
        : null}
    </>
  );
};

export default function ModelPricingEditor({
  options,
  refresh,
  candidateModelNames = EMPTY_CANDIDATE_MODEL_NAMES,
  filterMode = 'all',
  allowAddModel = true,
  allowDeleteModel = true,
  showConflictFilter = true,
  listDescription = '',
  emptyTitle = '',
  emptyDescription = '',
}) {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [addVisible, setAddVisible] = useState(false);
  const [batchVisible, setBatchVisible] = useState(false);
  const [newModelName, setNewModelName] = useState('');

  const {
    selectedModel,
    selectedModelName,
    selectedModelNames,
    setSelectedModelName,
    setSelectedModelNames,
    searchText,
    setSearchText,
    currentPage,
    setCurrentPage,
    loading,
    conflictOnly,
    setConflictOnly,
    filteredModels,
    pagedData,
    selectedWarnings,
    previewRows,
    isOptionalFieldEnabled,
    handleOptionalFieldToggle,
    handleNumericFieldChange,
    handleBillingModeChange,
    handleBillingExprChange,
    handleRequestRuleExprChange,
    handlePerRequestSubtypeChange,
    handlePerRequestRuleChange,
    handleSubmit,
    addModel,
    deleteModel,
    applySelectedModelPricing,
  } = useModelPricingEditorState({
    options,
    refresh,
    t,
    candidateModelNames,
    filterMode,
  });

  const getExprModeLabel = useCallback(
    (model) => {
      if (model?.billingMode !== 'tiered_expr') {
        return '';
      }
      return (model.billingExpr || '').includes('tier(')
        ? t('阶梯计费')
        : t('表达式计费');
    },
    [t],
  );

  const columns = useMemo(
    () => [
      {
        title: t('模型名称'),
        dataIndex: 'name',
        key: 'name',
        render: (text, record) => (
          <Space>
            <Button
              theme='borderless'
              type='tertiary'
              onClick={() => setSelectedModelName(record.name)}
              style={{
                padding: 0,
                color:
                  record.name === selectedModelName
                    ? 'var(--semi-color-primary)'
                    : undefined,
              }}
            >
              {text}
            </Button>
            {selectedModelNames.includes(record.name) ? (
              <Tag color='green' shape='circle'>
                {t('已勾选')}
              </Tag>
            ) : null}
            {record.hasConflict ? (
              <Tag color='red' shape='circle'>
                {t('矛盾')}
              </Tag>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('计费方式'),
        dataIndex: 'billingMode',
        key: 'billingMode',
        render: (_, record) => (
          <Tag
            color={
              record.billingMode === 'per-request'
                ? 'teal'
                : record.billingMode === 'tiered_expr'
                  ? 'amber'
                  : 'violet'
            }
          >
            {record.billingMode === 'per-request'
              ? t('按次计费')
              : record.billingMode === 'tiered_expr'
                ? getExprModeLabel(record)
                : t('按量计费')}
          </Tag>
        ),
      },
      {
        title: t('价格摘要'),
        dataIndex: 'summary',
        key: 'summary',
        render: (_, record) => buildSummaryText(record, t),
      },
      {
        title: t('操作'),
        key: 'action',
        render: (_, record) => (
          <Space>
            {allowDeleteModel ? (
              <Button
                size='small'
                type='danger'
                icon={<IconDelete />}
                onClick={() => deleteModel(record.name)}
              />
            ) : null}
          </Space>
        ),
      },
    ],
    [
      allowDeleteModel,
      deleteModel,
      getExprModeLabel,
      selectedModelName,
      selectedModelNames,
      setSelectedModelName,
      t,
    ],
  );

  const handleAddModel = () => {
    if (addModel(newModelName)) {
      setNewModelName('');
      setAddVisible(false);
    }
  };

  const rowSelection = {
    selectedRowKeys: selectedModelNames,
    onChange: (selectedRowKeys) => setSelectedModelNames(selectedRowKeys),
  };

  return (
    <>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space wrap className='mt-2'>
          {allowAddModel ? (
            <Button
              icon={<IconPlus />}
              onClick={() => setAddVisible(true)}
              style={isMobile ? { width: '100%' } : undefined}
            >
              {t('添加模型')}
            </Button>
          ) : null}
          <Button
            type='primary'
            icon={<IconSave />}
            loading={loading}
            onClick={handleSubmit}
            style={isMobile ? { width: '100%' } : undefined}
          >
            {t('应用更改')}
          </Button>
          <Button
            disabled={!selectedModel || selectedModelNames.length === 0}
            onClick={() => setBatchVisible(true)}
            style={isMobile ? { width: '100%' } : undefined}
          >
            {t('批量应用当前模型价格')}
            {selectedModelNames.length > 0
              ? ` (${selectedModelNames.length})`
              : ''}
          </Button>
          <Input
            prefix={<IconSearch />}
            placeholder={t('搜索模型名称')}
            value={searchText}
            onChange={(value) => setSearchText(value)}
            style={{ width: isMobile ? '100%' : 220 }}
            showClear
          />
          {showConflictFilter ? (
            <Checkbox
              checked={conflictOnly}
              onChange={(event) => setConflictOnly(event.target.checked)}
            >
              {t('仅显示矛盾倍率')}
            </Checkbox>
          ) : null}
        </Space>

        {listDescription ? (
          <div className='text-sm text-gray-500'>{listDescription}</div>
        ) : null}
        {selectedModelNames.length > 0 ? (
          <div
            style={{
              width: '100%',
              padding: '10px 12px',
              borderRadius: 8,
              background: 'var(--semi-color-primary-light-default)',
              border: '1px solid var(--semi-color-primary)',
              color: 'var(--semi-color-primary)',
              fontWeight: 600,
            }}
          >
            {t('已勾选 {{count}} 个模型', { count: selectedModelNames.length })}
          </div>
        ) : null}

        <div
          style={{
            width: '100%',
            display: 'grid',
            gap: 16,
            gridTemplateColumns: isMobile
              ? 'minmax(0, 1fr)'
              : 'minmax(300px, 0.8fr) minmax(480px, 1.2fr)',
          }}
        >
          <Card
            bodyStyle={{ padding: 0 }}
            style={isMobile ? { order: 2 } : undefined}
          >
            <div style={{ overflowX: 'auto' }}>
              <Table
                columns={columns}
                dataSource={pagedData}
                rowKey='name'
                rowSelection={rowSelection}
                pagination={{
                  currentPage,
                  pageSize: PAGE_SIZE,
                  total: filteredModels.length,
                  onPageChange: (page) => setCurrentPage(page),
                  showTotal: true,
                  showSizeChanger: false,
                }}
                empty={
                  <div style={{ textAlign: 'center', padding: '20px' }}>
                    {emptyTitle || t('暂无模型')}
                  </div>
                }
                onRow={(record) => ({
                  style: {
                    background: selectedModelNames.includes(record.name)
                      ? 'var(--semi-color-success-light-default)'
                      : record.name === selectedModelName
                        ? 'var(--semi-color-primary-light-default)'
                        : undefined,
                    boxShadow: selectedModelNames.includes(record.name)
                      ? 'inset 4px 0 0 var(--semi-color-success)'
                      : record.name === selectedModelName
                        ? 'inset 4px 0 0 var(--semi-color-primary)'
                        : undefined,
                    transition: 'background 0.2s ease, box-shadow 0.2s ease',
                  },
                  onClick: () => setSelectedModelName(record.name),
                })}
                scroll={isMobile ? { x: 720 } : undefined}
              />
            </div>
          </Card>

          <Card
            style={isMobile ? { order: 1 } : undefined}
            title={selectedModel ? selectedModel.name : t('模型计费编辑器')}
            headerExtraContent={
              selectedModel ? (
                <Tag
                  color={
                    selectedModel.billingMode === 'per-request'
                      ? 'teal'
                      : selectedModel.billingMode === 'tiered_expr'
                        ? 'amber'
                        : 'blue'
                  }
                >
                  {selectedModel.billingMode === 'per-request'
                    ? t('按次计费')
                    : selectedModel.billingMode === 'tiered_expr'
                      ? getExprModeLabel(selectedModel)
                      : t('按量计费')}
                </Tag>
              ) : null
            }
          >
            {!selectedModel ? (
              <Empty
                title={emptyTitle || t('暂无模型')}
                description={
                  emptyDescription || t('请先新增模型或从左侧列表选择一个模型')
                }
              />
            ) : (
              <div>
                <div className='mb-4'>
                  <div className='mb-2 font-medium text-gray-700'>
                    {t('计费方式')}
                  </div>
                  <RadioGroup
                    type='button'
                    value={selectedModel.billingMode}
                    onChange={(event) =>
                      handleBillingModeChange(event.target.value)
                    }
                  >
                    <Radio value='per-token'>{t('按量计费')}</Radio>
                    <Radio value='per-request'>{t('按次计费')}</Radio>
                    <Radio value='tiered_expr'>{t('表达式/阶梯计费')}</Radio>
                  </RadioGroup>
                  <div className='mt-2 text-xs text-gray-500'>
                    {t(
                      '普通按量/按次直接填价格就行；如果价格要跟请求参数或请求头联动，请切到表达式/阶梯计费。',
                    )}
                  </div>
                </div>

                {selectedWarnings.length > 0 ? (
                  <Card
                    bodyStyle={{ padding: 12 }}
                    style={{
                      marginBottom: 16,
                      background: 'var(--semi-color-warning-light-default)',
                    }}
                  >
                    <div className='font-medium mb-2'>{t('当前提示')}</div>
                    {selectedWarnings.map((warning) => (
                      <div key={warning} className='text-sm text-gray-700 mb-1'>
                        {warning}
                      </div>
                    ))}
                  </Card>
                ) : null}

                {selectedModel.billingMode === 'per-request' ? (
                  <PerRequestPricingEditor
                    model={selectedModel}
                    onPriceChange={handleNumericFieldChange}
                    onSubtypeChange={handlePerRequestSubtypeChange}
                    onRuleChange={handlePerRequestRuleChange}
                    t={t}
                  />
                ) : selectedModel.billingMode === 'tiered_expr' ? (
                  <TieredPricingEditor
                    model={selectedModel}
                    onExprChange={handleBillingExprChange}
                    requestRuleExpr={selectedModel.requestRuleExpr}
                    onRequestRuleExprChange={handleRequestRuleExprChange}
                    t={t}
                  />
                ) : (
                  <>
                    <Card
                      bodyStyle={{ padding: 16 }}
                      style={{
                        marginBottom: 16,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <div className='font-medium mb-3'>{t('基础价格')}</div>
                      <PriceInput
                        label={t('输入价格')}
                        value={selectedModel.inputPrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('inputPrice', value)
                        }
                      />
                      {selectedModel.completionRatioLocked ? (
                        <Banner
                          type='warning'
                          bordered
                          fullMode={false}
                          closeIcon={null}
                          style={{ marginBottom: 12 }}
                          title={t('补全价格已锁定')}
                          description={t(
                            '该模型补全倍率由后端固定为 {{ratio}}。补全价格不能在这里修改。',
                            {
                              ratio: selectedModel.lockedCompletionRatio || '-',
                            },
                          )}
                        />
                      ) : null}
                      <PriceInput
                        label={t('补全价格')}
                        value={selectedModel.completionPrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('completionPrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'completionPrice',
                            )}
                            disabled={selectedModel.completionRatioLocked}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'completionPrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'completionPrice',
                          )
                        }
                        disabled={
                          !hasValue(selectedModel.inputPrice) ||
                          selectedModel.completionRatioLocked
                        }
                        extraText={
                          selectedModel.completionRatioLocked
                            ? t(
                                '后端固定倍率：{{ratio}}。该字段仅展示换算后的价格。',
                                {
                                  ratio:
                                    selectedModel.lockedCompletionRatio || '-',
                                },
                              )
                            : !isOptionalFieldEnabled(
                                  selectedModel,
                                  'completionPrice',
                                )
                              ? t('当前未启用，需要时再打开即可。')
                              : ''
                        }
                      />
                      <PriceInput
                        label={t('缓存读取价格')}
                        value={selectedModel.cachePrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('cachePrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'cachePrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle('cachePrice', checked)
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(selectedModel, 'cachePrice')
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(selectedModel, 'cachePrice')
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                      <PriceInput
                        label={t('缓存创建价格')}
                        value={selectedModel.createCachePrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('createCachePrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'createCachePrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'createCachePrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'createCachePrice',
                          )
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'createCachePrice',
                          )
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                    </Card>

                    <Card
                      bodyStyle={{ padding: 16 }}
                      style={{
                        marginBottom: 16,
                        background: 'var(--semi-color-fill-0)',
                      }}
                    >
                      <div className='mb-3'>
                        <div className='font-medium'>{t('扩展价格')}</div>
                        <div className='text-xs text-gray-500 mt-1'>
                          {t('这些价格都是可选项，不填也可以。')}
                        </div>
                      </div>
                      <PriceInput
                        label={t('图片输入价格')}
                        value={selectedModel.imagePrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('imagePrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'imagePrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle('imagePrice', checked)
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(selectedModel, 'imagePrice')
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(selectedModel, 'imagePrice')
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                      <PriceInput
                        label={t('音频输入价格')}
                        value={selectedModel.audioInputPrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('audioInputPrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'audioInputPrice',
                            )}
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'audioInputPrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioInputPrice',
                          )
                        }
                        disabled={!hasValue(selectedModel.inputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioInputPrice',
                          )
                            ? t('当前未启用，需要时再打开即可。')
                            : ''
                        }
                      />
                      <PriceInput
                        label={t('音频补全价格')}
                        value={selectedModel.audioOutputPrice}
                        placeholder={t('输入 $/1M tokens')}
                        onChange={(value) =>
                          handleNumericFieldChange('audioOutputPrice', value)
                        }
                        headerAction={
                          <Switch
                            size='small'
                            checked={isOptionalFieldEnabled(
                              selectedModel,
                              'audioOutputPrice',
                            )}
                            disabled={
                              !isOptionalFieldEnabled(
                                selectedModel,
                                'audioInputPrice',
                              )
                            }
                            onChange={(checked) =>
                              handleOptionalFieldToggle(
                                'audioOutputPrice',
                                checked,
                              )
                            }
                          />
                        }
                        hidden={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioOutputPrice',
                          )
                        }
                        disabled={!hasValue(selectedModel.audioInputPrice)}
                        extraText={
                          !isOptionalFieldEnabled(
                            selectedModel,
                            'audioInputPrice',
                          )
                            ? t('请先开启并填写音频输入价格。')
                            : !isOptionalFieldEnabled(
                                  selectedModel,
                                  'audioOutputPrice',
                                )
                              ? t('当前未启用，需要时再打开即可。')
                              : ''
                        }
                      />
                    </Card>
                  </>
                )}

                <Card
                  bodyStyle={{ padding: 16 }}
                  style={{ background: 'var(--semi-color-fill-0)' }}
                >
                  <div className='font-medium mb-3'>{t('保存预览')}</div>
                  <div className='text-xs text-gray-500 mb-3'>
                    {t(
                      '下面展示这个模型保存后会写入哪些后端字段，便于和原始 JSON 编辑框保持一致。',
                    )}
                  </div>
                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns: 'minmax(140px, 180px) 1fr',
                      gap: 8,
                    }}
                  >
                    {previewRows.map((row) => (
                      <React.Fragment key={row.key}>
                        <Text strong>{row.label}</Text>
                        <Text>{row.value}</Text>
                      </React.Fragment>
                    ))}
                  </div>
                </Card>
              </div>
            )}
          </Card>
        </div>
      </Space>

      {allowAddModel ? (
        <Modal
          title={t('添加模型')}
          visible={addVisible}
          onCancel={() => {
            setAddVisible(false);
            setNewModelName('');
          }}
          onOk={handleAddModel}
        >
          <Input
            value={newModelName}
            placeholder={t('输入模型名称，例如 gpt-4.1')}
            onChange={(value) => setNewModelName(value)}
          />
        </Modal>
      ) : null}

      <Modal
        title={t('批量应用当前模型价格')}
        visible={batchVisible}
        onCancel={() => setBatchVisible(false)}
        onOk={() => {
          if (applySelectedModelPricing()) {
            setBatchVisible(false);
          }
        }}
      >
        <div className='text-sm text-gray-600'>
          {selectedModel
            ? t(
                '将把当前编辑中的模型 {{name}} 的价格配置，批量应用到已勾选的 {{count}} 个模型。',
                {
                  name: selectedModel.name,
                  count: selectedModelNames.length,
                },
              )
            : t('请先选择一个作为模板的模型')}
        </div>
        {selectedModel ? (
          <div className='text-xs text-gray-500 mt-3'>
            {t(
              '适合同系列模型一起定价，例如把 gpt-5.1 的价格批量同步到 gpt-5.1-high、gpt-5.1-low 等模型。',
            )}
          </div>
        ) : null}
      </Modal>
    </>
  );
}
