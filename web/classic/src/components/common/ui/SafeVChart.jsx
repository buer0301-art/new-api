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

import React, { useMemo } from 'react';

const PALETTE = [
  '#3b82f6',
  '#ef4444',
  '#10b981',
  '#f59e0b',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
  '#f97316',
  '#6366f1',
  '#14b8a6',
];

const numberValue = (value) => {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? numeric : 0;
};

const getValues = (spec) => spec?.data?.[0]?.values || [];

const getColor = (spec, key, index) => {
  const specified = spec?.color?.specified;
  if (specified?.[key]) {
    return specified[key];
  }
  const range = spec?.color?.range;
  if (Array.isArray(range) && range.length > 0) {
    return range[index % range.length];
  }
  return PALETTE[index % PALETTE.length];
};

const groupBy = (values, field) =>
  values.reduce((groups, item) => {
    const key = String(item[field] ?? '');
    if (!groups.has(key)) {
      groups.set(key, []);
    }
    groups.get(key).push(item);
    return groups;
  }, new Map());

const buildLinePath = (points) =>
  points
    .map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x} ${point.y}`)
    .join(' ');

const ChartTitle = ({ spec }) => {
  if (!spec?.title?.visible) {
    return null;
  }

  return (
    <div className='mb-2 flex items-baseline gap-3'>
      <div className='text-sm font-medium text-gray-700'>{spec.title.text}</div>
      {spec.title.subtext && (
        <div className='text-xs text-gray-400'>{spec.title.subtext}</div>
      )}
    </div>
  );
};

const EmptyChart = () => (
  <div className='flex h-full w-full items-center justify-center text-xs text-gray-400'>
    暂无数据
  </div>
);

const MiniLineChart = ({ spec, values }) => {
  const width = 100;
  const height = 40;
  const yField = spec.yField || 'y';
  const color = spec?.line?.style?.stroke || PALETTE[0];
  const maxValue = Math.max(...values.map((item) => numberValue(item[yField])), 1);
  const minValue = Math.min(...values.map((item) => numberValue(item[yField])), 0);
  const spread = maxValue - minValue || 1;
  const points = values.map((item, index) => ({
    x: values.length === 1 ? width / 2 : (index / (values.length - 1)) * width,
    y: height - ((numberValue(item[yField]) - minValue) / spread) * (height - 6) - 3,
  }));

  return (
    <svg className='h-full w-full' viewBox={`0 0 ${width} ${height}`}>
      <path d={buildLinePath(points)} fill='none' stroke={color} strokeWidth='2' />
    </svg>
  );
};

const LineChart = ({ spec, values, width, height, padding }) => {
  const xField = spec.xField || 'x';
  const yField = spec.yField || 'y';
  const seriesField = spec.seriesField;
  const groups = seriesField ? groupBy(values, seriesField) : new Map([['', values]]);
  const xValues = Array.from(new Set(values.map((item) => String(item[xField] ?? ''))));
  const maxValue = Math.max(...values.map((item) => numberValue(item[yField])), 1);
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;

  return (
    <svg className='h-full w-full' viewBox={`0 0 ${width} ${height}`}>
      <line x1={padding.left} y1={height - padding.bottom} x2={width - padding.right} y2={height - padding.bottom} stroke='#e5e7eb' />
      <line x1={padding.left} y1={padding.top} x2={padding.left} y2={height - padding.bottom} stroke='#e5e7eb' />
      {Array.from(groups.entries()).map(([series, items], groupIndex) => {
        const itemMap = new Map(items.map((item) => [String(item[xField] ?? ''), item]));
        const points = xValues.map((xValue, index) => {
          const item = itemMap.get(xValue);
          const value = numberValue(item?.[yField]);
          return {
            x: padding.left + (xValues.length === 1 ? plotWidth / 2 : (index / (xValues.length - 1)) * plotWidth),
            y: padding.top + plotHeight - (value / maxValue) * plotHeight,
          };
        });
        return (
          <path
            key={series}
            d={buildLinePath(points)}
            fill='none'
            stroke={getColor(spec, series, groupIndex)}
            strokeWidth='2'
          />
        );
      })}
      {xValues.slice(0, 6).map((label, index) => (
        <text
          key={`${label}-${index}`}
          x={padding.left + (xValues.length === 1 ? plotWidth / 2 : (index / Math.max(xValues.slice(0, 6).length - 1, 1)) * plotWidth)}
          y={height - 12}
          textAnchor='middle'
          fontSize='11'
          fill='#9ca3af'
        >
          {label}
        </text>
      ))}
    </svg>
  );
};

const BarChart = ({ spec, values, width, height, padding }) => {
  const xField = spec.xField || 'x';
  const yField = spec.yField || 'y';
  const horizontal = spec.direction === 'horizontal';
  const valueField = horizontal ? xField : yField;
  const labelField = horizontal ? yField : xField;
  const maxValue = Math.max(
    ...values.map((item) => numberValue(item[valueField])),
    1,
  );
  const shownValues = values.slice(0, horizontal ? 10 : 16);
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const barGap = 6;
  const barSize = Math.max(
    4,
    (horizontal ? plotHeight : plotWidth) / Math.max(shownValues.length, 1) - barGap,
  );

  return (
    <svg className='h-full w-full' viewBox={`0 0 ${width} ${height}`}>
      <line x1={padding.left} y1={height - padding.bottom} x2={width - padding.right} y2={height - padding.bottom} stroke='#e5e7eb' />
      <line x1={padding.left} y1={padding.top} x2={padding.left} y2={height - padding.bottom} stroke='#e5e7eb' />
      {shownValues.map((item, index) => {
        const key = String(item[labelField] ?? item.Model ?? item.User ?? index);
        const value = numberValue(item[valueField]);
        const color = getColor(spec, key, index);
        if (horizontal) {
          const y = padding.top + index * (barSize + barGap);
          const barWidth = (value / maxValue) * plotWidth;
          return (
            <g key={`${key}-${index}`}>
              <rect x={padding.left} y={y} width={barWidth} height={barSize} rx='3' fill={color} />
              <text x={padding.left - 8} y={y + barSize / 2 + 4} textAnchor='end' fontSize='11' fill='#6b7280'>
                {key.slice(0, 14)}
              </text>
            </g>
          );
        }
        const x = padding.left + index * (barSize + barGap);
        const barHeight = (value / maxValue) * plotHeight;
        return (
          <g key={`${key}-${index}`}>
            <rect x={x} y={padding.top + plotHeight - barHeight} width={barSize} height={barHeight} rx='3' fill={color} />
            <text x={x + barSize / 2} y={height - 12} textAnchor='middle' fontSize='10' fill='#9ca3af'>
              {key.slice(0, 8)}
            </text>
          </g>
        );
      })}
    </svg>
  );
};

const PieChart = ({ spec, values, width, height }) => {
  const valueField = spec.valueField || 'value';
  const categoryField = spec.categoryField || 'type';
  const total = values.reduce((sum, item) => sum + numberValue(item[valueField]), 0);
  const radius = Math.min(width, height) * 0.28;
  const cx = width * 0.42;
  const cy = height * 0.52;
  let currentAngle = -Math.PI / 2;

  return (
    <svg className='h-full w-full' viewBox={`0 0 ${width} ${height}`}>
      {values.map((item, index) => {
        const value = numberValue(item[valueField]);
        const angle = total > 0 ? (value / total) * Math.PI * 2 : 0;
        const start = currentAngle;
        const end = currentAngle + angle;
        currentAngle = end;
        const largeArc = angle > Math.PI ? 1 : 0;
        const x1 = cx + radius * Math.cos(start);
        const y1 = cy + radius * Math.sin(start);
        const x2 = cx + radius * Math.cos(end);
        const y2 = cy + radius * Math.sin(end);
        const category = String(item[categoryField] ?? index);
        const path = `M ${cx} ${cy} L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArc} 1 ${x2} ${y2} Z`;
        return <path key={`${category}-${index}`} d={path} fill={getColor(spec, category, index)} />;
      })}
      {values.slice(0, 8).map((item, index) => {
        const category = String(item[categoryField] ?? index);
        return (
          <g key={`legend-${category}`} transform={`translate(${width * 0.72}, ${paddingTopForLegend(index)})`}>
            <rect width='10' height='10' rx='2' fill={getColor(spec, category, index)} />
            <text x='16' y='9' fontSize='11' fill='#6b7280'>{category.slice(0, 18)}</text>
          </g>
        );
      })}
    </svg>
  );
};

const paddingTopForLegend = (index) => 58 + index * 22;

const SafeVChart = ({ spec, className = '', style }) => {
  const chart = useMemo(() => {
    const values = getValues(spec).filter((item) => item && typeof item === 'object');
    const hasData = values.some((item) => {
      const field = spec?.yField || spec?.valueField || 'value';
      return numberValue(item[field]) > 0;
    });

    return { values, hasData };
  }, [spec]);

  if (!spec) {
    return <EmptyChart />;
  }

  const isMini = spec.height === 40 || spec.width === 100;
  const width = isMini ? 100 : 640;
  const height = isMini ? 40 : 320;
  const padding = {
    top: 24,
    right: 28,
    bottom: 38,
    left: spec.direction === 'horizontal' ? 110 : 44,
  };

  if (!chart.values.length || !chart.hasData) {
    return (
      <div className={`h-full w-full ${className}`} style={style}>
        {!isMini && <ChartTitle spec={spec} />}
        <EmptyChart />
      </div>
    );
  }

  return (
    <div className={`h-full w-full ${className}`} style={style}>
      {!isMini && <ChartTitle spec={spec} />}
      <div className={isMini ? 'h-full w-full' : 'h-[calc(100%-32px)] w-full'}>
        {isMini && <MiniLineChart spec={spec} values={chart.values} />}
        {!isMini && spec.type === 'pie' && (
          <PieChart spec={spec} values={chart.values} width={width} height={height} />
        )}
        {!isMini && (spec.type === 'bar') && (
          <BarChart spec={spec} values={chart.values} width={width} height={height} padding={padding} />
        )}
        {!isMini && (spec.type === 'line' || spec.type === 'area') && (
          <LineChart spec={spec} values={chart.values} width={width} height={height} padding={padding} />
        )}
      </div>
    </div>
  );
};

export default SafeVChart;
