<template>
  <section class="dashboard content">
    <header class="columns">
      <div class="column is-two-thirds">
        <h1 class="title is-5">
          {{ $utils.niceDate(new Date()) }}
        </h1>
      </div>
    </header>

    <section class="counts wrap">
      <div class="tile is-ancestor">
        <div class="tile is-vertical is-12">
          <div class="tile">
            <div class="tile is-parent is-vertical relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="lists">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="format-list-bulleted-square" />
                      {{ $utils.niceNumber(counts.lists.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.list', counts.lists.total) }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.public) }}</label>
                        {{ $t('lists.types.public') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.private) }}</label>
                        {{ $t('lists.types.private') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.optinSingle) }}</label>
                        {{ $t('lists.optins.single') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.lists.optinDouble) }}</label>
                        {{ $t('lists.optins.double') }}
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- lists -->

              <article class="tile is-child notification" data-cy="campaigns">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="rocket-launch-outline" />
                      {{ $utils.niceNumber(counts.campaigns.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.campaign', counts.campaigns.total) }}
                    </p>
                  </div>
                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li v-for="(num, status) in counts.campaigns.byStatus" :key="status">
                        <label for="#" :data-cy="`campaigns-${status}`">{{ num }}</label>
                        {{ $t(`campaigns.status.${status}`) }}
                        <span v-if="status === 'running'" class="spinner is-tiny">
                          <b-loading :is-full-page="false" active />
                        </span>
                      </li>
                    </ul>
                  </div>
                </div>
              </article><!-- campaigns -->
            </div><!-- block -->

            <div class="tile is-parent relative">
              <b-loading v-if="isCountsLoading" active :is-full-page="false" />
              <article class="tile is-child notification" data-cy="subscribers">
                <div class="columns is-mobile">
                  <div class="column is-6">
                    <p class="title">
                      <b-icon icon="account-multiple" />
                      {{ $utils.niceNumber(counts.subscribers.total) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $tc('globals.terms.subscriber', counts.subscribers.total) }}
                    </p>
                  </div>

                  <div class="column is-6">
                    <ul class="no has-text-grey">
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.subscribers.blocklisted) }}</label>
                        {{ $t('subscribers.status.blocklisted') }}
                      </li>
                      <li>
                        <label for="#">{{ $utils.niceNumber(counts.subscribers.orphans) }}</label>
                        {{ $t('dashboard.orphanSubs') }}
                      </li>
                    </ul>
                  </div><!-- subscriber breakdown -->
                </div><!-- subscriber columns -->
                <hr />
                <div class="columns" data-cy="messages">
                  <div class="column is-12">
                    <p class="title">
                      <b-icon icon="email-outline" />
                      {{ $utils.niceNumber(counts.messages) }}
                    </p>
                    <p class="is-size-6 has-text-grey">
                      {{ $t('dashboard.messagesSent') }}
                    </p>
                  </div>
                </div>
              </article><!-- subscribers -->
            </div>
          </div>
          <div class="tile is-parent relative">
            <b-loading v-if="isChartsLoading" active :is-full-page="false" />
            <article class="tile is-child notification charts">
              <div class="columns">
                <div class="column is-6">
                  <h3 class="title is-size-6">
                    {{ $t('dashboard.campaignViews') }}
                  </h3><br />
                  <chart type="line" v-if="campaignViews" :data="campaignViews" />
                </div>
                <div class="column is-6">
                  <h3 class="title is-size-6 has-text-right">
                    {{ $t('dashboard.linkClicks') }}
                  </h3><br />
                  <chart type="line" v-if="campaignClicks" :data="campaignClicks" />
                </div>
              </div>
            </article>
          </div>
        </div>
      </div><!-- tile block -->
      <p v-if="settings['app.cache_slow_queries']" class="has-text-grey">
        *{{ $t('globals.messages.slowQueriesCached') }}
        <a href="https://listmonk.app/maintenance/performance/" target="_blank" rel="noopener noreferer"
          class="has-text-grey">
          <b-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
        </a>
      </p>
    </section>
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
        campaigns: {},
        messages: 0,
      },
    };
  },

  methods: {
    makeChart(data) {
      if (data.length === 0) {
        return {};
      }
      return {
        labels: data.map((d) => dayjs(d.date).format('DD MMM')),
        datasets: [
          {
            data: [...data.map((d) => d.count)],
            borderColor: colors.primary,
            borderWidth: 2,
            pointHoverBorderWidth: 5,
            pointBorderWidth: 0.5,
          },
        ],
      };
    },
  },

  computed: {
    ...mapState(['settings']),
    dayjs() {
      return dayjs;
    },
  },

  mounted() {
    // Pull the counts.
    this.$api.getDashboardCounts().then((data) => {
      this.counts = data;
      this.isCountsLoading = false;
    });

    // Pull the charts.
    this.$api.getDashboardCharts().then((data) => {
      this.isChartsLoading = false;
      this.campaignViews = this.makeChart(data.campaignViews);
      this.campaignClicks = this.makeChart(data.linkClicks);
    });
  },
});
</script>
