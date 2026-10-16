import Alpine from 'alpinejs';
import Chart from 'chart.js/auto';

import { api } from '../main.js';

const chartColorRed = '#ee7d5b';
const chartColors = [
  '#0055d4',
  '#FFB50D',
  '#41AC9C',
  chartColorRed,
  '#7FC7BC',
  '#3a82d6',
  '#688ED9',
  '#FFC43D',
];

const DEFAULT_DONUT = {
  type: 'doughnut',
  options: {
    responsive: true,
    cutout: '70%',
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: '#fff',
        borderColor: '#ddd',
        borderWidth: 1,
        titleColor: '#666',
        bodyColor: '#666',
        bodyFont: { size: 15 },
        bodySpacing: 10,
        padding: 10,
        callbacks: {
          label: (item) => {
            const data = item.chart.data.datasets[item.datasetIndex];
            return formatCount(data.data[item.dataIndex], data.sent[item.dataIndex]);
          },
        },
      },
    },
  },
};

const DEFAULT_LINE = {
  type: 'line',
  options: {
    responsive: true,
    lineTension: 0.5,
    maintainAspectRatio: false,
    interaction: { intersect: false, axis: 'index' },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: '#fff',
        borderColor: '#ddd',
        borderWidth: 1,
        bodyColor: '#666',
        displayColors: true,
        bodyFont: { size: 15 },
        bodySpacing: 10,
        padding: 10,
        callbacks: {
          labelColor: (item) => ({
            backgroundColor: item.dataset.borderColor,
            borderColor: 'transparent',
          }),
        },
      },
    },
    scales: {
      x: { grid: { display: false } },
      y: { grid: { display: false }, ticks: { precision: 0 } },
    },
  },
};

const DEFAULT_BAR = {
  type: 'bar',
  options: {
    responsive: true,
    indexAxis: 'y',
    barThickness: 40,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: '#fff',
        borderColor: '#ddd',
        borderWidth: 1,
        titleColor: '#666',
        bodyColor: '#666',
        bodyFont: { size: 15 },
        bodySpacing: 10,
        padding: 10,
      },
    },
    scales: {
      x: { grid: { display: false } },
      y: { grid: { display: false } },
    },
  },
};

// Zero-pad a number to two digits.
const pad = (n) => String(n).padStart(2, '0');

