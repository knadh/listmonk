<template>
  <section class="dashboard">
    <header class="hstack justify-between mb-6">
      <div class="hstack">
        <h1>{{ $utils.niceDate(new Date()) }}</h1>
        <span class="badge success">{{ $t('menu.dashboard') }}</span>
      </div>
      <button type="button" class="outline small" @click="fetchData">
        <oat-icon icon="refresh" />
        {{ $t('globals.buttons.refresh') }}
      </button>
    </header>

    <section class="row">
      <div class="col-3">
        <article class="card stat-card" :aria-busy="isCountsLoading ? 'true' : 'false'" data-cy="lists">
          <header>
            <small class="text-light">{{ $tc('globals.terms.list', 2) }}</small>
            <div class="stat-value">
              <oat-icon icon="format-list-bulleted-square" />
              {{ nice(counts.lists.total) }}
            </div>
            <small class="text-light hstack gap-2">
              <span class="badge public">{{ nice(counts.lists.public) }} {{ $t('lists.types.public') }}</span>
              <span class="badge private">{{ nice(counts.lists.private) }} {{ $t('lists.types.private') }}</span>
            </small>
          </header>
        </article>
      </div>

      <div class="col-3">
        <article class="card stat-card" :aria-busy="isCountsLoading ? 'true' : 'false'" data-cy="subscribers">
          <header>
            <small class="text-light">{{ $tc('globals.terms.subscriber', 2) }}</small>
            <div class="stat-value">
              <oat-icon icon="account-multiple" />
              {{ nice(counts.subscribers.total) }}
            </div>
            <small class="text-light hstack gap-2">
              <span class="badge blocklisted">{{ nice(counts.subscribers.blocklisted) }} {{ $t('subscribers.status.blocklisted') }}</span>
              <span class="badge outline">{{ nice(counts.subscribers.orphans) }} {{ $t('dashboard.orphanSubs') }}</span>
            </small>
          </header>
        </article>
      </div>

      <div class="col-3">
        <article class="card stat-card" :aria-busy="isCountsLoading ? 'true' : 'false'" data-cy="campaigns">
          <header>
            <small class="text-light">{{ $tc('globals.terms.campaign', 2) }}</small>
            <div class="stat-value">
              <oat-icon icon="rocket-launch-outline" />
              {{ nice(counts.campaigns.total) }}
            </div>
            <small class="text-light hstack gap-2">
              <span v-for="status in primaryCampaignStatuses" :key="status" :class="['badge', status]">
                {{ statusCount(status) }} {{ $t(`campaigns.status.${status}`) }}
              </span>
            </small>
          </header>
        </article>
      </div>

      <div class="col-3">
        <article class="card stat-card" :aria-busy="isCountsLoading ? 'true' : 'false'" data-cy="messages">
          <header>
            <small class="text-light">{{ $t('dashboard.messagesSent') }}</small>
            <div class="stat-value">
              <oat-icon icon="email-outline" />
              {{ nice(counts.messages) }}
            </div>
            <small class="text-light">
              <span class="badge">{{ nice(messagesPerSubscriber) }} / {{ $tc('globals.terms.subscriber', 1) }}</span>
            </small>
          </header>
        </article>
      </div>
    </section>

    <section class="row mt-6">
      <div class="col-6">
        <article class="card chart-card" :aria-busy="isChartsLoading ? 'true' : 'false'" data-spinner="overlay">
          <header class="hstack justify-between">
            <div>
              <h3>{{ $t('dashboard.campaignViews') }}</h3>
              <small class="text-light">{{ nice(chartTotal(campaignViews)) }} total</small>
            </div>
            <span class="badge success">{{ $t('globals.terms.analytics') }}</span>
          </header>
          <chart type="line" v-if="campaignViews" :data="campaignViews" />
        </article>
      </div>

      <div class="col-6">
        <article class="card chart-card" :aria-busy="isChartsLoading ? 'true' : 'false'" data-spinner="overlay">
          <header class="hstack justify-between">
            <div>
              <h3>{{ $t('dashboard.linkClicks') }}</h3>
              <small class="text-light">{{ nice(chartTotal(campaignClicks)) }} total</small>
            </div>
            <span class="badge warning">{{ $t('analytics.links') }}</span>
          </header>
          <chart type="line" v-if="campaignClicks" :data="campaignClicks" />
        </article>
      </div>
    </section>

    <section class="row mt-6">
      <div class="col-6">
        <article class="card">
          <header>
            <h3>{{ $tc('globals.terms.campaign', 2) }}</h3>
            <p class="text-light text-7">{{ $t('globals.fields.status') }}</p>
          </header>
          <div class="table">
            <table>
              <thead>
                <tr>
                  <th>{{ $t('globals.fields.status') }}</th>
                  <th class="align-right">{{ $t('analytics.count') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(num, status) in campaignStatuses" :key="status">
                  <td>
                    <span :class="['badge', status]">
                      {{ $t(`campaigns.status.${status}`) }}
                    </span>
                  </td>
                  <td class="align-right">{{ nice(num) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>
      </div>

      <div class="col-6">
        <article class="card">
          <header>
            <h3>{{ $tc('globals.terms.list', 2) }}</h3>
            <p class="text-light text-7">{{ $t('globals.fields.type') }} / {{ $t('lists.optin') }}</p>
          </header>
          <div class="dashboard-mix">
            <div>
              <div class="hstack justify-between">
                <strong>{{ $t('lists.types.public') }}</strong>
                <span class="badge public">{{ nice(counts.lists.public) }}</span>
              </div>
            </div>
            <div>
              <div class="hstack justify-between">
                <strong>{{ $t('lists.types.private') }}</strong>
                <span class="badge private">{{ nice(counts.lists.private) }}</span>
              </div>
            </div>
            <div>
              <div class="hstack justify-between">
                <strong>{{ $t('lists.optins.single') }}</strong>
                <span class="badge single">{{ nice(counts.lists.optinSingle) }}</span>
              </div>
            </div>
            <div>
              <div class="hstack justify-between">
                <strong>{{ $t('lists.optins.double') }}</strong>
                <span class="badge double">{{ nice(counts.lists.optinDouble) }}</span>
              </div>
            </div>
          </div>
        </article>
      </div>
    </section>

    <div v-if="settings['app.cache_slow_queries']" role="alert" class="mt-6">
      *{{ $t('globals.messages.slowQueriesCached') }}
      <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer">
        <oat-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
      </a>
    </div>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';
import { colors } from '../constants';
import Chart from '../components/Chart.vue';

export default Vue.extend({
  components: {
    Chart,
  },

  data() {
    return {
      isChartsLoading: true,
      isCountsLoading: true,
      campaignViews: null,
      campaignClicks: null,
      counts: {
        lists: {},
        subscribers: {},
        campaigns: {
          byStatus: {},
        },
        messages: 0,
      },
    };
  },

  methods: {
    fetchData() {
      this.isCountsLoading = true;
      this.isChartsLoading = true;

      this.$api.getDashboardCounts().then((data) => {
        this.counts = {
          ...data,
          campaigns: {
            ...data.campaigns,
            byStatus: data.campaigns.byStatus || {},
          },
        };
        this.isCountsLoading = false;
      });

      this.$api.getDashboardCharts().then((data) => {
        this.isChartsLoading = false;
        this.campaignViews = this.makeChart(data.campaignViews);
        this.campaignClicks = this.makeChart(data.linkClicks);
      });
    },

    makeChart(data) {
      if (data.length === 0) {
        return { labels: [], datasets: [{ data: [] }] };
      }
      return {
        labels: data.map((d) => dayjs(d.date).format('DD MMM')),
        datasets: [
          {
            data: data.map((d) => d.count),
            borderColor: colors.primary,
            backgroundColor: `${colors.primary}22`,
            borderWidth: 2,
            fill: true,
            tension: 0.35,
            pointHoverBorderWidth: 5,
            pointBorderWidth: 0.5,
          },
        ],
      };
    },

    nice(n) {
      return this.$utils.niceNumber(n || 0);
    },

    statusCount(status) {
      return this.nice(this.counts.campaigns.byStatus?.[status]);
    },

    chartTotal(chart) {
      return chart?.datasets?.[0]?.data?.reduce((sum, n) => sum + n, 0) || 0;
    },
  },

  computed: {
    ...mapState(['settings']),

    messagesPerSubscriber() {
      if (!this.counts.subscribers.total) {
        return 0;
      }
      return Math.round((this.counts.messages || 0) / this.counts.subscribers.total);
    },

    campaignStatuses() {
      return this.counts.campaigns.byStatus || {};
    },

    primaryCampaignStatuses() {
      return Object.keys(this.campaignStatuses).slice(0, 3);
    },
  },

  created() {
    this.$root.$on('page.refresh', this.fetchData);
  },

  destroyed() {
    this.$root.$off('page.refresh', this.fetchData);
  },

  mounted() {
    this.fetchData();
  },
});
</script>

<style>
.dashboard > header h1 {
  margin: 0;
}

.stat-card header {
  display: grid;
  gap: var(--space-2);
}

.stat-value {
  align-items: center;
  display: flex;
  gap: var(--space-2);
  font-size: var(--text-2);
  font-weight: var(--font-semibold);
  line-height: 1.1;
}

.chart-card {
  min-height: 24rem;
}

.chart-card .chart {
  height: 18rem;
}

.dashboard-mix {
  display: grid;
  gap: var(--space-4);
}
</style>
