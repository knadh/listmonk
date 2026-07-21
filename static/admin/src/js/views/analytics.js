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
            const total = data.data.reduce((acc, val) => acc + val, 0);
            const val = data.data[item.dataIndex];
            const percentage = ((val / total) * 100).toFixed(2);
            return `${val} (${percentage}%)`;
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

// CampaignTag is the object used in the <ot-taginput> campaign selector.
class CampaignTag {
  constructor(c) {
    this.id = c.id;
    this.name = c.name;
  }

  toString() {
    return `#${this.id}: ${this.name}`;
  }
}

// analyticsForm is the Alpine component for the campaign analytics page.
function analyticsForm() {
  let queryTimer = null;

  return {
    campaigns: (window._selectedCampaigns || []).map((c) => new CampaignTag(c)),

    // Set on submit to give feedback during the full-page reload.
    submitting: false,

    // Populate the selector's suggestions with the campaigns API.
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
          const data = await api('analytics.campaigns', `/campaigns?${params.toString()}`, 'GET');
          el.list.replaceChildren(...(data.results || [])
            .filter((c) => !chosen.has(c.id))
            .map((c) => {
              const o = new Option(`#${c.id}: ${c.name}`);
              o.data = new CampaignTag(c);
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

  // eslint-disable-next-line no-new
  new window.Chart(canvas, {
    type: def.type,
    data,
    options: { ...def.options, ...(extraOptions || {}) },
  });
}

// Render a line chart (time series per campaign) and its donut (totals per campaign).
function renderCounts(key, camps, rows) {
  const datasets = camps.map((c, n) => ({
    label: `#${c.id}: ${c.name}`,
    data: rows.filter((item) => item.campaign_id === c.id)
      .map((item) => ({ x: toLabel(item.timestamp), y: item.count })),
    borderColor: chartColors[n % chartColors.length],
    borderWidth: 2,
    pointHoverBorderWidth: 5,
    pointBorderWidth: 0.5,
  }));

  const labels = [];
  const totals = camps.map((c) => {
    labels.push(`#${c.id}: ${c.name}`);
    return rows.reduce((a, item) => (item.campaign_id === c.id ? a + item.count : a), 0);
  });

  draw(`chart-${key}`, DEFAULT_LINE, { datasets });
  draw(`chart-${key}-donut`, DEFAULT_DONUT, {
    labels,
    datasets: [{ data: totals, backgroundColor: chartColors, borderWidth: 6 }],
  });
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
  if (!d || !window.Chart) {
    return;
  }

  const camps = d.campaigns || [];
  renderCounts('views', camps, d.views || []);
  renderCounts('clicks', camps, d.clicks || []);
  renderCounts('bounces', camps, d.bounces || []);
  renderLinks(d.links || []);
}

document.addEventListener('alpine:init', () => {
  window.Alpine.data('analyticsForm', analyticsForm);
}, { once: true });

renderCharts();
