import { lineChart } from '../charts.js';

(() => {
  const d = window._dashboardCharts || {};
  lineChart('chart-views', d.campaign_views);
  lineChart('chart-clicks', d.link_clicks);
})();
