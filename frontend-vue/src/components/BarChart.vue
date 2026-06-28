<template>
  <section class="bar-chart">
    <canvas class="bar-chart-canvas" />
  </section>
</template>

<script>
import Chart from 'chart.js/auto';

const DEFAULT = {
  type: 'bar',
  data: {},
  options: {
    responsive: true,
    indexAxis: 'y',
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        backgroundColor: '#fff',
        borderColor: '#ddd',
        borderWidth: 1,
        titleColor: '#666',
        bodyColor: '#666',
        bodyFont: {
          size: 15,
        },
        bodySpacing: 10,
        padding: 10,
      },
    },
  },
};

export default {
  name: 'BarChart',

  props: {
    data: { type: Object, default: () => { } },
  },

  mounted() {
    const ctx = this.$el.querySelector('.bar-chart-canvas');
    this.chart = new Chart(ctx, { ...DEFAULT, data: this.$props.data });
  },
};
</script>
