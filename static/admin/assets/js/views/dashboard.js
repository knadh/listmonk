import Chart from 'chart.js/auto';

const PRIMARY = getComputedStyle(document.documentElement).getPropertyValue('--primary').trim() || '#0055d4';
const FONT_FAMILY = "'Inter', sans-serif";

// Format an ISO date ('2024-06-01') to a short 'DD Mon' label.
function label(iso) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }

  return d.toLocaleDateString(undefined, { day: '2-digit', month: 'short' });
}

// Draw a filled line chart of {count, date} rows on the given canvas.
function lineChart(id, rows) {
  const canvas = document.getElementById(id);
  if (!canvas) {
    return;
  }

  const data = rows || [];
  Chart.defaults.font.family = FONT_FAMILY;

  // eslint-disable-next-line no-new
  new Chart(canvas, {
    type: 'line',
    data: {
      labels: data.map((d) => label(d.date)),
      datasets: [{
        data: data.map((d) => d.count),
        borderColor: PRIMARY,
        backgroundColor: `${PRIMARY}08`,
        borderWidth: 2,
        fill: true,
        tension: 0.35,
        pointBorderWidth: 0.5,
        pointHoverBorderWidth: 5,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { grid: { display: false }, border: { display: false } },
        y: { grid: { display: false }, border: { display: false }, ticks: { precision: 0 } },
      },
    },
  });
}

(() => {
  const d = window._dashboardCharts || {};
  lineChart('chart-views', d.campaign_views);
  lineChart('chart-clicks', d.link_clicks);
})();
