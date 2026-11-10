import Chart from 'chart.js/auto';
import { niceDate } from './utils.js';

const FONT_FAMILY = "'Geist', sans-serif";

// Resolve theme variables to a concrete color for canvas rendering.
export function chartThemeColor(name) {
  const el = document.createElement('div');
  el.style.color = `var(--${name})`;
  document.documentElement.appendChild(el);
  const color = getComputedStyle(el).color;
  el.remove();
  return color;
}

// Format date-only buckets without shifting the day in western timezones.
function label(iso) {
  return niceDate(`${iso}T00:00:00`) || iso;
}

// Draw {count, date} rows, optionally adding a dashed comparison series.
export function lineChart(id, rows, comparison) {
  const canvas = document.getElementById(id);
  if (!canvas) {
    return;
  }

  const data = rows || [];
  if (!data.some((d) => d.count > 0) && !comparison?.some((d) => d.count > 0)) {
    const emptyState = canvas.parentElement.querySelector('template');
    if (emptyState) {
      canvas.parentElement.replaceChildren(emptyState.content.cloneNode(true));
      return;
    }
  }

  // Get the primary colour.
  const color = chartThemeColor('primary') || '#0055d4';

  Chart.defaults.font.family = FONT_FAMILY;

  const datasets = [{
    label: canvas.getAttribute('aria-label'),
    data: data.map((d) => d.count),
    borderColor: color,
    backgroundColor: `${color.replace('rgb(', 'rgba(').replace(')', ', 0.0314)')}`,
    borderWidth: 2,
    fill: true,
    tension: 0.35,
    pointBorderWidth: 0.5,
    pointHoverBorderWidth: 5,
  }];
  if (comparison) {
    const counts = new Map(comparison.map((d) => [d.date, d.count]));
    datasets.push({
      ...datasets[0],
      label: canvas.dataset.extraLabel,
      data: data.map((d) => counts.get(d.date) || 0),
      borderDash: [6, 4],
      fill: false,
    });
  }

  // eslint-disable-next-line no-new
  new Chart(canvas, {
    type: 'line',
    data: { labels: data.map((d) => label(d.date)), datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: !!comparison } },
      scales: {
        x: {
          grid: { display: false },
          border: { display: false },
          ticks: { font: { size: 10 }, color: chartThemeColor('muted-foreground') },
        },
        y: { grid: { display: false }, border: { display: false }, ticks: { precision: 0 } },
      },
    },
  });
}