// Format a timestamp string into a chart axis label.
function toLabel(s) {
  const d = new Date(s);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// AnalyticsTag is the object used in the <ot-taginput> analytics selector.
class AnalyticsTag {
  constructor(c) {
    this.id = c.id;
    this.name = c.name;
  }

  toString() {
    return `#${this.id}: ${this.name}`;
  }
}

// analyticsForm is the Alpine component for the analytics page.
function analyticsForm(from, to) {
  let queryTimer = null;

  return {
    items: (window._selectedItems || []).map((c) => new AnalyticsTag(c)),

    // Set on submit to give feedback during the full-page reload.
    submitting: false,

    from,
    to,

    get rangeLabel() {
      return [this.from, this.to]
        .map((value) => value ? value.slice(0, 16).replace('T', ' ') : '…')
        .join(' – ');
    },

    onDateRange(amount, unit) {
      const to = new Date();
      const from = new Date(to);
      if (unit === 'month') {
        // Clamp to the last day when the target month is short.
        from.setDate(1);
        from.setMonth(from.getMonth() - amount);

        const lastDay = new Date(from.getFullYear(), from.getMonth() + 1, 0).getDate();
        from.setDate(Math.min(to.getDate(), lastDay));
      } else if (unit === 'day') {
        from.setDate(from.getDate() - amount);
      } else {
        from.setTime(from.getTime() - amount * 60 * 60 * 1000);
      }

      const localDate = (date) => new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
      this.from = localDate(from);
      this.to = localDate(to);
      this.$refs.dates.hidePopover();
      this.$nextTick(() => this.$root.requestSubmit());
    },

    // Populate the selector's suggestions with the resource API.
    onQuery(el) {
      clearTimeout(queryTimer);
      queryTimer = setTimeout(async () => {
        const ti = el.closest('ot-taginput');
        const chosen = new Set((ti ? ti.value : []).map((c) => c.id));
        const params = new URLSearchParams({
          query: el.value.trim(),
          order_by: 'created_at',
          order: 'DESC',
          no_body: 'true',
          per_page: '10',
        });

        try {
          const data = await api(`analytics.${window._analyticsResource}`, `/${window._analyticsResource}?${params.toString()}`, 'GET');
          el.list.replaceChildren(...(data.results || [])
            .filter((c) => !chosen.has(c.id))
            .map((c) => {
              const o = new Option(`#${c.id}: ${c.name}`);
              o.data = new AnalyticsTag(c);
              return o;
            }));
        } catch (err) { /* api() shows a toast. */ }
      }, 200);
    },
  };
}

// Create a Chart.js instance on the canvas with the given id.
function draw(id, def, data, extraOptions) {
  const canvas = document.getElementById(id);
  if (!canvas) {
    return;
  }

  return new Chart(canvas, {
    type: def.type,
    data,
    options: { ...def.options, ...(extraOptions || {}) },
  });
}

// Use the campaign's sent count instead of the chart total.
function formatCount(count, sent) {
  return sent > 0 ? `${count} (${((count / sent) * 100).toFixed(2)}%)` : `${count}`;
}

// Group each series once for chart points and legend totals.
function groupCounts(rows, idKey) {
  const groups = new Map();
  rows.forEach((row) => {
    const id = row[idKey];
    if (!groups.has(id)) {
      groups.set(id, { points: [], total: 0 });
    }
    const group = groups.get(id);
    group.points.push({ x: toLabel(row.timestamp), y: row.count });
    group.total += row.count;
  });
  return groups;
}

// Render per-item time series and totals, with optional unsubscription lines.
function renderCounts(key, items, rows, unsubscriptions) {
  const idKey = window._analyticsResource === 'lists' ? 'list_id' : 'campaign_id';
  const counts = groupCounts(rows, idKey);
  const unsubs = groupCounts(unsubscriptions || [], idKey);
  const datasets = items.map((c, n) => ({
    label: `#${c.id}: ${c.name}`,
    data: counts.get(c.id)?.points || [],
    borderColor: chartColors[n % chartColors.length],
    backgroundColor: chartColors[n % chartColors.length],
    borderWidth: 2,
    pointHoverBorderWidth: 5,
    pointBorderWidth: 0.5,
  }));

  const labels = datasets.map((d) => d.label);
  const totals = items.map((c) => counts.get(c.id)?.total || 0);

  if (unsubscriptions) {
    const label = document.getElementById(`chart-${key}`)?.dataset.extraLabel;
    datasets.push(...items.map((c, n) => ({
      ...datasets[n],
      label: `${datasets[n].label}: ${label}`,
      data: unsubs.get(c.id)?.points || [],
      borderDash: [6, 4],
    })));
  }

  draw(`chart-${key}`, DEFAULT_LINE, { datasets });
  const donut = draw(`chart-${key}-donut`, DEFAULT_DONUT, {
    labels,
    datasets: [{
      data: totals,
      sent: items.map((c) => c.sent),
      backgroundColor: chartColors,
      borderWidth: 6,
    }],
  });

  const legend = document.getElementById(`chart-${key}-legend`);
  if (legend) {
    legend.querySelectorAll('[data-index]').forEach((item) => {
      const n = Number(item.dataset.index);
      item.querySelector('[data-marker]').style.color = datasets[n].borderColor;
      item.querySelector('[data-count]').textContent = formatCount(totals[n], items[n].sent);
      const unsubCount = item.querySelector('[data-unsubscription-count]');
      if (unsubCount) {
        unsubCount.textContent = unsubs.get(items[n].id)?.total || 0;
      }

      const highlight = () => {
        donut.setActiveElements(item.matches(':hover, :focus-within') ? [{ datasetIndex: 0, index: n }] : []);
        donut.update('none');
      };
      ['mouseenter', 'mouseleave', 'focusin', 'focusout'].forEach((event) => {
        item.addEventListener(event, highlight);
      });
    });
  }
}

// Render the link-clicks bar chart.
function renderLinks(rows) {
  const urls = rows.map((l) => l.url);
  const labels = rows.map((l) => {
    try {
      const u = new URL(l.url);
      if (l.url.length > 80) {
        return `${u.hostname}${u.pathname.substr(0, 50)}..`;
      }
      return u.hostname + u.pathname;
    } catch {
      return l.url;
    }
  });

  draw('chart-links', DEFAULT_BAR, {
    labels,
    datasets: [{ data: rows.map((l) => l.count), backgroundColor: chartColors }],
  }, {
    onClick: (e) => {
      const bars = e.chart.getElementsAtEventForMode(e, 'nearest', { intersect: true }, true);
      if (bars.length > 0) {
        window.open(urls[bars[0].index], '_blank', 'noopener noreferrer');
      }
    },
  });
}

// Render all charts from the server-embedded analytics data.
function renderCharts() {
  const d = window._analytics;
  if (!d) {
    return;
  }

  const camps = d.items || [];
  renderCounts('subscriptions', camps, d.subscriptions || [], d.unsubscriptions || []);
  renderCounts('views', camps, d.views || []);
  renderCounts('clicks', camps, d.clicks || []);

  if (d.unique_views) {
    renderCounts('unique_views', camps, d.unique_views);
  }

  if (d.unique_clicks) {
    renderCounts('unique_clicks', camps, d.unique_clicks);
  }

  renderCounts('bounces', camps, d.bounces || []);
  renderLinks(d.links || []);
}

document.addEventListener('alpine:init', () => {
  Alpine.data('analyticsForm', analyticsForm);
}, { once: true });

renderCharts();
