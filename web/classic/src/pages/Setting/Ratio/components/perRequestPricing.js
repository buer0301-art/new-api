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

export const PER_REQUEST_RULES_KEY = 'per_request_pricing.rules';

const MEDIA_CONFIG = {
  image: {
    unit: 'image',
    defaultResolution: '',
    label: 'image',
  },
  video: {
    unit: 'second',
    defaultResolution: '',
    label: 'video',
  },
};

const isPlainObject = (value) =>
  !!value && typeof value === 'object' && !Array.isArray(value);

const toTrimmedString = (value) =>
  typeof value === 'string' ? value.trim() : '';

const toFiniteNumber = (value) => {
  if (typeof value === 'string' && value.trim() === '') {
    return null;
  }
  const parsed = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

const normalizeResolutionKey = (value) =>
  value.trim().replace(/\s+/g, '').toLowerCase().replace(/\*/g, 'x');

const findConfiguredResolution = (resolutions, targetResolution) => {
  const targetKey = normalizeResolutionKey(targetResolution);
  if (!targetKey) return '';
  return (
    resolutions.find(
      (resolution) => normalizeResolutionKey(resolution) === targetKey,
    ) || ''
  );
};

const createPriceRowId = () => {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2);
};

const normalizeRule = (rule) => {
  const mediaType =
    rule?.media_type === 'image' || rule?.media_type === 'video'
      ? rule.media_type
      : undefined;
  const unit =
    rule?.unit === 'image' ||
    rule?.unit === 'second' ||
    rule?.unit === 'request'
      ? rule.unit
      : undefined;
  const prices = {};

  if (isPlainObject(rule?.prices)) {
    Object.entries(rule.prices).forEach(([resolution, value]) => {
      const normalizedResolution = toTrimmedString(resolution);
      const normalizedPrice = toFiniteNumber(value);
      if (normalizedResolution && normalizedPrice !== null) {
        prices[normalizedResolution] = normalizedPrice;
      }
    });
  }

  const normalized = {
    media_type: mediaType || 'image',
    unit: unit || 'image',
    prices,
    default_resolution: toTrimmedString(rule?.default_resolution),
    fallback_enabled: Boolean(rule?.fallback_enabled),
  };

  if (mediaType) {
    normalized.media_type = mediaType;
    if (mediaType === 'image') {
      normalized.unit = MEDIA_CONFIG[mediaType].unit;
    } else {
      normalized.unit =
        unit === 'second' || unit === 'request'
          ? unit
          : MEDIA_CONFIG[mediaType].unit;
    }
  } else if (unit) {
    normalized.unit = unit;
  }

  return normalized;
};

export function parsePerRequestRules(value) {
  const raw = (value ?? '').trim();
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw);
    if (!isPlainObject(parsed)) return {};
    const result = {};
    Object.entries(parsed).forEach(([model, rule]) => {
      const trimmedModel = toTrimmedString(model);
      if (!trimmedModel || !isPlainObject(rule)) return;
      result[trimmedModel] = normalizeRule(rule);
    });
    return result;
  } catch {
    return {};
  }
}

export function stringifyPerRequestRules(rules) {
  const result = {};
  Object.entries(rules || {}).forEach(([model, rule]) => {
    const trimmedModel = toTrimmedString(model);
    if (!trimmedModel) return;

    const normalized = normalizeRule(rule);
    const defaultResolution = findConfiguredResolution(
      Object.keys(normalized.prices),
      normalized.default_resolution,
    );
    if (
      !normalized.media_type ||
      !normalized.unit ||
      Object.keys(normalized.prices).length === 0 ||
      !normalized.default_resolution ||
      !defaultResolution
    ) {
      return;
    }

    result[trimmedModel] = {
      media_type: normalized.media_type,
      unit: normalized.unit,
      prices: normalized.prices,
      default_resolution: defaultResolution,
      fallback_enabled: normalized.fallback_enabled,
    };
  });
  return JSON.stringify(result, null, 2);
}

export function summarizePerRequestRule(rule, t) {
  if (!rule || !rule.media_type) return '';
  const mediaType = rule.media_type;
  const mediaLabel = MEDIA_CONFIG[mediaType]?.label || mediaType;
  const unitLabel =
    mediaType === 'image'
      ? t?.('张') || 'image'
      : rule.unit === 'request'
        ? t?.('次') || 'request'
        : t?.('秒') || 's';
  const entries = Object.entries(rule.prices || {})
    .map(
      ([resolution, price]) =>
        `${resolution} $${Number(price)
          .toFixed(3)
          .replace(/\.?0+$/, '')}/${unitLabel}`,
    )
    .filter(Boolean);

  return entries.length > 0
    ? `${mediaLabel[0].toUpperCase()}${mediaLabel.slice(1)} · ${entries.join(
        ' · ',
      )}`
    : mediaLabel;
}

export function createDefaultPriceRows(mediaType) {
  if (!MEDIA_CONFIG[mediaType]) return [];
  return [
    {
      id: createPriceRowId(),
      resolution: '',
      price: '',
      enabled: true,
    },
  ];
}

export function createPriceRowsFromResolutions(resolutions) {
  return (resolutions || []).map((resolution) => ({
    id: createPriceRowId(),
    resolution,
    price: '',
    enabled: true,
  }));
}

export function createPriceRowsFromRule(mediaType, rule) {
  if (rule?.media_type === mediaType && isPlainObject(rule.prices)) {
    return Object.entries(rule.prices).map(([resolution, price]) => ({
      id: createPriceRowId(),
      resolution,
      price: String(price),
      enabled: true,
    }));
  }
  return createDefaultPriceRows(mediaType);
}

export function createEmptyPriceRow() {
  return {
    id: createPriceRowId(),
    resolution: '',
    price: '',
    enabled: true,
  };
}

export function getConfiguredDefaultResolution(mediaType, rule) {
  if (rule?.media_type === mediaType && rule.default_resolution) {
    return rule.default_resolution;
  }
  return MEDIA_CONFIG[mediaType]?.defaultResolution || '';
}

export function buildRuleFromRows(
  mediaType,
  rows,
  defaultResolution,
  unit = MEDIA_CONFIG[mediaType]?.unit,
) {
  const config = MEDIA_CONFIG[mediaType];
  if (!config) return null;

  const normalizedPrices = {};
  const pricedRows = [];
  const seen = new Set();

  rows.forEach((row) => {
    if (!row.enabled) return;
    const resolution = toTrimmedString(row.resolution);
    const resolutionKey = normalizeResolutionKey(resolution);
    if (!resolution || seen.has(resolutionKey)) return;
    seen.add(resolutionKey);
    const price = toFiniteNumber(row.price);
    if (price !== null && price >= 0) {
      normalizedPrices[resolution] = price;
      pricedRows.push(resolution);
    }
  });

  if (Object.keys(normalizedPrices).length === 0) return null;

  const fallbackResolution =
    findConfiguredResolution(pricedRows, defaultResolution) ||
    pricedRows[0];
  if (!fallbackResolution) return null;

  return {
    media_type: mediaType,
    unit:
      mediaType === 'video' && (unit === 'second' || unit === 'request')
        ? unit
        : config.unit,
    prices: normalizedPrices,
    default_resolution: fallbackResolution,
    fallback_enabled: false,
  };
}
